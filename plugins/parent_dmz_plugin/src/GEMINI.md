# Parent Portal DMZ Sync Lua Runtime (`plugins/parent_dmz_plugin/src`)

This directory contains the Lua 5.1 (`gopher-lua`) entrypoint for `uk.org.optimis.parent.dmz`.

## Files

* [`init.lua`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/src/init.lua):
  * `pull_dmz_inbox(ctx, args)`: Drains queued parent absence submissions via `ctx.http.post("https://dmz.parent.optimis.org.uk/v1/inbox/pull", ...)` prior to opening any SQLite write transaction, parses the response via `ctx.json.decode(resp.body)`, performs coarse-grained $O(1)$ batch lookups via `ctx.dal.pupils.get_by_ids_batch` and `ctx.dal.guardians.list_for_pupils_batch(..., { require_parental_responsibility = true })`, stamps `processed_at` with `ctx.clock.now_utc()`, and persists `ACCEPTED` vs `REJECTED_SAFEGUARDING_BLOCK` rows into `plugin_parent_dmz.dmz_inbox_submissions`.
  * `push_sanitized_projection(ctx, args)`: Fetches pupil, guardian, and AM/PM session attendance summaries via `ctx.dal.pupils.get_by_id`, `ctx.dal.guardians.list_for_pupil`, and `ctx.dal.attendance.get_pupil_summary`, strips statutory `UPN`/`ULN` and confidential household addresses, serializes the projection via `ctx.json.encode`, pushes the sanitized JSON payload via `ctx.http.post("https://dmz.parent.optimis.org.uk/v1/projections/push", ...)`, and records the snapshot in `plugin_parent_dmz.dmz_projection_snapshots`.
  * `list_dmz_sync_history(ctx, args)`: Lists persisted inbox submissions and projection snapshots from `plugin_parent_dmz.db`.
  * `export_pupil_data(ctx, args)` & `purge_pupil_data(ctx, args)`: Mandatory UK GDPR SAR export and retention purge hooks (auto-bound via `manifest.json` `gdpr_hooks`) across `dmz_inbox_submissions` and `dmz_projection_snapshots`.
