package host

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

const (
	defaultMaxDBSizeMB  = 500
	defaultSQLitePageSz = 4096
)

var sqlSchemaPrefixRegex = regexp.MustCompile(`(?i)\b([a-z0-9_]+)\s*\.\s*([a-z0-9_]+)\b`)

type Config struct {
	DataDirPath      string
	SchoolURN        string
	HostKMSKeyHandle string
	PluginsDir       string
}

type EndpointSpec struct {
	Scopes         []string `json:"scopes"`
	Mutates        bool     `json:"mutates"`
	TimeoutMS      int      `json:"timeout_ms,omitempty"`
	Events         []string `json:"events,omitempty"`
	Event          string   `json:"event,omitempty"`
	Cron           string   `json:"cron,omitempty"`
	MaxConcurrency int      `json:"max_concurrency,omitempty"`
}

type EgressSpec struct {
	AllowedDomains []string `json:"allowed_domains"`
	TimeoutMS      int      `json:"timeout_ms,omitempty"`
}

type DatabaseSpec struct {
	Alias       string `json:"alias,omitempty"`
	MaxDBSizeMB int    `json:"max_db_size_mb,omitempty"`
}

type GDPRHooksSpec struct {
	ExportPupilData string `json:"export_pupil_data"`
	PurgePupilData  string `json:"purge_pupil_data"`
}

type WidgetSpec struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Entry string `json:"entry"`
}

type Manifest struct {
	ID             string                  `json:"id"`
	Title          string                  `json:"title,omitempty"`
	Version        string                  `json:"version"`
	CoreDALVersion string                  `json:"core_dal_version"`
	MaxConcurrency int                     `json:"max_concurrency,omitempty"`
	MaxMemoryMB    int                     `json:"max_memory_mb,omitempty"`
	Database       *DatabaseSpec           `json:"database,omitempty"`
	GDPRHooks      *GDPRHooksSpec          `json:"gdpr_hooks,omitempty"`
	Egress         EgressSpec              `json:"egress"`
	Endpoints      map[string]EndpointSpec `json:"endpoints"`
	Workers        map[string]EndpointSpec `json:"workers,omitempty"`
	Widgets        []WidgetSpec            `json:"widgets,omitempty"`
}

type PluginState string

const (
	PluginStateActive             PluginState = "ACTIVE"
	PluginStateUninstalledDormant PluginState = "UNINSTALLED_DORMANT"
)

type LoadedPlugin struct {
	Manifest         Manifest
	LuaSource        string
	DirPath          string
	Folder           string
	State            PluginState
	ManifestModTime  time.Time
	LuaModTime       time.Time
	IsSyntheticClone bool
}

type CallerContext struct {
	Type        string   `json:"type"`
	UserID      string   `json:"user_id"`
	StaffID     string   `json:"staff_id"`
	ActiveRoles []string `json:"active_roles"`
	Scopes      []string `json:"scopes"`
	RequestID   string   `json:"request_id"`
}

type RPCResponse struct {
	OK        bool   `json:"ok"`
	Data      any    `json:"data,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
}

type Event struct {
	Name    string
	Payload map[string]any
	Hop     uint8
}

type Host struct {
	cfg     Config
	coreDAL *dal.CoreDAL

	mu                sync.RWMutex
	closed            bool
	plugins           map[string]*LoadedPlugin
	serviceLogs       []string
	diagnostics       []string
	diagnosticEntries []DiagnosticEntry
	diagnosticCap     int
	subscribers       map[string][]chan Event
	workerPools       map[string]*pluginEventWorkerPool

	eventsDispatchedTotal  atomic.Uint64
	eventsDroppedTotal     atomic.Uint64
	workerInvocationsTotal atomic.Uint64
}

func NewHost(cfg Config) (*Host, error) {
	if strings.TrimSpace(cfg.DataDirPath) == "" {
		return nil, errors.New("missing mandatory config: DataDirPath")
	}
	if strings.TrimSpace(cfg.SchoolURN) == "" {
		return nil, errors.New("missing mandatory config: SchoolURN")
	}
	if strings.TrimSpace(cfg.HostKMSKeyHandle) == "" {
		return nil, errors.New("missing mandatory config: HostKMSKeyHandle")
	}
	if strings.TrimSpace(cfg.PluginsDir) == "" {
		return nil, errors.New("missing mandatory config: PluginsDir")
	}

	if err := os.MkdirAll(cfg.DataDirPath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create DataDirPath: %w", err)
	}

	coreDAL, err := dal.NewCoreDAL(cfg.DataDirPath, cfg.SchoolURN)
	if err != nil {
		return nil, err
	}

	h := &Host{
		cfg:               cfg,
		coreDAL:           coreDAL,
		plugins:           make(map[string]*LoadedPlugin),
		serviceLogs:       make([]string, 0, 128),
		diagnostics:       make([]string, 0, DefaultDiagnosticRingBufferCapacity),
		diagnosticEntries: make([]DiagnosticEntry, 0, DefaultDiagnosticRingBufferCapacity),
		diagnosticCap:     DefaultDiagnosticRingBufferCapacity,
		subscribers:       make(map[string][]chan Event),
		workerPools:       make(map[string]*pluginEventWorkerPool),
	}

	coreDAL.SetPostCommitEventHook(func(evt dal.CoreDomainEvent) {
		h.dispatchEvents([]Event{
			{
				Name:    evt.Name,
				Payload: evt.Payload,
				Hop:     evt.Hop,
			},
		})
	})

	h.logInfo(fmt.Sprintf("Host initialized for school URN %s", cfg.SchoolURN))
	return h, nil
}

func (h *Host) DAL() *dal.CoreDAL {
	return h.coreDAL
}

func (h *Host) Close() error {
	h.stopAllPluginWorkerPools()
	return h.coreDAL.Close()
}

func (h *Host) logInfo(msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	entry := fmt.Sprintf("[%s] INFO %s", time.Now().UTC().Format(time.RFC3339Nano), msg)
	h.serviceLogs = append(h.serviceLogs, entry)
}

func (h *Host) logError(msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	entry := fmt.Sprintf("[%s] ERROR %s", time.Now().UTC().Format(time.RFC3339Nano), msg)
	h.serviceLogs = append(h.serviceLogs, entry)
}

func (h *Host) recordDiagnostic(pluginID string, method string, errClass string, duration time.Duration) {
	h.recordDiagnosticDetailed(pluginID, method, errClass, nil, duration, nil, nil)
}

func (h *Host) LogCursor() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return strconv.Itoa(len(h.serviceLogs))
}

func (h *Host) LogsSince(cursor string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	idx, err := strconv.Atoi(cursor)
	if err != nil || idx < 0 {
		idx = 0
	}
	if idx >= len(h.serviceLogs) {
		return []string{}
	}
	out := make([]string, len(h.serviceLogs[idx:]))
	copy(out, h.serviceLogs[idx:])
	return out
}

func (h *Host) Diagnostics() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]string, len(h.diagnostics))
	copy(out, h.diagnostics)
	return out
}

func (h *Host) SubscribeEvents(eventName string) <-chan Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan Event, 16)
	h.subscribers[eventName] = append(h.subscribers[eventName], ch)
	return ch
}

func (h *Host) dispatchEvents(events []Event) {
	for _, evt := range events {
		if evt.Hop == 0 {
			evt.Hop = 1
		}
		if evt.Hop > maxEventHopLimit {
			h.eventsDroppedTotal.Add(1)
			h.logInfo(fmt.Sprintf("Dropped event %s exceeding max hop count (hop=%d > %d)", evt.Name, evt.Hop, maxEventHopLimit))
			continue
		}

		h.eventsDispatchedTotal.Add(1)

		h.mu.RLock()
		rawSubs := h.subscribers[evt.Name]
		subs := make([]chan Event, len(rawSubs))
		copy(subs, rawSubs)

		var matchingPools []*pluginEventWorkerPool
		for pluginID, pool := range h.workerPools {
			if len(pool.subscribedEvents[evt.Name]) == 0 {
				continue
			}
			if p, ok := h.plugins[pluginID]; ok && p.State == PluginStateUninstalledDormant {
				continue
			}
			matchingPools = append(matchingPools, pool)
		}
		h.mu.RUnlock()

		for _, ch := range subs {
			select {
			case ch <- evt:
			default:
			}
		}

		for _, pool := range matchingPools {
			select {
			case pool.queue <- evt:
			default:
				h.eventsDroppedTotal.Add(1)
				h.logError(fmt.Sprintf("Dropped event %s for plugin %s: per-plugin buffered channel full", evt.Name, pool.pluginID))
			}
		}
	}
}

func readPluginFromDisk(dirPath string, folder string) (*LoadedPlugin, error) {
	manifestPath := filepath.Join(dirPath, "manifest.json")
	manifestStat, err := os.Stat(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat manifest.json: %w", err)
	}
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest.json: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest.json: %w", err)
	}
	if strings.TrimSpace(manifest.ID) == "" {
		return nil, errors.New("manifest.json missing mandatory id")
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return nil, errors.New("manifest.json missing mandatory version")
	}
	if strings.TrimSpace(manifest.CoreDALVersion) == "" {
		return nil, errors.New("manifest.json missing mandatory core_dal_version")
	}
	if len(manifest.Endpoints) == 0 {
		return nil, errors.New("manifest.json must declare at least one endpoint")
	}
	if manifest.Database != nil {
		if err := dal.ValidatePluginDatabaseAlias(manifest.Database.Alias); err != nil {
			return nil, err
		}
		if manifest.GDPRHooks == nil ||
			strings.TrimSpace(manifest.GDPRHooks.ExportPupilData) == "" ||
			strings.TrimSpace(manifest.GDPRHooks.PurgePupilData) == "" {
			return nil, fmt.Errorf(
				"manifest.json for plugin %s declares database.alias (%s) and must declare mandatory gdpr_hooks (export_pupil_data and purge_pupil_data)",
				manifest.ID,
				manifest.Database.Alias,
			)
		}
	}

	if len(manifest.Widgets) > 0 {
		seenWidgetIDs := make(map[string]struct{}, len(manifest.Widgets))
		for idx, w := range manifest.Widgets {
			wid := strings.TrimSpace(w.ID)
			wtitle := strings.TrimSpace(w.Title)
			wentry := strings.TrimSpace(w.Entry)
			if wid == "" || wtitle == "" || wentry == "" {
				return nil, fmt.Errorf("manifest.json widget[%d] for plugin %s is missing mandatory id, title, or entry", idx, manifest.ID)
			}
			if _, dup := seenWidgetIDs[wid]; dup {
				return nil, fmt.Errorf("manifest.json for plugin %s declares duplicate widget id %q", manifest.ID, wid)
			}
			seenWidgetIDs[wid] = struct{}{}

			relEntry, err := normalizeWidgetEntryRelPath(wentry)
			if err != nil {
				return nil, fmt.Errorf("manifest.json widget %q for plugin %s has invalid entry %q: %w", wid, manifest.ID, wentry, err)
			}
			uiDistDir := filepath.Join(dirPath, "ui", "dist")
			if distInfo, distErr := os.Stat(uiDistDir); distErr == nil && distInfo.IsDir() {
				widgetFullPath := filepath.Join(uiDistDir, relEntry)
				if st, statErr := os.Stat(widgetFullPath); statErr != nil || st.IsDir() {
					return nil, fmt.Errorf("manifest.json widget %q entry file not found at %s", wid, widgetFullPath)
				}
			}
		}
	}

	luaPath := filepath.Join(dirPath, "src", "init.lua")
	luaStat, err := os.Stat(luaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat src/init.lua: %w", err)
	}
	luaBytes, err := os.ReadFile(luaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read src/init.lua: %w", err)
	}

	luaSource := string(luaBytes)
	if err := validateManifestWorkersAndLuaSource(manifest, luaSource); err != nil {
		return nil, err
	}

	return &LoadedPlugin{
		Manifest:        manifest,
		LuaSource:       luaSource,
		DirPath:         dirPath,
		Folder:          folder,
		State:           PluginStateActive,
		ManifestModTime: manifestStat.ModTime(),
		LuaModTime:      luaStat.ModTime(),
	}, nil
}

func normalizeWidgetEntryRelPath(rawEntry string) (string, error) {
	trimmed := strings.TrimSpace(rawEntry)
	if trimmed == "" {
		return "", errors.New("empty widget entry path")
	}
	trimmed = strings.TrimPrefix(trimmed, "/")
	if strings.HasPrefix(trimmed, "ui/dist/") {
		trimmed = strings.TrimPrefix(trimmed, "ui/dist/")
	} else if strings.HasPrefix(trimmed, "ui/") {
		trimmed = strings.TrimPrefix(trimmed, "ui/")
	}
	clean := filepath.Clean(trimmed)
	if clean == "." || clean == "" || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("disallowed widget entry path: %s", rawEntry)
	}
	return clean, nil
}

func (h *Host) LoadPlugin(pluginFolder string) error {
	if strings.TrimSpace(pluginFolder) == "" {
		return errors.New("missing mandatory parameter: pluginFolder")
	}

	dirPath := filepath.Join(h.cfg.PluginsDir, pluginFolder)
	loaded, err := readPluginFromDisk(dirPath, pluginFolder)
	if err != nil {
		return err
	}

	if err := h.provisionAndMigratePluginDB(context.Background(), loaded); err != nil {
		return err
	}

	h.mu.Lock()
	h.plugins[loaded.Manifest.ID] = loaded
	h.mu.Unlock()

	h.startOrReplacePluginWorkerPool(loaded)

	h.logInfo(fmt.Sprintf("Loaded plugin %s v%s", loaded.Manifest.ID, loaded.Manifest.Version))
	return nil
}

func (h *Host) GetLoadedPlugin(pluginID string) (*LoadedPlugin, bool) {
	h.mu.RLock()
	p, ok := h.plugins[pluginID]
	h.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if p.IsSyntheticClone {
		return p, true
	}

	manifestStat, mErr := os.Stat(filepath.Join(p.DirPath, "manifest.json"))
	luaStat, lErr := os.Stat(filepath.Join(p.DirPath, "src", "init.lua"))
	if mErr == nil && lErr == nil {
		if !manifestStat.ModTime().Equal(p.ManifestModTime) || !luaStat.ModTime().Equal(p.LuaModTime) {
			if reloaded, err := readPluginFromDisk(p.DirPath, p.Folder); err == nil {
				reloaded.State = p.State
				if migErr := h.provisionAndMigratePluginDB(context.Background(), reloaded); migErr == nil {
					h.mu.Lock()
					h.plugins[pluginID] = reloaded
					h.mu.Unlock()
					h.startOrReplacePluginWorkerPool(reloaded)
					h.logInfo(fmt.Sprintf("Hot-reloaded plugin %s v%s from disk", reloaded.Manifest.ID, reloaded.Manifest.Version))
					return reloaded, true
				}
			}
		}
	}

	return p, true
}

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
	if plugin.State == PluginStateUninstalledDormant {
		h.recordDiagnostic(pluginID, methodName, "PLUGIN_DORMANT_QUARANTINE", time.Since(start))
		h.logError(fmt.Sprintf("Blocked RPC %s.%s because plugin is in UNINSTALLED_DORMANT quarantine state", pluginID, methodName))
		return nil, fmt.Errorf("PLUGIN_DORMANT_QUARANTINE: plugin %s is uninstalled and in dormant GDPR retention quarantine", pluginID)
	}

	endpointSpec, declared := plugin.Manifest.Endpoints[methodName]
	if !declared {
		h.recordDiagnostic(pluginID, methodName, "UNDECLARED_ENDPOINT", time.Since(start))
		h.logError(fmt.Sprintf("Default-deny blocked undeclared endpoint %s on %s", methodName, pluginID))
		return nil, fmt.Errorf("endpoint %q is not declared in manifest.json for plugin %s", methodName, pluginID)
	}

	if !hasAllScopes(caller.Scopes, endpointSpec.Scopes) {
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

	// Capability-sandboxed Lua VM (no io, os, debug, package, dofile)
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
	// Strip dangerous globals
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

	ctxTable := h.buildLuaCtxTable(L, execCtx, plugin, endpointSpec, caller, &bufferedEvents, &writeTx, lockTracker)
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
			h.recordDiagnosticDetailed(pluginID, methodName, "SQLITE_COMMIT_ERROR", commitErr, time.Since(start), lockTracker, argSecrets)
			return nil, fmt.Errorf("failed to commit plugin_db write transaction: %w", commitErr)
		}
	}

	dataVal := luaValueToGo(L.GetField(retTbl, "data"))

	// Post-commit asynchronous Event Bus dispatch
	if len(bufferedEvents) > 0 {
		h.dispatchEvents(bufferedEvents)
	}

	h.logInfo(fmt.Sprintf("Executed RPC %s.%s in %dms", pluginID, methodName, time.Since(start).Milliseconds()))
	return &RPCResponse{
		OK:   true,
		Data: dataVal,
	}, nil
}

func validatePluginDBSQLScope(sqlQuery string, alias string) error {
	trimmed := strings.TrimSpace(sqlQuery)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "attach ") || strings.HasPrefix(lower, "detach ") {
		return errors.New("ATTACH and DETACH statements are forbidden in ctx.plugin_db")
	}

	matches := sqlSchemaPrefixRegex.FindAllStringSubmatch(lower, -1)
	for _, m := range matches {
		schemaPrefix := m[1]
		if schemaPrefix == "excluded" || schemaPrefix == "new" || schemaPrefix == "old" {
			continue
		}
		if schemaPrefix == "main" || schemaPrefix == "temp" || strings.HasPrefix(schemaPrefix, "plugin_") {
			if schemaPrefix != strings.ToLower(alias) {
				return fmt.Errorf("ctx.plugin_db is scoped to %s; access to schema %q is forbidden", alias, schemaPrefix)
			}
		}
	}
	return nil
}

func luaTableToSQLArgs(L *lua.LState, argIdx int) ([]any, error) {
	if L.GetTop() < argIdx {
		return nil, nil
	}
	lv := L.Get(argIdx)
	if lv == lua.LNil {
		return nil, nil
	}
	tbl, ok := lv.(*lua.LTable)
	if !ok {
		return nil, errors.New("SQL params argument must be a Lua table")
	}
	maxn := tbl.MaxN()
	args := make([]any, 0, maxn)
	for i := 1; i <= maxn; i++ {
		item := tbl.RawGetInt(i)
		switch v := item.(type) {
		case *lua.LNilType:
			args = append(args, nil)
		case lua.LBool:
			args = append(args, bool(v))
		case lua.LString:
			args = append(args, string(v))
		case lua.LNumber:
			f := float64(v)
			if f == float64(int64(f)) {
				args = append(args, int64(f))
			} else {
				args = append(args, f)
			}
		default:
			args = append(args, v.String())
		}
	}
	return args, nil
}

func markLuaTableAsArray(L *lua.LState, tbl *lua.LTable) {
	mt := L.NewTable()
	L.SetField(mt, "__is_array", lua.LTrue)
	L.SetMetatable(tbl, mt)
}

func isMarkedLuaArray(tbl *lua.LTable) bool {
	mt, ok := tbl.Metatable.(*lua.LTable)
	if !ok || mt == nil {
		return false
	}
	return mt.RawGetString("__is_array") == lua.LTrue
}

func (h *Host) buildLuaCtxTable(
	L *lua.LState,
	execCtx context.Context,
	plugin *LoadedPlugin,
	endpointSpec EndpointSpec,
	caller CallerContext,
	bufferedEvents *[]Event,
	writeTx **sql.Tx,
	lockTracker *sqlLockTracker,
) *lua.LTable {
	ctxTbl := L.NewTable()

	// 1. ctx.caller
	callerTbl := L.NewTable()
	L.SetField(callerTbl, "type", lua.LString(caller.Type))
	L.SetField(callerTbl, "user_id", lua.LString(caller.UserID))
	L.SetField(callerTbl, "staff_id", lua.LString(caller.StaffID))
	L.SetField(callerTbl, "request_id", lua.LString(caller.RequestID))
	rolesTbl := L.NewTable()
	for i, r := range caller.ActiveRoles {
		rolesTbl.RawSetInt(i+1, lua.LString(r))
	}
	L.SetField(callerTbl, "active_roles", rolesTbl)
	L.SetField(ctxTbl, "caller", callerTbl)

	// 2. ctx.dal (scoped by declared endpoint scopes)
	dalTbl := h.buildLuaDALTable(L, execCtx, endpointSpec, caller)
	L.SetField(ctxTbl, "dal", dalTbl)

	// 3. ctx.plugin_db (scoped to attached plugin_<name>.db)
	pluginDBTbl := L.NewTable()
	L.SetField(pluginDBTbl, "query", L.NewFunction(func(L *lua.LState) int {
		sqlQuery := strings.TrimSpace(L.OptString(1, ""))
		if sqlQuery == "" {
			L.RaiseError("plugin_db.query requires a non-empty SQL query")
			return 0
		}
		if plugin.Manifest.Database == nil || strings.TrimSpace(plugin.Manifest.Database.Alias) == "" {
			L.RaiseError("plugin %s does not declare database.alias in manifest.json", plugin.Manifest.ID)
			return 0
		}
		alias := strings.TrimSpace(plugin.Manifest.Database.Alias)
		if err := validatePluginDBSQLScope(sqlQuery, alias); err != nil {
			L.RaiseError("plugin_db.query scope violation: %s", err.Error())
			return 0
		}
		sqlArgs, err := luaTableToSQLArgs(L, 2)
		if err != nil {
			L.RaiseError("plugin_db.query invalid params: %s", err.Error())
			return 0
		}

		var rows *sql.Rows
		if *writeTx != nil {
			rows, err = (*writeTx).QueryContext(execCtx, sqlQuery, sqlArgs...)
		} else {
			rows, err = h.coreDAL.ReadDB().QueryContext(execCtx, sqlQuery, sqlArgs...)
		}
		if err != nil {
			L.RaiseError("plugin_db.query failed: %s", err.Error())
			return 0
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			L.RaiseError("plugin_db.query failed to read columns: %s", err.Error())
			return 0
		}

		resultTbl := L.NewTable()
		markLuaTableAsArray(L, resultTbl)
		rowIndex := 1

		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				L.RaiseError("plugin_db.query row scan failed: %s", err.Error())
				return 0
			}

			rowTbl := L.NewTable()
			for i, colName := range cols {
				rawVal := vals[i]
				switch v := rawVal.(type) {
				case nil:
					L.SetField(rowTbl, colName, lua.LNil)
				case int64:
					L.SetField(rowTbl, colName, lua.LNumber(v))
				case float64:
					L.SetField(rowTbl, colName, lua.LNumber(v))
				case bool:
					L.SetField(rowTbl, colName, lua.LBool(v))
				case string:
					L.SetField(rowTbl, colName, lua.LString(v))
				case []byte:
					L.SetField(rowTbl, colName, lua.LString(string(v)))
				default:
					L.SetField(rowTbl, colName, lua.LString(fmt.Sprintf("%v", v)))
				}
			}
			resultTbl.RawSetInt(rowIndex, rowTbl)
			rowIndex++
		}
		if err := rows.Err(); err != nil {
			L.RaiseError("plugin_db.query iteration error: %s", err.Error())
			return 0
		}

		L.Push(resultTbl)
		return 1
	}))

	if endpointSpec.Mutates {
		L.SetField(pluginDBTbl, "exec", L.NewFunction(func(L *lua.LState) int {
			sqlQuery := strings.TrimSpace(L.OptString(1, ""))
			if sqlQuery == "" {
				L.RaiseError("plugin_db.exec requires a non-empty SQL statement")
				return 0
			}
			if plugin.Manifest.Database == nil || strings.TrimSpace(plugin.Manifest.Database.Alias) == "" {
				L.RaiseError("plugin %s does not declare database.alias in manifest.json", plugin.Manifest.ID)
				return 0
			}
			alias := strings.TrimSpace(plugin.Manifest.Database.Alias)
			if err := validatePluginDBSQLScope(sqlQuery, alias); err != nil {
				L.RaiseError("plugin_db.exec scope violation: %s", err.Error())
				return 0
			}
			sqlArgs, err := luaTableToSQLArgs(L, 2)
			if err != nil {
				L.RaiseError("plugin_db.exec invalid params: %s", err.Error())
				return 0
			}

			if *writeTx == nil {
				waitStart := time.Now()
				tx, txErr := h.coreDAL.WriteDB().BeginTx(execCtx, nil)
				if lockTracker != nil {
					lockTracker.lockWaitDuration += time.Since(waitStart)
				}
				if txErr != nil {
					L.RaiseError("plugin_db.exec failed to start WriteDB transaction: %s", txErr.Error())
					return 0
				}
				if lockTracker != nil {
					lockTracker.writeHoldStart = time.Now()
				}
				*writeTx = tx
			}

			res, execErr := (*writeTx).ExecContext(execCtx, sqlQuery, sqlArgs...)
			if execErr != nil {
				L.RaiseError("plugin_db.exec failed: %s", execErr.Error())
				return 0
			}

			rowsAffected, _ := res.RowsAffected()
			lastInsertID, _ := res.LastInsertId()

			resTbl := L.NewTable()
			L.SetField(resTbl, "rows_affected", lua.LNumber(rowsAffected))
			L.SetField(resTbl, "last_insert_id", lua.LNumber(lastInsertID))
			L.Push(resTbl)
			return 1
		}))
	}
	L.SetField(ctxTbl, "plugin_db", pluginDBTbl)

	// 4. ctx.events (post-commit buffered emission with hop-count increment)
	eventsTbl := L.NewTable()
	L.SetField(eventsTbl, "emit", L.NewFunction(func(L *lua.LState) int {
		evtName := strings.TrimSpace(L.OptString(1, ""))
		if evtName == "" {
			L.RaiseError("ctx.events.emit requires a non-empty event_name")
			return 0
		}
		if err := dal.ValidateEventSubscriptionName(evtName); err != nil {
			L.RaiseError("ctx.events.emit invalid event_name: %s", err.Error())
			return 0
		}
		var payload map[string]any
		if L.GetTop() >= 2 {
			if tbl, ok := L.Get(2).(*lua.LTable); ok {
				if m, ok := luaValueToGo(tbl).(map[string]any); ok {
					payload = m
				}
			}
		}
		if payload == nil {
			payload = map[string]any{}
		}
		parentHop := dal.EventHopFromContext(execCtx)
		nextHop := parentHop + 1
		if nextHop == 0 {
			nextHop = 1
		}
		*bufferedEvents = append(*bufferedEvents, Event{
			Name:    evtName,
			Payload: payload,
			Hop:     nextHop,
		})
		return 0
	}))
	L.SetField(ctxTbl, "events", eventsTbl)

	// 5. ctx.http (egress.timeout_ms, CheckRedirect domain whitelist verification & open-write-tx halt)
	httpTbl := h.buildLuaHTTPEgressTable(L, execCtx, plugin, writeTx, lockTracker)
	L.SetField(ctxTbl, "http", httpTbl)

	// 6. ctx.xml (deterministic DfE CBDS XML builder & XSD schema validator)
	xmlTbl := h.buildLuaXMLTable(L)
	L.SetField(ctxTbl, "xml", xmlTbl)

	return ctxTbl
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


func goMapToLuaTable(L *lua.LState, m map[string]any) *lua.LTable {
	tbl := L.NewTable()
	for k, v := range m {
		L.SetField(tbl, k, goValueToLua(L, v))
	}
	return tbl
}

func goValueToLua(L *lua.LState, v any) lua.LValue {
	if v == nil {
		return lua.LNil
	}
	switch val := v.(type) {
	case string:
		return lua.LString(val)
	case bool:
		return lua.LBool(val)
	case int:
		return lua.LNumber(val)
	case uint8:
		return lua.LNumber(val)
	case int64:
		return lua.LNumber(val)
	case float64:
		return lua.LNumber(val)
	case []string:
		arr := L.NewTable()
		markLuaTableAsArray(L, arr)
		for i, item := range val {
			arr.RawSetInt(i+1, lua.LString(item))
		}
		return arr
	case []any:
		arr := L.NewTable()
		markLuaTableAsArray(L, arr)
		for i, item := range val {
			arr.RawSetInt(i+1, goValueToLua(L, item))
		}
		return arr
	case map[string]any:
		return goMapToLuaTable(L, val)
	default:
		return lua.LString(fmt.Sprintf("%v", val))
	}
}

func luaValueToGo(lv lua.LValue) any {
	switch v := lv.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LString:
		return string(v)
	case lua.LNumber:
		return float64(v)
	case *lua.LTable:
		maxn := v.MaxN()
		if maxn > 0 {
			arr := make([]any, 0, maxn)
			for i := 1; i <= maxn; i++ {
				arr = append(arr, luaValueToGo(v.RawGetInt(i)))
			}
			return arr
		}
		if isMarkedLuaArray(v) {
			return []any{}
		}
		m := make(map[string]any)
		v.ForEach(func(key, val lua.LValue) {
			m[key.String()] = luaValueToGo(val)
		})
		return m
	default:
		return v.String()
	}
}
