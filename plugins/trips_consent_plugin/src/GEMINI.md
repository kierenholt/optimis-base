# Trip Consent Plugin Lua Runtime (`plugins/trips_consent_plugin/src`)

This directory contains the capability-sandboxed Lua 5.1 (`gopher-lua`) backend for `uk.org.optimis.ops.trips`.

## Files

* [`init.lua`](file:///home/kieren/Documents/code/optimis/plugins/trips_consent_plugin/src/init.lua):
  * `create_trip(ctx, args)`: Verifies the caller's `:own_groups` access to `group_id` via `ctx.dal.pupils.list_by_group` (`strict_own_groups = true`), persists the visit in `plugin_trips.trips`, and returns the active group roster.
  * `record_trip_consent(ctx, args)`: Verifies `pupil_id` belongs to the trip's active group roster, queries `ctx.dal.guardians.list_for_pupil(pupil_id, { require_parental_responsibility = true, ... })`, fails fast with `GUARDIAN_NOT_AUTHORIZED_FOR_CONSENT` if `guardian_person_id` is not an unrestricted `PR` guardian (`has_parental_responsibility = 1` and `court_order_restricted = 0`), and upserts the verified consent record into `plugin_trips.trip_consents`.
  * `generate_trip_emergency_pack(ctx, args)`: Queries consented pupils from `plugin_trips.trip_consents`, batch-fetches SEN (`E`/`K` + primary need) and Medical (`IHCP` + emergency response plan) alerts via `ctx.dal.pupils.list_sen_and_medical_batch`, and returns the structured Trip Emergency Manifest.
  * `export_pupil_data(ctx, args)` & `purge_pupil_data(ctx, args)`: UK GDPR SAR export hook (`redactable_fields = { "dietary_medical_notes" }`) and retention purge hook.
