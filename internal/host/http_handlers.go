package host

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

func (h *Host) ExecuteRPC(
	ctx context.Context,
	pluginID string,
	methodName string,
	caller CallerContext,
	args map[string]any,
) (*RPCResponse, error) {
	start := time.Now()

	if strings.TrimSpace(pluginID) == "" {
		return nil, errors.New("missing mandatory parameter: pluginID")
	}
	if strings.TrimSpace(methodName) == "" {
		return nil, errors.New("missing mandatory parameter: methodName")
	}
	if strings.TrimSpace(caller.UserID) == "" {
		return nil, errors.New("missing mandatory caller UserID")
	}

	plugin, exists := h.GetLoadedPlugin(pluginID)
	if !exists {
		h.logError(fmt.Sprintf("Plugin not found: %s", pluginID))
		return nil, fmt.Errorf("plugin not loaded: %s", pluginID)
	}
	pluginID = plugin.Manifest.ID
	if plugin.State == PluginStateUninstalledDormant {
		h.recordDiagnostic(pluginID, methodName, "PLUGIN_DORMANT_QUARANTINE", time.Since(start))
		h.logError(fmt.Sprintf("Blocked RPC %s.%s because plugin is in UNINSTALLED_DORMANT quarantine state", pluginID, methodName))
		return nil, fmt.Errorf("PLUGIN_DORMANT_QUARANTINE: plugin %s is uninstalled and in dormant GDPR retention quarantine", pluginID)
	}

	endpointSpec, declared := plugin.Manifest.Endpoints[methodName]
	if !declared && plugin.Manifest.GDPRHooks != nil {
		if methodName == strings.TrimSpace(plugin.Manifest.GDPRHooks.ExportPupilData) && methodName != "" {
			endpointSpec = EndpointSpec{
				Scopes:    []string{"dal.pupils.read:all"},
				Mutates:   false,
				TimeoutMS: 2500,
			}
			declared = true
		} else if methodName == strings.TrimSpace(plugin.Manifest.GDPRHooks.PurgePupilData) && methodName != "" {
			endpointSpec = EndpointSpec{
				Scopes:    []string{"dal.pupils.read:all"},
				Mutates:   true,
				TimeoutMS: 2500,
			}
			declared = true
		}
	}
	if !declared {
		h.recordDiagnostic(pluginID, methodName, "UNDECLARED_ENDPOINT", time.Since(start))
		h.logError(fmt.Sprintf("Default-deny blocked undeclared endpoint %s on %s", methodName, pluginID))
		return nil, fmt.Errorf("endpoint %q is not declared in manifest.json for plugin %s", methodName, pluginID)
	}

	if (!hasAllScopes(caller.Scopes, endpointSpec.Scopes) || !hasRequiredEndpointRoles(caller, endpointSpec.RequireRoles)) && !isCallerAdminOrDPO(caller, false) {
		h.recordDiagnostic(pluginID, methodName, "UNAUTHORIZED_SCOPE", time.Since(start))
		h.logError(fmt.Sprintf("Unauthorized scope for caller %s on %s.%s", caller.UserID, pluginID, methodName))
		return nil, fmt.Errorf("caller lacks required DAL scopes for %s.%s", pluginID, methodName)
	}

	timeoutMS := endpointSpec.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 2500
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	L := newSandboxedLuaState(execCtx)
	defer L.Close()

	var bufferedEvents []Event
	var writeTx *sql.Tx
	lockTracker := &sqlLockTracker{}
	argSecrets := collectArgSecretStrings(args)
	defer func() {
		if writeTx != nil {
			_ = writeTx.Rollback()
			lockTracker.finishHold()
		}
	}()

	ctxTable := h.buildLuaCtxTable(L, execCtx, plugin, endpointSpec, caller, &bufferedEvents, &writeTx, lockTracker, &argSecrets)
	argsTable := goMapToLuaTable(L, args)

	if err := L.DoString(plugin.LuaSource); err != nil {
		h.recordDiagnosticDetailed(pluginID, methodName, "LUA_LOAD_ERROR", err, time.Since(start), lockTracker, argSecrets)
		h.logError(fmt.Sprintf("Lua load error in %s: %v", pluginID, err))
		return nil, fmt.Errorf("failed to load plugin Lua script: %w", err)
	}

	fn := L.GetGlobal(methodName)
	if fn.Type() != lua.LTFunction {
		h.recordDiagnosticDetailed(pluginID, methodName, "MISSING_LUA_FUNCTION", nil, time.Since(start), lockTracker, argSecrets)
		return nil, fmt.Errorf("declared endpoint %q is not a function in init.lua", methodName)
	}

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, ctxTable, argsTable); err != nil {
		h.recordDiagnosticDetailed(pluginID, methodName, "LUA_RUNTIME_ERROR", err, time.Since(start), lockTracker, argSecrets)
		h.logError(fmt.Sprintf("Lua runtime error in %s.%s", pluginID, methodName))
		return nil, fmt.Errorf("lua execution error in %s.%s: %w", pluginID, methodName, err)
	}

	ret := L.Get(-1)
	L.Pop(1)

	retTbl, ok := ret.(*lua.LTable)
	if !ok {
		h.recordDiagnosticDetailed(pluginID, methodName, "INVALID_RETURN_TYPE", nil, time.Since(start), lockTracker, argSecrets)
		return nil, fmt.Errorf("endpoint %s.%s must return a table", pluginID, methodName)
	}

	okVal := L.GetField(retTbl, "ok")
	if okVal != lua.LTrue {
		if writeTx != nil {
			_ = writeTx.Rollback()
			writeTx = nil
			lockTracker.finishHold()
		}
		var errCode string
		if ecVal := L.GetField(retTbl, "error_code"); ecVal != lua.LNil {
			errCode = strings.TrimSpace(ecVal.String())
		}
		var errMsg string
		if emVal := L.GetField(retTbl, "error"); emVal != lua.LNil {
			errMsg = strings.TrimSpace(emVal.String())
		} else if emVal := L.GetField(retTbl, "message"); emVal != lua.LNil {
			errMsg = strings.TrimSpace(emVal.String())
		}
		if errCode == "" {
			if errMsg != "" {
				errCode = errMsg
			} else {
				errCode = "PLUGIN_RPC_FAILED"
			}
		}
		h.emitStructuredStdout(fmt.Sprintf(
			"type=rpc plugin=%s method=%s caller=%s status=error error_code=%s duration_ms=%d sqlite_write_hold_ms=%d",
			pluginID,
			methodName,
			caller.StaffID,
			errCode,
			time.Since(start).Milliseconds(),
			lockTracker.writeHoldDuration.Milliseconds(),
		))
		return &RPCResponse{
			OK:        false,
			ErrorCode: errCode,
			Error:     errMsg,
		}, nil
	}

	dataField := L.GetField(retTbl, "data")
	if dataTbl, isTbl := dataField.(*lua.LTable); isTbl && writeTx == nil {
		if dataTbl.RawGetString("__unhandled_egress_timeout") == lua.LTrue {
			errMsg := dataTbl.RawGetString("error").String()
			if errMsg == "" {
				errMsg = "EGRESS_TIMEOUT"
			}
			return nil, fmt.Errorf("lua execution error in %s.%s: %s", pluginID, methodName, errMsg)
		}
	}

	if writeTx != nil {
		commitErr := writeTx.Commit()
		writeTx = nil
		lockTracker.finishHold()
		if commitErr != nil {
			h.recordDiagnosticDetailed(pluginID, methodName, "SQLITE_COMMIT_ERROR", commitErr, time.Since(start), lockTracker, argSecrets)
			return nil, fmt.Errorf("failed to commit plugin_db write transaction: %w", commitErr)
		}
	}

	dataVal := luaValueToGo(dataField)

	// Post-commit asynchronous Event Bus dispatch
	if len(bufferedEvents) > 0 {
		for _, evt := range bufferedEvents {
			h.emitStructuredStdout(fmt.Sprintf(
				"type=event_emit plugin=%s event=%s hop=%d",
				pluginID,
				evt.Name,
				evt.Hop,
			))
		}
		h.dispatchEvents(bufferedEvents)
	}

	h.emitStructuredStdout(fmt.Sprintf(
		"type=rpc plugin=%s method=%s caller=%s status=ok duration_ms=%d sqlite_write_hold_ms=%d",
		pluginID,
		methodName,
		caller.StaffID,
		time.Since(start).Milliseconds(),
		lockTracker.writeHoldDuration.Milliseconds(),
	))
	h.logInfo(fmt.Sprintf("Executed RPC %s.%s in %dms", pluginID, methodName, time.Since(start).Milliseconds()))
	return &RPCResponse{
		OK:      true,
		Data:    dataVal,
		Mutated: endpointSpec.Mutates,
	}, nil
}

func hasRequiredEndpointRoles(caller CallerContext, requireRoles []string) bool {
	if len(requireRoles) == 0 {
		return true
	}
	for _, reqRole := range requireRoles {
		target := strings.ToUpper(strings.TrimSpace(reqRole))
		if target == "" {
			continue
		}
		for _, active := range caller.ActiveRoles {
			if strings.ToUpper(strings.TrimSpace(active)) == target {
				return true
			}
		}
	}
	return false
}

func hasAllScopes(callerScopes []string, requiredScopes []string) bool {
	set := make(map[string]struct{}, len(callerScopes))
	for _, s := range callerScopes {
		set[s] = struct{}{}
	}
	for _, req := range requiredScopes {
		if _, ok := set[req]; ok {
			continue
		}
		if strings.HasSuffix(req, ":own_groups") {
			allScope := strings.TrimSuffix(req, ":own_groups") + ":all"
			if _, ok := set[allScope]; ok {
				continue
			}
		}
		return false
	}
	return true
}

func hasDomainScope(scopes []string, prefix string) bool {
	for _, s := range scopes {
		if s == prefix || strings.HasPrefix(s, prefix+":") {
			return true
		}
	}
	return false
}

func (h *Host) registerHostLogAndFixtureRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/test/logs/cursor", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"cursor": h.LogCursor(),
		})
	})

	mux.HandleFunc("GET /api/test/logs", func(w http.ResponseWriter, r *http.Request) {
		since := strings.TrimSpace(r.URL.Query().Get("since"))
		if since == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": "missing mandatory query parameter: since",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"entries": h.LogsSince(since),
		})
	})

	mux.HandleFunc("POST /api/test/seed-pupil", func(w http.ResponseWriter, r *http.Request) {
		var in dal.SeedPupilInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		pupil, err := h.coreDAL.SeedPupil(r.Context(), in)
		if err != nil {
			h.logError(fmt.Sprintf("Failed to seed pupil: %v", err))
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.logInfo(fmt.Sprintf("Seeded pupil record %s", pupil.ID))
		writeJSON(w, http.StatusCreated, map[string]any{
			"pupil_id": pupil.ID,
			"pupil":    pupil,
		})
	})

	mux.HandleFunc("DELETE /api/test/seed-pupil/{pupil_id}", func(w http.ResponseWriter, r *http.Request) {
		pupilID := strings.TrimSpace(r.PathValue("pupil_id"))
		if pupilID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": "missing mandatory path parameter: pupil_id",
			})
			return
		}
		if err := h.coreDAL.DeletePupil(r.Context(), pupilID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.logInfo(fmt.Sprintf("Deleted seeded pupil record %s", pupilID))
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": true,
		})
	})

	mux.HandleFunc("POST /api/test/seed-milestone1-fixtures", func(w http.ResponseWriter, r *http.Request) {
		var in dal.Milestone1SeedInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		seeded, err := h.coreDAL.SeedMilestone1Fixtures(r.Context(), in)
		if err != nil {
			h.logError(fmt.Sprintf("Failed to seed Milestone 1 fixtures: %v", err))
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.logInfo(fmt.Sprintf("Seeded Milestone 1 fixtures %s", seeded.FixtureID))
		writeJSON(w, http.StatusCreated, seeded)
	})

	mux.HandleFunc("DELETE /api/test/seed-milestone1-fixtures/{fixture_id}", func(w http.ResponseWriter, r *http.Request) {
		fixtureID := strings.TrimSpace(r.PathValue("fixture_id"))
		if fixtureID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": "missing mandatory path parameter: fixture_id",
			})
			return
		}
		if err := h.coreDAL.DeleteMilestone1Fixtures(r.Context(), fixtureID); err != nil {
			h.logError(fmt.Sprintf("Failed to delete Milestone 1 fixtures %s: %v", fixtureID, err))
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.logInfo(fmt.Sprintf("Deleted Milestone 1 fixtures %s", fixtureID))
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": true,
		})
	})
}

func (h *Host) handlePluginRPCRequest(w http.ResponseWriter, r *http.Request) {
	plugID := strings.TrimSpace(r.PathValue("plug_id"))
	method := strings.TrimSpace(r.PathValue("method"))
	if plugID == "" || method == "" {
		h.logError("RPC rejected: missing plug_id or method")
		writeJSON(w, http.StatusBadRequest, RPCResponse{
			OK:        false,
			ErrorCode: "MISSING_PLUGIN_OR_METHOD",
		})
		return
	}

	var body struct {
		Args map[string]any `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.logError(fmt.Sprintf("RPC rejected for %s.%s: invalid JSON payload: %v", plugID, method, err))
		writeJSON(w, http.StatusBadRequest, RPCResponse{
			OK:        false,
			ErrorCode: "INVALID_JSON_PAYLOAD",
		})
		return
	}
	if body.Args == nil {
		body.Args = map[string]any{}
	}

	caller, callerErr := h.ResolveRequestCaller(r, "")
	if callerErr != nil {
		h.logError(fmt.Sprintf("RPC caller resolution failed for %s.%s: %v", plugID, method, callerErr))
		writeJSON(w, http.StatusBadRequest, RPCResponse{
			OK:        false,
			ErrorCode: callerErr.Error(),
		})
		return
	}

	plugin, exists := h.GetLoadedPluginByPlugID(plugID)
	if !exists {
		h.logError(fmt.Sprintf("Plugin not found for plug_id: %s", plugID))
		writeJSON(w, http.StatusBadRequest, RPCResponse{
			OK:        false,
			ErrorCode: fmt.Sprintf("plugin not loaded: %s", plugID),
		})
		return
	}

	res, err := h.ExecuteRPC(r.Context(), plugin.Manifest.ID, method, caller, body.Args)
	if err != nil {
		status := http.StatusBadRequest
		if plugin.Manifest.ID != "uk.org.optimis.test" && strings.Contains(err.Error(), "caller lacks required DAL scopes") {
			status = http.StatusForbidden
		}
		writeJSON(w, status, RPCResponse{
			OK:        false,
			ErrorCode: err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, res)
}
