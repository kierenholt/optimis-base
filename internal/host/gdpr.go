package host

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

type SARExportBundle struct {
	PupilID         string           `json:"pupil_id"`
	SchoolURN       string           `json:"school_urn"`
	GeneratedAt     string           `json:"generated_at"`
	GeneratedByUser string           `json:"generated_by_user"`
	Sections        []dal.SARSection `json:"sections"`
}

type SARFinalizeInput struct {
	PupilID  string           `json:"pupil_id"`
	Sections []dal.SARSection `json:"sections"`
}

type SARFinalizedPackage struct {
	PupilID           string           `json:"pupil_id"`
	SchoolURN         string           `json:"school_urn"`
	FinalizedAt       string           `json:"finalized_at"`
	RedactedByStaffID string           `json:"redacted_by_staff_id"`
	RedactedByUserID  string           `json:"redacted_by_user_id"`
	Status            string           `json:"status"`
	Sections          []dal.SARSection `json:"sections"`
}

type RetentionPurgeResult struct {
	PupilID          string         `json:"pupil_id"`
	PurgedAt         string         `json:"purged_at"`
	PurgedByStaffID  string         `json:"purged_by_staff_id"`
	CorePurged       bool           `json:"core_purged"`
	PluginPurgedRows map[string]int `json:"plugin_purged_rows"`
	Status           string         `json:"status"`
}

func isCallerAdminOrDPO(caller CallerContext, requireWrite bool) bool {
	for _, r := range caller.ActiveRoles {
		if strings.EqualFold(r, "ADMIN") || strings.EqualFold(r, "DPO") {
			return true
		}
	}
	if requireWrite {
		return hasAllScopes(caller.Scopes, []string{"dal.pupils.write:all", "dal.staff.write:admin"})
	}
	return hasAllScopes(caller.Scopes, []string{"dal.pupils.read:all", "dal.staff.write:admin"})
}

func (h *Host) SetPluginState(pluginID string, state PluginState) error {
	trimmedID := strings.TrimSpace(pluginID)
	if trimmedID == "" {
		return errors.New("missing mandatory parameter: pluginID")
	}
	if state != PluginStateActive && state != PluginStateUninstalledDormant {
		return fmt.Errorf("invalid plugin state %q: must be ACTIVE or UNINSTALLED_DORMANT", state)
	}

	h.mu.Lock()
	p, exists := h.plugins[trimmedID]
	if !exists {
		h.mu.Unlock()
		return fmt.Errorf("plugin not found: %s", trimmedID)
	}
	p.State = state
	var poolToStop *pluginEventWorkerPool
	if state == PluginStateUninstalledDormant {
		poolToStop = h.workerPools[trimmedID]
		delete(h.workerPools, trimmedID)
	}
	h.serviceLogs = append(
		h.serviceLogs,
		fmt.Sprintf("[%s] INFO Plugin %s state transitioned to %s", time.Now().UTC().Format(time.RFC3339Nano), trimmedID, state),
	)
	h.mu.Unlock()

	if poolToStop != nil {
		poolToStop.stop()
	} else if state == PluginStateActive {
		h.startOrReplacePluginWorkerPool(p)
	}
	return nil
}

func (h *Host) UninstallPlugin(pluginID string) error {
	return h.SetPluginState(pluginID, PluginStateUninstalledDormant)
}

func (h *Host) listAttachedPluginsSorted() []*LoadedPlugin {
	h.mu.RLock()
	defer h.mu.RUnlock()
	list := make([]*LoadedPlugin, 0, len(h.plugins))
	for _, p := range h.plugins {
		if p.Manifest.Database != nil && strings.TrimSpace(p.Manifest.Database.Alias) != "" {
			list = append(list, p)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Manifest.ID < list[j].Manifest.ID
	})
	return list
}

func (h *Host) invokePluginGDPRHook(
	ctx context.Context,
	plugin *LoadedPlugin,
	hookFnName string,
	mutates bool,
	caller CallerContext,
	pupilID string,
) (any, error) {
	start := time.Now()
	execCtx, cancel := context.WithTimeout(ctx, 5000*time.Millisecond)
	defer cancel()

	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer L.Close()
	for _, lib := range []struct {
		name string
		fn   lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenPackage},
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		L.Push(L.NewFunction(lib.fn))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	L.SetGlobal("dofile", lua.LNil)
	L.SetGlobal("loadfile", lua.LNil)
	L.SetGlobal("package", lua.LNil)
	L.SetGlobal("require", lua.LNil)
	L.SetGlobal("os", lua.LNil)
	L.SetGlobal("io", lua.LNil)
	L.SetGlobal("debug", lua.LNil)
	L.SetContext(execCtx)

	var bufferedEvents []Event
	var writeTx *sql.Tx
	defer func() {
		if writeTx != nil {
			_ = writeTx.Rollback()
		}
	}()

	hookSpec := EndpointSpec{
		Scopes:    []string{"dal.pupils.read:all", "dal.census.read:all"},
		Mutates:   mutates,
		TimeoutMS: 5000,
	}

	ctxTable := h.buildLuaCtxTable(L, execCtx, plugin, hookSpec, caller, &bufferedEvents, &writeTx)
	argsTable := goMapToLuaTable(L, map[string]any{
		"pupil_id": pupilID,
	})

	if err := L.DoString(plugin.LuaSource); err != nil {
		h.recordDiagnostic(plugin.Manifest.ID, hookFnName, "LUA_LOAD_ERROR", time.Since(start))
		return nil, fmt.Errorf("failed to load Lua source for GDPR hook %s.%s: %w", plugin.Manifest.ID, hookFnName, err)
	}

	fn := L.GetGlobal(hookFnName)
	if fn.Type() != lua.LTFunction {
		h.recordDiagnostic(plugin.Manifest.ID, hookFnName, "MISSING_GDPR_HOOK", time.Since(start))
		return nil, fmt.Errorf("declared gdpr_hook %q is not a function in %s init.lua", hookFnName, plugin.Manifest.ID)
	}

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, ctxTable, argsTable); err != nil {
		h.recordDiagnostic(plugin.Manifest.ID, hookFnName, "LUA_GDPR_HOOK_ERROR", time.Since(start))
		return nil, fmt.Errorf("GDPR hook %s.%s execution failed: %w", plugin.Manifest.ID, hookFnName, err)
	}

	ret := L.Get(-1)
	L.Pop(1)

	retTbl, ok := ret.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("GDPR hook %s.%s must return a table", plugin.Manifest.ID, hookFnName)
	}
	if L.GetField(retTbl, "ok") != lua.LTrue {
		if writeTx != nil {
			_ = writeTx.Rollback()
			writeTx = nil
		}
		errCode := L.GetField(retTbl, "error_code").String()
		return nil, fmt.Errorf("GDPR hook %s.%s returned ok=false (%s)", plugin.Manifest.ID, hookFnName, errCode)
	}

	if writeTx != nil {
		if err := writeTx.Commit(); err != nil {
			writeTx = nil
			return nil, fmt.Errorf("failed to commit GDPR hook %s.%s write transaction: %w", plugin.Manifest.ID, hookFnName, err)
		}
		writeTx = nil
	}

	return luaValueToGo(L.GetField(retTbl, "data")), nil
}

func parsePluginSARSections(plugin *LoadedPlugin, rawData any) ([]dal.SARSection, error) {
	rawArr, ok := rawData.([]any)
	if !ok {
		if m, mOk := rawData.(map[string]any); mOk {
			rawArr, ok = m["sections"].([]any)
		}
	}
	if !ok {
		return nil, fmt.Errorf("plugin %s export_pupil_data must return an array of section tables", plugin.Manifest.ID)
	}

	sections := make([]dal.SARSection, 0, len(rawArr))
	for _, item := range rawArr {
		secMap, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("plugin %s export_pupil_data section entry must be a table", plugin.Manifest.ID)
		}
		title, _ := secMap["title"].(string)
		if strings.TrimSpace(title) == "" {
			return nil, fmt.Errorf("plugin %s export_pupil_data section missing mandatory title", plugin.Manifest.ID)
		}

		records := make([]map[string]any, 0)
		if rawRecs, ok := secMap["records"].([]any); ok {
			for _, r := range rawRecs {
				if recMap, ok := r.(map[string]any); ok {
					records = append(records, recMap)
				}
			}
		}

		redactable := make([]string, 0)
		if rawRed, ok := secMap["redactable_fields"].([]any); ok {
			for _, f := range rawRed {
				if fStr, ok := f.(string); ok && strings.TrimSpace(fStr) != "" {
					redactable = append(redactable, strings.TrimSpace(fStr))
				}
			}
		}

		sections = append(sections, dal.SARSection{
			SourceID:         plugin.Manifest.ID,
			SourceType:       "PLUGIN",
			PluginState:      string(plugin.State),
			Title:            title,
			Records:          records,
			RedactableFields: redactable,
		})
	}
	return sections, nil
}

func (h *Host) ExportPupilSAR(ctx context.Context, caller CallerContext, pupilID string) (*SARExportBundle, error) {
	trimmedPupilID := strings.TrimSpace(pupilID)
	if trimmedPupilID == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if !isCallerAdminOrDPO(caller, false) {
		h.logError(fmt.Sprintf("Unauthorized SAR export attempt by caller %s (staff %s)", caller.UserID, caller.StaffID))
		return nil, errors.New("UNAUTHORIZED_ADMIN_GDPR_SCOPE: SAR export requires System Admin / DPO authority")
	}

	coreSections, err := h.coreDAL.ExportCorePupilSAR(ctx, trimmedPupilID)
	if err != nil {
		return nil, err
	}

	allSections := make([]dal.SARSection, 0, len(coreSections)+4)
	allSections = append(allSections, coreSections...)

	for _, plugin := range h.listAttachedPluginsSorted() {
		if plugin.Manifest.GDPRHooks == nil || strings.TrimSpace(plugin.Manifest.GDPRHooks.ExportPupilData) == "" {
			return nil, fmt.Errorf("plugin %s attaches a database but is missing gdpr_hooks.export_pupil_data", plugin.Manifest.ID)
		}
		rawExport, err := h.invokePluginGDPRHook(
			ctx,
			plugin,
			strings.TrimSpace(plugin.Manifest.GDPRHooks.ExportPupilData),
			false,
			caller,
			trimmedPupilID,
		)
		if err != nil {
			return nil, err
		}
		pluginSecs, err := parsePluginSARSections(plugin, rawExport)
		if err != nil {
			return nil, err
		}
		allSections = append(allSections, pluginSecs...)
	}

	h.logInfo(fmt.Sprintf("Exported UK GDPR SAR package for pupil (%d sections) by staff %s", len(allSections), caller.StaffID))
	return &SARExportBundle{
		PupilID:         trimmedPupilID,
		SchoolURN:       h.cfg.SchoolURN,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		GeneratedByUser: caller.UserID,
		Sections:        allSections,
	}, nil
}

func (h *Host) FinalizePupilSAR(ctx context.Context, caller CallerContext, in SARFinalizeInput) (*SARFinalizedPackage, error) {
	trimmedPupilID := strings.TrimSpace(in.PupilID)
	if trimmedPupilID == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if len(in.Sections) == 0 {
		return nil, errors.New("missing mandatory parameter: sections")
	}
	if !isCallerAdminOrDPO(caller, false) {
		return nil, errors.New("UNAUTHORIZED_ADMIN_GDPR_SCOPE: SAR redaction finalize requires System Admin / DPO authority")
	}

	// Verify pupil exists in core.db
	if _, err := h.coreDAL.ExportCorePupilSAR(ctx, trimmedPupilID); err != nil {
		return nil, err
	}

	h.logInfo(fmt.Sprintf("Finalized redacted UK GDPR SAR bundle for pupil by staff %s", caller.StaffID))
	return &SARFinalizedPackage{
		PupilID:           trimmedPupilID,
		SchoolURN:         h.cfg.SchoolURN,
		FinalizedAt:       time.Now().UTC().Format(time.RFC3339),
		RedactedByStaffID: caller.StaffID,
		RedactedByUserID:  caller.UserID,
		Status:            "SAR_EXPORT_FINALIZED",
		Sections:          in.Sections,
	}, nil
}

func (h *Host) PurgePupilData(ctx context.Context, caller CallerContext, pupilID string) (*RetentionPurgeResult, error) {
	trimmedPupilID := strings.TrimSpace(pupilID)
	if trimmedPupilID == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if !isCallerAdminOrDPO(caller, true) {
		h.logError(fmt.Sprintf("Unauthorized GDPR retention purge attempt by caller %s (staff %s)", caller.UserID, caller.StaffID))
		return nil, errors.New("UNAUTHORIZED_ADMIN_GDPR_SCOPE: statutory retention purge requires System Admin / DPO write authority")
	}

	// 1. Pre-flight check: enforce KCSIE safeguarding retention lock BEFORE touching any plugin or core table
	if err := h.coreDAL.CheckPupilRetentionPurgeEligible(ctx, trimmedPupilID); err != nil {
		h.logError(fmt.Sprintf("Blocked retention purge due to KCSIE retention guard: %v", err))
		return nil, err
	}

	// 2. Invoke gdpr_hooks.purge_pupil_data across all attached plugins (both ACTIVE and UNINSTALLED_DORMANT)
	pluginPurgedRows := make(map[string]int)
	for _, plugin := range h.listAttachedPluginsSorted() {
		if plugin.Manifest.GDPRHooks == nil || strings.TrimSpace(plugin.Manifest.GDPRHooks.PurgePupilData) == "" {
			return nil, fmt.Errorf("plugin %s attaches a database but is missing gdpr_hooks.purge_pupil_data", plugin.Manifest.ID)
		}
		rawPurge, err := h.invokePluginGDPRHook(
			ctx,
			plugin,
			strings.TrimSpace(plugin.Manifest.GDPRHooks.PurgePupilData),
			true,
			caller,
			trimmedPupilID,
		)
		if err != nil {
			return nil, err
		}
		purgedCount := 0
		if m, ok := rawPurge.(map[string]any); ok {
			if f, ok := m["purged_rows"].(float64); ok {
				purgedCount = int(f)
			}
		}
		pluginPurgedRows[plugin.Manifest.ID] = purgedCount
	}

	// 3. Purge pupil records across core.db
	if err := h.coreDAL.PurgeCorePupilData(ctx, trimmedPupilID); err != nil {
		return nil, err
	}

	h.logInfo(fmt.Sprintf("Completed statutory GDPR retention purge across Core and %d attached plugin(s) by staff %s", len(pluginPurgedRows), caller.StaffID))
	return &RetentionPurgeResult{
		PupilID:          trimmedPupilID,
		PurgedAt:         time.Now().UTC().Format(time.RFC3339),
		PurgedByStaffID:  caller.StaffID,
		CorePurged:       true,
		PluginPurgedRows: pluginPurgedRows,
		Status:           "RETENTION_PURGE_COMPLETED",
	}, nil
}

func (h *Host) RegisterGDPRRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/admin/gdpr/sar/export", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PupilID string `json:"pupil_id"`
			StaffID string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		caller, err := h.ResolveRequestCaller(r, body.StaffID)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		bundle, err := h.ExportPupilSAR(r.Context(), caller, body.PupilID)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "UNAUTHORIZED_ADMIN_GDPR_SCOPE") {
				status = http.StatusForbidden
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":     true,
			"bundle": bundle,
		})
	})

	mux.HandleFunc("POST /api/admin/gdpr/sar/finalize", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PupilID  string           `json:"pupil_id"`
			StaffID  string           `json:"staff_id"`
			Sections []dal.SARSection `json:"sections"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		caller, err := h.ResolveRequestCaller(r, body.StaffID)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		pkg, err := h.FinalizePupilSAR(r.Context(), caller, SARFinalizeInput{
			PupilID:  body.PupilID,
			Sections: body.Sections,
		})
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "UNAUTHORIZED_ADMIN_GDPR_SCOPE") {
				status = http.StatusForbidden
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"package": pkg,
		})
	})

	mux.HandleFunc("POST /api/admin/gdpr/purge", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PupilID string `json:"pupil_id"`
			StaffID string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		caller, err := h.ResolveRequestCaller(r, body.StaffID)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		res, err := h.PurgePupilData(r.Context(), caller, body.PupilID)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "UNAUTHORIZED_ADMIN_GDPR_SCOPE") {
				status = http.StatusForbidden
			} else if strings.Contains(err.Error(), "KCSIE_RETENTION_LOCK") {
				status = http.StatusConflict
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":     true,
			"result": res,
		})
	})

	mux.HandleFunc("POST /api/admin/plugins/{plugin_id}/state", func(w http.ResponseWriter, r *http.Request) {
		pluginID := strings.TrimSpace(r.PathValue("plugin_id"))
		if pluginID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory path parameter: plugin_id",
			})
			return
		}
		var body struct {
			State   string `json:"state"`
			StaffID string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		caller, err := h.ResolveRequestCaller(r, body.StaffID)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}
		if !isCallerAdminOrDPO(caller, true) {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":         false,
				"error_code": "UNAUTHORIZED_ADMIN_GDPR_SCOPE: changing plugin lifecycle state requires System Admin authority",
			})
			return
		}
		if err := h.SetPluginState(pluginID, PluginState(strings.TrimSpace(body.State))); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":        true,
			"plugin_id": pluginID,
			"state":     strings.TrimSpace(body.State),
		})
	})

	mux.HandleFunc("POST /api/test/seed-gdpr-fixtures", func(w http.ResponseWriter, r *http.Request) {
		var in dal.GDPRTestFixtureInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		in.HostKMSKeyHandle = h.cfg.HostKMSKeyHandle
		seeded, err := h.coreDAL.SeedGDPRTestFixtures(r.Context(), in)
		if err != nil {
			h.logError(fmt.Sprintf("Failed to seed GDPR fixtures: %v", err))
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.ResetDevPersonaSessions()
		h.logInfo(fmt.Sprintf("Seeded GDPR test fixtures %s", seeded.FixtureID))
		writeJSON(w, http.StatusCreated, seeded)
	})

	mux.HandleFunc("DELETE /api/test/seed-gdpr-fixtures/{fixture_id}", func(w http.ResponseWriter, r *http.Request) {
		fixtureID := strings.TrimSpace(r.PathValue("fixture_id"))
		if fixtureID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": "missing mandatory path parameter: fixture_id",
			})
			return
		}
		if err := h.coreDAL.DeleteGDPRTestFixtures(r.Context(), fixtureID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.ResetDevPersonaSessions()
		h.logInfo(fmt.Sprintf("Deleted GDPR test fixtures %s", fixtureID))
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": true,
		})
	})
}

