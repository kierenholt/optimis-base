# Plugin Contract Feedback — Vertical 1: Timetable & Statutory Session Attendance (`uk.org.optimis.timetable.attendance`)

**Plugin Under Review:** [`plugins/timetable_attendance_plugin`](file:///home/kieren/Documents/code/optimis/plugins/timetable_attendance_plugin/GEMINI.md) (`uk.org.optimis.timetable.attendance`)  
**E2E Verification Suite:** [`tests/e2e/timetable_attendance_plugin_vertical.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/timetable_attendance_plugin_vertical.spec.ts)  
**Architecture Blueprint Reference:** [`GEMINI.md` §9 (Plugin Contract Specification) & §10 (Runtime & Resource Contract)](file:///home/kieren/Documents/code/optimis/GEMINI.md)

---

## 1. What Worked Well (Do Not Loosen)

1. **Zero-DB Pure Core DAL Plugins (`database` & `gdpr_hooks` Optionality):**
   * Allowing [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/timetable_attendance_plugin/manifest.json) to omit `database` and `gdpr_hooks` when a plugin only orchestrates Core DAL domains (`groups`, `pupils`, `attendance`) worked cleanly.
   * The Go Host ([`internal/host/plugin_lifecycle.go`](file:///home/kieren/Documents/code/optimis/internal/host/plugin_lifecycle.go)) only requires `gdpr_hooks` when `database.alias` is present, avoiding boilerplate no-op SAR/purge hooks on stateless workflow plugins.
2. **Dynamic `:own_groups` Enforcement Inside the Go DAL:**
   * Having [`GetStaffTimetableSlotsScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_groups_attendance.go#L187), [`ListPupilsByGroupScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go#L118), [`ListPupilSenAndMedicalBatch`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go#L194), and [`UpsertSessionAttendanceBatchScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_groups_attendance.go#L350) enforce temporal `group_staff_assignments` and `group_pupil_memberships` in Go meant [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/timetable_attendance_plugin/src/init.lua) needed **zero** custom RBAC or effective-dating SQL logic.
3. **ReadDB vs WriteDB Routing via `mutates: true/false`:**
   * Declaring `mutates: false` on `load_staff_timetable` and `load_slot_roster_with_flags` and `mutates: true` on `submit_session_attendance_batch` in [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/timetable_attendance_plugin/manifest.json) gave hard guarantees at the [`dal_bindings.go`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L395) boundary without the plugin author thinking about SQLite connection pools.

---

## 2. Friction Points & Recommended Contract Adjustments

### A. `dal.pupils.read` vs `dal.sen.read` / `dal.medical.read` Scope Granularity
* **Observation:** In [`internal/dal/persona_auth.go`](file:///home/kieren/Documents/code/optimis/internal/dal/persona_auth.go#L555), the `TEACHER` role grants `dal.sen.read:own_groups` and `dal.medical.read:own_groups` in addition to `dal.pupils.read:own_groups`. However, in [`internal/host/dal_bindings.go`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L78-L198), `ctx.dal.pupils.list_sen_and_medical_batch` is gated solely behind `dal.pupils.read`.
* **Recommended Adjustment:**
  * Gate `ctx.dal.pupils.list_sen_and_medical_batch` (or split/sub-scope it) so that calling it requires `dal.sen.read:<mode>` and `dal.medical.read:<mode>` in `manifest.json` (in addition to `dal.pupils.read:<mode>`). That way, a third-party homework or gradebook plugin granted `dal.pupils.read:own_groups` does not automatically receive sensitive medical/IHCP conditions unless explicitly requested in `manifest.json` and approved by the DPO.

### B. Lua DAL Field Naming & Shape Consistency (`pupilToLuaTable` vs SQL Schema)
* **Observation:** While implementing [`plugins/timetable_attendance_plugin/src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/timetable_attendance_plugin/src/init.lua#L138), we hit a subtle mismatch between the Core SQLite schema (`persons` uses `legal_first_name`, `legal_last_name`, `preferred_first_name`, `preferred_last_name`, and `pupil_year_enrolments` has `nc_year_group`) and [`pupilToLuaTable`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L55-L67), which only projected `{ id, upn, first_name, last_name, academic_year_id, effective_from, effective_to }`.
* **Recommended Adjustment:**
  * Expand [`dal.Pupil`](file:///home/kieren/Documents/code/optimis/internal/dal/dal.go#L27) and [`pupilToLuaTable`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L55) to include `preferred_first_name`, `preferred_last_name`, `legal_first_name`, `legal_last_name`, and `nc_year_group`. In UK schools, classroom registers must display a pupil's **preferred name** to teachers while statutory exports use legal names.

### C. Declarative `data-mis-rpc` + `data-mis-args` in [`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js) for Dynamic Forms
* **Observation:** Sections 6C & 9.4 emphasize declarative `data-mis-rpc="<method>"` + `data-mis-args='<json>'` attributes on HTML elements. For static actions that works cleanly, but for a 30-pupil attendance register table ([`plugins/timetable_attendance_plugin/ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/timetable_attendance_plugin/ui/dist/index.html#L277-L307)), the UI had to re-serialize the entire `marks` array into the submit button's `data-mis-args` attribute on every `<select>` or `<input>` change event so clicking the declarative button sent the latest form state.
* **Recommended Adjustment:**
  * Ensure `mis-sdk verify` (Section 8B Check #1) treats both declarative `data-mis-rpc="<method>"` tags and programmatic `window.mis.rpc("<method>", args)` calls as valid UI bindings, **or** add optional `data-mis-args-fn="<globalCallback>"` / `<form data-mis-rpc="<method>">` support in [`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js) so dynamic forms can be serialized lazily at click/submit time.

### D. Standardized School & Temporal Context Injection (`ctx.school`)
* **Observation:** Every Lua endpoint in [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/timetable_attendance_plugin/src/init.lua#L8-L21) currently has to thread `{ academic_year_id = args.academic_year_id, reference_date = args.reference_date }` into every `ctx.dal.*` call (`get_staff_timetable_slots`, `list_by_group`, `list_sen_and_medical_batch`, `get_group_session`, `upsert_session_batch`).
* **Recommended Adjustment:**
  * While [`extractScopeFromLuaOpts`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L33-L53) already falls back to [`ResolveScopeDefaults`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go#L48) (`CURRENT` academic year and today's UTC date) when `opts` is omitted, exposing a read-only `ctx.school` table (`{ urn, academic_year_id, reference_date, current_cycle_week, current_day_of_week_id }`) on `ctx` would let plugins know the school's active academic calendar and Week A/Week B cycle without requiring the frontend to pass it on every RPC call.

### E. Hierarchical Scope Superseding (`:all` Satisfying `:own_groups` for Cover / SLT Staff)
* **Observation:** [`resolveDomainScopeMode`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L24-L31) currently inspects `endpointSpec.Scopes` from `manifest.json` rather than intersecting with the **caller's** highest granted scope. Because `timetable_attendance_plugin`'s `manifest.json` declares `dal.pupils.read:own_groups`, even if a Head of Department (`HOD`) or `ADMIN` (who holds `dal.pupils.read:all`) opens the plugin to take cover attendance for an absent colleague's class, `resolveDomainScopeMode` restricts the query to `ScopeOwnGroups` (`caller.StaffID`).
* **Recommended Adjustment:**
  * In [`internal/host/host.go`](file:///home/kieren/Documents/code/optimis/internal/host/host.go) and [`internal/host/dal_bindings.go`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L24), when an endpoint declares `dal.<domain>.<action>:own_groups`:
    1. Allow a caller holding `dal.<domain>.<action>:all` to satisfy the endpoint scope check and execute with `ScopeAll` (or allow cover teachers assigned via `staff_cover_assignments` to pass `:own_groups` for covered slots).
    2. Keep ordinary `:own_groups` callers strictly restricted to `ScopeOwnGroups`.

---

## 3. Overall Verdict

The core security and isolation boundaries of the Plugin Contract (capability-sandboxed Lua, `ReadDB`/`WriteDB` split, post-commit Event Bus, and `connect-src 'none'` iframe bridge) are **not too restrictive**—they kept the vertical plugin small (~220 lines of Lua), fast (1–2ms RPC execution), and safe by default. Addressing the **5 ergonomic edges above** (especially `:all` superseding `:own_groups` for cover/SLT staff, preferred pupil names on `ctx.dal.pupils`, and `ctx.school` calendar metadata) will make real-world school workflows significantly smoother before locking down `cmd/mis-sdk` (`mis-sdk verify`).
