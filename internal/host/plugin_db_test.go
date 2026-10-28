package host_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"optimis/internal/dal"
	"optimis/internal/host"
)

func TestLoadPlugin_WithDatabaseAliasAndMigrations_ProvisionsAttachedDBAndEnforcesMaxPageCount_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	// Verify plugin_test.db is attached on both WriteDB and ReadDB with max_page_count = 64MB (16384 pages)
	var writeMaxPages int
	if err := h.DAL().WriteDB().QueryRowContext(ctx, `PRAGMA plugin_test.max_page_count;`).Scan(&writeMaxPages); err != nil {
		t.Fatalf("failed to query PRAGMA plugin_test.max_page_count on WriteDB: %v", err)
	}
	if writeMaxPages != 16384 {
		t.Errorf("expected WriteDB PRAGMA plugin_test.max_page_count to be 16384 (64MB), got %d", writeMaxPages)
	}

	var readMaxPages int
	if err := h.DAL().ReadDB().QueryRowContext(ctx, `PRAGMA plugin_test.max_page_count;`).Scan(&readMaxPages); err != nil {
		t.Fatalf("failed to query PRAGMA plugin_test.max_page_count on ReadDB: %v", err)
	}
	if readMaxPages != 16384 {
		t.Errorf("expected ReadDB PRAGMA plugin_test.max_page_count to be 16384 (64MB), got %d", readMaxPages)
	}

	// Verify migration table recorded 001_init.sql inside plugin_test.db
	var migrationCount int
	if err := h.DAL().ReadDB().QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM plugin_test.__plugin_migrations WHERE filename = '001_init.sql'`,
	).Scan(&migrationCount); err != nil {
		t.Fatalf("failed to query plugin_test.__plugin_migrations on ReadDB: %v", err)
	}
	if migrationCount != 1 {
		t.Errorf("expected 1 applied migration record for 001_init.sql, got %d", migrationCount)
	}
}

func TestExecuteRPC_PluginDBExecAndQuery_PersistsViaWriteDBAndReadsConcurrentlyViaReadDBPool_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	seeded, err := h.DAL().SeedPupil(ctx, dal.SeedPupilInput{
		UPN:            "P123456789099",
		FirstName:      "Katherine",
		LastName:       "Johnson",
		AcademicYearID: "2026-2027",
		EffectiveFrom:  "2026-09-01",
	})
	if err != nil {
		t.Fatalf("failed to seed pupil: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeletePupil(context.Background(), seeded.ID)
	})

	caller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_1",
		StaffID:     "stf_001",
		ActiveRoles: []string{"teacher"},
		Scopes:      []string{"dal.pupils.read:all"},
		RequestID:   "req_plugin_db_rw",
	}

	saveRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.test", "save_pupil_note", caller, map[string]any{
		"pupil_id": seeded.ID,
		"note_id":  "note_001",
		"note":     "Solved orbital trajectory equations",
	})
	if err != nil {
		t.Fatalf("ExecuteRPC save_pupil_note failed: %v", err)
	}
	if !saveRes.OK {
		t.Fatalf("expected saveRes.OK == true, got false (%s)", saveRes.ErrorCode)
	}

	// Concurrently read across multiple ReadDB connections to verify ConnectHook synchronization
	const workers = 6
	var wg sync.WaitGroup
	errCh := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			listRes, qErr := h.ExecuteRPC(ctx, "uk.org.optimis.test", "list_pupil_notes", caller, map[string]any{
				"pupil_id": seeded.ID,
			})
			if qErr != nil {
				errCh <- qErr
				return
			}
			dataMap, ok := listRes.Data.(map[string]any)
			if !ok {
				t.Errorf("expected map[string]any data, got %T", listRes.Data)
				return
			}
			if dataMap["count"] != float64(1) {
				t.Errorf("expected count == 1 on ReadDB connection, got %v", dataMap["count"])
			}
			if dataMap["first_note"] != "Solved orbital trajectory equations" {
				t.Errorf("expected first_note to match, got %v", dataMap["first_note"])
			}
			if dataMap["max_page_count"] != float64(16384) {
				t.Errorf("expected max_page_count == 16384 on ReadDB connection, got %v", dataMap["max_page_count"])
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for qErr := range errCh {
		t.Fatalf("concurrent ReadDB query failed: %v", qErr)
	}

	// Verify GDPR export_pupil_data and purge_pupil_data leave zero orphaned records
	exportRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.test", "export_pupil_data", caller, map[string]any{
		"pupil_id": seeded.ID,
	})
	if err != nil || !exportRes.OK {
		t.Fatalf("export_pupil_data failed: err=%v res=%+v", err, exportRes)
	}

	purgeRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.test", "purge_pupil_data", caller, map[string]any{
		"pupil_id": seeded.ID,
	})
	if err != nil || !purgeRes.OK {
		t.Fatalf("purge_pupil_data failed: err=%v res=%+v", err, purgeRes)
	}

	afterPurgeRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.test", "list_pupil_notes", caller, map[string]any{
		"pupil_id": seeded.ID,
	})
	if err != nil || !afterPurgeRes.OK {
		t.Fatalf("list_pupil_notes after purge failed: err=%v res=%+v", err, afterPurgeRes)
	}
	afterMap := afterPurgeRes.Data.(map[string]any)
	if afterMap["count"] != float64(0) {
		t.Errorf("expected 0 remaining notes after purge_pupil_data, got %v", afterMap["count"])
	}
}

func TestLoadPlugin_FailingMigrationInUpgrade_RollsBackFileSnapshotAutomatically_test(t *testing.T) {
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "optimis_mig_rollback_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	dataDir := filepath.Join(tempDir, "data")
	pluginsDir := filepath.Join(tempDir, "plugins")
	pluginDir := filepath.Join(pluginsDir, "rollback_plugin")
	migrationsDir := filepath.Join(pluginDir, "migrations")
	srcDir := filepath.Join(pluginDir, "src")

	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("failed to create migrations dir: %v", err)
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	manifestJSON := `{
  "id": "uk.org.optimis.rollback",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "database": {
    "alias": "plugin_rollback"
  },
  "gdpr_hooks": {
    "export_pupil_data": "export_pupil_data",
    "purge_pupil_data": "purge_pupil_data"
  },
  "egress": {
    "allowed_domains": []
  },
  "endpoints": {
    "seed_item": {
      "scopes": ["dal.pupils.read:all"],
      "mutates": true
    },
    "count_items": {
      "scopes": ["dal.pupils.read:all"],
      "mutates": false
    }
  }
}`
	luaSrc := `
function seed_item(ctx, args)
  ctx.plugin_db.exec("INSERT INTO items (id, label) VALUES (?, ?)", { args.id, args.label })
  return { ok = true, data = { seeded = true } }
end

function count_items(ctx, args)
  local rows = ctx.plugin_db.query("SELECT COUNT(*) AS cnt FROM items")
  local pages = ctx.plugin_db.query("PRAGMA plugin_rollback.max_page_count")
  return {
    ok = true,
    data = {
      cnt = rows[1].cnt,
      max_page_count = pages[1].max_page_count
    }
  }
end

function export_pupil_data(ctx, args)
  return { ok = true, data = {} }
end

function purge_pupil_data(ctx, args)
  return { ok = true, data = { purged_rows = 0 } }
end
`
	mig001 := `CREATE TABLE items (id TEXT PRIMARY KEY, label TEXT NOT NULL);`

	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("failed to write manifest.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "init.lua"), []byte(luaSrc), 0o644); err != nil {
		t.Fatalf("failed to write init.lua: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "001_init.sql"), []byte(mig001), 0o644); err != nil {
		t.Fatalf("failed to write 001_init.sql: %v", err)
	}

	h, err := host.NewHost(host.Config{
		DataDirPath:      dataDir,
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms-test-key-001",
		PluginsDir:       pluginsDir,
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	logCursor := h.LogCursor()
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("--- TAPPED SERVICE LOGS ---\n%s", strings.Join(h.LogsSince(logCursor), "\n"))
		}
		h.Close()
	})

	if err := h.LoadPlugin("rollback_plugin"); err != nil {
		t.Fatalf("initial LoadPlugin failed: %v", err)
	}

	caller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_1",
		StaffID:     "stf_001",
		ActiveRoles: []string{"teacher"},
		Scopes:      []string{"dal.pupils.read:all"},
		RequestID:   "req_rb_1",
	}

	if _, err := h.ExecuteRPC(ctx, "uk.org.optimis.rollback", "seed_item", caller, map[string]any{
		"id":    "item_1",
		"label": "Pre-upgrade preserved item",
	}); err != nil {
		t.Fatalf("seed_item failed: %v", err)
	}

	// Now write a second migration 002_succeeds.sql and a third migration 003_broken.sql
	// Even if 002 runs before 003 fails, both BEGIN IMMEDIATE and the file-level snapshot rollback
	// must restore plugin_rollback.db to the exact pre-upgrade state (only 1 row in items, no 002 table).
	mig002 := `CREATE TABLE extra_audit (id TEXT PRIMARY KEY); INSERT INTO items (id, label) VALUES ('item_2', 'Should be rolled back');`
	mig003 := `INSERT INTO items (id, label) VALUES ('item_3', 'Unclosed syntax; INVALID SQL STATEMENT HERE;`

	if err := os.WriteFile(filepath.Join(migrationsDir, "002_extra.sql"), []byte(mig002), 0o644); err != nil {
		t.Fatalf("failed to write 002_extra.sql: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "003_broken.sql"), []byte(mig003), 0o644); err != nil {
		t.Fatalf("failed to write 003_broken.sql: %v", err)
	}

	if err := h.LoadPlugin("rollback_plugin"); err == nil {
		t.Fatalf("expected LoadPlugin to fail fast when 003_broken.sql has invalid SQL, got nil")
	}

	// Verify snapshot file was cleaned up after restoration and original DB state is 100% preserved
	if _, statErr := os.Stat(filepath.Join(dataDir, "plugin_rollback.db.snapshot")); !os.IsNotExist(statErr) {
		t.Errorf("expected plugin_rollback.db.snapshot to be cleaned up after rollback, statErr=%v", statErr)
	}

	countRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.rollback", "count_items", caller, map[string]any{})
	if err != nil {
		t.Fatalf("count_items after rollback failed: %v", err)
	}
	dataMap := countRes.Data.(map[string]any)
	if dataMap["cnt"] != float64(1) {
		t.Errorf("expected exactly 1 item preserved from pre-upgrade snapshot, got %v", dataMap["cnt"])
	}
	// Default max_db_size_mb fallback is 500 MB -> 500 * 256 = 128000 pages
	if dataMap["max_page_count"] != float64(128000) {
		t.Errorf("expected default max_page_count fallback 128000 (500MB), got %v", dataMap["max_page_count"])
	}
}

func TestLoadPlugin_InvalidDatabaseAlias_FailsFast_test(t *testing.T) {
	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	pluginDir := filepath.Join(pluginsDir, "bad_alias_plugin")
	if err := os.MkdirAll(filepath.Join(pluginDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	manifestJSON := `{
  "id": "uk.org.optimis.badalias",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "database": {
    "alias": "invalid_alias_without_prefix"
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

	if err := h.LoadPlugin("bad_alias_plugin"); err == nil {
		t.Fatalf("expected LoadPlugin to fail fast for database.alias not matching ^plugin_[a-z0-9_]+$, got nil")
	}
}

func TestExecuteRPC_PluginDBScopeAndWriteTxHttpGuard_FailsFastAndRollsBack_test(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	pluginDir := filepath.Join(pluginsDir, "guard_plugin")
	if err := os.MkdirAll(filepath.Join(pluginDir, "migrations"), 0o755); err != nil {
		t.Fatalf("failed to create migrations dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pluginDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	manifestJSON := `{
  "id": "uk.org.optimis.guard",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "database": {
    "alias": "plugin_guard"
  },
  "gdpr_hooks": {
    "export_pupil_data": "export_pupil_data",
    "purge_pupil_data": "purge_pupil_data"
  },
  "egress": {
    "allowed_domains": ["https://api.example.org"]
  },
  "endpoints": {
    "query_main_schema": {
      "scopes": [],
      "mutates": false
    },
    "exec_then_http": {
      "scopes": [],
      "mutates": true
    },
    "count_entries": {
      "scopes": [],
      "mutates": false
    }
  }
}`
	luaSrc := `
function query_main_schema(ctx, args)
  local rows = ctx.plugin_db.query("SELECT * FROM main.pupils")
  return { ok = true, data = rows }
end

function exec_then_http(ctx, args)
  ctx.plugin_db.exec("INSERT INTO guard_items (id, val) VALUES ('g1', 'uncommitted')")
  ctx.http.get("https://api.example.org/sync")
  return { ok = true, data = {} }
end

function count_entries(ctx, args)
  local rows = ctx.plugin_db.query("SELECT COUNT(*) AS cnt FROM guard_items")
  return { ok = true, data = { cnt = rows[1].cnt } }
end

function export_pupil_data(ctx, args)
  return { ok = true, data = {} }
end

function purge_pupil_data(ctx, args)
  return { ok = true, data = { purged_rows = 0 } }
end
`
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("failed to write manifest.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "src", "init.lua"), []byte(luaSrc), 0o644); err != nil {
		t.Fatalf("failed to write init.lua: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "migrations", "001_init.sql"), []byte(`CREATE TABLE guard_items (id TEXT PRIMARY KEY, val TEXT NOT NULL);`), 0o644); err != nil {
		t.Fatalf("failed to write 001_init.sql: %v", err)
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

	if err := h.LoadPlugin("guard_plugin"); err != nil {
		t.Fatalf("LoadPlugin failed: %v", err)
	}

	caller := host.CallerContext{
		Type:      "USER",
		UserID:    "usr_teacher_1",
		StaffID:   "stf_001",
		Scopes:    []string{},
		RequestID: "req_guard_1",
	}

	// 1. Accessing main.pupils via ctx.plugin_db.query must fail fast
	if _, err := h.ExecuteRPC(ctx, "uk.org.optimis.guard", "query_main_schema", caller, map[string]any{}); err == nil {
		t.Fatalf("expected fail-fast error when querying main.pupils via ctx.plugin_db.query, got nil")
	}

	// 2. Calling ctx.http.get inside an open write transaction must halt/rollback the write transaction and fail fast
	if _, err := h.ExecuteRPC(ctx, "uk.org.optimis.guard", "exec_then_http", caller, map[string]any{}); err == nil {
		t.Fatalf("expected fail-fast error when invoking ctx.http.get inside an open SQLite write transaction, got nil")
	}

	// Verify the uncommitted INSERT before ctx.http.get was rolled back
	countRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.guard", "count_entries", caller, map[string]any{})
	if err != nil {
		t.Fatalf("count_entries failed: %v", err)
	}
	if countRes.Data.(map[string]any)["cnt"] != float64(0) {
		t.Errorf("expected 0 entries after write transaction rollback on ctx.http violation, got %v", countRes.Data)
	}
}

