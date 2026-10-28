# Parent Portal DMZ Sync Plugin Migrations (`plugins/parent_dmz_plugin/migrations`)

This folder holds the SQLite schema migrations applied inside `BEGIN IMMEDIATE` transactions onto `plugin_parent_dmz.db`.

## Files

* [`001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/migrations/001_init.sql):
  * `dmz_inbox_submissions`: Audit log of pulled parent absence submissions from the Public Cloud DMZ write-inbox, recording `submission_id`, `pupil_id`, `submitted_by_guardian_person_id`, `guardian_name`, `submission_type`, `date`, `session`, `reason` (redactable in SAR exports), and `status` (`'ACCEPTED'` vs `'REJECTED_SAFEGUARDING_BLOCK'`).
  * `dmz_projection_snapshots`: Audit log of outbound sanitized read-projections pushed to the Public Cloud DMZ, enforcing `upn_stripped = 1` and storing `present_sessions`, `total_sessions`, `authorized_guardian_ids_json`, `redacted_confidential_address_count`, `sanitized_payload_json`, and `notes` (redactable in SAR exports).
