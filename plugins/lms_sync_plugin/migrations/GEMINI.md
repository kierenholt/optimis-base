# Google Classroom LMS Sync SQLite Migrations (`plugins/lms_sync_plugin/migrations`)

This directory holds the sequential SQL migration scripts (`*.sql`) executed by the Go Host Container against the attached per-plugin SQLite database (`plugin_lms_sync.db`) declared in [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/manifest.json).

---

## Migration Files

* [`001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/migrations/001_init.sql):
  * `lms_sync_logs`: Stores outbound Google Classroom roster sync audit records (`id`, `event_name`, `hop`, `membership_id`, `group_id`, `pupil_id`, `academic_year_id`, `effective_from`, `effective_to`, `roster_count`, `target_url`, `http_status`, `status`, `external_sync_id`, `caller_type`, `caller_staff_id`, `sync_notes`, `synced_at`), indexed by `pupil_id` (`idx_lms_sync_logs_pupil`) and `group_id` (`idx_lms_sync_logs_group`).
