package host_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"optimis/internal/dal"
	"optimis/internal/host"
)

func setupSparxMathsPluginTestHost(t *testing.T) (*host.Host, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "optimis_sparx_maths_test_*")
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
		HostKMSKeyHandle: "kms-sparx-maths-test-key-001",
		PluginsDir:       pluginsDir,
	}

	h, err := host.NewHost(cfg)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to initialize host: %v", err)
	}

	h.ConfigureEgressMode(false)

	if _, err := h.InstallPlugin(context.Background(), "sparx_maths_plugin"); err != nil {
		_ = h.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to install sparx_maths_plugin: %v", err)
	}
	if err := h.ActivatePlugin("uk.co.sparx.maths"); err != nil {
		_ = h.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to activate uk.co.sparx.maths: %v", err)
	}

	logCursor := h.LogCursor()

	cleanup := func() {
		if t.Failed() {
			logs := h.LogsSince(logCursor)
			t.Logf("--- TAPPED SERVICE LOGS ---\n%s", strings.Join(logs, "\n"))
			t.Logf("--- TAPPED DIAGNOSTIC RING BUFFER ---\n%s", strings.Join(h.Diagnostics(), "\n"))
		}
		_ = h.RemovePlugin(context.Background(), "uk.co.sparx.maths")
		_ = h.Close()
		_ = os.RemoveAll(tempDir)
	}

	return h, cleanup
}

func TestSparxMathsPlugin_TwoPhaseSyncDiagnosticApiKeyScrubbingAndGdprHooks_PersistsScoresAndStripsSecrets_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupSparxMathsPluginTestHost(t)
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

	const sparxURL = "https://sync.sparxmaths.uk/v1/homework/sync"
	const secretAPIKey = "spx_live_secret_go_test_key_8877665544332211"

	if err := h.RegisterMockEgressRule(host.MockEgressRule{
		Method: "POST",
		URL:    sparxURL,
		Status: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"sync_batch_id":"sparx_batch_go_01","homework_week":"2026-W44","scores":[{"pupil_id":"` + seeded.ActivePupilID + `","homework_week":"2026-W44","completion_rate":94,"xp_earned":390,"bookwork_accuracy":91,"teacher_note":"Completed all compulsory tasks with Ada Lovelace"}]}`,
	}); err != nil {
		t.Fatalf("RegisterMockEgressRule failed: %v", err)
	}
	t.Cleanup(func() {
		h.ClearMockEgressRules()
	})

	teacherCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_t1_" + seeded.FixtureID,
		StaffID:     seeded.TeacherStaffID,
		ActiveRoles: []string{"TEACHER"},
		Scopes: []string{
			"dal.groups.read:own_groups",
			"dal.pupils.read:own_groups",
		},
		RequestID: "req_sparx_t1",
	}

	adminCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_admin_sparx",
		StaffID:     "stf_admin_sparx",
		ActiveRoles: []string{"ADMIN"},
		Scopes: []string{
			"dal.groups.read:all",
			"dal.pupils.read:all",
		},
		RequestID: "req_sparx_admin",
	}

	// 1. Configure vendor API key in plugin_sparx.db (committed before HTTP call)
	cfgResp, err := h.ExecuteRPC(ctx, "uk.co.sparx.maths", "configure_vendor_credentials", teacherCaller, map[string]any{
		"api_key":           secretAPIKey,
		"sync_endpoint_url": sparxURL,
	})
	if err != nil || !cfgResp.OK {
		t.Fatalf("expected configure_vendor_credentials to succeed, got resp=%+v err=%v", cfgResp, err)
	}

	// 2. Trigger two-phase Read/Egress -> WriteDB sync
	syncResp, err := h.ExecuteRPC(ctx, "uk.co.sparx.maths", "sync_group_homework", teacherCaller, map[string]any{
		"group_id":         seeded.OwnGroupID,
		"academic_year_id": seeded.AcademicYearID,
		"reference_date":   seeded.ReferenceDate,
		"homework_week":    "2026-W44",
	})
	if err != nil || !syncResp.OK {
		t.Fatalf("expected sync_group_homework to succeed, got resp=%+v err=%v", syncResp, err)
	}
	syncData, _ := syncResp.Data.(map[string]any)
	if syncData["two_phase_pattern"] != "READ_EGRESS_THEN_WRITEDB_OK" {
		t.Errorf("expected two_phase_pattern READ_EGRESS_THEN_WRITEDB_OK, got %v", syncData["two_phase_pattern"])
	}
	if syncData["persisted_scores_count"] != float64(1) {
		t.Errorf("expected persisted_scores_count 1, got %v", syncData["persisted_scores_count"])
	}

	// 3. Trigger simulated fault with stored API key & verify WriteDB-before-egress guard
	_, faultErr := h.ExecuteRPC(ctx, "uk.co.sparx.maths", "sync_group_homework", teacherCaller, map[string]any{
		"group_id":         seeded.OwnGroupID,
		"academic_year_id": seeded.AcademicYearID,
		"reference_date":   seeded.ReferenceDate,
		"homework_week":    "2026-W44",
		"simulate_fault":   "VENDOR_AUTH_FAULT_WITH_STORED_KEY",
	})
	if faultErr == nil {
		t.Fatalf("expected simulated fault to return an error")
	}

	_, txFaultErr := h.ExecuteRPC(ctx, "uk.co.sparx.maths", "sync_group_homework", teacherCaller, map[string]any{
		"group_id":         seeded.OwnGroupID,
		"academic_year_id": seeded.AcademicYearID,
		"reference_date":   seeded.ReferenceDate,
		"homework_week":    "2026-W44",
		"simulate_fault":   "WRITE_TX_BEFORE_EGRESS",
	})
	if txFaultErr == nil || !strings.Contains(txFaultErr.Error(), "disallowed inside an open SQLite write transaction") {
		t.Fatalf("expected open SQLite write transaction halt error, got %v", txFaultErr)
	}

	// 4. Export Diagnostic Bundle and verify vendor API key and pupil IDs are scrubbed
	bundle, err := h.ExportDiagnosticBundle(adminCaller, "uk.co.sparx.maths")
	if err != nil {
		t.Fatalf("ExportDiagnosticBundle failed: %v", err)
	}
	bundleJSONBytes, _ := json.Marshal(bundle)
	bundleJSON := string(bundleJSONBytes)

	for _, forbidden := range []string{
		secretAPIKey,
		"8877665544332211",
		seeded.ActivePupilID,
	} {
		if strings.Contains(bundleJSON, forbidden) || strings.Contains(bundle.LogArchive, forbidden) {
			t.Errorf("diagnostic bundle leaked forbidden secret/PII %q", forbidden)
		}
	}

	// 5. Verify UK GDPR export_pupil_data & purge_pupil_data hooks
	exportResp, err := h.ExecuteRPC(ctx, "uk.co.sparx.maths", "export_pupil_data", adminCaller, map[string]any{
		"pupil_id": seeded.ActivePupilID,
	})
	if err != nil || !exportResp.OK {
		t.Fatalf("export_pupil_data failed: %v", err)
	}

	purgeResp, err := h.ExecuteRPC(ctx, "uk.co.sparx.maths", "purge_pupil_data", adminCaller, map[string]any{
		"pupil_id": seeded.ActivePupilID,
	})
	if err != nil || !purgeResp.OK {
		t.Fatalf("purge_pupil_data failed: %v", err)
	}

	exportAfterResp, err := h.ExecuteRPC(ctx, "uk.co.sparx.maths", "export_pupil_data", adminCaller, map[string]any{
		"pupil_id": seeded.ActivePupilID,
	})
	if err != nil || !exportAfterResp.OK {
		t.Fatalf("export_pupil_data after purge failed: %v", err)
	}
	exportAfterData, _ := exportAfterResp.Data.(map[string]any)
	sectionsAfter, _ := exportAfterData["sections"].([]any)
	secAfter0, _ := sectionsAfter[0].(map[string]any)
	recordsAfter, _ := secAfter0["records"].([]any)
	if len(recordsAfter) != 0 {
		t.Fatalf("expected 0 orphaned records in plugin_sparx.db after purge_pupil_data, got %d", len(recordsAfter))
	}
}
