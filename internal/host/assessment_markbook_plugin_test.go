package host_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"optimis/internal/dal"
	"optimis/internal/host"
)

func setupAssessmentMarkbookPluginTestHost(t *testing.T) (*host.Host, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "optimis_assessment_markbook_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	pluginsDir, err := filepath.Abs(filepath.Join("..", "..", "plugins"))
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to resolve plugins dir path: %v", err)
	}

	cfg := host.Config{
		DataDirPath:      tempDir,
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms-markbook-test-key-001",
		PluginsDir:       pluginsDir,
	}

	h, err := host.NewHost(cfg)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to initialize host: %v", err)
	}

	if _, err := h.InstallPlugin(context.Background(), "assessment_markbook_plugin"); err != nil {
		_ = h.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to install assessment_markbook_plugin: %v", err)
	}
	if err := h.ActivatePlugin("uk.org.optimis.assessment.markbook"); err != nil {
		_ = h.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to activate uk.org.optimis.assessment.markbook: %v", err)
	}

	logCursor := h.LogCursor()

	cleanup := func() {
		if t.Failed() {
			logs := h.LogsSince(logCursor)
			t.Logf("--- TAPPED SERVICE LOGS ---\n%s", strings.Join(logs, "\n"))
			t.Logf("--- TAPPED DIAGNOSTIC RING BUFFER ---\n%s", strings.Join(h.Diagnostics(), "\n"))
		}
		_ = h.RemovePlugin(context.Background(), "uk.org.optimis.assessment.markbook")
		_ = h.Close()
		_ = os.RemoveAll(tempDir)
	}

	return h, cleanup
}

func TestAssessmentMarkbookPlugin_MatSchemaSyncOwnGroupsVsAllRollupAndGdprHooks_EnforcesScopesAndRedactsTeacherComments_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupAssessmentMarkbookPluginTestHost(t)
	t.Cleanup(cleanup)

	seeded, err := h.DAL().SeedMilestone1Fixtures(ctx, dal.Milestone1SeedInput{
		AcademicYearID: "ay_2026_2027",
		ReferenceDate:  "2026-10-28",
	})
	if err != nil {
		t.Fatalf("SeedMilestone1Fixtures failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeleteMilestone1Fixtures(context.Background(), seeded.FixtureID)
	})

	teacherCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_mb",
		StaffID:     seeded.TeacherStaffID,
		ActiveRoles: []string{"TEACHER"},
		Scopes: []string{
			"dal.groups.read:own_groups",
			"dal.pupils.read:own_groups",
		},
		RequestID: "req_mb_teacher_1",
	}

	hodCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_hod_mb",
		StaffID:     "stf_hod_mb",
		ActiveRoles: []string{"HOD"},
		Scopes: []string{
			"dal.groups.read:all",
			"dal.pupils.read:all",
		},
		RequestID: "req_mb_hod_1",
	}

	adminCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_admin_mb",
		StaffID:     "stf_admin_mb",
		ActiveRoles: []string{"ADMIN"},
		Scopes: []string{
			"dal.groups.read:all",
			"dal.pupils.read:all",
		},
		RequestID: "req_mb_admin_1",
	}

	// 1. Before MAT grading schema is synced into __system_config, create_markbook_column fails fast with MAT_GRADING_SCHEMA_NOT_SYNCED
	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.assessment.markbook", "create_markbook_column", teacherCaller, map[string]any{
		"group_id":         seeded.OwnGroupID,
		"title":            "Should Fail Without MAT Sync",
		"grading_scale":    "9-1",
		"academic_year_id": seeded.AcademicYearID,
		"reference_date":   seeded.ReferenceDate,
	})
	if err == nil || !strings.Contains(err.Error(), "MAT_GRADING_SCHEMA_NOT_SYNCED") {
		t.Fatalf("expected MAT_GRADING_SCHEMA_NOT_SYNCED error before MAT policy sync, got: %v", err)
	}

	// 2. Sync MAT grading schema into __system_config
	if _, err := h.DAL().SetSystemConfigValue(ctx, "mat_id", "MAT_OPTIMIS_TRUST_01"); err != nil {
		t.Fatalf("SetSystemConfigValue mat_id failed: %v", err)
	}
	if _, err := h.DAL().SetSystemConfigValue(ctx, "mat_grading_schema_version", "KS4_9_TO_1_V2026"); err != nil {
		t.Fatalf("SetSystemConfigValue mat_grading_schema_version failed: %v", err)
	}

	// 3. Teacher (:own_groups) creates a column and saves a grade for own set (OwnGroupID), and is blocked from OtherGroupID and get_department_attainment_rollup
	colResp, err := h.ExecuteRPC(ctx, "uk.org.optimis.assessment.markbook", "create_markbook_column", teacherCaller, map[string]any{
		"group_id":         seeded.OwnGroupID,
		"title":            "Autumn Algebra Paper 1",
		"grading_scale":    "9-1",
		"academic_year_id": seeded.AcademicYearID,
		"reference_date":   seeded.ReferenceDate,
	})
	if err != nil || !colResp.OK {
		t.Fatalf("expected Teacher create_markbook_column on OwnGroupID to succeed, got resp=%+v err=%v", colResp, err)
	}
	colData, _ := colResp.Data.(map[string]any)
	colMap, _ := colData["column"].(map[string]any)
	colID, _ := colMap["id"].(string)

	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.assessment.markbook", "create_markbook_column", teacherCaller, map[string]any{
		"group_id":         seeded.OtherGroupID,
		"title":            "Unauthorized Set 2 Column",
		"grading_scale":    "9-1",
		"academic_year_id": seeded.AcademicYearID,
		"reference_date":   seeded.ReferenceDate,
	})
	if err == nil || !strings.Contains(err.Error(), "access denied under own_groups scope") {
		t.Fatalf("expected Teacher create_markbook_column on OtherGroupID to fail with own_groups error, got: %v", err)
	}

	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.assessment.markbook", "get_department_attainment_rollup", teacherCaller, map[string]any{
		"academic_year_id": seeded.AcademicYearID,
		"reference_date":   seeded.ReferenceDate,
	})
	if err == nil || !strings.Contains(err.Error(), "caller lacks required DAL scopes") {
		t.Fatalf("expected Teacher get_department_attainment_rollup to fail with unauthorized scope, got: %v", err)
	}

	saveResp, err := h.ExecuteRPC(ctx, "uk.org.optimis.assessment.markbook", "save_pupil_grades", teacherCaller, map[string]any{
		"column_id":        colID,
		"group_id":         seeded.OwnGroupID,
		"academic_year_id": seeded.AcademicYearID,
		"reference_date":   seeded.ReferenceDate,
		"entries": []any{
			map[string]any{
				"pupil_id":        seeded.ActivePupilID,
				"grade":           "9",
				"teacher_comment": "Completed higher-tier proof alongside Neville Longbottom.",
			},
		},
	})
	if err != nil || !saveResp.OK {
		t.Fatalf("expected Teacher save_pupil_grades on OwnGroupID to succeed, got resp=%+v err=%v", saveResp, err)
	}

	// 4. HoD (:all) creates column & saves grade on Set 2 (OtherGroupID) and runs cross-set attainment rollup
	hodColResp, err := h.ExecuteRPC(ctx, "uk.org.optimis.assessment.markbook", "create_markbook_column", hodCaller, map[string]any{
		"group_id":         seeded.OtherGroupID,
		"title":            "Autumn Algebra Paper 1",
		"grading_scale":    "9-1",
		"academic_year_id": seeded.AcademicYearID,
		"reference_date":   seeded.ReferenceDate,
	})
	if err != nil || !hodColResp.OK {
		t.Fatalf("expected HoD create_markbook_column on OtherGroupID to succeed, got resp=%+v err=%v", hodColResp, err)
	}
	hodColData, _ := hodColResp.Data.(map[string]any)
	hodColMap, _ := hodColData["column"].(map[string]any)
	hodColID, _ := hodColMap["id"].(string)

	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.assessment.markbook", "save_pupil_grades", hodCaller, map[string]any{
		"column_id":        hodColID,
		"group_id":         seeded.OtherGroupID,
		"academic_year_id": seeded.AcademicYearID,
		"reference_date":   seeded.ReferenceDate,
		"entries": []any{
			map[string]any{
				"pupil_id":        seeded.OtherGroupPupilID,
				"grade":           "6",
				"teacher_comment": "Consistent effort in Set 2.",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected HoD save_pupil_grades on OtherGroupID to succeed, got err=%v", err)
	}

	rollupResp, err := h.ExecuteRPC(ctx, "uk.org.optimis.assessment.markbook", "get_department_attainment_rollup", hodCaller, map[string]any{
		"academic_year_id": seeded.AcademicYearID,
		"reference_date":   seeded.ReferenceDate,
		"group_ids":        []any{seeded.OwnGroupID, seeded.OtherGroupID},
	})
	if err != nil || !rollupResp.OK {
		t.Fatalf("expected HoD get_department_attainment_rollup to succeed, got resp=%+v err=%v", rollupResp, err)
	}
	rollupData, _ := rollupResp.Data.(map[string]any)
	if rollupData["highest_attaining_group_id"] != seeded.OwnGroupID {
		t.Errorf("expected highest_attaining_group_id=%s, got %v", seeded.OwnGroupID, rollupData["highest_attaining_group_id"])
	}

	// 5. Verify UK GDPR export_pupil_data (redactable teacher_comment) and purge_pupil_data
	exportResp, err := h.ExecuteRPC(ctx, "uk.org.optimis.assessment.markbook", "export_pupil_data", adminCaller, map[string]any{
		"pupil_id": seeded.ActivePupilID,
	})
	if err != nil || !exportResp.OK {
		t.Fatalf("export_pupil_data failed: %v", err)
	}
	sections, _ := exportResp.Data.([]any)
	if len(sections) != 1 {
		t.Fatalf("expected 1 SAR section, got %d", len(sections))
	}
	sec0, _ := sections[0].(map[string]any)
	records, _ := sec0["records"].([]any)
	if len(records) != 1 {
		t.Fatalf("expected 1 grade record before purge, got %d", len(records))
	}

	purgeResp, err := h.ExecuteRPC(ctx, "uk.org.optimis.assessment.markbook", "purge_pupil_data", adminCaller, map[string]any{
		"pupil_id": seeded.ActivePupilID,
	})
	if err != nil || !purgeResp.OK {
		t.Fatalf("purge_pupil_data failed: %v", err)
	}

	exportAfterResp, err := h.ExecuteRPC(ctx, "uk.org.optimis.assessment.markbook", "export_pupil_data", adminCaller, map[string]any{
		"pupil_id": seeded.ActivePupilID,
	})
	if err != nil || !exportAfterResp.OK {
		t.Fatalf("export_pupil_data after purge failed: %v", err)
	}
	sectionsAfter, _ := exportAfterResp.Data.([]any)
	secAfter0, _ := sectionsAfter[0].(map[string]any)
	recordsAfter, _ := secAfter0["records"].([]any)
	if len(recordsAfter) != 0 {
		t.Fatalf("expected 0 records after purge_pupil_data, got %d", len(recordsAfter))
	}
}
