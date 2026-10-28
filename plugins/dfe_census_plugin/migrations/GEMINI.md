# DfE Census Plugin SQLite Migrations (`plugins/dfe_census_plugin/migrations`)

This directory holds the sequential SQL migration scripts (`*.sql`) executed by the Go Host Container against the attached per-plugin SQLite database (`plugin_dfe_census.db`) declared in [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/manifest.json).

---

## 1. Migration Safety & Snapshot Contract

* **Snapshot Rollback (`plugin_dfe_census.db.snapshot`)**: Prior to applying any pending migration scripts, [`Host.provisionAndMigratePluginDB`](file:///home/kieren/Documents/code/optimis/internal/host/plugin_migration.go#L52) checkpoints WAL frames and takes a file-level backup of `plugin_dfe_census.db`, executing DDL inside `BEGIN IMMEDIATE` and rolling back automatically on error.
* **Expand-and-Contract Compliance**: Schema additions preserve backwards compatibility across minor updates.

---

## 2. Migration Files

* [`001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/migrations/001_init.sql):
  * `census_returns`: Stores termly DfE School Census return metadata, summary rollups (`on_roll_total`, `sen_n_count`, `sen_k_count`, `sen_e_count`, `fsm_count`, `lac_count`, `possible_sessions`, `present_sessions`, `auth_absence_sessions`, `unauth_absence_sessions`), validated `<Message>` CBDS `xml_payload`, `xsd_valid`, and DfE COLLECT submission status (`collect_submission_id`, `collect_status`, `submitted_at`).
  * `census_pupil_snapshots`: Stores per-pupil statutory snapshot rows (`return_id`, `pupil_id`, `upn`, `uln`, `nc_year_group`, `enrolment_status`, `sen_provision_code`, `primary_sen_need_code`, `fsm_eligible`, `looked_after_flag`, `lac_episode_status`, session totals, and redactable `notes`) indexed by `pupil_id` (`idx_census_pupil_snapshots_pupil_id`) to power UK GDPR `export_pupil_data` and `purge_pupil_data` hooks.
