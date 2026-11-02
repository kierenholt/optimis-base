package sdk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/host"
)

var luaTestErrLineRegex = regexp.MustCompile(`([^\s:]+\.lua):(\d+):`)

type TestRunReport struct {
	PluginID string
	Passed   int
	Failed   int
}

func RunPluginTests(pluginDir string, defaultPersona string, out io.Writer) (*TestRunReport, error) {
	trimmedDir := strings.TrimSpace(pluginDir)
	if trimmedDir == "" {
		return nil, errors.New("missing mandatory plugin directory path")
	}
	absDir, err := filepath.Abs(trimmedDir)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = io.Discard
	}

	manifest, err := ReadAndValidateManifest(absDir)
	if err != nil {
		return nil, err
	}

	testsDir := filepath.Join(absDir, "tests")
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		return nil, fmt.Errorf("missing or unreadable tests/ directory in %s: %w", absDir, err)
	}

	var testFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), "_test.lua") {
			testFiles = append(testFiles, filepath.Join(testsDir, entry.Name()))
		}
	}
	sort.Strings(testFiles)
	if len(testFiles) == 0 {
		return nil, fmt.Errorf("no *_test.lua files found in %s", testsDir)
	}

	tmpDir, err := os.MkdirTemp("", "mis-test-run-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	pluginsRoot := filepath.Join(tmpDir, "plugins")
	stagedPluginDir := filepath.Join(pluginsRoot, "target_plugin")
	if err := copyDirectory(absDir, stagedPluginDir); err != nil {
		return nil, err
	}

	h, err := host.NewHost(host.Config{
		DataDirPath:      filepath.Join(tmpDir, "data"),
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms_test_key_01",
		PluginsDir:       pluginsRoot,
	})
	if err != nil {
		return nil, err
	}
	defer h.Close()

	h.SetStructuredStdout(out)

	ctx := context.Background()
	if err := SeedSyntheticDevCohort(ctx, h.DAL()); err != nil {
		return nil, fmt.Errorf("failed to seed synthetic school cohort: %w", err)
	}

	if err := h.LoadPlugin("target_plugin"); err != nil {
		return nil, fmt.Errorf("failed to load plugin %s: %w", manifest.ID, err)
	}

	activePersona := strings.TrimSpace(defaultPersona)
	if activePersona == "" {
		activePersona = "stf_tch_7a"
	}

	report := &TestRunReport{
		PluginID: manifest.ID,
	}

	for _, testFile := range testFiles {
		relFile, _ := filepath.Rel(absDir, testFile)
		rawLua, err := os.ReadFile(testFile)
		if err != nil {
			return report, err
		}

		L := newSandboxedLuaState()
		if err := L.DoString(string(rawLua)); err != nil {
			L.Close()
			file, line := parseLuaTestErrorLocation(relFile, err)
			_, _ = fmt.Fprintf(out, "type=lua_err plugin=%s file=%s line=%d error_class=LUA_TEST_LOAD_ERROR err=%q\n", manifest.ID, file, line, err.Error())
			report.Failed++
			return report, fmt.Errorf("failed to load %s: %w", relFile, err)
		}

		var testFuncNames []string
		L.G.Global.ForEach(func(k, v lua.LValue) {
			if k.Type() == lua.LTString && v.Type() == lua.LTFunction {
				name := k.String()
				if strings.HasPrefix(name, "test_") {
					testFuncNames = append(testFuncNames, name)
				}
			}
		})
		sort.Strings(testFuncNames)

		for _, fnName := range testFuncNames {
			currentPersona := activePersona
			tTbl := buildLuaTestHelperTable(ctx, L, h, manifest.ID, &currentPersona)
			fn := L.GetGlobal(fnName)
			callErr := L.CallByParam(lua.P{
				Fn:      fn,
				NRet:    0,
				Protect: true,
			}, tTbl)
			if callErr != nil {
				report.Failed++
				file, line := parseLuaTestErrorLocation(relFile, callErr)
				_, _ = fmt.Fprintf(
					out,
					"type=lua_err plugin=%s test=%s file=%s line=%d error_class=LUA_TEST_ASSERTION_ERROR err=%q\n",
					manifest.ID,
					fnName,
					file,
					line,
					callErr.Error(),
				)
			} else {
				report.Passed++
				_, _ = fmt.Fprintf(
					out,
					"type=test_pass plugin=%s test=%s file=%s status=PASS\n",
					manifest.ID,
					fnName,
					relFile,
				)
			}
		}
		L.Close()
	}

	if report.Failed > 0 {
		return report, fmt.Errorf("%d Lua test(s) failed in %s", report.Failed, manifest.ID)
	}
	return report, nil
}

func buildLuaTestHelperTable(
	ctx context.Context,
	L *lua.LState,
	h *host.Host,
	pluginID string,
	currentPersona *string,
) *lua.LTable {
	tTbl := L.NewTable()

	L.SetField(tTbl, "set_persona", L.NewFunction(func(L *lua.LState) int {
		next := strings.TrimSpace(L.CheckString(1))
		if next == "" {
			L.RaiseError("t.set_persona requires a non-empty staff_id")
			return 0
		}
		*currentPersona = next
		return 0
	}))

	L.SetField(tTbl, "rpc", L.NewFunction(func(L *lua.LState) int {
		method := strings.TrimSpace(L.CheckString(1))
		var argsMap map[string]any
		if L.GetTop() >= 2 && L.Get(2) != lua.LNil {
			tbl := L.CheckTable(2)
			if converted, ok := luaValueToGoValue(tbl).(map[string]any); ok {
				argsMap = converted
			}
		}
		if argsMap == nil {
			argsMap = make(map[string]any)
		}

		personaID := *currentPersona
		if L.GetTop() >= 3 && L.Get(3).Type() == lua.LTString {
			if override := strings.TrimSpace(L.CheckString(3)); override != "" {
				personaID = override
			}
		}

		caller, err := resolveTestPersonaCaller(ctx, h, personaID)
		if err != nil {
			L.RaiseError("failed to resolve persona %s: %v", personaID, err)
			return 0
		}

		resp, rpcErr := h.ExecuteRPC(ctx, pluginID, method, caller, argsMap)
		if rpcErr != nil {
			out := L.NewTable()
			L.SetField(out, "ok", lua.LFalse)
			L.SetField(out, "error_code", lua.LString(rpcErr.Error()))
			L.SetField(out, "error", lua.LString(rpcErr.Error()))
			L.Push(out)
			return 1
		}

		out := L.NewTable()
		L.SetField(out, "ok", lua.LBool(resp.OK))
		if resp.Data != nil {
			L.SetField(out, "data", goValueToLuaValue(L, resp.Data))
		}
		if resp.ErrorCode != "" {
			L.SetField(out, "error_code", lua.LString(resp.ErrorCode))
		}
		if resp.Error != "" {
			L.SetField(out, "error", lua.LString(resp.Error))
		}
		L.Push(out)
		return 1
	}))

	return tTbl
}

func resolveTestPersonaCaller(ctx context.Context, h *host.Host, staffID string) (host.CallerContext, error) {
	persona, err := h.DAL().ResolveStaffPersonaByID(ctx, staffID)
	if err != nil {
		return host.CallerContext{}, err
	}
	return host.CallerContext{
		Type:        "USER",
		UserID:      persona.UserID,
		StaffID:     persona.StaffID,
		ActiveRoles: persona.ActiveRoles,
		Scopes:      persona.Scopes,
		RequestID:   "req_test_" + persona.StaffID,
	}, nil
}

func parseLuaTestErrorLocation(defaultRelFile string, err error) (string, int) {
	if err == nil {
		return defaultRelFile, 0
	}
	normalized := strings.ReplaceAll(err.Error(), "<string>:", defaultRelFile+":")
	if m := luaTestErrLineRegex.FindStringSubmatch(normalized); len(m) == 3 {
		lineNum, _ := strconv.Atoi(m[2])
		return m[1], lineNum
	}
	return defaultRelFile, 0
}

func luaValueToGoValue(val lua.LValue) any {
	switch v := val.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	case *lua.LTable:
		maxn := v.MaxN()
		if maxn > 0 {
			arr := make([]any, 0, maxn)
			for i := 1; i <= maxn; i++ {
				arr = append(arr, luaValueToGoValue(v.RawGetInt(i)))
			}
			return arr
		}
		m := make(map[string]any)
		v.ForEach(func(k, item lua.LValue) {
			if k.Type() == lua.LTString {
				m[k.String()] = luaValueToGoValue(item)
			}
		})
		return m
	default:
		return val.String()
	}
}

func goValueToLuaValue(L *lua.LState, val any) lua.LValue {
	if val == nil {
		return lua.LNil
	}
	switch v := val.(type) {
	case bool:
		return lua.LBool(v)
	case int:
		return lua.LNumber(v)
	case int64:
		return lua.LNumber(v)
	case float64:
		return lua.LNumber(v)
	case string:
		return lua.LString(v)
	case []any:
		tbl := L.NewTable()
		for i, item := range v {
			tbl.RawSetInt(i+1, goValueToLuaValue(L, item))
		}
		return tbl
	case []map[string]any:
		tbl := L.NewTable()
		for i, item := range v {
			tbl.RawSetInt(i+1, goValueToLuaValue(L, item))
		}
		return tbl
	case map[string]any:
		tbl := L.NewTable()
		for k, item := range v {
			L.SetField(tbl, k, goValueToLuaValue(L, item))
		}
		return tbl
	default:
		return lua.LString(fmt.Sprintf("%v", v))
	}
}
