# Test Plugin SQLite Migrations (`plugins/test_plugin/migrations`)

This directory holds the sequential SQL migration scripts (`*.sql`) executed by the Go Host Container against the attached per-plugin SQLite database (`plugin_test.db`) declared in [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/manifest.json).

---

## 1. Migration Execution & Snapshot Rollback Contract

* **Lexicographical Ordering**: All `*.sql` files in this folder are sorted lexicographically (`001_...`, `002_...`) and tracked in `plugin_test.__plugin_migrations` (`filename`, `applied_at`).
* **Automatic File-Level Snapshot (`plugin_test.db.snapshot`)**: Prior to applying any pending migration files, [`Host.provisionAndMigratePluginDB`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L320-L447) checkpoints WAL frames (`PRAGMA wal_checkpoint(TRUNCATE)`) and creates a byte-for-byte file copy at `<DATA_DIR_PATH>/plugin_test.db.snapshot`.
* **`BEGIN IMMEDIATE` Atomic DDL & Rollback**: Pending migrations run inside a `BEGIN IMMEDIATE` transaction via [`Host.runMigrationsInImmediateTx`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L449-L504) with `PRAGMA max_page_count` enforced. If any migration fails, the transaction rolls back, stale `-wal`/`-shm` files are removed, and `plugin_test.db.snapshot` is restored automatically.
* **Expand-and-Contract Rule**: Minor plugin version updates must be non-destructive (no dropping or renaming columns) so mid-day code rollbacks remain compatible.

---

## 2. Migration Files

* [`001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/migrations/001_init.sql): Creates the `pupil_notes` table (`id`, `pupil_id`, `note`, `created_by`) and `idx_pupil_notes_pupil_id` index inside `plugin_test.db` for `save_pupil_note`, `list_pupil_notes`, `export_pupil_data`, and `purge_pupil_data`.
* [`002_worker_event_logs.sql`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/migrations/002_worker_event_logs.sql): Creates the `worker_event_logs` table (`id`, `pupil_id`, `event_name`, `hop`, `caller_type`, `caller_staff_id`, `created_at`) and `idx_worker_event_logs_pupil_id` index inside `plugin_test.db` for background Event Bus worker auditing (`on_core_event_audit`, `on_loop_relay`, `list_worker_event_logs`, and `purge_pupil_data`).

