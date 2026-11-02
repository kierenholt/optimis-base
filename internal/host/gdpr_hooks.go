package host

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

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
	canonicalKey := trimmedID
	if !exists {
		for mID, candidate := range h.plugins {
			if candidate.PlugID == trimmedID {
				p = candidate
				canonicalKey = mID
				exists = true
				break
			}
		}
	}
	if !exists {
		h.mu.Unlock()
		return fmt.Errorf("plugin not found: %s", trimmedID)
	}
	p.State = state
	p.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_ = h.coreDAL.SetInstalledPluginState(context.Background(), canonicalKey, string(state))
	var poolToStop *pluginEventWorkerPool
	if state == PluginStateUninstalledDormant {
		poolToStop = h.workerPools[canonicalKey]
		delete(h.workerPools, canonicalKey)
	}
	h.serviceLogs = append(
		h.serviceLogs,
		fmt.Sprintf("[%s] INFO Plugin %s (%s) state transitioned to %s", time.Now().UTC().Format(time.RFC3339Nano), canonicalKey, p.PlugID, state),
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
	lockTracker := &sqlLockTracker{}
	argSecrets := []string{pupilID}
	defer func() {
		if writeTx != nil {
			_ = writeTx.Rollback()
			lockTracker.finishHold()
		}
	}()

	hookSpec := EndpointSpec{
		Scopes:    []string{"dal.pupils.read:all", "dal.census.read:all"},
		Mutates:   mutates,
		TimeoutMS: 5000,
	}

	ctxTable := h.buildLuaCtxTable(L, execCtx, plugin, hookSpec, caller, &bufferedEvents, &writeTx, lockTracker, &argSecrets)
	argsTable := goMapToLuaTable(L, map[string]any{
		"pupil_id": pupilID,
	})

	if err := L.DoString(plugin.LuaSource); err != nil {
		h.recordDiagnosticDetailed(plugin.Manifest.ID, hookFnName, "LUA_LOAD_ERROR", err, time.Since(start), lockTracker, argSecrets)
		return nil, fmt.Errorf("failed to load Lua source for GDPR hook %s.%s: %w", plugin.Manifest.ID, hookFnName, err)
	}

	fn := L.GetGlobal(hookFnName)
	if fn.Type() != lua.LTFunction {
		h.recordDiagnosticDetailed(plugin.Manifest.ID, hookFnName, "MISSING_GDPR_HOOK", nil, time.Since(start), lockTracker, argSecrets)
		return nil, fmt.Errorf("declared gdpr_hook %q is not a function in %s init.lua", hookFnName, plugin.Manifest.ID)
	}

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, ctxTable, argsTable); err != nil {
		h.recordDiagnosticDetailed(plugin.Manifest.ID, hookFnName, "LUA_GDPR_HOOK_ERROR", err, time.Since(start), lockTracker, argSecrets)
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
			lockTracker.finishHold()
		}
		errCode := L.GetField(retTbl, "error_code").String()
		return nil, fmt.Errorf("GDPR hook %s.%s returned ok=false (%s)", plugin.Manifest.ID, hookFnName, errCode)
	}

	if writeTx != nil {
		commitErr := writeTx.Commit()
		writeTx = nil
		lockTracker.finishHold()
		if commitErr != nil {
			return nil, fmt.Errorf("failed to commit GDPR hook %s.%s write transaction: %w", plugin.Manifest.ID, hookFnName, commitErr)
		}
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

func (h *Host) InvokePluginExportHook(ctx context.Context, pluginID string, caller CallerContext, pupilID string) ([]dal.SARSection, error) {
	h.mu.RLock()
	plugin, exists := h.plugins[strings.TrimSpace(pluginID)]
	h.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", pluginID)
	}
	if plugin.Manifest.GDPRHooks == nil || strings.TrimSpace(plugin.Manifest.GDPRHooks.ExportPupilData) == "" {
		return nil, fmt.Errorf("plugin %s is missing gdpr_hooks.export_pupil_data", plugin.Manifest.ID)
	}
	rawExport, err := h.invokePluginGDPRHook(
		ctx,
		plugin,
		strings.TrimSpace(plugin.Manifest.GDPRHooks.ExportPupilData),
		false,
		caller,
		strings.TrimSpace(pupilID),
	)
	if err != nil {
		return nil, err
	}
	return parsePluginSARSections(plugin, rawExport)
}

func (h *Host) InvokePluginPurgeHook(ctx context.Context, pluginID string, caller CallerContext, pupilID string) (any, error) {
	h.mu.RLock()
	plugin, exists := h.plugins[strings.TrimSpace(pluginID)]
	h.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", pluginID)
	}
	if plugin.Manifest.GDPRHooks == nil || strings.TrimSpace(plugin.Manifest.GDPRHooks.PurgePupilData) == "" {
		return nil, fmt.Errorf("plugin %s is missing gdpr_hooks.purge_pupil_data", plugin.Manifest.ID)
	}
	return h.invokePluginGDPRHook(
		ctx,
		plugin,
		strings.TrimSpace(plugin.Manifest.GDPRHooks.PurgePupilData),
		true,
		caller,
		strings.TrimSpace(pupilID),
	)
}
