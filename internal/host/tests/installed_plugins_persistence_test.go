package tests

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"optimis/internal/dal"
)

var isoUtcRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)

func TestInstalledPlugins_LifecycleStateAndTimestamps_StoresIsoUtcAndUpdatesOnStateChange_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedEmptyTestHost(t)
	t.Cleanup(cleanup)

	// 1. Install test_plugin
	loaded, err := h.InstallPlugin(ctx, "test_plugin")
	if err != nil {
		t.Fatalf("InstallPlugin(test_plugin) failed: %v", err)
	}

	if loaded.PlugID == "" || len(loaded.PlugID) != 8 {
		t.Fatalf("expected 8-char plug_id, got %q", loaded.PlugID)
	}

	// 2. Query installed_plugins table directly
	rec, err := h.DAL().GetInstalledPlugin(ctx, loaded.Manifest.ID)
	if err != nil {
		t.Fatalf("GetInstalledPlugin(%s) failed: %v", loaded.Manifest.ID, err)
	}

	if rec.ManifestID != "uk.org.optimis.test" {
		t.Errorf("expected ManifestID uk.org.optimis.test, got %s", rec.ManifestID)
	}
	if rec.PlugID != loaded.PlugID {
		t.Errorf("expected PlugID %s, got %s", loaded.PlugID, rec.PlugID)
	}
	if rec.State != "UNINSTALLED_DORMANT" {
		t.Errorf("expected state UNINSTALLED_DORMANT, got %s", rec.State)
	}
	if rec.Version != "0.2.0" {
		t.Errorf("expected version 0.2.0, got %s", rec.Version)
	}
	if rec.DatabaseAlias != "plugin_test" {
		t.Errorf("expected database_alias plugin_test, got %s", rec.DatabaseAlias)
	}
	if !isoUtcRegex.MatchString(rec.InstalledAt) {
		t.Errorf("expected installed_at to match ISO-8601 UTC regex, got %q", rec.InstalledAt)
	}
	if !isoUtcRegex.MatchString(rec.UpdatedAt) {
		t.Errorf("expected updated_at to match ISO-8601 UTC regex, got %q", rec.UpdatedAt)
	}
	if strings.Contains(rec.InstalledAt, " ") {
		t.Errorf("installed_at contains space (SQLite CURRENT_TIMESTAMP leak): %q", rec.InstalledAt)
	}

	// 3. Verify __system_config does NOT have plugin_instance_id key written with CURRENT_TIMESTAMP
	var legacyVal string
	err = h.DAL().ReadDB().QueryRowContext(
		ctx,
		`SELECT value FROM __system_config WHERE key = ?`,
		"plugin_instance_id."+loaded.Manifest.ID,
	).Scan(&legacyVal)
	if err == nil {
		t.Errorf("expected __system_config to NOT contain legacy prefix key, found value %q", legacyVal)
	}

	// 4. Activate the plugin -> state transitions to ACTIVE and updated_at advances
	time.Sleep(10 * time.Millisecond)
	if err := h.ActivatePlugin(loaded.PlugID); err != nil {
		t.Fatalf("ActivatePlugin failed: %v", err)
	}

	activeRec, err := h.DAL().GetInstalledPlugin(ctx, loaded.PlugID)
	if err != nil {
		t.Fatalf("GetInstalledPlugin by plug_id failed: %v", err)
	}
	if activeRec.State != "ACTIVE" {
		t.Errorf("expected state ACTIVE after activation, got %s", activeRec.State)
	}
	if !isoUtcRegex.MatchString(activeRec.UpdatedAt) {
		t.Errorf("expected updated_at to match ISO-8601 UTC regex, got %q", activeRec.UpdatedAt)
	}

	// 5. Remove plugin -> state transitions to UNINSTALLED_DORMANT in SQLite
	if err := h.RemovePlugin(ctx, loaded.PlugID); err != nil {
		t.Fatalf("RemovePlugin failed: %v", err)
	}

	postRemoveRec, err := h.DAL().GetInstalledPlugin(ctx, loaded.Manifest.ID)
	if err != nil {
		t.Fatalf("GetInstalledPlugin after remove failed: %v", err)
	}
	if postRemoveRec.State != "UNINSTALLED_DORMANT" {
		t.Errorf("expected state UNINSTALLED_DORMANT in DB after remove, got %s", postRemoveRec.State)
	}
}

func TestInstalledPlugins_LegacySystemConfigBackfill_MigratesPrefixKeysIntoInstalledPlugins_test(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "optimis_legacy_backfill_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "core.db")

	// 1. Manually create core.db with legacy __system_config containing plugin_instance_id.*
	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open raw sqlite db: %v", err)
	}
	_, err = rawDB.Exec(`
		CREATE TABLE __system_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		);
		INSERT INTO __system_config (key, value) VALUES
			('school_urn', '100001'),
			('plugin_instance_id.uk.org.legacy.markbook', 'leg12345'),
			('plugin_instance_id.uk.org.legacy.attendance', 'leg67890');
	`)
	rawDB.Close()
	if err != nil {
		t.Fatalf("failed to seed legacy DB: %v", err)
	}

	// 2. Boot CoreDAL which runs initSchema
	coreDAL, err := dal.NewCoreDAL(tempDir, "100001")
	if err != nil {
		t.Fatalf("NewCoreDAL failed on legacy DB: %v", err)
	}
	defer coreDAL.Close()

	// 3. Verify legacy keys were backfilled into installed_plugins
	rec1, err := coreDAL.GetInstalledPlugin(context.Background(), "uk.org.legacy.markbook")
	if err != nil {
		t.Fatalf("GetInstalledPlugin(uk.org.legacy.markbook) failed: %v", err)
	}
	if rec1.PlugID != "leg12345" {
		t.Errorf("expected plug_id leg12345, got %s", rec1.PlugID)
	}

	rec2, err := coreDAL.GetInstalledPlugin(context.Background(), "leg67890")
	if err != nil {
		t.Fatalf("GetInstalledPlugin by plug_id leg67890 failed: %v", err)
	}
	if rec2.ManifestID != "uk.org.legacy.attendance" {
		t.Errorf("expected manifest_id uk.org.legacy.attendance, got %s", rec2.ManifestID)
	}
}

func TestEnsurePluginInstanceID_CollisionAndPersistence_AllocatesIndexedOpaqueIdAndPersistsInInstalledPlugins_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedEmptyTestHost(t)
	t.Cleanup(cleanup)

	// Pre-seed an existing installed plugin
	err := h.DAL().UpsertInstalledPlugin(ctx, dal.InstalledPluginRecord{
		ManifestID:         "uk.org.optimis.existing",
		PlugID:             "ext12345",
		Version:            "1.0.0",
		State:              "ACTIVE",
		Folder:             "existing",
		ApprovedScopesJSON: "[]",
	})
	if err != nil {
		t.Fatalf("UpsertInstalledPlugin failed: %v", err)
	}

	// Verify IsPlugIDAllocated detects ext12345
	allocated, err := h.DAL().IsPlugIDAllocated(ctx, "ext12345", "uk.org.optimis.other")
	if err != nil {
		t.Fatalf("IsPlugIDAllocated failed: %v", err)
	}
	if !allocated {
		t.Errorf("expected ext12345 to be allocated")
	}

	// IsPlugIDAllocated for same manifest ID should return false (it's own ID)
	ownAllocated, err := h.DAL().IsPlugIDAllocated(ctx, "ext12345", "uk.org.optimis.existing")
	if err != nil {
		t.Fatalf("IsPlugIDAllocated for same manifest failed: %v", err)
	}
	if ownAllocated {
		t.Errorf("expected ext12345 for same manifest to not be considered a collision")
	}

	// Random unallocated ID returns false
	unallocated, err := h.DAL().IsPlugIDAllocated(ctx, "zzzzzzzz", "")
	if err != nil {
		t.Fatalf("IsPlugIDAllocated for random ID failed: %v", err)
	}
	if unallocated {
		t.Errorf("expected zzzzzzzz to not be allocated")
	}
}
