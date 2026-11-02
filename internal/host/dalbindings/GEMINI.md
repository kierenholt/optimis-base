# Core DAL Lua Capability Bindings (`internal/host/dalbindings`)

This subpackage translates capability-scoped Core DAL methods into sandboxed Lua functions exposed to plugins under `ctx.dal.<domain>` (Section 9.2 of the Architecture Blueprint).

## Architectural Role & Boundaries

* **Decoupled from Host Runtime:** The package does not depend on `*host.Host`. It receives `*dal.CoreDAL`, `types.EndpointSpec`, and `types.CallerContext` via [`Bind`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/bindings.go#L31).
* **Security & Scoping Invariant:** Every domain method enforces caller RBAC (`:all` vs `:own_groups`), mutability verification (`mutates: true`), and temporal reference resolution.
* **Deterministic Lua Tables:** Arrays are tagged with `__is_array` metatables via `markLuaTableAsArray` for lossless JSON round-trips.

## File Catalog

* [`bindings.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/bindings.go): Core entrypoint `Bind()`, `Binder` struct, scope resolution (`resolveEffectiveDomainScopeMode`), PII scrubber helper, and table conversion utilities.
* [`academic.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/academic.go): Binds `ctx.dal.academic` (`list_years`, `upsert_year`, `list_terms`, `upsert_term`, `list_days_of_week`, `upsert_day_of_week`, `list_special_days`, `upsert_special_day`, `list_alert_levels`).
* [`pupils.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/pupils.go): Binds `ctx.dal.pupils` (`get_by_id`, `get_by_ids_batch`, `list_all`, `list_by_group`, `upsert_pupil_admission`, `upsert_year_enrolment`, `promote_pupil_enrolment`).
* [`pupils_pastoral.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/pupils_pastoral.go): Pastoral statutory extensions (`list_fsm_periods_batch`, `upsert_fsm_period`, `list_exclusions`, `upsert_exclusion`, `list_alternative_provisions`, `upsert_alternative_provision`, `list_post16_learning_aims`, `upsert_post16_learning_aim`, `list_pastoral_notes`, `upsert_pastoral_note`).
* [`sen.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/sen.go): Binds `ctx.dal.sen` (`list_batch`, `list_register`, `list_need_types`, `upsert_need_type`, `upsert_sen_record`, `upsert_sen_need`, `upsert_sen_assistant`).
* [`medical.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/medical.go): Binds `ctx.dal.medical` (`list_batch`, `list_conditions`, `upsert_condition`).
* [`groups.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/groups.go): Core groups domain entrypoint, access guards (`assert_caller_assigned`, `is_caller_assigned`), and roster read/write bindings.
* [`groups_timetable.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/groups_timetable.go): Timetable sub-domain bindings for slots, periods, and free staff query.
* [`groups_curriculum.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/groups_curriculum.go): Curriculum sub-domain bindings for rooms, courses, departments, year groups, and houses.
* [`staff.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/staff.go): Binds `ctx.dal.staff` (`list_active`, `list_all`, `upsert_staff_member`, `list_roles`, `list_scr_register`, `upsert_scr_record`, `list_contracts`, `upsert_contract`, `list_absences`, `upsert_absence`, `list_cover_assignments`, `upsert_cover_assignment`).
* [`attendance.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/attendance.go): Binds `ctx.dal.attendance` (`list_codes`, `upsert_code`, `get_group_lesson`, `upsert_lesson_batch`, `list_session_marks`, `list_register_submissions`).
* [`guardians.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/guardians.go): Binds `ctx.dal.guardians` (`upsert_guardian_contact`, `list_siblings_for_pupil`, `list_households`) and `ctx.dal.safeguarding` (`create_postbox`).
* [`census.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/census.go): Binds `ctx.dal.census` (`get_statutory_rollup`).
* [`dashboard.go`](file:///home/kieren/Documents/code/optimis/internal/host/dalbindings/dashboard.go): Binds `ctx.dal.dashboard` (`list_available_widgets`, `list_role_defaults`, `get_role_default`, `save_role_default`, `append_widget_to_role`, `get_staff_layout`, `save_staff_layout`, `reset_staff_layout`).
