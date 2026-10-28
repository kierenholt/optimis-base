# DfE Census Plugin Sandboxed Lua Runtime (`plugins/dfe_census_plugin/src`)

This directory contains [`init.lua`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/src/init.lua), the Lua 5.1 (`gopher-lua`) implementation for `uk.gov.dfe.census` executed inside the Go Host's capability-sandboxed VM.

---

## Exported Endpoints & UK GDPR Hooks

* [`generate_and_validate_census_return(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/src/init.lua#L102-L249):
  * Requires scope `dal.census.read:all` (`mutates: true`).
  * Invokes `ctx.dal.census.get_statutory_rollup({ academic_year_id, census_date, term_code, attendance_start_date, attendance_end_date })`.
  * Constructs the DfE CBDS `<Message>` XML tree (`<Header>` + `<Pupils><Pupil>`) via `ctx.xml.build` and validates it against `"dfe_school_census_v1.xsd"` via `ctx.xml.validate_xsd`.
  * Returns `{ ok = false, error_code = "XSD_VALIDATION_FAILED: ..." }` if any statutory XSD rule fails.
  * Persists the validated return into `plugin_dfe_census.census_returns` and per-pupil records into `plugin_dfe_census.census_pupil_snapshots`.
* [`submit_census_return(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/src/init.lua#L251-L321):
  * Requires scope `dal.census.read:all` (`mutates: false` — executes with zero open SQLite write transactions).
  * Loads the XML return from `plugin_dfe_census.census_returns` (or `args.xml_payload`), re-validates against `"dfe_school_census_v1.xsd"`, and dispatches `ctx.http.post` to `https://collect.dfe.gov.uk/v1/school-census/submit` (routed to the Host's Mock DfE COLLECT service when `ENABLE_MOCK_DFE_CENSUS_SERVICE=true`, or failing fast with `MOCK_DFE_CENSUS_SERVICE_DISABLED` when `false`).
* [`list_census_returns(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/src/init.lua#L323-L334):
  * Lists persisted census returns from `plugin_dfe_census.census_returns`.
* [`export_pupil_data(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/src/init.lua#L336-L370):
  * UK GDPR Subject Access Request (SAR) hook returning section `"DfE Statutory School Census Snapshots"` with `redactable_fields = { "notes" }` from `plugin_dfe_census.census_pupil_snapshots`.
* [`purge_pupil_data(ctx, args)`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/src/init.lua#L372-L396):
  * UK GDPR statutory retention purge hook permanently deleting all snapshot rows for `pupil_id` from `plugin_dfe_census.census_pupil_snapshots`.
