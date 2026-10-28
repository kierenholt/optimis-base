# Parent Portal DMZ Sync Lua Runtime (`plugins/parent_dmz_plugin/src`)

This directory contains the Lua 5.1 (`gopher-lua`) entrypoint for `uk.org.optimis.parent.dmz`.

## Files

* [`init.lua`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/src/init.lua):
  * `pull_dmz_inbox(ctx, args)`: Drains queued parent absence submissions via `ctx.http.post("https://dmz.parent.optimis.org.uk/v1/inbox/pull", ...)` prior to opening any SQLite write transaction, validates guardian Parental Responsibility and court-order safeguarding blocks via `ctx.dal.guardians.list_for_pupil(..., { require_parental_responsibility = true })`, and persists `ACCEPTED` vs `REJECTED_SAFEGUARDING_BLOCK` rows into `plugin_parent_dmz.dmz_inbox_submissions`.
  * `push_sanitized_projection(ctx, args)`: Fetches pupil, guardian, and AM/PM session attendance summaries via `ctx.dal.pupils.get_by_id`, `ctx.dal.guardians.list_for_pupil`, and `ctx.dal.attendance.get_pupil_summary`, strips statutory `UPN` and confidential household addresses, pushes the sanitized JSON payload via `ctx.http.post("https://dmz.parent.optimis.org.uk/v1/projections/push", ...)`, and records the snapshot in `plugin_parent_dmz.dmz_projection_snapshots`.
  * `list_dmz_sync_history(ctx, args)`: Lists persisted inbox submissions and projection snapshots from `plugin_parent_dmz.db`.
  * `export_pupil_data(ctx, args)` & `purge_pupil_data(ctx, args)`: Mandatory UK GDPR SAR export and retention purge hooks across `dmz_inbox_submissions` and `dmz_projection_snapshots`.
