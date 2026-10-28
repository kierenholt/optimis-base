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

func TestLoadPlugin_DatabaseAliasWithoutGDPRHooks_FailsFast_test(t *testing.T) {
	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	pluginDir := filepath.Join(pluginsDir, "missing_gdpr_plugin")
	if err := os.MkdirAll(filepath.Join(pluginDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create plugin src dir: %v", err)
	}

	manifestJSON := `{
  "id": "uk.org.optimis.missinggdpr",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "database": {
    "alias": "plugin_missing_gdpr"
  },
  "egress": { "allowed_domains": [] },
  "endpoints": {
    "ping": { "scopes": [], "mutates": false }
  }
}`
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("failed to write manifest.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "src", "init.lua"), []byte(`function ping(ctx, args) return { ok = true, data = {} } end`), 0o644); err != nil {
		t.Fatalf("failed to write init.lua: %v", err)
	}

	h, err := host.NewHost(host.Config{
		DataDirPath:      filepath.Join(tempDir, "data"),
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms-test-key-001",
		PluginsDir:       pluginsDir,
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	t.Cleanup(func() {
		_ = h.Close()
	})

	err = h.LoadPlugin("missing_gdpr_plugin")
	if err == nil {
		t.Fatalf("expected LoadPlugin to fail fast when database.alias is declared without gdpr_hooks, got nil")
	}
	if !strings.Contains(err.Error(), "gdpr_hooks") {
		t.Errorf("expected error message to mention gdpr_hooks, got: %v", err)
	}
}

func TestExportPupilSAR_AggregatesCoreAndActiveAndDormantPlugins_ReturnsStructuredSections_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	seeded, err := h.DAL().SeedGDPRTestFixtures(ctx, dal.GDPRTestFixtureInput{
		AcademicYearID:   "ay_2026_2027",
		ReferenceDate:    "2026-10-20",
		HostKMSKeyHandle: "kms-test-key-001",
	})
	if err != nil {
		t.Fatalf("SeedGDPRTestFixtures failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeleteGDPRTestFixtures(context.Background(), seeded.FixtureID)
	})

	adminCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_admin_1",
		StaffID:     seeded.AdminStaffID,
		ActiveRoles: []string{"ADMIN"},
		Scopes: []string{
			"dal.pupils.read:all",
			"dal.pupils.write:all",
			"dal.staff.read:admin",
			"dal.staff.write:admin",
		},
		RequestID: "req_sar_export_1",
	}

	// 1. Seed a plugin note in plugin_test.db
	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.test", "save_pupil_note", adminCaller, map[string]any{
		"pupil_id": seeded.PupilID,
		"note_id":  "note_sar_1",
		"note":     "Shared worksheet with third-party student Hermione Granger.",
	})
	if err != nil {
		t.Fatalf("save_pupil_note failed: %v", err)
	}

	// 2. Transition plugin to UNINSTALLED_DORMANT quarantine state (Section 8 Gap 3 & Section 9.3.4)
	if err := h.SetPluginState("uk.org.optimis.test", host.PluginStateUninstalledDormant); err != nil {
		t.Fatalf("SetPluginState to UNINSTALLED_DORMANT failed: %v", err)
	}

	// Verify normal RPC is blocked while dormant
	if _, rpcErr := h.ExecuteRPC(ctx, "uk.org.optimis.test", "list_pupil_notes", adminCaller, map[string]any{
		"pupil_id": seeded.PupilID,
	}); rpcErr == nil || !strings.Contains(rpcErr.Error(), "PLUGIN_DORMANT_QUARANTINE") {
		t.Fatalf("expected ExecuteRPC on dormant plugin to fail with PLUGIN_DORMANT_QUARANTINE, got: %v", rpcErr)
	}

	// 3. Export SAR bundle across Core and Dormant Plugin
	bundle, err := h.ExportPupilSAR(ctx, adminCaller, seeded.PupilID)
	if err != nil {
		t.Fatalf("ExportPupilSAR failed: %v", err)
	}
	if bundle.PupilID != seeded.PupilID {
		t.Errorf("expected PupilID %s, got %s", seeded.PupilID, bundle.PupilID)
	}

	var foundCoreProfile, foundCorePastoral, foundPluginNotes bool
	for _, sec := range bundle.Sections {
		if sec.SourceID == "core" && sec.Title == "Core Pupil Profile & Enrolments" {
			foundCoreProfile = true
			if len(sec.Records) == 0 {
				t.Errorf("expected Core Pupil Profile & Enrolments to have records")
			}
		}
		if sec.SourceID == "core" && sec.Title == "Core Pastoral Notes" {
			foundCorePastoral = true
			if len(sec.Records) != 1 {
				t.Errorf("expected 1 Core Pastoral Note, got %d", len(sec.Records))
			}
			if len(sec.RedactableFields) == 0 {
				t.Errorf("expected Core Pastoral Notes to declare redactable_fields")
			}
		}
		if sec.SourceID == "uk.org.optimis.test" && sec.Title == "Plugin Pupil Notes" {
			foundPluginNotes = true
			if sec.PluginState != string(host.PluginStateUninstalledDormant) {
				t.Errorf("expected PluginState UNINSTALLED_DORMANT, got %s", sec.PluginState)
			}
			if len(sec.Records) != 1 {
				t.Errorf("expected 1 Plugin Pupil Note record, got %d", len(sec.Records))
			}
		}
	}

	if !foundCoreProfile || !foundCorePastoral || !foundPluginNotes {
		t.Fatalf("missing expected SAR sections: coreProfile=%v corePastoral=%v pluginNotes=%v", foundCoreProfile, foundCorePastoral, foundPluginNotes)
	}
}

func TestPurgePupilData_WithKCSIESafeguardingLockAndEligiblePurge_EnforcesRetentionAndPurgesZeroOrphans_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	seeded, err := h.DAL().SeedGDPRTestFixtures(ctx, dal.GDPRTestFixtureInput{
		AcademicYearID:   "ay_2026_2027",
		ReferenceDate:    "2026-10-20",
		HostKMSKeyHandle: "kms-test-key-001",
	})
	if err != nil {
		t.Fatalf("SeedGDPRTestFixtures failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeleteGDPRTestFixtures(context.Background(), seeded.FixtureID)
	})

	adminCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_admin_1",
		StaffID:     seeded.AdminStaffID,
		ActiveRoles: []string{"ADMIN"},
		Scopes: []string{
			"dal.pupils.read:all",
			"dal.pupils.write:all",
			"dal.staff.read:admin",
			"dal.staff.write:admin",
		},
		RequestID: "req_purge_1",
	}

	// Seed plugin note for eligible pupil
	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.test", "save_pupil_note", adminCaller, map[string]any{
		"pupil_id": seeded.PupilID,
		"note_id":  "note_purge_1",
		"note":     "Note to be purged on retention expiry",
	})
	if err != nil {
		t.Fatalf("save_pupil_note failed: %v", err)
	}

	// 1. Attempting to purge ProtectedPupilID (has safeguarding_notes with retention_purge_eligible=0) MUST fail fast
	_, err = h.PurgePupilData(ctx, adminCaller, seeded.ProtectedPupilID)
	if err == nil || !strings.Contains(err.Error(), "KCSIE_RETENTION_LOCK") {
		t.Fatalf("expected PurgePupilData on KCSIE-protected pupil to fail with KCSIE_RETENTION_LOCK, got: %v", err)
	}

	// 2. Purge eligible PupilID while plugin is in UNINSTALLED_DORMANT state
	if err := h.SetPluginState("uk.org.optimis.test", host.PluginStateUninstalledDormant); err != nil {
		t.Fatalf("SetPluginState failed: %v", err)
	}

	purgeRes, err := h.PurgePupilData(ctx, adminCaller, seeded.PupilID)
	if err != nil {
		t.Fatalf("PurgePupilData on eligible pupil failed: %v", err)
	}
	if !purgeRes.CorePurged {
		t.Errorf("expected CorePurged == true")
	}
	if purgeRes.PluginPurgedRows["uk.org.optimis.test"] != 1 {
		t.Errorf("expected 1 purged row in uk.org.optimis.test, got %d", purgeRes.PluginPurgedRows["uk.org.optimis.test"])
	}

	// 3. Verify zero orphaned rows remain in plugin_test.db or core.db
	var pluginNoteCount int
	if err := h.DAL().ReadDB().QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM plugin_test.pupil_notes WHERE pupil_id = ?`,
		seeded.PupilID,
	).Scan(&pluginNoteCount); err != nil {
		t.Fatalf("failed querying plugin_test.pupil_notes: %v", err)
	}
	if pluginNoteCount != 0 {
		t.Errorf("expected 0 remaining plugin notes after purge, got %d", pluginNoteCount)
	}

	var corePupilCount int
	if err := h.DAL().ReadDB().QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM pupils WHERE id = ?`,
		seeded.PupilID,
	).Scan(&corePupilCount); err != nil {
		t.Fatalf("failed querying core pupils: %v", err)
	}
	if corePupilCount != 0 {
		t.Errorf("expected 0 remaining core pupil rows after purge, got %d", corePupilCount)
	}
}
