package host

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	lua "github.com/yuin/gopher-lua"
)

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
