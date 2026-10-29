# Assessment & Markbook Plugin Migrations (`plugins/assessment_markbook_plugin/migrations`)

This folder holds the SQLite schema migrations applied inside `BEGIN IMMEDIATE` transactions onto `plugin_markbook.db`.

## Files

* [`001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/assessment_markbook_plugin/migrations/001_init.sql):
  * `markbook_columns`: Defines per-group assessment columns (`id`, `group_id`, `academic_year_id`, `title`, `grading_scale` constrained to `'9-1'` or `'WT,EXS,GDS'`, `mat_schema_version`, `assessment_date`, `created_by_staff_id`, `created_at`).
  * `markbook_grades`: Stores per-pupil grades and teacher comments (`id`, `column_id`, `group_id`, `pupil_id`, `academic_year_id`, `grade`, `grade_points`, `teacher_comment` redactable in UK GDPR SAR exports, `recorded_by_staff_id`, `updated_at`, with `UNIQUE(column_id, pupil_id)`).
