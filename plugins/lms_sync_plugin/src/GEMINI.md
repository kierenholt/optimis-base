# Google Classroom LMS Sync Sandboxed Lua Runtime (`plugins/lms_sync_plugin/src`)

This directory contains [`init.lua`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua), the Lua 5.1 (`gopher-lua`) implementation for `uk.org.optimis.lms.classroom` executed inside the Go Host's capability-sandboxed VM.

---

## Exported Endpoints, Workers & UK GDPR Hooks

* [`upsert_pupil_membership(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L3-L51):
  * Requires scopes `dal.groups.write:all`, `dal.groups.read:all`, `dal.pupils.read:all` (`mutates: true`).
  * Calls `ctx.dal.groups.upsert_pupil_membership`, which commits the membership update to `core.db` and emits `core.group.membership_changed` post-commit.
* [`on_membership_changed(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L85-L238):
  * Background Event Bus worker subscribed to `core.group.membership_changed` under `SYSTEM` service principal (`dal.pupils.read:all`, `dal.groups.read:all`, `mutates: true`).
  * Reads the updated group roster (including `preferred_first_name`, `preferred_last_name`, and `nc_year_group`) via `ctx.dal.pupils.list_by_group`, encodes the JSON payload via `ctx.json.encode`, executes outbound `ctx.http.post` to `https://classroom.googleapis.com/v1/courses/roster:sync` **before** opening any SQLite write transaction, inspects `resp.ok` / `resp.error_code` and parses `resp.body` via `ctx.json.decode`, and records `SYNCED` or `FAILED` status in `plugin_lms_sync.lms_sync_logs` using `ctx.crypto.uuid()` and `ctx.clock.now_utc()`.
* [`get_group_roster(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L53-L83) & [`list_sync_logs(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L240-L296):
  * Read-only endpoints for inspecting Core group membership (`:own_groups` with `:all` hierarchical scope subsumption) and `plugin_lms_sync.lms_sync_logs`.
* [`export_pupil_data(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L298-L327) & [`purge_pupil_data(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L329-L347):
  * Auto-bound UK GDPR SAR export (`"Google Classroom LMS Roster Sync Logs"`, `redactable_fields = { "sync_notes" }`) and zero-orphan statutory retention purge hooks.
