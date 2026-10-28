# Test Plugin Sandboxed Lua Runtime (`plugins/test_plugin/src`)

This directory contains [`init.lua`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/init.lua), the Lua 5.1 (`gopher-lua`) entrypoint executed inside the Go Host's capability-sandboxed VM (`io`, `os`, `debug`, `package`, `dofile`, `loadfile`, and `require` stripped).

---

## Exported Endpoints & UK GDPR Hooks

Every function receives `(ctx, args)` and returns `{ ok = true, data = ... }` (or raises a fail-fast Lua error when mandatory parameters or capabilities are missing):

* [`get_pupil_record(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/init.lua#L1-L34): Verifies all 5 injected `ctx` namespaces (`caller`, `dal`, `plugin_db`, `events`, `http`), fetches a pupil via `ctx.dal.pupils.get_by_id`, and emits `test.pupil_fetched` on the post-commit Event Bus.
* [`save_pupil_note(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/init.lua#L36-L68): Mutating endpoint (`mutates: true`) executing `ctx.plugin_db.exec` on `WriteDB` to upsert a row in `plugin_test.pupil_notes` and returning the updated note list via `ctx.plugin_db.query`.
* [`list_pupil_notes(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/init.lua#L70-L96): Read-only endpoint (`mutates: false`) executing `ctx.plugin_db.query` against `ReadDB` to return pupil notes and `PRAGMA plugin_test.max_page_count`.
* [`export_pupil_data(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/init.lua#L98-L119): UK GDPR Subject Access Request (SAR) export hook returning structured section tables (`title`, `records`, `redactable_fields`) from `plugin_test.db`.
* [`purge_pupil_data(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/init.lua#L121-L138): UK GDPR retention purge hook deleting all rows tied to `pupil_id` in `plugin_test.db`.
* [`get_group_roster_summary(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/init.lua#L140-L180), [`get_pupil_guardians(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/init.lua#L182-L204), [`record_group_am_attendance(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/init.lua#L206-L248), & [`submit_safeguarding_postbox(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/init.lua#L250-L274): Coarse-grained `:own_groups` Core DAL domain endpoints and write-only safeguarding postbox handler.
