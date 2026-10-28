# Core SQLite Schema (`schema/core`)

This directory defines the 45-table modular SQLite schema for `core.db`, executed in lexical order (`00` through `06`) by [`CoreDAL.initSchema`](file:///home/kieren/Documents/code/optimis/internal/dal/dal.go#L98-L155).

## Architectural Invariants

* **Strict Core vs. Plugin Boundary**: Holds only Statutory, Safeguarding, and Common Non-Proprietary UK All-Through (N1–Y13) school data. Custom gradebooks, behaviour point schemes, catering, and third-party sync state live in attached `plugin_<name>.db` files.
* **Temporal Effective Dating**: Enrolments (`pupil_year_enrolments`), group memberships (`group_pupil_memberships`), staff assignments (`group_staff_assignments`), guardian relationships (`guardian_relationships`), and SEN records (`pupil_sen_records`) carry `effective_from` and `effective_to` boundaries.
* **Safeguarding & Court-Order Isolation**: `guardian_relationships.court_order_restricted = 1` enforces zero contact/portal permissions via SQLite `CHECK` constraints, and `safeguarding_notes` / `safeguarding_audit_logs` enforce envelope encryption and append-only audit triggers.

## Key Files

* [`SCHEMA_MAP.md`](file:///home/kieren/Documents/code/optimis/schema/core/SCHEMA_MAP.md): Master agent navigation map indexing all 45 tables and 75 foreign keys across domains `00`–`06`.
* [`00_system_and_calendar.sql`](file:///home/kieren/Documents/code/optimis/schema/core/00_system_and_calendar.sql): `__system_config`, `academic_years`, `academic_terms`, `days_of_week`, `academic_special_days`, `year_groups`, `houses`, `departments`, `buildings`, `rooms`, `alert_levels`.
* [`01_identity_and_people.sql`](file:///home/kieren/Documents/code/optimis/schema/core/01_identity_and_people.sql): `persons`, `staff`, `staff_qualifications`, `staff_absences`, `pupils`, `pupil_year_enrolments`, `pupil_admissions`, `pupil_ctf_history`.
* [`02_households_and_guardians.sql`](file:///home/kieren/Documents/code/optimis/schema/core/02_households_and_guardians.sql): `households`, `household_memberships`, `guardian_relationships`, `pupil_court_order_blocks`.
* [`03_groups_and_timetable.sql`](file:///home/kieren/Documents/code/optimis/schema/core/03_groups_and_timetable.sql): `courses`, `groups`, `group_staff_assignments`, `group_pupil_memberships`, `timetable_periods`, `timetable_slots`, `timetable_room_exceptions`.
* [`04_statutory_attendance.sql`](file:///home/kieren/Documents/code/optimis/schema/core/04_statutory_attendance.sql): `attendance_codes`, `attendance_session_marks`, `attendance_lesson_marks`, `attendance_register_submissions`, `attendance_future_absences`.
* [`05_sen_medical_and_characteristics.sql`](file:///home/kieren/Documents/code/optimis/schema/core/05_sen_medical_and_characteristics.sql): `sen_need_types`, `pupil_sen_records`, `pupil_sen_needs`, `pupil_ehcp_reviews`, `pupil_medical_conditions`, `pupil_dietary_requirements`, `pupil_fsm_periods`, `pupil_looked_after_episodes`.
* [`06_safeguarding_vault.sql`](file:///home/kieren/Documents/code/optimis/schema/core/06_safeguarding_vault.sql): `safeguarding_notes` and `safeguarding_audit_logs` with append-only and retention-lock SQLite triggers.
