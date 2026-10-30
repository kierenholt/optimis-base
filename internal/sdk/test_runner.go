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

	"optimis/internal/dal"
	"optimis/internal/host"
)

var luaTestErrLineRegex = regexp.MustCompile(`([^\s:]+\.lua):(\d+):`)

type TestRunReport struct {
	PluginID string
	Passed   int
	Failed   int
}

func SeedSyntheticDevCohort(ctx context.Context, coreDAL *dal.CoreDAL) error {
	tx, err := coreDAL.WriteDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const ayID = "ay_2026_2027"
	const effFrom = "2026-09-01"

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO academic_years (id, name, status, sequence_number, first_day, last_day)
		 VALUES (?, '2026/2027', 'CURRENT', (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM academic_years), ?, '2027-07-23')
		 ON CONFLICT(id) DO NOTHING`,
		ayID, effFrom,
	); err != nil {
		return err
	}

	roles := []struct {
		id, cat, name, short, desc, scopes string
	}{
		{
			"ADMIN", "STAFF", "System Administrator / Headteacher", "ADM", "Full school administrative access",
			`["dal.census.read:all","dal.pupils.read:all","dal.pupils.write:all","dal.staff.read:admin","dal.staff.write:admin","dal.groups.read:all","dal.groups.write:all","dal.attendance.read:all","dal.attendance.write:all","dal.guardians.read:all","dal.guardians.write:all","dal.sen.read:all","dal.medical.read:all","plugin.config.write"]`,
		},
		{
			"DSL", "STAFF", "Designated Safeguarding Lead", "DSL", "KCSIE Safeguarding Vault read & triage authority",
			`["dal.safeguarding.create","dal.safeguarding.read","dal.pupils.read:all","dal.guardians.read:all","dal.attendance.read:all","dal.sen.read:all","dal.medical.read:all"]`,
		},
		{
			"HOD", "STAFF", "Head of Department", "HOD", "Departmental curriculum and cohort lead",
			`["dal.pupils.read:all","dal.groups.read:all","dal.groups.write:all","dal.attendance.read:all","dal.sen.read:all","dal.medical.read:all","dal.safeguarding.create","plugin.config.write"]`,
		},
		{
			"TEACHER", "STAFF", "Classroom Teacher", "TCH", "Scoped access to assigned teaching and registration groups",
			`["dal.pupils.read:own_groups","dal.groups.read:own_groups","dal.attendance.read:own_groups","dal.attendance.write:own_groups","dal.guardians.read:own_groups","dal.sen.read:own_groups","dal.medical.read:own_groups","dal.safeguarding.create"]`,
		},
		{
			"SUPPORT", "STAFF", "School Office & Pastoral Support", "SUP", "Attendance officer, First Aider, and administrative support",
			`["dal.pupils.read:all","dal.attendance.read:all","dal.attendance.write:all","dal.guardians.read:all","dal.medical.read:all","dal.safeguarding.create"]`,
		},
	}
	for _, r := range roles {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO roles (id, category, name, name_short, description, default_scopes_json)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET default_scopes_json = excluded.default_scopes_json`,
			r.id, r.cat, r.name, r.short, r.desc, r.scopes,
		); err != nil {
			return err
		}
	}

	staffRows := []struct {
		staffID, personID, code, title, first, last, staffType, jobTitle, roleID string
		isDSL                                                                    int
	}{
		{"stf_admin_01", "per_sdk_adm01", "ADM01", "Dame", "Eleanor", "Vance", "LEADERSHIP", "Executive Headteacher", "ADMIN", 1},
		{"stf_dsl_01", "per_sdk_dsl01", "DSL01", "Mr", "Marcus", "Thorne", "LEADERSHIP", "Designated Safeguarding Lead", "DSL", 1},
		{"stf_senco_01", "per_sdk_sen01", "SEN01", "Dr", "Amina", "Patel", "LEADERSHIP", "SENCO", "HOD", 1},
		{"stf_hod_maths", "per_sdk_hod01", "HOD01", "Mr", "David", "Sterling", "TEACHING", "Head of Mathematics", "HOD", 0},
		{"stf_tch_7a", "per_sdk_tch7a", "TCH7A", "Ms", "Hannah", "Abbott", "TEACHING", "Form Tutor 7A", "TEACHER", 0},
		{"stf_tch_7b", "per_sdk_tch7b", "TCH7B", "Mr", "Neville", "Longbottom", "TEACHING", "Form Tutor 7B", "TEACHER", 0},
		{"stf_support_01", "per_sdk_sup01", "SUP01", "Mrs", "Clara", "Oswald", "SUPPORT", "Attendance & Medical Officer", "SUPPORT", 0},
	}

	for _, s := range staffRows {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO persons (id, external_sso_sub, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, status)
			 VALUES (?, ?, ?, ?, ?, ?, ?, 'U', 'ACTIVE')
			 ON CONFLICT(id) DO NOTHING`,
			s.personID, "sso_"+s.staffID, s.title, s.first, s.last, s.first, s.last,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from)
			 VALUES (?, ?, 1, ?)
			 ON CONFLICT(person_id, role_id, effective_from) DO NOTHING`,
			s.personID, s.roleID, effFrom,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO staff (id, person_id, staff_code, staff_type, job_title, qts_flag, is_dsl, hire_date)
			 VALUES (?, ?, ?, ?, ?, 1, ?, ?)
			 ON CONFLICT(id) DO NOTHING`,
			s.staffID, s.personID, s.code, s.staffType, s.jobTitle, s.isDSL, effFrom,
		); err != nil {
			return err
		}
	}

	groups := []struct {
		id, groupType, code, name, leadStaffID string
	}{
		{"grp_reg_7a", "REGISTRATION_FORM", "7A", "Year 7 Form 7A", "stf_tch_7a"},
		{"grp_reg_7b", "REGISTRATION_FORM", "7B", "Year 7 Form 7B", "stf_tch_7b"},
		{"grp_mat_7x1", "TEACHING_CLASS", "7Ma/X1", "Year 7 Maths Set 1", "stf_tch_7a"},
	}
	for _, g := range groups {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO groups (id, academic_year_id, group_type, code, name, takes_attendance, effective_from, effective_to)
			 VALUES (?, ?, ?, ?, ?, 1, ?, NULL)
			 ON CONFLICT(id) DO NOTHING`,
			g.id, ayID, g.groupType, g.code, g.name, effFrom,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO group_staff_assignments (id, group_id, staff_id, role, academic_year_id, effective_from, effective_to)
			 VALUES (?, ?, ?, 'LEAD_TEACHER', ?, ?, NULL)
			 ON CONFLICT(id) DO NOTHING`,
			"gsa_"+g.id, g.id, g.leadStaffID, ayID, effFrom,
		); err != nil {
			return err
		}
	}

	pupils := []struct {
		pupilID, personID, upn, first, last, yearGroup, regGroupID string
	}{
		{"pup_yr7_alice", "per_sdk_alice", "U100001202601", "Alice", "iddell", "7", "grp_reg_7a"},
		{"pup_yr7_noah", "per_sdk_noah", "U100001202602", "Noah", "Patel", "7", "grp_reg_7a"},
		{"pup_yr7_zara", "per_sdk_zara", "U100001202603", "Zara", "Khan", "7", "grp_reg_7b"},
		{"pup_pri_maya", "per_sdk_maya", "U100001202604", "Maya", "Lin", "4", "grp_reg_7a"},
	}
	for _, p := range pupils {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO persons (id, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, status)
			 VALUES (?, ?, ?, ?, ?, 'F', 'ACTIVE')
			 ON CONFLICT(id) DO NOTHING`,
			p.personID, p.first, p.last, p.first, p.last,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO pupils (id, person_id, upn, ethnicity_code, first_language_code)
			 VALUES (?, ?, ?, 'WBRI', 'ENG')
			 ON CONFLICT(id) DO NOTHING`,
			p.pupilID, p.personID, p.upn,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, enrolment_status, effective_from, effective_to)
			 VALUES (?, ?, ?, ?, 'C', ?, NULL)
			 ON CONFLICT(id) DO NOTHING`,
			"enr_"+p.pupilID, p.pupilID, ayID, p.yearGroup, effFrom,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from, effective_to)
			 VALUES (?, ?, ?, ?, ?, NULL)
			 ON CONFLICT(id) DO NOTHING`,
			"gpm_"+p.pupilID+"_"+p.regGroupID, p.regGroupID, p.pupilID, ayID, effFrom,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
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
