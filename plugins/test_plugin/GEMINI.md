# Synthetic Test & Fixture Plugin (`plugins/test_plugin`)

This directory contains the unpacked reference and test fixture plugin (`uk.org.optimis.test`) mounted by the Go Host Container during local development and E2E testing.

## Plugin Architecture & Contract

1. **Manifest Contract ([`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/manifest.json))**:
   - Declares plugin ID `uk.org.optimis.test`, `core_dal_version: "~1.0"`, attached database alias `plugin_test` (`max_db_size_mb: 64`), and explicit method-level scopes and `mutates` flags for each interactive RPC endpoint:
     - `get_pupil_record` (`dal.pupils.read:all`, `mutates: false`)
     - `save_pupil_note` (`dal.pupils.read:all`, `mutates: true`)
     - `list_pupil_notes` (`dal.pupils.read:all`, `mutates: false`)
     - `get_group_roster_summary` (`dal.pupils.read:own_groups`, `dal.sen.read:own_groups`, `dal.medical.read:own_groups`, `dal.groups.read:own_groups`, `mutates: false`)
     - `get_pupil_guardians` (`dal.pupils.read:own_groups`, `dal.guardians.read:own_groups`, `mutates: false`)
     - `record_group_am_attendance` (`dal.attendance.read:own_groups`, `dal.attendance.write:own_groups`, `mutates: true`)
     - `submit_safeguarding_postbox` (`dal.safeguarding.create`, `mutates: true`)
   - Declares UK GDPR hooks (`export_pupil_data` and `purge_pupil_data`).

2. **Sandboxed Lua Runtime Entrypoint ([`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/init.lua))**:
   - Implements all endpoints and GDPR hooks declared in `manifest.json` using the injected `ctx.dal`, `ctx.plugin_db`, and `ctx.events` capabilities.
   - Enforces the write-only safeguarding postbox invariant by verifying `ctx.dal.safeguarding.read == nil` and `ctx.dal.safeguarding.get_by_id == nil` inside Lua.

3. **Sandboxed Micro-Frontend UI ([`ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/ui/dist/index.html))**:
   - Runs inside `<iframe sandbox="allow-scripts">` under `connect-src 'none'` CSP.
   - Completes the `mis:ready` / `mis:init` handshake with the parent React Host Shell and dispatches declarative `data-mis-rpc` actions over `window.postMessage`.

4. **Per-Plugin Attached SQLite Migrations ([`migrations/`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/migrations/GEMINI.md))**:
   - [`migrations/001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/migrations/001_init.sql): Provisions the `pupil_notes` table and `idx_pupil_notes_pupil_id` index inside `plugin_test.db` inside `BEGIN IMMEDIATE` with automatic file-snapshot rollback protection.
