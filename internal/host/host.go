package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

type Config struct {
	DataDirPath      string
	SchoolURN        string
	HostKMSKeyHandle string
	PluginsDir       string
}

type EndpointSpec struct {
	Scopes    []string `json:"scopes"`
	Mutates   bool     `json:"mutates"`
	TimeoutMS int      `json:"timeout_ms,omitempty"`
}

type EgressSpec struct {
	AllowedDomains []string `json:"allowed_domains"`
	TimeoutMS      int      `json:"timeout_ms,omitempty"`
}

type DatabaseSpec struct {
	Alias       string `json:"alias,omitempty"`
	MaxDBSizeMB int    `json:"max_db_size_mb,omitempty"`
}

type Manifest struct {
	ID             string                  `json:"id"`
	Version        string                  `json:"version"`
	CoreDALVersion string                  `json:"core_dal_version"`
	Database       *DatabaseSpec           `json:"database,omitempty"`
	Egress         EgressSpec              `json:"egress"`
	Endpoints      map[string]EndpointSpec `json:"endpoints"`
}

type LoadedPlugin struct {
	Manifest        Manifest
	LuaSource       string
	DirPath         string
	Folder          string
	ManifestModTime time.Time
	LuaModTime      time.Time
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

	mu          sync.RWMutex
	plugins     map[string]*LoadedPlugin
	serviceLogs []string
	diagnostics []string
	subscribers map[string][]chan Event
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
		cfg:         cfg,
		coreDAL:     coreDAL,
		plugins:     make(map[string]*LoadedPlugin),
		serviceLogs: make([]string, 0, 128),
		diagnostics: make([]string, 0, 64),
		subscribers: make(map[string][]chan Event),
	}
	h.logInfo(fmt.Sprintf("Host initialized for school URN %s", cfg.SchoolURN))
	return h, nil
}

func (h *Host) DAL() *dal.CoreDAL {
	return h.coreDAL
}

func (h *Host) Close() error {
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
	h.mu.Lock()
	defer h.mu.Unlock()
	entry := fmt.Sprintf("plugin=%s method=%s error_class=%s duration_ms=%d", pluginID, method, errClass, duration.Milliseconds())
	h.diagnostics = append(h.diagnostics, entry)
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
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, evt := range events {
		if evt.Hop > 3 {
			continue
		}
		subs := h.subscribers[evt.Name]
		for _, ch := range subs {
			select {
			case ch <- evt:
			default:
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

	luaPath := filepath.Join(dirPath, "src", "init.lua")
	luaStat, err := os.Stat(luaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat src/init.lua: %w", err)
	}
	luaBytes, err := os.ReadFile(luaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read src/init.lua: %w", err)
	}

	return &LoadedPlugin{
		Manifest:        manifest,
		LuaSource:       string(luaBytes),
		DirPath:         dirPath,
		Folder:          folder,
		ManifestModTime: manifestStat.ModTime(),
		LuaModTime:      luaStat.ModTime(),
	}, nil
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

	h.mu.Lock()
	h.plugins[loaded.Manifest.ID] = loaded
	h.mu.Unlock()

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

	manifestStat, mErr := os.Stat(filepath.Join(p.DirPath, "manifest.json"))
	luaStat, lErr := os.Stat(filepath.Join(p.DirPath, "src", "init.lua"))
	if mErr == nil && lErr == nil {
		if !manifestStat.ModTime().Equal(p.ManifestModTime) || !luaStat.ModTime().Equal(p.LuaModTime) {
			if reloaded, err := readPluginFromDisk(p.DirPath, p.Folder); err == nil {
				h.mu.Lock()
				h.plugins[pluginID] = reloaded
				h.mu.Unlock()
				h.logInfo(fmt.Sprintf("Hot-reloaded plugin %s v%s from disk", reloaded.Manifest.ID, reloaded.Manifest.Version))
				return reloaded, true
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

	ctxTable := h.buildLuaCtxTable(L, execCtx, plugin, endpointSpec, caller, &bufferedEvents)
	argsTable := goMapToLuaTable(L, args)

	if err := L.DoString(plugin.LuaSource); err != nil {
		h.recordDiagnostic(pluginID, methodName, "LUA_LOAD_ERROR", time.Since(start))
		h.logError(fmt.Sprintf("Lua load error in %s: %v", pluginID, err))
		return nil, fmt.Errorf("failed to load plugin Lua script: %w", err)
	}

	fn := L.GetGlobal(methodName)
	if fn.Type() != lua.LTFunction {
		h.recordDiagnostic(pluginID, methodName, "MISSING_LUA_FUNCTION", time.Since(start))
		return nil, fmt.Errorf("declared endpoint %q is not a function in init.lua", methodName)
	}

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, ctxTable, argsTable); err != nil {
		h.recordDiagnostic(pluginID, methodName, "LUA_RUNTIME_ERROR", time.Since(start))
		h.logError(fmt.Sprintf("Lua runtime error in %s.%s", pluginID, methodName))
		return nil, fmt.Errorf("lua execution error in %s.%s: %w", pluginID, methodName, err)
	}

	ret := L.Get(-1)
	L.Pop(1)

	retTbl, ok := ret.(*lua.LTable)
	if !ok {
		h.recordDiagnostic(pluginID, methodName, "INVALID_RETURN_TYPE", time.Since(start))
		return nil, fmt.Errorf("endpoint %s.%s must return a table", pluginID, methodName)
	}

	okVal := L.GetField(retTbl, "ok")
	if okVal != lua.LTrue {
		errCode := L.GetField(retTbl, "error_code").String()
		return &RPCResponse{
			OK:        false,
			ErrorCode: errCode,
		}, nil
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

func (h *Host) buildLuaCtxTable(
	L *lua.LState,
	execCtx context.Context,
	plugin *LoadedPlugin,
	endpointSpec EndpointSpec,
	caller CallerContext,
	bufferedEvents *[]Event,
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
	dalTbl := L.NewTable()
	if hasDomainScope(endpointSpec.Scopes, "dal.pupils.read") {
		pupilsTbl := L.NewTable()
		L.SetField(pupilsTbl, "get_by_id", L.NewFunction(func(L *lua.LState) int {
			pupilID := strings.TrimSpace(L.OptString(1, ""))
			if pupilID == "" {
				L.RaiseError("missing mandatory parameter: pupil_id")
				return 0
			}
			pupil, err := h.coreDAL.GetPupilByID(execCtx, pupilID)
			if err != nil {
				L.RaiseError("dal.pupils.get_by_id failed: %s", err.Error())
				return 0
			}
			pTbl := L.NewTable()
			L.SetField(pTbl, "id", lua.LString(pupil.ID))
			L.SetField(pTbl, "upn", lua.LString(pupil.UPN))
			L.SetField(pTbl, "first_name", lua.LString(pupil.FirstName))
			L.SetField(pTbl, "last_name", lua.LString(pupil.LastName))
			L.SetField(pTbl, "academic_year_id", lua.LString(pupil.AcademicYearID))
			L.SetField(pTbl, "effective_from", lua.LString(pupil.EffectiveFrom))
			L.Push(pTbl)
			return 1
		}))
		L.SetField(dalTbl, "pupils", pupilsTbl)
	}
	L.SetField(ctxTbl, "dal", dalTbl)

	// 3. ctx.plugin_db
	pluginDBTbl := L.NewTable()
	L.SetField(pluginDBTbl, "query", L.NewFunction(func(L *lua.LState) int {
		sqlQuery := strings.TrimSpace(L.OptString(1, ""))
		if sqlQuery == "" {
			L.RaiseError("plugin_db.query requires a non-empty SQL query")
			return 0
		}
		L.Push(L.NewTable())
		return 1
	}))
	if endpointSpec.Mutates {
		L.SetField(pluginDBTbl, "exec", L.NewFunction(func(L *lua.LState) int {
			sqlQuery := strings.TrimSpace(L.OptString(1, ""))
			if sqlQuery == "" {
				L.RaiseError("plugin_db.exec requires a non-empty SQL statement")
				return 0
			}
			resTbl := L.NewTable()
			L.SetField(resTbl, "rows_affected", lua.LNumber(0))
			L.Push(resTbl)
			return 1
		}))
	}
	L.SetField(ctxTbl, "plugin_db", pluginDBTbl)

	// 4. ctx.events (post-commit buffered emission)
	eventsTbl := L.NewTable()
	L.SetField(eventsTbl, "emit", L.NewFunction(func(L *lua.LState) int {
		evtName := strings.TrimSpace(L.OptString(1, ""))
		if evtName == "" {
			L.RaiseError("ctx.events.emit requires a non-empty event_name")
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
		*bufferedEvents = append(*bufferedEvents, Event{
			Name:    evtName,
			Payload: payload,
			Hop:     1,
		})
		return 0
	}))
	L.SetField(ctxTbl, "events", eventsTbl)

	// 5. ctx.http (default-deny domain check)
	httpTbl := L.NewTable()
	L.SetField(httpTbl, "get", L.NewFunction(func(L *lua.LState) int {
		targetURL := strings.TrimSpace(L.OptString(1, ""))
		if !isDomainAllowed(targetURL, plugin.Manifest.Egress.AllowedDomains) {
			L.RaiseError("egress domain not permitted by manifest.json: %s", targetURL)
			return 0
		}
		respTbl := L.NewTable()
		L.SetField(respTbl, "status", lua.LNumber(200))
		L.Push(respTbl)
		return 1
	}))
	L.SetField(httpTbl, "post", L.NewFunction(func(L *lua.LState) int {
		targetURL := strings.TrimSpace(L.OptString(1, ""))
		if !isDomainAllowed(targetURL, plugin.Manifest.Egress.AllowedDomains) {
			L.RaiseError("egress domain not permitted by manifest.json: %s", targetURL)
			return 0
		}
		respTbl := L.NewTable()
		L.SetField(respTbl, "status", lua.LNumber(200))
		L.Push(respTbl)
		return 1
	}))
	L.SetField(ctxTbl, "http", httpTbl)

	return ctxTbl
}

func hasAllScopes(callerScopes []string, requiredScopes []string) bool {
	set := make(map[string]struct{}, len(callerScopes))
	for _, s := range callerScopes {
		set[s] = struct{}{}
	}
	for _, req := range requiredScopes {
		if _, ok := set[req]; !ok {
			return false
		}
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

func isDomainAllowed(targetURL string, allowed []string) bool {
	if targetURL == "" {
		return false
	}
	for _, origin := range allowed {
		if origin != "" && strings.HasPrefix(targetURL, origin) {
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
	case int64:
		return lua.LNumber(val)
	case float64:
		return lua.LNumber(val)
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
		m := make(map[string]any)
		v.ForEach(func(key, val lua.LValue) {
			m[key.String()] = luaValueToGo(val)
		})
		return m
	default:
		return v.String()
	}
}
