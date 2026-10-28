# Synthetic Test & Fixture Plugin (`plugins/test_plugin`)

This directory contains the unpacked reference and test fixture plugin (`uk.org.optimis.test`) mounted by the Go Host Container during local development and E2E testing.

## Plugin Architecture & Contract

1. **Manifest Contract ([`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/manifest.json))**:
   - Declares plugin ID `uk.org.optimis.test`, `core_dal_version: "~1.0"`, attached database alias `plugin_test` (`max_db_size_mb: 64`), `egress.allowed_domains` (`["https://sync.sparxmaths.uk", "https://api.dfe.gov.uk"]`), `egress.timeout_ms: 1500`, and explicit method-level scopes and `mutates` flags for each interactive RPC endpoint:
     - `get_pupil_record` (`dal.pupils.read:all`, `mutates: false`)
     - `save_pupil_note` (`dal.pupils.read:all`, `mutates: true`)
     - `list_pupil_notes` (`dal.pupils.read:all`, `mutates: false`)
     - `get_group_roster_summary` (`dal.pupils.read:own_groups`, `dal.groups.read:own_groups`, `mutates: false`)
     - `get_pupil_guardians` (`dal.guardians.read:own_groups`, `mutates: false`)
     - `record_group_am_attendance` (`dal.attendance.read:own_groups`, `dal.attendance.write:own_groups`, `mutates: true`)
     - `submit_safeguarding_postbox` (`dal.safeguarding.create`, `mutates: true`)
      - `sync_external_sparx_get`, `push_external_sparx_post`, & `mutate_then_illegal_egress` (`dal.pupils.read:own_groups`, `ctx.http` egress endpoints)
      - `trigger_loop_relay` (`dal.pupils.read:own_groups`, `mutates: false`) & `list_worker_event_logs` (`dal.pupils.read:own_groups`, `mutates: false`)
   - Declares background Event Bus workers (`on_core_event_audit` subscribed to `core.attendance.session_recorded`, `core.safeguarding.postbox_created`, and `core.group.membership_changed`; `on_loop_relay` subscribed to `uk.org.optimis.test.loop_relay`).
   - Declares UK GDPR hooks (`export_pupil_data` and `purge_pupil_data`).

2. **Sandboxed Lua Runtime Entrypoint ([`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/init.lua))**:
   - Implements all endpoints, background event workers, and GDPR hooks declared in `manifest.json` using the injected `ctx.dal`, `ctx.plugin_db`, `ctx.http`, and `ctx.events` capabilities.
   - Enforces the write-only safeguarding postbox invariant by verifying `ctx.dal.safeguarding.read == nil` and `ctx.dal.safeguarding.get_by_id == nil` inside Lua.

3. **Sandboxed Micro-Frontend UI ([`ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/ui/dist/index.html))**:
   - Runs inside `<iframe sandbox="allow-scripts">` under `connect-src 'none'` CSP and loads the official [`/sdk/mis-theme.css`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-theme.css) (`--mis-*` Light & High-Contrast variables) and [`/sdk/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js) SDK (`window.mis.onInit`, `window.mis.rpc()`, declarative `data-mis-rpc` delegation, and `data-mis-link` entity deep-links).

4. **Per-Plugin Attached SQLite Migrations ([`migrations/`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/migrations/GEMINI.md))**:
   - [`migrations/001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/migrations/001_init.sql): Provisions the `pupil_notes` table and `idx_pupil_notes_pupil_id` index inside `plugin_test.db` inside `BEGIN IMMEDIATE` with automatic file-snapshot rollback protection.
   - [`migrations/002_worker_event_logs.sql`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/migrations/002_worker_event_logs.sql): Non-destructive Expand-and-Contract migration adding `worker_event_logs` (`id`, `pupil_id`, `worker_method`, `event_name`, `hop`, `caller_type`, `summary_json`) for background Event Bus worker auditing.

