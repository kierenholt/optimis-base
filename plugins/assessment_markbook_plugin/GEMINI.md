# Department Assessment & Markbook Plugin (`plugins/assessment_markbook_plugin`)

This directory contains the first-party **Department Assessment & Markbook Plugin** (`uk.org.optimis.assessment.markbook`), implementing MAT-standardized assessment grading (`9–1` and `WT,EXS,GDS`), Classroom Teacher (`:own_groups`) vs Head of Department (`:all`) scope contrast, cross-set attainment comparison, and UK GDPR SAR teacher comment redaction (Sections 1, 3, 5, 6 & 9 of the Architecture Blueprint).

---

## 1. Architectural Responsibilities

* **MAT Policy Sync & Standardized Grading Scales (`POST /api/sync/mat/pull-policies` $\rightarrow$ `__system_config` $\rightarrow$ `ctx.school.mat_grading_schema`)**:
  * Verifies that MAT policies (`mat_id`, `mat_grading_schema_version`, and structured `mat_grading_schema`) have been pulled into `__system_config` via `ctx.school.mat_grading_schema`, failing fast with `MAT_GRADING_SCHEMA_NOT_SYNCED` if unsynced.
  * Dynamically validates grade tokens and numeric point mappings against `ctx.school.mat_grading_schema.scales`:
    * Secondary GCSE `9-1` (`9, 8, 7, 6, 5, 4, 3, 2, 1, U` $\rightarrow$ `9..0` points).
    * Primary KS2 `WT,EXS,GDS` (`GDS, EXS, WT` $\rightarrow$ `3..1` points).
* **`:own_groups` (Classroom Teacher) vs `:all` (Head of Department) Scope Contrast (`ctx.dal.groups.assert_caller_assigned` & `ctx.plugin_db.exec_batch`)**:
  * `create_markbook_column` & `save_pupil_grades` declare `dal.groups.read:own_groups` and `dal.pupils.read:own_groups` and enforce group write ownership via `ctx.dal.groups.assert_caller_assigned(group_id, opts)` and batch persistence via `ctx.plugin_db.exec_batch`. Classroom Teachers can create columns and record grades only for their assigned sets and active on-roll pupils (blocked from other sets and mid-year transferred pupils with `access denied under own_groups scope`), while HoDs holding `:all` can manage any department set via hierarchical `:all` scope subsumption.
  * `get_department_attainment_rollup` declares `dal.groups.read:all` and `dal.pupils.read:all`, restricting cross-set attainment comparison to HoDs and SLT/Admins (rejecting Classroom Teachers with `403`).
* **UK GDPR SAR Redaction & Retention Purge (`gdpr_hooks`)**:
  * Implements mandatory `export_pupil_data` (returning section `"Assessment Markbook Grades & Teacher Comments"` with `redactable_fields = { "teacher_comment" }`) and `purge_pupil_data` (removing all rows for `pupil_id` in `plugin_markbook.markbook_grades`).

---

## 2. Directory Structure

* [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/assessment_markbook_plugin/manifest.json): Default-deny permission manifest declaring `id: "uk.org.optimis.assessment.markbook"`, `database.alias: "plugin_markbook"`, `gdpr_hooks`, and interactive endpoints.
* [`migrations/`](file:///home/kieren/Documents/code/optimis/plugins/assessment_markbook_plugin/migrations/GEMINI.md): Expand-and-Contract SQLite DDL migrations for `plugin_markbook.db`.
* [`src/`](file:///home/kieren/Documents/code/optimis/plugins/assessment_markbook_plugin/src/GEMINI.md): Pure-Go Lua 5.1 (`gopher-lua`) implementation (`init.lua`).
* [`ui/`](file:///home/kieren/Documents/code/optimis/plugins/assessment_markbook_plugin/ui/GEMINI.md): Offline-bundled sandboxed micro-frontend (`ui/dist/index.html`) running inside `<iframe sandbox="allow-scripts">` with `connect-src 'none'` CSP.
