package host_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"optimis/internal/dal"
	"optimis/internal/host"
)

func setupLmsSyncPluginTestHost(t *testing.T) (*host.Host, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "optimis_lms_sync_test_*")
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
		HostKMSKeyHandle: "kms-lms-sync-test-key-001",
		PluginsDir:       pluginsDir,
	}

	h, err := host.NewHost(cfg)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to initialize host: %v", err)
	}

	h.ConfigureEgressMode(false)

	if _, err := h.InstallPlugin(context.Background(), "lms_sync_plugin"); err != nil {
		_ = h.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to install lms_sync_plugin: %v", err)
	}
	if err := h.ActivatePlugin("uk.org.optimis.lms.classroom"); err != nil {
		_ = h.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to activate uk.org.optimis.lms.classroom: %v", err)
	}

	logCursor := h.LogCursor()

	cleanup := func() {
		if t.Failed() {
			logs := h.LogsSince(logCursor)
			t.Logf("--- TAPPED SERVICE LOGS ---\n%s", strings.Join(logs, "\n"))
			t.Logf("--- TAPPED DIAGNOSTIC RING BUFFER ---\n%s", strings.Join(h.Diagnostics(), "\n"))
		}
		_ = h.RemovePlugin(context.Background(), "uk.org.optimis.lms.classroom")
		_ = h.Close()
		_ = os.RemoveAll(tempDir)
	}

	return h, cleanup
}

func TestLmsSyncPlugin_UpsertPupilMembershipAndGdprHooks_FiresPostCommitEventWorkerSyncsClassroomAndPurgesPupilData_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupLmsSyncPluginTestHost(t)
	t.Cleanup(cleanup)

	if err := h.RegisterMockEgressRule(host.MockEgressRule{
		Method: "POST",
		URL:    "https://classroom.googleapis.com/v1/courses/roster:sync",
		Status: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"sync_receipt_id":"gclass_sync_go_test_1","status":"SYNCED"}`,
	}); err != nil {
		t.Fatalf("RegisterMockEgressRule failed: %v", err)
	}
	t.Cleanup(func() {
		h.ClearMockEgressRules()
	})

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

	adminCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_admin_lms",
		StaffID:     "stf_admin_lms",
		ActiveRoles: []string{"ADMIN"},
		Scopes: []string{
			"dal.groups.write:all",
			"dal.groups.read:all",
			"dal.pupils.read:all",
		},
		RequestID: "req_lms_admin_1",
	}

	// 1. Transfer active pupil to other_group_id via upsert_pupil_membership
	upsertResp, err := h.ExecuteRPC(ctx, "uk.org.optimis.lms.classroom", "upsert_pupil_membership", adminCaller, map[string]any{
		"group_id":         seeded.OtherGroupID,
		"pupil_id":         seeded.ActivePupilID,
		"academic_year_id": seeded.AcademicYearID,
		"effective_from":   seeded.ReferenceDate,
	})
	if err != nil || !upsertResp.OK {
		t.Fatalf("expected upsert_pupil_membership to succeed, got resp=%+v err=%v", upsertResp, err)
	}

	// 2. Wait for post-commit Event Bus worker on_membership_changed to sync with mock Google Classroom and persist log
	var syncLogs []any
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		logsResp, lErr := h.ExecuteRPC(ctx, "uk.org.optimis.lms.classroom", "list_sync_logs", adminCaller, map[string]any{
			"pupil_id": seeded.ActivePupilID,
			"group_id": seeded.OtherGroupID,
		})
		if lErr == nil && logsResp.OK {
			dataMap, _ := logsResp.Data.(map[string]any)
			if rawLogs, ok := dataMap["logs"].([]any); ok && len(rawLogs) > 0 {
				syncLogs = rawLogs
				break
			}
		}
		time.Sleep(40 * time.Millisecond)
	}

	if len(syncLogs) == 0 {
		t.Fatalf("expected worker on_membership_changed to persist at least 1 sync log row in plugin_lms_sync.db")
	}

	firstLog, _ := syncLogs[0].(map[string]any)
	if firstLog["status"] != "SYNCED" {
		t.Errorf("expected sync log status SYNCED, got %v", firstLog["status"])
	}
	if firstLog["external_sync_id"] != "gclass_sync_go_test_1" {
		t.Errorf("expected external_sync_id gclass_sync_go_test_1, got %v", firstLog["external_sync_id"])
	}
	if firstLog["caller_type"] != "SYSTEM" {
		t.Errorf("expected worker caller_type SYSTEM, got %v", firstLog["caller_type"])
	}

	// 3. Verify UK GDPR export_pupil_data & purge_pupil_data hooks
	exportResp, err := h.ExecuteRPC(ctx, "uk.org.optimis.lms.classroom", "export_pupil_data", adminCaller, map[string]any{
		"pupil_id": seeded.ActivePupilID,
	})
	if err != nil || !exportResp.OK {
		t.Fatalf("export_pupil_data failed: %v", err)
	}
	exportData, _ := exportResp.Data.(map[string]any)
	sections, _ := exportData["sections"].([]any)
	if len(sections) != 1 {
		t.Fatalf("expected 1 GDPR SAR section from export_pupil_data, got %d", len(sections))
	}
	sec0, _ := sections[0].(map[string]any)
	recordsBefore, _ := sec0["records"].([]any)
	if len(recordsBefore) == 0 {
		t.Fatalf("expected >= 1 record in GDPR SAR export before purge")
	}

	purgeResp, err := h.ExecuteRPC(ctx, "uk.org.optimis.lms.classroom", "purge_pupil_data", adminCaller, map[string]any{
		"pupil_id": seeded.ActivePupilID,
	})
	if err != nil || !purgeResp.OK {
		t.Fatalf("purge_pupil_data failed: %v", err)
	}

	exportAfterResp, err := h.ExecuteRPC(ctx, "uk.org.optimis.lms.classroom", "export_pupil_data", adminCaller, map[string]any{
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
		t.Fatalf("expected 0 orphaned records in plugin_lms_sync.db after purge_pupil_data, got %d", len(recordsAfter))
	}
}
