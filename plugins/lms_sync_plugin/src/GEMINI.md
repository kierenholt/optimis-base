# Google Classroom LMS Sync Sandboxed Lua Runtime (`plugins/lms_sync_plugin/src`)

This directory contains [`init.lua`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua), the Lua 5.1 (`gopher-lua`) implementation for `uk.org.optimis.lms.classroom` executed inside the Go Host's capability-sandboxed VM.

---

## Exported Endpoints, Workers & UK GDPR Hooks

* [`upsert_pupil_membership(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L3-L51):
  * Requires scopes `dal.groups.write:all`, `dal.groups.read:all`, `dal.pupils.read:all` (`mutates: true`).
  * Calls `ctx.dal.groups.upsert_pupil_membership`, which commits the membership update to `core.db` and emits `core.group.membership_changed` post-commit.
* [`on_membership_changed(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L82-L194):
  * Background Event Bus worker subscribed to `core.group.membership_changed` under `SYSTEM` service principal (`dal.pupils.read:all`, `dal.groups.read:all`, `mutates: true`).
  * Reads the updated group roster via `ctx.dal.pupils.list_by_group`, executes outbound `ctx.http.post` to `https://classroom.googleapis.com/v1/courses/roster:sync` **before** opening any SQLite write transaction, and records `SYNCED` or `FAILED` status in `plugin_lms_sync.lms_sync_logs`.
* [`get_group_roster(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L53-L80) & [`list_sync_logs(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L196-L253):
  * Read-only endpoints for inspecting Core group membership and `plugin_lms_sync.lms_sync_logs`.
* [`export_pupil_data(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L255-L283) & [`purge_pupil_data(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L285-L303):
  * UK GDPR SAR export (`"Google Classroom LMS Roster Sync Logs"`, `redactable_fields = { "sync_notes" }`) and zero-orphan statutory retention purge hooks.
