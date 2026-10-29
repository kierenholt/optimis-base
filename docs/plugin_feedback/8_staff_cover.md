# Plugin Contract Feedback — Vertical 8: Cover Manager & Supply Teacher (`uk.org.optimis.ops.cover`)

**Plugin Under Review:** [`plugins/staff_cover_plugin`](file:///home/kieren/Documents/code/optimis/plugins/staff_cover_plugin/GEMINI.md) (`uk.org.optimis.ops.cover`)  
**E2E Verification Suite:** [`tests/e2e/staff_cover_plugin_vertical.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/staff_cover_plugin_vertical.spec.ts)  
**Architecture Blueprint Reference:** [`GEMINI.md` §3, §5, §6, §9 & §10](file:///home/kieren/Documents/code/optimis/GEMINI.md)

**Contract Surfaces Exercised:**
* Attached Plugin SQLite DB (`database.alias: "plugin_cover"`)
* Staff Timetable Lookup (`ctx.dal.groups.get_staff_timetable_slots`)
* Temporal Staff Group Assignment (`ctx.dal.groups.upsert_staff_assignment`) dynamically elevating `:own_groups` access
* Scoped Class Roster + Batch SEN/Medical Alerts (`ctx.dal.pupils.list_by_group`, `ctx.dal.pupils.list_sen_and_medical_batch`)
* Statutory `AM`/`PM` Session Attendance (`ctx.dal.attendance.get_group_session`, `ctx.dal.attendance.upsert_session_batch`)
* Method-Level Scoped RBAC Mixing `:all` (Cover Manager) and `:own_groups` (Cover / Supply Teacher)
* UK GDPR SAR Export & Retention Purge Hooks (`export_pupil_data` with `redactable_fields: ["register_comment"]` & `purge_pupil_data`)

---

## 1. What Worked Well (Keep As-Is)

1. **Temporal `group_staff_assignments` Driving Immediate `:own_groups` Access (`ctx.dal.groups.upsert_staff_assignment`)**
   * Because [`CoreDAL`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go#L112-L162) evaluates `:own_groups` dynamically against `group_staff_assignments` (`effective_from <= reference_date AND (effective_to IS NULL OR effective_to >= reference_date)`), adding [`UpsertGroupStaffAssignmentScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_groups_attendance.go#L435-L492) (`ctx.dal.groups.upsert_staff_assignment`) immediately unlocked `:own_groups` access across **all** downstream Core domains (`pupils`, `sen`, `medical`, and `attendance`) for the cover date.
   * Zero cover-specific special cases were needed inside [`ListPupilSenAndMedicalBatch`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go#L276), [`GetGroupSessionAttendanceScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_groups_attendance.go#L244), or [`UpsertSessionAttendanceBatchScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_groups_attendance.go#L350).

2. **Method-Level `:all` vs. `:own_groups` Separation + Hierarchical Scope Subsumption (`1B`)**
   * Declaring `dal.groups.read:all` and `dal.groups.write:all` on `load_absent_teacher_timetable` and `record_absence_and_assign_cover`, while declaring `:own_groups` scopes on `load_cover_class_roster` and `submit_cover_attendance_batch` in [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/staff_cover_plugin/manifest.json), worked cleanly:
     * Regular Classroom/Supply Teachers (`TEACHER` role, holding only `:own_groups`) are blocked by the Host Gateway from self-assigning cover (`caller lacks required DAL scopes`).
     * Once assigned by the Cover Manager (`ADMIN` / `HOD`), the Supply Teacher's `:own_groups` scope permits loading the class roster, viewing SEN/Medical alerts, and submitting the statutory `AM`/`PM` attendance register for that group on `cover_date`.
     * Thanks to `1B` ([`resolveEffectiveDomainScopeMode`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L34-L56)), a Cover Manager (`:all`) can also invoke `load_cover_class_roster` to inspect any cover class register without being blocked by `ensureGroupInOwnGroups`.

3. **Clean Boundary Between `core.db` and `plugin_cover.db`**
   * Recording the statutory temporal teaching assignment (`group_staff_assignments`) and statutory session marks (`attendance_session_marks`, `attendance_register_submissions`) in `core.db` while storing operational cover workflow data (`absence_reason`, `cover_notes`, `cover_pupil_register_logs`) in `plugin_cover.db` avoided polluting the Core schema with proprietary cover-management columns while keeping statutory attendance 100% in `core.db`.

---

## 2. Where the Contract Is Too Restrictive or Needs Adjustment

### 2.1 Day-Level (`effective_from` / `effective_to`) vs. Period/Slot-Level Granularity in `group_staff_assignments`
* **Observed Friction:**
  In [`schema/core/02_groups_timetable.sql`](file:///home/kieren/Documents/code/optimis/schema/core/02_groups_timetable.sql#L46-L61), `group_staff_assignments` is effective-dated at **calendar-day granularity** (`effective_from`, `effective_to`) and constrains `role` via `CHECK (role IN ('LEAD_TEACHER', 'CO_TEACHER', 'TUTOR', 'SUPPORT'))`.
  1. **Missing `'COVER_TEACHER'` Role:** Because `'COVER_TEACHER'` is not in the SQL `CHECK` constraint, [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/staff_cover_plugin/src/init.lua#L156) had to assign `role = "CO_TEACHER"`. In a real school MIS, permanent co-teachers and single-day supply/cover teachers need distinct role codes so timetable views and SLT reports don't confuse a supply cover assignment with a permanent co-teacher.
  2. **Whole-Day Group Access vs. Single-Period Cover:** Granting `effective_from = cover_date, effective_to = cover_date` on `group_id` grants the cover teacher `:own_groups` access to that `group_id` for the entire calendar day. If a class has a double period or both `AM` and `PM` lessons on the same day and the supply teacher only covers Period 1 (`AM`), they still hold `:own_groups` access to that group during `PM`.
  3. **Revoking / Reassigning Cover:** If a Cover Manager reassigns cover from Supply Teacher A to Supply Teacher B before the lesson starts, there is currently no `ctx.dal.groups.revoke_staff_assignment` or `effective_to` deactivation helper to strip Supply Teacher A's `:own_groups` grant on that same `effective_from` date.
* **Recommended Contract Adjustment:**
  * Expand the `group_staff_assignments.role` constraint in [`schema/core/02_groups_timetable.sql`](file:///home/kieren/Documents/code/optimis/schema/core/02_groups_timetable.sql#L51) to include `'COVER_TEACHER'`.
  * Add `ctx.dal.groups.delete_staff_assignment(input, opts)` (or allow `upsert_staff_assignment` to soft-revoke/delete a temporary cover row when cover is reassigned).

### 2.2 `ctx.dal.pupils.list_by_group` Defaults to `ScopeAll` Unless `{ strict_own_groups = true }` Is Passed (`1C` Side-Effect)
* **Observed Friction:**
  When Enhancement `1C` (Relaxed Core Pupil Read Visibility) was introduced in [`internal/host/dal_bindings.go`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L69-L77), `pupilsReadMode` was set to `dal.ScopeAll` for all `ctx.dal.pupils.*` read methods, and [`ctx.dal.pupils.list_by_group`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L320-L351) was changed to only enforce `pupilsMode` (`ScopeOwnGroups`) if the Lua caller explicitly passes `{ strict_own_groups = true }` in `opts`.
  * While `ctx.dal.pupils.list_sen_and_medical_batch` and `ctx.dal.attendance.get_group_session` still enforce `ScopeOwnGroups` by default, a plugin author who writes `ctx.dal.pupils.list_by_group(group_id, { academic_year_id = ..., reference_date = ... })` without knowing about the undocumented `strict_own_groups = true` flag will inadvertently bypass `:own_groups` on the initial pupil list query.
* **Recommended Contract Adjustment:**
  * Invert the default on `ctx.dal.pupils.list_by_group`: when an endpoint declares `dal.pupils.read:own_groups` and the caller only holds `:own_groups`, `ctx.dal.pupils.list_by_group` should enforce `ScopeOwnGroups` **by default** (requiring no special flag).
  * Plugins that intentionally want school-wide pastoral group browsing under `1C` already have the explicit `ctx.dal.groups.list_group_pupils(group_id, opts)` and `ctx.dal.pupils.list_all(opts)` methods (or can pass `{ allow_all_groups = true }`). Security boundaries declared in `manifest.json` should always be enforced by default without opt-in flags in Lua.

### 2.3 Persona Switching in the Host UI Shell Does Not Notify Mounted Keep-Alive Iframes (`mis:persona:changed`)
* **Observed Friction:**
  Under Section 9A (Keep-Alive LRU Pool), switching personas via the Topbar Persona Switcher ([`ui/src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx#L390-L435)) updates the parent React Shell's `activePersona` state and sends the new `X-MIS-Staff-ID` header on subsequent RPC calls.
  * However, mounted plugin iframes in the Keep-Alive pool are **not sent a `postMessage` notification** when the user switches personas, and `window.mis.activeRoles`, `window.mis.scopes`, `window.mis.permittedMethods`, and `window.mis.context.staff_id` inside [`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js) remain frozen at the persona that was active when the iframe first mounted.
  * For multi-role operational plugins like `staff_cover_plugin` (where a Cover Manager assigns cover and then switches to the Cover Teacher persona in local dev/testing, or where UI buttons are conditionally enabled based on `window.mis.permittedMethods`), the iframe has no lifecycle hook to react to a persona change.
* **Recommended Contract Adjustment:**
  * When `handlePersonaSwitch` completes in [`ui/src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx), broadcast an updated `mis:init` (or `mis:persona:changed`) message containing the new caller's `{ context, permitted_methods, scopes, active_roles }` to all mounted iframes in the Keep-Alive pool and Dashboard Widget Grid, and expose `window.mis.onPersonaChanged(callback)` in [`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js).

### 2.4 Coarse-Grained Timetable Availability Lookup for Cover Management (`ctx.dal.groups`)
* **Observed Friction:**
  [`ctx.dal.groups.get_staff_timetable_slots(staff_id, cycle_week, day_of_week_id, opts)`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L538-L586) queries timetable slots for a **single `staff_id`**.
  * When a Cover Manager selects an absent teacher's `P1_AM` slot (`slot_id` / `period_id`), the natural next step in a production Cover Manager UI is to list **which active teachers are free during that period** (i.e. have no `timetable_slots` row for that `cycle_week`, `day_of_week_id`, and `period_id`).
  * With only single-staff `get_staff_timetable_slots`, computing free teachers in Lua requires calling `ctx.dal.staff.list_active()` (~95 staff in our QA seed) and executing 95 separate `get_staff_timetable_slots` DAL queries ($N+1$ loop).
* **Recommended Contract Adjustment:**
  * Add a coarse-grained batch method on `ctx.dal.groups` (gated by `dal.groups.read:all`), such as:
    * `ctx.dal.groups.get_staff_timetable_slots_batch(staff_ids, cycle_week, day_of_week_id, opts)` or
    * `ctx.dal.groups.list_free_staff_for_period(period_id, opts)`
  * This keeps the Lua VM boundary coarse-grained (Section 7A) and avoids $N+1$ query loops when rendering cover candidate pickers.

### 2.5 Hardcoded Plugin ID Branching in HTTP Status Code Mapping (`internal/host/server.go`)
* **Observed Friction:**
  In [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L186-L202), when `ExecuteRPC` returns an error containing `"caller lacks required DAL scopes"`, the HTTP gateway decides whether to return HTTP `403 Forbidden` vs. HTTP `400 Bad Request` by checking a hardcoded list of plugin IDs (`uk.org.optimis.parent.dmz`, `uk.org.optimis.assessment.markbook`, `uk.org.optimis.ops.cover`, etc.).
* **Recommended Contract Adjustment:**
  * Remove per-plugin ID branching in [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L186-L202). All plugins should share a single, deterministic HTTP status and `{ ok: false, error_code }` contract on RBAC scope denials.

---

## 3. Summary Verdict

The Plugin Contract is **not overly restrictive** for daily school operations workflows: combining `ctx.dal.groups.get_staff_timetable_slots`, temporal `ctx.dal.groups.upsert_staff_assignment`, `:own_groups`-scoped `ctx.dal.pupils.list_by_group` / `list_sen_and_medical_batch`, and `ctx.dal.attendance.upsert_session_batch` with an isolated `plugin_cover.db` allowed the entire Cover Manager & Supply Teacher vertical to be built in ~320 lines of Lua with full UK GDPR SAR/purge compliance.

Addressing the **5 adjustments above**—especially adding `'COVER_TEACHER'` + assignment revocation to `group_staff_assignments`, enforcing `ScopeOwnGroups` by default on `list_by_group`, broadcasting persona switches to Keep-Alive iframes, and adding a batch timetable availability query—will remove the remaining edge cases before locking the `mis-sdk verify` contract.
