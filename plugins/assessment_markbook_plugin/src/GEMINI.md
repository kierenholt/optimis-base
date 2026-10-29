# Assessment & Markbook Lua Runtime (`plugins/assessment_markbook_plugin/src`)

This directory contains the Lua 5.1 (`gopher-lua`) entrypoint for `uk.org.optimis.assessment.markbook`.

## Files

* [`init.lua`](file:///home/kieren/Documents/code/optimis/plugins/assessment_markbook_plugin/src/init.lua):
  * `get_markbook_workspace(ctx, args)`: Returns the synced MAT grading schema version from `ctx.school.mat_grading_schema_version`, the caller's scoped teaching groups (`ctx.dal.groups.list_groups`), active class roster (`ctx.dal.pupils.list_by_group`), markbook columns, and saved pupil grades.
  * `create_markbook_column(ctx, args)`: Verifies MAT grading schema sync (`MAT_GRADING_SCHEMA_NOT_SYNCED`), enforces `:own_groups` vs `:all` access to `group_id` via `ctx.dal.pupils.list_by_group`, validates `grading_scale` (`9-1` or `WT,EXS,GDS`), and inserts into `plugin_markbook.markbook_columns`.
  * `save_pupil_grades(ctx, args)`: Verifies MAT schema sync, enforces caller group scope and active pupil membership on `reference_date`, validates each grade against the column's MAT scale (`INVALID_MAT_GRADE`), and upserts rows into `plugin_markbook.markbook_grades`.
  * `get_department_attainment_rollup(ctx, args)`: Requires `dal.groups.read:all` and `dal.pupils.read:all` (HoD/Admin only), aggregating graded counts, mean attainment points, and highest-attaining set across department teaching classes.
  * `export_pupil_data(ctx, args)` & `purge_pupil_data(ctx, args)`: Mandatory UK GDPR SAR export (`title = "Assessment Markbook Grades & Teacher Comments"`, `redactable_fields = { "teacher_comment" }`) and zero-orphan retention purge hooks on `plugin_markbook.db`.
