# Optimis `core.db` Schema Map — Agent Navigation Entrypoint

> **Agent Instructions:** Always read this file **first** before inspecting or modifying database tables. Every domain SQL file in [`schema/core/`](./) is under **235 lines** so you can read any domain in a single `view_file` call.

---

## 1. Domain Files & Bounded Contexts

| Domain File | Tables | Lines | DAL Scopes (`ctx.dal.*`) | Summary |
| :--- | ---: | ---: | :--- | :--- |
| [`00_system_and_academic_cycle.sql`](./00_system_and_academic_cycle.sql) | 7 | 110 | `host.internal`, `dal.academic.*`, `dal.census.read:all` | `__system_config`, Academic Years, Terms, Special Closure Days, Days of Week, Alert Levels, GDPR Retention Rules |
| [`01_identity_pupils_and_staff.sql`](./01_identity_pupils_and_staff.sql) | 9 | 216 | `dal.identity.*`, `dal.pupils.*`, `dal.staff.*`, `dal.census.read:all` | 1-row-per-person `persons`, `pupils` (DfE CBDS), & `staff` (SWC/SCR); temporal `person_roles`, `pupil_year_enrolments` (incl. EYFS hours), `staff_contracts`, `staff_absences`, `staff_cover_assignments` |
| [`02_households_and_guardians.sql`](./02_households_and_guardians.sql) | 3 | 100 | `dal.guardians.*` | BS7666 `households` (`is_confidential_address`), temporal `household_memberships`, `guardian_relationships` (`has_parental_responsibility` & full `court_order_restricted` lockdown `CHECK`) |
| [`03_groups_and_timetable.sql`](./03_groups_and_timetable.sql) | 10 | 177 | `dal.groups.*` | `year_groups`, `houses`, `departments`, `rooms`, `courses`, unified `groups`, temporal `group_pupil_memberships` & `group_staff_assignments` (`:own_groups` RBAC), `timetable_periods` & `timetable_slots` |
| [`04_statutory_attendance.sql`](./04_statutory_attendance.sql) | 4 | 103 | `dal.attendance.*`, `dal.census.read:all` | DfE `attendance_codes`, statutory `attendance_session_marks` (`AM`/`PM`), `attendance_lesson_marks`, `attendance_register_submissions` |
| [`05_sen_medical_and_characteristics.sql`](./05_sen_medical_and_characteristics.sql) | 10 | 228 | `dal.sen.*`, `dal.medical.*`, `dal.pupils.*`, `dal.census.read:all` | `pupil_fsm_periods`, `sen_need_types`, `pupil_sen_records` (`N`/`K`/`E`), `pupil_sen_needs` (Rank 1 & 2), `pupil_sen_assistants`, `pupil_suspensions_exclusions`, `pupil_alternative_provisions`, `pupil_post16_learning_aims`, `pupil_medical_conditions`, `pupil_pastoral_notes` |
| [`06_safeguarding_vault.sql`](./06_safeguarding_vault.sql) | 2 | 77 | `dal.safeguarding.create` / `dal.safeguarding.read` | Envelope-encrypted `safeguarding_notes` (protected by KCSIE retention trigger) + append-only `safeguarding_audit_logs` with SQLite immutability triggers |

---

## 2. Complete `core.db` Entity-Relationship Reference (45 Tables, 75 Indexed Foreign Keys)

| Table Name | File | Primary Key | Foreign Keys (`-> target`) | Temporal Columns | Key Invariants & DfE CBDS Alignment |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `__system_config` | `00` | `key` | `-` | `updated_at` | Self-contained runtime config (`school_urn`, OAuth settings, manifests) |
| `academic_years` | `00` | `id` | `-` | `first_day`, `last_day` | `status IN ('PAST','CURRENT','UPCOMING')`; `CHECK (last_day >= first_day)` |
| `academic_terms` | `00` | `id` | `academic_year_id -> academic_years.id` | `first_day`, `last_day` | Maps to `AUTUMN`, `SPRING`, `SUMMER` Census windows |
| `days_of_week` | `00` | `id` | `-` | `-` | ISO `1..7`; stores `am_register_time` & `pm_register_time` |
| `academic_special_days` | `00` | `id` | `academic_term_id -> academic_terms.id` | `date` | Maps to DfE Attendance Code `#` (Planned Closure / INSET) |
| `alert_levels` | `00` | `id` | `-` | `-` | `LOW`, `MEDIUM`, `HIGH`, `CRITICAL` severity badges |
| `data_retention_policies` | `00` | `domain_key` | `-` | `-` | Statutory retention floors (e.g., SEN = DOB+31y, Safeguarding = DOB+25y) |
| `roles` | `01` | `id` | `-` | `-` | `STAFF`, `PUPIL`, `GUARDIAN`, `SYSTEM` role definitions |
| `persons` | `01` | `id` | `-` | **Non-temporal** (1 row/person) | Shared identity spine (`external_sso_sub`, legal vs preferred names, `dob`, `sex`) |
| `person_roles` | `01` | `(person_id, role_id, effective_from)` | `person_id -> persons.id`, `role_id -> roles.id` | `effective_from`, `effective_to` | Temporal role assignment |
| `pupils` | `01` | `id` | `person_id -> persons.id` (`NOT NULL UNIQUE`) | **Non-temporal** (1 row/pupil) | **DfE CBDS:** `upn` (`UNIQUE`), `uln`, `nhs_number`, `ethnicity_code`, `first_language_code`, `pupil_premium_flag`, `looked_after_flag` |
| `pupil_year_enrolments` | `01` | `id` | `pupil_id -> pupils.id`, `academic_year_id -> academic_years.id` | `effective_from`, `effective_to` | Temporal admission/roll: `nc_year_group` (`E1`..`14`), `enrolment_status` (`C`/`M`/`S`/`F`/`O`), EYFS `ey_funded_hours` & `ey_thirty_hour_code` |
| `staff` | `01` | `id` | `person_id -> persons.id` (`NOT NULL UNIQUE`) | **Non-temporal** (1 row/staff) | **DfE SWC & SCR:** `trn`, `qts_flag`, `hlta_flag`, `is_dsl`, `dbs_check_date`, `hire_date`, `termination_date` |
| `staff_contracts` | `01` | `id` | `staff_id -> staff.id` | `effective_from`, `effective_to` | **DfE SWC:** `swc_post_code`, `pay_scale`, `pay_spine_point`, `fte` |
| `staff_absences` | `01` | `id` | `staff_id -> staff.id`, `academic_year_id -> academic_years.id`, `approved_by_staff_id -> staff.id` | `start_date`, `end_date` | **DfE SWC:** `absence_category_code`, `working_days_lost`, `cover_required` |
| `staff_cover_assignments` | `01` | `id` | `staff_absence_id -> staff_absences.id`, `absent_staff_id -> staff.id`, `covering_staff_id -> staff.id` | `date` | Tracks daily lesson cover assignments |
| `households` | `02` | `id` | `-` | **Non-temporal** (1 row/place) | **BS7666 / CBDS:** `uprn`, `address_line_1`, `postcode`, `is_confidential_address` |
| `household_memberships` | `02` | `id` | `household_id -> households.id`, `person_id -> persons.id` | `effective_from`, `effective_to` | Temporal residency linking `ADULT` and `CHILD` persons to households |
| `guardian_relationships` | `02` | `id` | `pupil_id -> pupils.id`, `guardian_person_id -> persons.id` | `effective_from`, `effective_to` | **Mandatory Invariant:** `has_parental_responsibility`, `court_order_restricted` (`CHECK` zeroes out portal, collection, emergency, reports, and all contact channels when restricted) |
| `year_groups` | `03` | `id` | `head_of_year_staff_id -> staff.id` | `-` | Maps `nc_code` (`E1`..`14`) to `phase` and `key_stage` (`EYFS`..`KS5`) |
| `houses` | `03` | `id` | `-` | `-` | School house system |
| `departments` | `03` | `id` | `head_of_department_staff_id -> staff.id` | `-` | Subject departments / faculties |
| `rooms` | `03` | `id` | `-` | `-` | Physical teaching spaces & capacities |
| `courses` | `03` | `id` | `academic_year_id -> academic_years.id`, `department_id -> departments.id` | `-` | Curriculum courses & `dfe_subject_code` |
| `groups` | `03` | `id` | `academic_year_id -> academic_years.id`, `year_group_id -> year_groups.id`, `course_id -> courses.id`, `default_room_id -> rooms.id` | `effective_from`, `effective_to` | Unified group spine: `REGISTRATION_FORM`, `TEACHING_CLASS`, `INTERVENTION`, `CUSTOM` |
| `group_pupil_memberships` | `03` | `id` | `group_id -> groups.id`, `pupil_id -> pupils.id`, `academic_year_id -> academic_years.id` | `effective_from`, `effective_to` | **Mandatory Invariant:** Temporal pupil roster powering `:own_groups` RBAC |
| `group_staff_assignments` | `03` | `id` | `group_id -> groups.id`, `staff_id -> staff.id`, `academic_year_id -> academic_years.id` | `effective_from`, `effective_to` | **Mandatory Invariant:** Temporal staff roster powering `:own_groups` RBAC |
| `timetable_periods` | `03` | `id` | `academic_year_id -> academic_years.id`, `day_of_week_id -> days_of_week.id` | `-` | Supports 1- or 2-week rolling cycles (`cycle_week IN (1,2)`); maps to `AM`/`PM` |
| `timetable_slots` | `03` | `id` | `timetable_period_id -> timetable_periods.id`, `group_id -> groups.id`, `staff_id -> staff.id`, `room_id -> rooms.id` | `effective_from`, `effective_to` | Temporal scheduled class lessons |
| `attendance_codes` | `04` | `code` | `-` | `-` | Official DfE attendance codes (`/`, `\`, `L`, `I`, `M`, `N`, `O`, `U`, `#`, etc.) |
| `attendance_session_marks` | `04` | `id` | `pupil_id -> pupils.id`, `academic_year_id -> academic_years.id`, `attendance_code -> attendance_codes.code` | `date`, `session` (`AM`/`PM`) | **Mandatory Invariant:** All phases resolve down to statutory `AM`/`PM` sessions (`ON DELETE RESTRICT`) |
| `attendance_lesson_marks` | `04` | `id` | `pupil_id -> pupils.id`, `timetable_slot_id -> timetable_slots.id`, `group_id -> groups.id`, `attendance_code -> attendance_codes.code` | `date` | Period-by-period subject attendance marks (`ON DELETE RESTRICT`) |
| `attendance_register_submissions` | `04` | `id` | `group_id -> groups.id`, `taken_by_staff_id -> staff.id` | `date`, `taken_at` | Audit trail of completed registers (`ON DELETE RESTRICT`) |
| `pupil_fsm_periods` | `05` | `id` | `pupil_id -> pupils.id` | `effective_from`, `effective_to` | **DfE CBDS:** Statutory Free School Meal eligibility periods |
| `sen_need_types` | `05` | `code` | `-` | `-` | **DfE CBDS:** Official SEN Need codes (`ASD`, `SEMH`, `SPLD`, `SLCN`, etc.) |
| `pupil_sen_records` | `05` | `id` | `pupil_id -> pupils.id` | `effective_from`, `effective_to` | **DfE CBDS:** `provision_code IN ('N','K','E')`, EHCP dates, IEP targets (DOB+31y retention) |
| `pupil_sen_needs` | `05` | `id` | `pupil_id -> pupils.id`, `sen_need_code -> sen_need_types.code`, `alert_level_id -> alert_levels.id` | `effective_from`, `effective_to` | **DfE CBDS:** Ranked primary (`1`) and secondary (`2`) SEN needs |
| `pupil_sen_assistants` | `05` | `id` | `pupil_id -> pupils.id`, `staff_id -> staff.id` | `effective_from`, `effective_to` | TA / LSA support allocations |
| `pupil_suspensions_exclusions` | `05` | `id` | `pupil_id -> pupils.id`, `academic_year_id -> academic_years.id` | `start_date`, `end_date` | **DfE Census:** Suspensions & Permanent Exclusions (up to 3 DfE reasons, `sessions_missed`) |
| `pupil_alternative_provisions` | `05` | `id` | `pupil_id -> pupils.id`, `academic_year_id -> academic_years.id` | `effective_from`, `effective_to` | **DfE Census:** Off-site Alternative Provision placements (`provider_urn`/`ukprn`) |
| `pupil_post16_learning_aims` | `05` | `id` | `pupil_id -> pupils.id`, `academic_year_id -> academic_years.id`, `course_id -> courses.id` | `effective_from`, `planned_end_date`, `actual_end_date` | **DfE Census (KS5):** Post-16 Learning Aims (`qn_code`, `planned_learning_hours`, `completion_status`) |
| `pupil_medical_conditions` | `05` | `id` | `pupil_id -> pupils.id`, `alert_level_id -> alert_levels.id` | `effective_from`, `effective_to` | Medical conditions, allergies, IHCP flags, and emergency response plans |
| `pupil_pastoral_notes` | `05` | `id` | `pupil_id -> pupils.id`, `created_by_staff_id -> staff.id` | `created_at` | Non-safeguarding pastoral notes (redactable in GDPR SAR UI) |
| `safeguarding_notes` | `06` | `id` | `pupil_id -> pupils.id`, `reported_by_staff_id -> staff.id`, `assigned_dsl_staff_id -> staff.id` | `incident_date`, `created_at` | **Envelope Encrypted Vault:** `kms_key_id`, `wrapped_dek`, `nonce`, `ciphertext_payload` + deletion-guard trigger |
| `safeguarding_audit_logs` | `06` | `audit_id` | `note_id -> safeguarding_notes.id`, `pupil_id -> pupils.id`, `actor_person_id -> persons.id` | `timestamp` | Append-only audit log protected by `BEFORE UPDATE/DELETE` SQLite triggers |
