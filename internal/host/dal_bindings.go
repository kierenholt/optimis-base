package host

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

var piiScrubPattern = regexp.MustCompile(`(?i)\b(pup_[a-z0-9_]+|per_[a-z0-9_]+|[A-Z][0-9]{12})\b`)

func scrubDiagnosticToken(raw string) string {
	return piiScrubPattern.ReplaceAllString(raw, "[REDACTED]")
}

func callerHasAllRole(activeRoles []string) bool {
	for _, r := range activeRoles {
		switch strings.ToUpper(strings.TrimSpace(r)) {
		case "ADMIN", "HOD", "DSL", "SUPPORT", "LEADERSHIP", "DPO", "SYSTEM_ADMIN":
			return true
		}
	}
	return false
}

func resolveEffectiveDomainScopeMode(endpointScopes []string, caller CallerContext, domainPrefix string) dal.ScopeMode {
	for _, s := range endpointScopes {
		if s == domainPrefix+":all" {
			return dal.ScopeAll
		}
	}
	if caller.Type == "USER" {
		hasAll := false
		hasOwn := false
		for _, s := range caller.Scopes {
			if s == domainPrefix+":all" {
				hasAll = true
			}
			if s == domainPrefix+":own_groups" {
				hasOwn = true
			}
		}
		if hasAll && (!hasOwn || callerHasAllRole(caller.ActiveRoles)) {
			return dal.ScopeAll
		}
	}
	return dal.ScopeOwnGroups
}

func resolveDomainScopeMode(scopes []string, domainPrefix string) dal.ScopeMode {
	for _, s := range scopes {
		if s == domainPrefix+":all" {
			return dal.ScopeAll
		}
	}
	return dal.ScopeOwnGroups
}

func extractScopeFromLuaOpts(
	ctx context.Context,
	d *dal.CoreDAL,
	mode dal.ScopeMode,
	staffID string,
	optsTbl *lua.LTable,
) (dal.DALQueryScope, error) {
	scope := dal.DALQueryScope{
		Mode:    mode,
		StaffID: staffID,
	}
	if optsTbl != nil {
		if ay := optsTbl.RawGetString("academic_year_id"); ay != lua.LNil {
			scope.AcademicYearID = strings.TrimSpace(ay.String())
		}
		if rd := optsTbl.RawGetString("reference_date"); rd != lua.LNil {
			scope.ReferenceDate = strings.TrimSpace(rd.String())
		}
	}
	return d.ResolveScopeDefaults(ctx, scope)
}

func extractPupilIDsAndOptsFromLua(L *lua.LState) ([]string, *lua.LTable, error) {
	firstTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		return nil, nil, errors.New("requires a table of pupil_ids")
	}

	var optsTbl *lua.LTable
	if L.GetTop() >= 2 {
		optsTbl, _ = L.Get(2).(*lua.LTable)
	}

	idsSourceTbl := firstTbl
	if nested, isTbl := firstTbl.RawGetString("pupil_ids").(*lua.LTable); isTbl {
		idsSourceTbl = nested
		if optsTbl == nil {
			optsTbl = firstTbl
		}
	}

	var pupilIDs []string
	maxn := idsSourceTbl.MaxN()
	for i := 1; i <= maxn; i++ {
		val := strings.TrimSpace(idsSourceTbl.RawGetInt(i).String())
		if val != "" {
			pupilIDs = append(pupilIDs, val)
		}
	}
	return pupilIDs, optsTbl, nil
}

func luaTableTrimmedString(tbl *lua.LTable, key string) string {
	if tbl == nil {
		return ""
	}
	v := tbl.RawGetString(key)
	if v == lua.LNil {
		return ""
	}
	return strings.TrimSpace(v.String())
}

func staffToLuaTable(L *lua.LState, st dal.StaffSummary) *lua.LTable {
	sTbl := L.NewTable()
	L.SetField(sTbl, "id", lua.LString(st.StaffID))
	L.SetField(sTbl, "staff_id", lua.LString(st.StaffID))
	L.SetField(sTbl, "person_id", lua.LString(st.PersonID))
	L.SetField(sTbl, "staff_code", lua.LString(st.StaffCode))
	L.SetField(sTbl, "display_name", lua.LString(st.DisplayName))
	L.SetField(sTbl, "first_name", lua.LString(st.FirstName))
	L.SetField(sTbl, "last_name", lua.LString(st.LastName))
	L.SetField(sTbl, "job_title", lua.LString(st.JobTitle))
	L.SetField(sTbl, "staff_type", lua.LString(st.StaffType))
	L.SetField(sTbl, "is_dsl", lua.LBool(st.IsDSL))
	return sTbl
}

func (h *Host) bindLuaDALStaff(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec EndpointSpec,
	dalTbl *lua.LTable,
) {
	if !hasDomainScope(endpointSpec.Scopes, "dal.staff.read") {
		return
	}
	staffTbl := L.NewTable()
	L.SetField(staffTbl, "list_active", L.NewFunction(func(L *lua.LState) int {
		staffList, err := h.coreDAL.ListActiveStaff(execCtx)
		if err != nil {
			L.RaiseError("dal.staff.list_active failed: %s", err.Error())
			return 0
		}
		arrTbl := L.NewTable()
		markLuaTableAsArray(L, arrTbl)
		for i, st := range staffList {
			arrTbl.RawSetInt(i+1, staffToLuaTable(L, st))
		}
		L.Push(arrTbl)
		return 1
	}))
	L.SetField(staffTbl, "get_by_id", L.NewFunction(func(L *lua.LState) int {
		staffID := strings.TrimSpace(L.OptString(1, ""))
		if staffID == "" {
			L.RaiseError("missing mandatory parameter: staff_id")
			return 0
		}
		st, err := h.coreDAL.GetStaffByID(execCtx, staffID)
		if err != nil {
			L.RaiseError("dal.staff.get_by_id failed: %s", err.Error())
			return 0
		}
		L.Push(staffToLuaTable(L, *st))
		return 1
	}))
	L.SetField(dalTbl, "staff", staffTbl)
}

func (h *Host) buildLuaDALTable(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec EndpointSpec,
	caller CallerContext,
) *lua.LTable {
	dalTbl := L.NewTable()

	h.bindLuaDALPupils(L, execCtx, endpointSpec, caller, dalTbl)
	h.bindLuaDALGroups(L, execCtx, endpointSpec, caller, dalTbl)
	h.bindLuaDALStaff(L, execCtx, endpointSpec, dalTbl)
	h.bindLuaDALAttendance(L, execCtx, endpointSpec, caller, dalTbl)
	h.bindLuaDALGuardiansAndSafeguarding(L, execCtx, endpointSpec, caller, dalTbl)
	h.bindLuaDALCensus(L, execCtx, endpointSpec, dalTbl)

	return dalTbl
}

func (h *Host) ExecuteWorker(
	ctx context.Context,
	pluginID string,
	workerName string,
	args map[string]any,
) (*RPCResponse, error) {
	start := time.Now()

	if strings.TrimSpace(pluginID) == "" {
		return nil, errors.New("missing mandatory parameter: pluginID")
	}
	if strings.TrimSpace(workerName) == "" {
		return nil, errors.New("missing mandatory parameter: workerName")
	}

	plugin, exists := h.GetLoadedPlugin(pluginID)
	if !exists {
		h.logError(fmt.Sprintf("Plugin not found for worker execution: %s", pluginID))
		return nil, fmt.Errorf("plugin not loaded: %s", pluginID)
	}

	workerSpec, declared := plugin.Manifest.Workers[workerName]
	if plugin.State == PluginStateUninstalledDormant {
		h.recordDiagnostic(pluginID, workerName, "PLUGIN_DORMANT_QUARANTINE", time.Since(start))
		return nil, fmt.Errorf("PLUGIN_DORMANT_QUARANTINE: plugin %s is uninstalled and in dormant GDPR retention quarantine", pluginID)
	}
	if !declared {
		h.recordDiagnostic(pluginID, workerName, "UNDECLARED_WORKER", time.Since(start))
		return nil, fmt.Errorf("worker %q is not declared in manifest.json for plugin %s", workerName, pluginID)
	}

	timeoutMS := workerSpec.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 30000
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	caller := CallerContext{
		Type:        "SYSTEM",
		UserID:      "system:" + pluginID,
		StaffID:     "",
		ActiveRoles: []string{"system_worker"},
		Scopes:      workerSpec.Scopes,
		RequestID:   "wrk_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12],
	}

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
	lockTracker := &sqlLockTracker{}
	argSecrets := collectArgSecretStrings(args)
	defer func() {
		if writeTx != nil {
			_ = writeTx.Rollback()
			lockTracker.finishHold()
		}
	}()

	ctxTable := h.buildLuaCtxTable(L, execCtx, plugin, workerSpec, caller, &bufferedEvents, &writeTx, lockTracker, &argSecrets)
	argsTable := goMapToLuaTable(L, args)

	if err := L.DoString(plugin.LuaSource); err != nil {
		h.recordDiagnosticDetailed(pluginID, workerName, "LUA_LOAD_ERROR", err, time.Since(start), lockTracker, argSecrets)
		return nil, fmt.Errorf("failed to load plugin Lua script: %w", err)
	}

	fn := L.GetGlobal(workerName)
	if fn.Type() != lua.LTFunction {
		h.recordDiagnosticDetailed(pluginID, workerName, "MISSING_LUA_FUNCTION", nil, time.Since(start), lockTracker, argSecrets)
		return nil, fmt.Errorf("declared worker %q is not a function in init.lua", workerName)
	}

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, ctxTable, argsTable); err != nil {
		h.recordDiagnosticDetailed(pluginID, workerName, "LUA_RUNTIME_ERROR", err, time.Since(start), lockTracker, argSecrets)
		return nil, fmt.Errorf("lua worker execution error in %s.%s: %w", pluginID, workerName, err)
	}

	ret := L.Get(-1)
	L.Pop(1)

	retTbl, ok := ret.(*lua.LTable)
	if !ok {
		h.recordDiagnosticDetailed(pluginID, workerName, "INVALID_RETURN_TYPE", nil, time.Since(start), lockTracker, argSecrets)
		return nil, fmt.Errorf("worker %s.%s must return a table", pluginID, workerName)
	}

	okVal := L.GetField(retTbl, "ok")
	if okVal != lua.LTrue {
		if writeTx != nil {
			_ = writeTx.Rollback()
			writeTx = nil
			lockTracker.finishHold()
		}
		errCode := L.GetField(retTbl, "error_code").String()
		return &RPCResponse{
			OK:        false,
			ErrorCode: errCode,
		}, nil
	}

	if writeTx != nil {
		commitErr := writeTx.Commit()
		writeTx = nil
		lockTracker.finishHold()
		if commitErr != nil {
			h.recordDiagnosticDetailed(pluginID, workerName, "SQLITE_COMMIT_ERROR", commitErr, time.Since(start), lockTracker, argSecrets)
			return nil, fmt.Errorf("failed to commit plugin_db write transaction in worker: %w", commitErr)
		}
	}

	dataVal := luaValueToGo(L.GetField(retTbl, "data"))
	if len(bufferedEvents) > 0 {
		h.dispatchEvents(bufferedEvents)
	}

	return &RPCResponse{
		OK:   true,
		Data: dataVal,
	}, nil
}
