# Plugin Contract Feedback — Vertical 2: Google Classroom LMS Sync (`uk.org.optimis.lms.classroom`)

This document captures engineering feedback on the **Optimis Plugin Contract** (Sections 5, 6, 9, and 10 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md)) gathered while implementing [`plugins/lms_sync_plugin`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/GEMINI.md) and its full-stack verification suites ([`tests/e2e/lms_sync_plugin_vertical.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/lms_sync_plugin_vertical.spec.ts) and [`internal/host/lms_sync_plugin_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/lms_sync_plugin_test.go)).

---

## 1. What Worked Exceptionally Well (Keep As-Is)

* **Structural Safety Invariant (`ctx.http` Disallowed Inside Open SQLite Write Transactions, §9.2 & §10.1)**:
  * Enforcing that [`ctx.http.post`](file:///home/kieren/Documents/code/optimis/internal/host/egress.go#L219) cannot execute while an SQLite write transaction is open was completely natural in [`on_membership_changed`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L85-L210):
    1. Read the current group roster via `ctx.dal.pupils.list_by_group` (`ReadDB`).
    2. Execute the outbound HTTPS call via `ctx.http.post` to `https://classroom.googleapis.com/v1/courses/roster:sync`.
    3. Record the sync outcome via `ctx.plugin_db.exec` (`WriteDB`) into `plugin_lms_sync.lms_sync_logs`.
  * This guardrail eliminates the risk of slow third-party EdTech APIs holding a local SQLite WAL lock without harming plugin ergonomics.
* **Post-Commit Event Bus + `SYSTEM` Worker Service Principal (§3, §9.1 & §10.2)**:
  * Having [`UpsertGroupPupilMembershipScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_groups_attendance.go) emit `core.group.membership_changed` strictly *after* the `WriteDB` transaction commits guaranteed that [`on_membership_changed`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L85) always read the freshly committed group roster when querying `ctx.dal.pupils.list_by_group`.
* **Mandatory UK GDPR Hooks (`export_pupil_data` & `purge_pupil_data`, §5C & §9.3)**:
  * Requiring `gdpr_hooks` whenever `database.alias` is declared took ~35 lines of Lua in [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L270-L319) and ensured `pupil_id` was indexed in [`migrations/001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/migrations/001_init.sql) from day one.

---

## 2. Where the Contract Was Too Restrictive or Caused Friction

### A. Missing Native JSON Encode/Decode in the Stripped Lua 5.1 Sandbox (`ctx.json`)
* **The Problem**: Because `package`, `require`, `dofile`, and `loadfile` are stripped from the `gopher-lua` sandbox ([`host.go`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L460)), Lua plugins have **no built-in JSON serializer or parser**.
* **Impact in `lms_sync_plugin`**:
  * In [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L114-L128), outbound JSON payloads had to be constructed manually using `string.format` and `table.concat`, and the Google Classroom JSON response (`resp.body`) had to be parsed with Lua pattern matching (`string.match(resp.body or "", '"sync_receipt_id"%s*:%s*"([^"]+)"')`).
  * For production REST integrations (Google Classroom, Microsoft Teams, Sparx Maths) that exchange nested JSON objects containing pupil names with apostrophes, quotes, or Unicode characters, manual string formatting is brittle and error-prone.
* **Recommendation**: Inject a deterministic `ctx.json.encode(lua_table)` and `ctx.json.decode(json_string)` helper namespace into `ctx` alongside [`ctx.xml`](file:///home/kieren/Documents/code/optimis/internal/host/xml_xsd.go).

### B. Binary `:own_groups` vs `:all` Scope Modifiers Restrict Middle-Leader (HoD / Year Lead) Workflows
* **The Problem**: Section 5A and [`domains.go`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go) only define two scope modifiers: `:own_groups` (groups the staff member personally teaches on the timetable) and `:all` (whole-school Admin/SLT).
* **Impact in `lms_sync_plugin`**:
  * Moving a pupil between Maths Set 1 and Maths Set 2 (`upsert_pupil_membership`) is a standard **Head of Department (HoD)** workflow. However, an HoD does not personally teach every set in their department (so `:own_groups` blocks them from transferring a pupil into another teacher's set), while schools do not want to grant every HoD whole-school `dal.groups.write:all` (and in [`SeedPersonaTestFixtures`](file:///home/kieren/Documents/code/optimis/internal/dal/persona_auth.go#L229), `HOD` only holds `dal.groups.read:all`).
* **Recommendation**: Introduce a `:department` (or `:managed_groups`) scope modifier resolved via `departments.head_staff_id` / `year_groups.head_staff_id` in the Core schema so Heads of Department and Pastoral Heads of Year can manage group memberships within their faculty/year group without needing whole-school `:all` write permissions.

### C. Declarative `data-mis-rpc` Buttons with Dynamic Form State
* **The Problem**: Section 6C / 9.4 emphasizes declarative `<button data-mis-rpc="method" data-mis-args='{...}'>` bindings intercepted by [`mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js).
* **Impact in `lms_sync_plugin`**:
  * Because [`mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js) registers a capture-phase click listener (`addEventListener("click", ..., true)`), any button carrying `data-mis-rpc` immediately dispatches whatever JSON is currently inside `data-mis-args` before a normal button click handler can read input fields. In [`ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/ui/dist/index.html), the UI script has to pre-populate `data-mis-args` ahead of time whenever state changes.
* **Recommendation**: Position `window.mis.rpc(method, args)` as the primary recommended SDK pattern for dynamic forms, or add support for `data-mis-form="#form-id"` in `mis-bridge.js` so form inputs are automatically serialized into `args` at click time.

### D. Worker Error Semantics vs. Persisting Failure Audit Rows in `plugin_<name>.db`
* **The Problem**: When an outbound integration sync fails in a background worker (`on_membership_changed`), the plugin almost always needs to write a `FAILED` audit row into `plugin_<name>.db` so the School Admin can inspect and retry failed syncs in the UI.
* **Impact in `lms_sync_plugin`**:
  * For HTTP `4xx`/`5xx` status codes, [`ctx.http.post`](file:///home/kieren/Documents/code/optimis/internal/host/egress.go#L219) returns `{ status = 503, body = ... }` without raising a Lua error, allowing [`on_membership_changed`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/src/init.lua#L146-L165) to record `status = "FAILED"` in `plugin_lms_sync.db`.
  * However, if `ctx.http.post` hits an operational network timeout (`EGRESS_TIMEOUT`) or connection refusal, the Go Host raises a Lua `error(...)`, which aborts the worker unless wrapped in `pcall`.
* **Recommendation**: Have `ctx.http.get` / `ctx.http.post` return `{ ok = false, status = 0, error_code = "EGRESS_TIMEOUT", error = "..." }` for operational network/timeout failures (while continuing to raise hard fail-fast Lua errors for contract/security violations such as undeclared domains or open SQLite write transactions).

### E. Duplicate `gdpr_hooks` and `endpoints` Declarations in `manifest.json`
* **The Problem**: `manifest.json` requires a top-level `gdpr_hooks` object (`export_pupil_data`, `purge_pupil_data`), which the Host invokes via [`ExportPupilSAR`](file:///home/kieren/Documents/code/optimis/internal/host/gdpr.go#L275) and [`PurgePupilData`](file:///home/kieren/Documents/code/optimis/internal/host/gdpr.go#L354). However, if a plugin or test also invokes those functions via `ExecuteRPC`, they must be duplicated under `endpoints` in [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/manifest.json#L38-L49).
* **Recommendation**: Keep `gdpr_hooks` exclusively in the `gdpr_hooks` manifest block (invoked only by the Host's GDPR SAR/Purge orchestrator) or automatically register declared `gdpr_hooks` as Admin/DPO-only endpoints in the Host without requiring duplicate declarations in `manifest.json`.

---

## 3. Summary Checklist of Recommended Contract Adjustments

| Surface | Current Contract | Recommended Adjustment |
| :--- | :--- | :--- |
| **Lua Runtime (`ctx`)** | `ctx.xml` only; no JSON parser/serializer in stripped Lua 5.1 VM | Add `ctx.json.encode(table)` and `ctx.json.decode(string)` to `ctx`. |
| **RBAC Scope Modifiers (§5A)** | `:own_groups` vs `:all` only | Add `:department` / `:managed_groups` for HoDs and Pastoral Year Leads. |
| **HTTP Egress (`ctx.http`, §9.2)** | Network timeouts throw a Lua `error(...)` | Return `{ ok = false, status = 0, error_code = "EGRESS_TIMEOUT" }` on network/timeout errors; keep hard Lua `error(...)` for policy violations (undeclared domain, open write tx). |
| **UI Bridge (`mis-bridge.js`, §9.4)** | Capture-phase click listener reads static `data-mis-args` | Support `data-mis-form="#form-id"` auto-serialization or document `window.mis.rpc()` as default for forms. |
| **Manifest (`manifest.json`, §9.1)** | `export_pupil_data` & `purge_pupil_data` often duplicated in `gdpr_hooks` and `endpoints` | Auto-bind `gdpr_hooks` for Host/DPO access without duplicating them in `endpoints`. |
