package host

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

func newSandboxedLuaState(execCtx context.Context) *lua.LState {
	// Capability-sandboxed Lua VM (no io, os, debug, package, dofile)
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
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
	return L
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
	extraSecrets *[]string,
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
	pluginDBTbl := h.buildLuaPluginDBTable(L, execCtx, plugin, endpointSpec, writeTx, lockTracker)
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

	// 7. ctx.json (safe JSON encode/decode without Lua string concatenation)
	jsonTbl := h.buildLuaJSONTable(L)
	L.SetField(ctxTbl, "json", jsonTbl)

	// 8. ctx.clock & ctx.crypto (capability-safe UTC timestamps and UUID v4 generation)
	clockTbl := L.NewTable()
	L.SetField(clockTbl, "now_utc", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(time.Now().UTC().Format(time.RFC3339)))
		return 1
	}))
	L.SetField(ctxTbl, "clock", clockTbl)

	cryptoTbl := L.NewTable()
	L.SetField(cryptoTbl, "uuid", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(uuid.NewString()))
		return 1
	}))
	L.SetField(cryptoTbl, "seal_secret", L.NewFunction(func(L *lua.LState) int {
		plaintext := strings.TrimSpace(L.OptString(1, ""))
		if plaintext == "" {
			L.RaiseError("ctx.crypto.seal_secret requires a non-empty plaintext string")
			return 0
		}
		sealed, sErr := h.sealPluginSecret(plugin.Manifest.ID, plaintext)
		if sErr != nil {
			L.RaiseError("ctx.crypto.seal_secret failed: %s", sErr.Error())
			return 0
		}
		if extraSecrets != nil && len(plaintext) >= 4 {
			*extraSecrets = append(*extraSecrets, plaintext)
		}
		L.Push(lua.LString(sealed))
		return 1
	}))
	L.SetField(cryptoTbl, "open_secret", L.NewFunction(func(L *lua.LState) int {
		ciphertext := strings.TrimSpace(L.OptString(1, ""))
		if ciphertext == "" {
			L.RaiseError("ctx.crypto.open_secret requires a non-empty ciphertext string")
			return 0
		}
		opened, oErr := h.openPluginSecret(plugin.Manifest.ID, ciphertext)
		if oErr != nil {
			L.RaiseError("ctx.crypto.open_secret failed: %s", oErr.Error())
			return 0
		}
		if extraSecrets != nil && len(opened) >= 4 {
			*extraSecrets = append(*extraSecrets, opened)
		}
		L.Push(lua.LString(opened))
		return 1
	}))
	L.SetField(ctxTbl, "crypto", cryptoTbl)

	// 9. ctx.school (read-only school URN, default temporal context, and synced MAT policy metadata from __system_config)
	schoolTbl := L.NewTable()
	L.SetField(schoolTbl, "urn", lua.LString(h.cfg.SchoolURN))
	if defScope, sErr := h.coreDAL.ResolveScopeDefaults(execCtx, dal.DALQueryScope{}); sErr == nil {
		L.SetField(schoolTbl, "academic_year_id", lua.LString(defScope.AcademicYearID))
		L.SetField(schoolTbl, "reference_date", lua.LString(defScope.ReferenceDate))
	} else {
		L.SetField(schoolTbl, "academic_year_id", lua.LString("ay_2026_2027"))
		L.SetField(schoolTbl, "reference_date", lua.LString(time.Now().UTC().Format("2006-01-02")))
	}
	matID, _ := h.coreDAL.GetSystemConfigValue(execCtx, "mat_id")
	matGradingSchemaVersion, _ := h.coreDAL.GetSystemConfigValue(execCtx, "mat_grading_schema_version")
	trimmedMATVersion := strings.TrimSpace(matGradingSchemaVersion)
	L.SetField(schoolTbl, "mat_id", lua.LString(strings.TrimSpace(matID)))
	L.SetField(schoolTbl, "mat_grading_schema_version", lua.LString(trimmedMATVersion))
	if trimmedMATVersion != "" {
		var schemaMap map[string]any
		if rawSchemaJSON, sErr := h.coreDAL.GetSystemConfigValue(execCtx, "mat_grading_schema_json"); sErr == nil && strings.TrimSpace(rawSchemaJSON) != "" {
			_ = json.Unmarshal([]byte(rawSchemaJSON), &schemaMap)
		}
		if len(schemaMap) == 0 {
			schemaMap = defaultMATGradingSchemaForVersion(trimmedMATVersion)
		}
		if len(schemaMap) > 0 {
			L.SetField(schoolTbl, "mat_grading_schema", goMapToLuaTable(L, schemaMap))
		}
	}
	L.SetField(ctxTbl, "school", schoolTbl)

	return ctxTbl
}

func (h *Host) derivePluginSecretKey(pluginID string) [32]byte {
	return sha256.Sum256([]byte("optimis-plugin-secret-v1:" + h.cfg.HostKMSKeyHandle + ":" + strings.TrimSpace(pluginID)))
}

func (h *Host) sealPluginSecret(pluginID string, plaintext string) (string, error) {
	key := h.derivePluginSecretKey(pluginID)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), []byte(strings.TrimSpace(pluginID)))
	return "enc:v1:" + base64.StdEncoding.EncodeToString(sealed), nil
}

func (h *Host) openPluginSecret(pluginID string, ciphertext string) (string, error) {
	const prefix = "enc:v1:"
	trimmed := strings.TrimSpace(ciphertext)
	if !strings.HasPrefix(trimmed, prefix) {
		return "", errors.New("invalid secret envelope prefix: expected enc:v1:")
	}
	rawBytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(trimmed, prefix))
	if err != nil {
		return "", fmt.Errorf("invalid base64 secret payload: %w", err)
	}
	key := h.derivePluginSecretKey(pluginID)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(rawBytes) < gcm.NonceSize() {
		return "", errors.New("malformed secret envelope: payload shorter than GCM nonce")
	}
	nonce := rawBytes[:gcm.NonceSize()]
	enc := rawBytes[gcm.NonceSize():]
	plainBytes, err := gcm.Open(nil, nonce, enc, []byte(strings.TrimSpace(pluginID)))
	if err != nil {
		return "", errors.New("failed to authenticate or decrypt plugin secret for this plugin_id")
	}
	return string(plainBytes), nil
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
	return luaValueToGoVisited(lv, make(map[*lua.LTable]bool))
}

func luaValueToGoVisited(lv lua.LValue, active map[*lua.LTable]bool) any {
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
		if active[v] {
			return nil
		}
		active[v] = true
		defer delete(active, v)

		maxn := v.MaxN()
		if maxn > 0 {
			arr := make([]any, 0, maxn)
			for i := 1; i <= maxn; i++ {
				arr = append(arr, luaValueToGoVisited(v.RawGetInt(i), active))
			}
			return arr
		}
		if isMarkedLuaArray(v) {
			return []any{}
		}
		m := make(map[string]any)
		v.ForEach(func(key, val lua.LValue) {
			m[key.String()] = luaValueToGoVisited(val, active)
		})
		return m
	default:
		return v.String()
	}
}
