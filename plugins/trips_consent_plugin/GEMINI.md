# Educational Visits, Parental Consent & Trip Emergency Pack Plugin (`plugins/trips_consent_plugin`)

This directory contains the Vertical 7 Educational Visits, Parental Consent (`PR`) Verification & Trip Emergency Pack plugin (`uk.org.optimis.ops.trips`), backed by an isolated attached SQLite database (`database.alias = "plugin_trips"` $\rightarrow$ `plugin_trips.db`).

---

## 1. Architectural Responsibilities & Contract Surface

* **Plugin Identifier**: `uk.org.optimis.ops.trips` (`v0.1.0`, `core_dal_version: "~1.0"`).
* **Attached Database**: `plugin_trips` (`max_db_size_mb: 64`), storing `trips` and `trip_consents`.
* **Core DAL Capabilities Exercised**:
  * `ctx.dal.pupils.list_by_group(group_id, opts)` (`dal.pupils.read:own_groups`, `dal.groups.read:own_groups`): Verifies `:own_groups` teacher assignment and loads active on-roll pupils for the trip group (excluding mid-year transferred pupils where `effective_to < reference_date`).
  * `ctx.dal.guardians.list_for_pupil(pupil_id, { require_parental_responsibility = true, ... })` (`dal.guardians.read:own_groups`): Enforces that parental consent can only be recorded from an adult holding statutory **Parental Responsibility (`has_parental_responsibility = 1`)** who is not blocked by a court order (`court_order_restricted = 0`), failing fast with `GUARDIAN_NOT_AUTHORIZED_FOR_CONSENT` otherwise.
  * `ctx.dal.pupils.list_sen_and_medical_batch(pupil_ids, opts)` (`dal.sen.read:own_groups`, `dal.medical.read:own_groups`): Enriches the generated Trip Emergency Manifest with SEN provision (`E`/`K` + primary need code) and Medical/IHCP alerts (`condition_name`, `has_ihcp`, `emergency_response_plan`) in a single coarse-grained batch query.
* **Mandatory UK GDPR Hooks (`gdpr_hooks`)**:
  * `export_pupil_data(ctx, args)`: Exports all `trip_consents` rows for `pupil_id` (joined with `trips.trip_name`, `destination`, `trip_date`), declaring `redactable_fields = { "dietary_medical_notes" }` so third-party pupil names in free-text consent notes can be redacted in the Host Admin SAR Redaction UI.
  * `purge_pupil_data(ctx, args)`: Permanently deletes all `trip_consents` rows for `pupil_id` and cleans up any orphaned `trips` records.

---

## 2. Key Files & Subdirectories

* [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/trips_consent_plugin/manifest.json): Default-deny routing, scope, attached database (`plugin_trips`), and UK GDPR hook contract.
* [`migrations/`](file:///home/kieren/Documents/code/optimis/plugins/trips_consent_plugin/migrations/GEMINI.md): Expand-and-Contract DDL migrations for `plugin_trips.db` ([`001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/trips_consent_plugin/migrations/001_init.sql)).
* [`src/`](file:///home/kieren/Documents/code/optimis/plugins/trips_consent_plugin/src/GEMINI.md): Capability-sandboxed Lua 5.1 implementation ([`init.lua`](file:///home/kieren/Documents/code/optimis/plugins/trips_consent_plugin/src/init.lua)).
* [`ui/`](file:///home/kieren/Documents/code/optimis/plugins/trips_consent_plugin/ui/GEMINI.md): Sandboxed `<iframe sandbox="allow-scripts">` offline UI bundle ([`dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/trips_consent_plugin/ui/dist/index.html)).
