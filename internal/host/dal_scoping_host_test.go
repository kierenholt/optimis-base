package host_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"optimis/internal/host"
)

type hostMilestone1Fixture struct {
	Host               *host.Host
	TempDir            string
	AcademicYearID     string
	ReferenceDate      string
	TeacherStaffID     string
	OtherTeacherID     string
	DSLStaffID         string
	OwnGroupID         string
	OtherGroupID       string
	ActivePupilID      string
	TransferredPupilID string
	OtherGroupPupilID  string
}

func setupMilestone1HostTest(t *testing.T, manifestJSON string, luaSource string) *hostMilestone1Fixture {
	t.Helper()

	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	pluginDir := filepath.Join(pluginsDir, "m1_plugin")
	if err := os.MkdirAll(filepath.Join(pluginDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create temp plugin dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("failed to write manifest.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "src", "init.lua"), []byte(luaSource), 0o644); err != nil {
		t.Fatalf("failed to write init.lua: %v", err)
	}

	h, err := host.NewHost(host.Config{
		DataDirPath:      filepath.Join(tempDir, "data"),
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms-host-test-key-001",
		PluginsDir:       pluginsDir,
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}

	if err := h.LoadPlugin("m1_plugin"); err != nil {
		_ = h.Close()
		t.Fatalf("failed to load m1_plugin: %v", err)
	}

	fx := &hostMilestone1Fixture{
		Host:               h,
		TempDir:            tempDir,
		AcademicYearID:     "ay_2026_2027",
		ReferenceDate:      "2026-10-27",
		TeacherStaffID:     "stf_teacher_1",
		OtherTeacherID:     "stf_teacher_2",
		DSLStaffID:         "stf_dsl_1",
		OwnGroupID:         "grp_10ma1",
		OtherGroupID:       "grp_10ma2",
		ActivePupilID:      "pup_active_1",
		TransferredPupilID: "pup_transferred_2",
		OtherGroupPupilID:  "pup_other_3",
	}

	seedStmts := []string{
		`INSERT INTO academic_years (id, name, status, sequence_number, first_day, last_day)
		 VALUES ('ay_2026_2027', '2026/2027', 'CURRENT', 1, '2026-09-01', '2027-07-23')`,
		`INSERT INTO days_of_week (id, name, name_short, is_school_day, school_open_time, am_register_time, pm_register_time, school_close_time)
		 VALUES (1, 'Monday', 'Mon', 1, '08:00', '08:45', '13:15', '15:30')`,
		`INSERT INTO alert_levels (id, name, name_short, color_hex, bg_color_hex, description, sequence_number)
		 VALUES ('HIGH', 'High Priority', 'HI', '#FFFFFF', '#D32F2F', 'Critical alert', 1)`,
		`INSERT INTO sen_need_types (code, name, broad_area, sequence_number)
		 VALUES ('ASD', 'Autistic Spectrum Disorder', 'COMMUNICATION_AND_INTERACTION', 1)`,
		`INSERT INTO attendance_codes (code, name, statistical_category, physical_status, sequence_number)
		 VALUES ('/', 'Present (AM)', 'PRESENT', 'ONSITE', 1),
		        ('\', 'Present (PM)', 'PRESENT', 'ONSITE', 2)`,
		`INSERT INTO persons (id, legal_first_name, legal_last_name, sex, dob, status)
		 VALUES ('per_pupil_1', 'Ada', 'Lovelace', 'F', '2012-12-10', 'ACTIVE'),
		        ('per_pupil_2', 'Charles', 'Babbage', 'M', '2012-10-15', 'ACTIVE'),
		        ('per_pupil_3', 'Alan', 'Turing', 'M', '2012-06-23', 'ACTIVE'),
		        ('per_teacher_1', 'Grace', 'Hopper', 'F', '1985-05-01', 'ACTIVE'),
		        ('per_teacher_2', 'Katherine', 'Johnson', 'F', '1984-08-26', 'ACTIVE'),
		        ('per_dsl_1', 'Margaret', 'Hamilton', 'F', '1980-08-17', 'ACTIVE'),
		        ('per_mother_1', 'Anne', 'Byron', 'F', '1988-03-14', 'ACTIVE'),
		        ('per_father_1', 'George', 'Byron', 'M', '1987-01-22', 'ACTIVE')`,
		`INSERT INTO pupils (id, person_id, upn, ethnicity_code, first_language_code)
		 VALUES ('pup_active_1', 'per_pupil_1', 'H823400026001', 'WBRI', 'ENG'),
		        ('pup_transferred_2', 'per_pupil_2', 'H823400026002', 'WBRI', 'ENG'),
		        ('pup_other_3', 'per_pupil_3', 'H823400026003', 'WBRI', 'ENG')`,
		`INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, enrolment_status, effective_from, effective_to)
		 VALUES ('enr_1', 'pup_active_1', 'ay_2026_2027', '10', 'C', '2026-09-01', NULL),
		        ('enr_2', 'pup_transferred_2', 'ay_2026_2027', '10', 'C', '2026-09-01', NULL),
		        ('enr_3', 'pup_other_3', 'ay_2026_2027', '10', 'C', '2026-09-01', NULL)`,
		`INSERT INTO staff (id, person_id, staff_code, trn, staff_type, job_title, qts_flag, is_dsl, hire_date)
		 VALUES ('stf_teacher_1', 'per_teacher_1', 'GHO', '7654321', 'TEACHING', 'Teacher of Maths', 1, 0, '2020-09-01'),
		        ('stf_teacher_2', 'per_teacher_2', 'KJO', '7654322', 'TEACHING', 'Teacher of Maths', 1, 0, '2021-09-01'),
		        ('stf_dsl_1', 'per_dsl_1', 'MHA', '1122334', 'LEADERSHIP', 'DSL', 1, 1, '2018-09-01')`,
		`INSERT INTO groups (id, academic_year_id, group_type, code, name, effective_from, effective_to)
		 VALUES ('grp_10ma1', 'ay_2026_2027', 'TEACHING_CLASS', '10Ma1', 'Year 10 Maths Set 1', '2026-09-01', NULL),
		        ('grp_10ma2', 'ay_2026_2027', 'TEACHING_CLASS', '10Ma2', 'Year 10 Maths Set 2', '2026-09-01', NULL)`,
		`INSERT INTO group_staff_assignments (id, group_id, staff_id, role, academic_year_id, effective_from, effective_to)
		 VALUES ('gsa_1', 'grp_10ma1', 'stf_teacher_1', 'LEAD_TEACHER', 'ay_2026_2027', '2026-09-01', NULL),
		        ('gsa_2', 'grp_10ma2', 'stf_teacher_2', 'LEAD_TEACHER', 'ay_2026_2027', '2026-09-01', NULL)`,
		`INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from, effective_to)
		 VALUES ('gpm_active_1', 'grp_10ma1', 'pup_active_1', 'ay_2026_2027', '2026-09-01', NULL),
		        ('gpm_trans_old', 'grp_10ma1', 'pup_transferred_2', 'ay_2026_2027', '2026-09-01', '2026-10-15'),
		        ('gpm_trans_new', 'grp_10ma2', 'pup_transferred_2', 'ay_2026_2027', '2026-10-16', NULL),
		        ('gpm_other_3', 'grp_10ma2', 'pup_other_3', 'ay_2026_2027', '2026-09-01', NULL)`,
	}

	for _, stmt := range seedStmts {
		if _, err := h.DAL().WriteDB().Exec(stmt); err != nil {
			_ = h.Close()
			t.Fatalf("failed to seed host test SQL (%s): %v", stmt, err)
		}
	}

	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("--- TAPPED SERVICE LOGS ---\n%s", strings.Join(h.LogsSince("0"), "\n"))
			t.Logf("--- TAPPED DIAGNOSTIC RING BUFFER ---\n%s", strings.Join(h.Diagnostics(), "\n"))
		}
		_ = h.Close()
	})

	return fx
}

func TestBuildLuaCtxTable_EndpointDeclaresOnlyPupilsAndGroupsOwnGroups_PopulatesDeclaredDomainsAndOmitsUndeclaredDomains_test(t *testing.T) {
	ctx := context.Background()

	manifest := `{
  "id": "uk.org.optimis.m1",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "egress": { "allowed_domains": [] },
  "endpoints": {
    "inspect_dal_namespaces": {
      "scopes": ["dal.pupils.read:own_groups", "dal.groups.read:own_groups"],
      "mutates": false
    }
  }
}`
	luaSrc := `
function inspect_dal_namespaces(ctx, args)
  return {
    ok = true,
    data = {
      has_pupils = (type(ctx.dal.pupils) == "table"),
      has_groups = (type(ctx.dal.groups) == "table"),
      has_attendance = (ctx.dal.attendance ~= nil),
      has_guardians = (ctx.dal.guardians ~= nil),
      has_safeguarding = (ctx.dal.safeguarding ~= nil)
    }
  }
end
`
	fx := setupMilestone1HostTest(t, manifest, luaSrc)

	caller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_1",
		StaffID:     fx.TeacherStaffID,
		ActiveRoles: []string{"teacher"},
		Scopes:      []string{"dal.pupils.read:own_groups", "dal.groups.read:own_groups"},
		RequestID:   "req_m1_namespaces",
	}

	res, err := fx.Host.ExecuteRPC(ctx, "uk.org.optimis.m1", "inspect_dal_namespaces", caller, map[string]any{})
	if err != nil {
		t.Fatalf("ExecuteRPC failed: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected res.OK == true, got error_code=%s", res.ErrorCode)
	}

	data := res.Data.(map[string]any)
	if data["has_pupils"] != true || data["has_groups"] != true {
		t.Errorf("expected declared domains pupils and groups to be populated on ctx.dal, got %+v", data)
	}
	if data["has_attendance"] != false || data["has_guardians"] != false || data["has_safeguarding"] != false {
		t.Errorf("SECURITY VIOLATION: undeclared domains must be nil on ctx.dal, got %+v", data)
	}
}

func TestExecuteRPC_LuaCallsPupilsListByGroupWithOwnGroupsScope_EnforcesCallerStaffIDAndTemporalMembership_test(t *testing.T) {
	ctx := context.Background()

	manifest := `{
  "id": "uk.org.optimis.m1",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "egress": { "allowed_domains": [] },
  "endpoints": {
    "fetch_group_pupils": {
      "scopes": ["dal.pupils.read:own_groups", "dal.groups.read:own_groups"],
      "mutates": false
    }
  }
}`
	luaSrc := `
function fetch_group_pupils(ctx, args)
  local pupils = ctx.dal.pupils.list_by_group(args.group_id, {
    reference_date = args.reference_date,
    academic_year_id = args.academic_year_id
  })
  return {
    ok = true,
    data = {
      count = #pupils,
      first_pupil_id = pupils[1] and pupils[1].id or ""
    }
  }
end
`
	fx := setupMilestone1HostTest(t, manifest, luaSrc)

	caller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_1",
		StaffID:     fx.TeacherStaffID,
		ActiveRoles: []string{"teacher"},
		Scopes:      []string{"dal.pupils.read:own_groups", "dal.groups.read:own_groups"},
		RequestID:   "req_m1_own_group",
	}

	// 1. Teacher 1 querying own group (grp_10ma1) on 2026-10-27 returns only active pupil (Ada)
	res, err := fx.Host.ExecuteRPC(ctx, "uk.org.optimis.m1", "fetch_group_pupils", caller, map[string]any{
		"group_id":         fx.OwnGroupID,
		"reference_date":   fx.ReferenceDate,
		"academic_year_id": fx.AcademicYearID,
	})
	if err != nil {
		t.Fatalf("ExecuteRPC for own group failed: %v", err)
	}
	data := res.Data.(map[string]any)
	if data["count"] != float64(1) || data["first_pupil_id"] != fx.ActivePupilID {
		t.Fatalf("expected 1 active pupil (%s) in own group, got %+v", fx.ActivePupilID, data)
	}

	// 2. Teacher 1 querying Teacher 2's group (grp_10ma2) under :own_groups MUST fail fast
	_, err = fx.Host.ExecuteRPC(ctx, "uk.org.optimis.m1", "fetch_group_pupils", caller, map[string]any{
		"group_id":         fx.OtherGroupID,
		"reference_date":   fx.ReferenceDate,
		"academic_year_id": fx.AcademicYearID,
	})
	if err == nil {
		t.Fatalf("expected fail-fast error when teacher with :own_groups queries another teacher's group %s", fx.OtherGroupID)
	}
}

func TestExecuteRPC_LuaCallsAttendanceRecordSessionBatchWithMutatesFalse_FailsFastAndRecordsDiagnostic_test(t *testing.T) {
	ctx := context.Background()

	manifest := `{
  "id": "uk.org.optimis.m1",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "egress": { "allowed_domains": [] },
  "endpoints": {
    "illegal_mutation_in_readonly_endpoint": {
      "scopes": ["dal.attendance.read:own_groups", "dal.attendance.write:own_groups"],
      "mutates": false
    }
  }
}`
	luaSrc := `
function illegal_mutation_in_readonly_endpoint(ctx, args)
  ctx.dal.attendance.record_session_batch(args.group_id, args.date, "AM", {
    { pupil_id = args.pupil_id, attendance_code = "/" }
  })
  return { ok = true, data = {} }
end
`
	fx := setupMilestone1HostTest(t, manifest, luaSrc)

	caller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_1",
		StaffID:     fx.TeacherStaffID,
		ActiveRoles: []string{"teacher"},
		Scopes:      []string{"dal.attendance.read:own_groups", "dal.attendance.write:own_groups"},
		RequestID:   "req_m1_illegal_mutate",
	}

	_, err := fx.Host.ExecuteRPC(ctx, "uk.org.optimis.m1", "illegal_mutation_in_readonly_endpoint", caller, map[string]any{
		"group_id": fx.OwnGroupID,
		"date":     fx.ReferenceDate,
		"pupil_id": fx.ActivePupilID,
	})
	if err == nil {
		t.Fatalf("expected fail-fast error when endpoint with mutates=false attempts to invoke record_session_batch")
	}
	if len(fx.Host.Diagnostics()) == 0 {
		t.Fatalf("expected diagnostic ring buffer entry to be recorded for failed mutation in read-only endpoint")
	}
}

func TestExecuteRPC_LuaCallsSafeguardingCreatePostbox_WritesEncryptedRecordAndExposesZeroSafeguardingReadFunctionToLua_test(t *testing.T) {
	ctx := context.Background()

	manifest := `{
  "id": "uk.org.optimis.m1",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "egress": { "allowed_domains": [] },
  "endpoints": {
    "submit_postbox": {
      "scopes": ["dal.safeguarding.create"],
      "mutates": true
    }
  }
}`
	luaSrc := `
function submit_postbox(ctx, args)
  local has_read_fn = (ctx.dal.safeguarding.read ~= nil or ctx.dal.safeguarding.get_by_id ~= nil)
  local receipt = ctx.dal.safeguarding.create_postbox({
    pupil_id = args.pupil_id,
    assigned_dsl_staff_id = args.dsl_staff_id,
    concern_category = "EMOTIONAL",
    incident_date = args.incident_date,
    summary = "Teacher postbox concern",
    verbatim_account = "Pupil upset in corridor",
    action_taken = "Reported to DSL"
  })
  return {
    ok = true,
    data = {
      note_id = receipt.note_id,
      status = receipt.status,
      has_read_fn = has_read_fn
    }
  }
end
`
	fx := setupMilestone1HostTest(t, manifest, luaSrc)

	caller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_1",
		StaffID:     fx.TeacherStaffID,
		ActiveRoles: []string{"teacher"},
		Scopes:      []string{"dal.safeguarding.create"},
		RequestID:   "req_m1_sfg_postbox",
	}

	res, err := fx.Host.ExecuteRPC(ctx, "uk.org.optimis.m1", "submit_postbox", caller, map[string]any{
		"pupil_id":      fx.ActivePupilID,
		"dsl_staff_id":  fx.DSLStaffID,
		"incident_date": fx.ReferenceDate,
	})
	if err != nil {
		t.Fatalf("ExecuteRPC submit_postbox failed: %v", err)
	}
	data := res.Data.(map[string]any)
	if data["has_read_fn"] != false {
		t.Fatalf("SECURITY VIOLATION: ctx.dal.safeguarding exposed a read function to the Lua sandbox")
	}
	noteID, _ := data["note_id"].(string)
	if !strings.HasPrefix(noteID, "sfg_") {
		t.Fatalf("expected safeguarding note_id starting with 'sfg_', got %q", noteID)
	}
}

func TestExecuteWorker_BackgroundWorkerWithManifestScopes_EnforcesPluginServicePrincipalScopesAndNullStaffID_test(t *testing.T) {
	ctx := context.Background()

	manifest := `{
  "id": "uk.org.optimis.m1",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "egress": { "allowed_domains": [] },
  "endpoints": {
    "noop": {
      "scopes": ["dal.groups.read:all"],
      "mutates": false
    }
  },
  "workers": {
    "nightly_group_sync": {
      "scopes": ["dal.groups.read:all", "dal.pupils.read:all"],
      "mutates": false
    }
  }
}`
	luaSrc := `
function noop(ctx, args)
  return { ok = true, data = {} }
end

function nightly_group_sync(ctx, args)
  local groups = ctx.dal.groups.list_groups({
    academic_year_id = args.academic_year_id,
    reference_date = args.reference_date,
    group_type = "TEACHING_CLASS"
  })
  return {
    ok = true,
    data = {
      caller_type = ctx.caller.type,
      caller_staff_id = ctx.caller.staff_id,
      group_count = #groups
    }
  }
end
`
	fx := setupMilestone1HostTest(t, manifest, luaSrc)

	res, err := fx.Host.ExecuteWorker(ctx, "uk.org.optimis.m1", "nightly_group_sync", map[string]any{
		"academic_year_id": fx.AcademicYearID,
		"reference_date":   fx.ReferenceDate,
	})
	if err != nil {
		t.Fatalf("ExecuteWorker failed: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected worker res.OK == true, got %s", res.ErrorCode)
	}

	data := res.Data.(map[string]any)
	if data["caller_type"] != "SYSTEM" {
		t.Errorf("expected worker ctx.caller.type == 'SYSTEM', got %v", data["caller_type"])
	}
	if data["caller_staff_id"] != "" {
		t.Errorf("expected worker ctx.caller.staff_id to be empty string per GEMINI.md §9.2, got %q", data["caller_staff_id"])
	}
	if data["group_count"] != float64(2) {
		t.Errorf("expected worker with :all scope to see 2 teaching groups, got %v", data["group_count"])
	}
}

func TestRecordDiagnostic_LuaRuntimeErrorDuringDalCall_ScrubsPupilIDsAndLocalVariablesFromRingBuffer_test(t *testing.T) {
	ctx := context.Background()

	manifest := `{
  "id": "uk.org.optimis.m1",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "egress": { "allowed_domains": [] },
  "endpoints": {
    "crash_with_sensitive_pii": {
      "scopes": ["dal.pupils.read:own_groups"],
      "mutates": false
    }
  }
}`
	luaSrc := `
function crash_with_sensitive_pii(ctx, args)
  local secret_pupil_id = "pup_active_1"
  local secret_upn = "H823400026001"
  local secret_name = "Ada Lovelace"
  error("crash while processing " .. secret_pupil_id .. " " .. secret_upn .. " " .. secret_name)
end
`
	fx := setupMilestone1HostTest(t, manifest, luaSrc)

	caller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_1",
		StaffID:     fx.TeacherStaffID,
		ActiveRoles: []string{"teacher"},
		Scopes:      []string{"dal.pupils.read:own_groups"},
		RequestID:   "req_m1_pii_scrub",
	}

	_, err := fx.Host.ExecuteRPC(ctx, "uk.org.optimis.m1", "crash_with_sensitive_pii", caller, map[string]any{
		"pupil_id": fx.ActivePupilID,
	})
	if err == nil {
		t.Fatalf("expected error from crashing Lua endpoint")
	}

	diags := fx.Host.Diagnostics()
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic entry to be recorded in ring buffer")
	}

	for _, d := range diags {
		if strings.Contains(d, "pup_active_1") || strings.Contains(d, "H823400026001") || strings.Contains(d, "Ada Lovelace") {
			t.Fatalf("SECURITY VIOLATION: diagnostic ring buffer entry leaked pupil PII: %q", d)
		}
	}
}
