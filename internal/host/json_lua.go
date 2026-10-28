package host

import (
	"encoding/json"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func (h *Host) buildLuaJSONTable(L *lua.LState) *lua.LTable {
	jsonTbl := L.NewTable()

	L.SetField(jsonTbl, "encode", L.NewFunction(func(L *lua.LState) int {
		if L.GetTop() < 1 {
			L.RaiseError("ctx.json.encode requires a value argument")
			return 0
		}
		goVal := luaValueToGo(L.Get(1))
		rawBytes, err := json.Marshal(goVal)
		if err != nil {
			L.RaiseError("ctx.json.encode failed to serialize JSON: %s", err.Error())
			return 0
		}
		L.Push(lua.LString(string(rawBytes)))
		return 1
	}))

	L.SetField(jsonTbl, "decode", L.NewFunction(func(L *lua.LState) int {
		rawStr := strings.TrimSpace(L.OptString(1, ""))
		if rawStr == "" {
			L.RaiseError("INVALID_JSON: ctx.json.decode requires a non-empty JSON string")
			return 0
		}
		var parsed any
		if err := json.Unmarshal([]byte(rawStr), &parsed); err != nil {
			L.RaiseError("INVALID_JSON: ctx.json.decode failed to parse JSON: %s", err.Error())
			return 0
		}
		L.Push(goValueToLua(L, parsed))
		return 1
	}))

	return jsonTbl
}
