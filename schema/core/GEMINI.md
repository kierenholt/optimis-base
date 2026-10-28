# Core SQLite Schema (`schema/core`)

This directory defines the 45-table modular SQLite schema for `core.db`, executed in lexical order (`00` through `06`) by [`CoreDAL.initSchema`](file:///home/kieren/Documents/code/optimis/internal/dal/dal.go#L98-L155).

## Architectural Invariants

* **Strict Core vs. Plugin Boundary**: Holds only Statutory, Safeguarding, and Common Non-Proprietary UK All-Through (N1–Y13) school data. Custom gradebooks, behaviour point schemes, catering, and third-party sync state live in attached `plugin_<name>.db` files.
* **Temporal Effective Dating**: Enrolments (`pupil_year_enrolments`), group memberships (`group_pupil_memberships`), staff assignments (`group_staff_assignments`), guardian relationships (`guardian_relationships`), and SEN records (`pupil_sen_records`) carry `effective_from` and `effective_to` boundaries.
* **Safeguarding & Court-Order Isolation**: `guardian_relationships.court_order_restricted = 1` enforces zero contact/portal permissions via SQLite `CHECK` constraints, and `safeguarding_notes` / `safeguarding_audit_logs` enforce envelope encryption and append-only audit triggers.

## Key Files

* [`SCHEMA_MAP.md`](file:///home/kieren/Documents/code/optimis/schema/core/SCHEMA_MAP.md): Master agent navigation map indexing all 45 tables and 75 foreign keys across domains `00`–`06`.
* [`00_system_and_academic_cycle.sql`](file:///home/kieren/Documents/code/optimis/schema/core/00_system_and_academic_cycle.sql): `__system_config`, `academic_years`, `academic_terms`, `days_of_week`, `academic_special_days`, `alert_levels`, `data_retention_policies`.
* [`01_identity_pupils_and_staff.sql`](file:///home/kieren/Documents/code/optimis/schema/core/01_identity_pupils_and_staff.sql): `roles`, `persons`, `person_roles`, `pupils` (DfE CBDS), `pupil_year_enrolments`, `staff` (SWC/SCR), `staff_contracts`, `staff_absences`, `staff_cover_assignments`.
* [`02_households_and_guardians.sql`](file:///home/kieren/Documents/code/optimis/schema/core/02_households_and_guardians.sql): `households` (BS7666 / `is_confidential_address`), `household_memberships`, `guardian_relationships` (`has_parental_responsibility` & `court_order_restricted` lockdown `CHECK`).
* [`03_groups_and_timetable.sql`](file:///home/kieren/Documents/code/optimis/schema/core/03_groups_and_timetable.sql): `year_groups`, `houses`, `departments`, `rooms`, `courses`, `groups`, `group_pupil_memberships`, `group_staff_assignments`, `timetable_periods`, `timetable_slots`.
* [`04_statutory_attendance.sql`](file:///home/kieren/Documents/code/optimis/schema/core/04_statutory_attendance.sql): `attendance_codes`, `attendance_session_marks` (`AM`/`PM`), `attendance_lesson_marks`, `attendance_register_submissions`.
* [`05_sen_medical_and_characteristics.sql`](file:///home/kieren/Documents/code/optimis/schema/core/05_sen_medical_and_characteristics.sql): `pupil_fsm_periods`, `sen_need_types`, `pupil_sen_records` (`N`/`K`/`E`), `pupil_sen_needs`, `pupil_sen_assistants`, `pupil_suspensions_exclusions`, `pupil_alternative_provisions`, `pupil_post16_learning_aims`, `pupil_medical_conditions`, `pupil_pastoral_notes`.
* [`06_safeguarding_vault.sql`](file:///home/kieren/Documents/code/optimis/schema/core/06_safeguarding_vault.sql): `safeguarding_notes` and `safeguarding_audit_logs` with append-only and KCSIE retention-lock SQLite triggers.
