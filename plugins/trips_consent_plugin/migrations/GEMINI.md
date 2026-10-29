# Trip Consent Plugin SQLite Migrations (`plugins/trips_consent_plugin/migrations`)

This directory contains the SQLite DDL migrations applied inside `BEGIN IMMEDIATE` to `plugin_trips.db` (`database.alias = "plugin_trips"`).

## Files

* [`001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/trips_consent_plugin/migrations/001_init.sql):
  * `trips`: Stores educational visits (`id`, `group_id`, `trip_name`, `destination`, `trip_date`, `academic_year_id`, `reference_date`, `created_by_staff_id`, `created_at`).
  * `trip_consents`: Stores verified parental consents per pupil per trip (`id`, `trip_id`, `pupil_id`, `guardian_person_id`, `guardian_name`, `relationship_code`, `emergency_phone`, `consent_status`, `dietary_medical_notes`, `recorded_by_staff_id`, `recorded_at`, with `UNIQUE(trip_id, pupil_id)` and indexes on `pupil_id` and `trip_id`).
