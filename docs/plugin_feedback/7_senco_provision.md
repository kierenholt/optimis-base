# Plugin Contract Evaluation — Vertical 9: SENCO & Classroom Differentiation (`uk.org.optimis.sen.provision`)

**Plugin Under Review:** [`plugins/senco_provision_plugin`](file:///home/kieren/Documents/code/optimis/plugins/senco_provision_plugin/GEMINI.md) (`uk.org.optimis.sen.provision`)  
**E2E Verification Suite:** [`tests/e2e/senco_provision_plugin_vertical.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/senco_provision_plugin_vertical.spec.ts)  
**Contract Surfaces Exercised:**
* Attached Plugin SQLite DB (`database.alias: "plugin_sen_provision"`)
* Asymmetric Read/Write Endpoint Scopes:
  * `log_iep_provision` (`scopes: ["dal.pupils.read:all", "dal.sen.read:all"]`, `mutates: true`)
  * `get_sen_differentiation_summary` (`scopes: ["dal.pupils.read:own_groups", "dal.sen.read:own_groups", "dal.groups.read:own_groups"]`, `mutates: false`)
* Batch SEN & Medical Hydration (`ctx.dal.pupils.list_sen_and_medical_batch`)
* Host Capability Helpers (`ctx.crypto.uuid()`, `ctx.clock.now_utc()`, `ctx.school`, `ctx.events.emit`)
* Home Dashboard Mini-Iframe Widget (`widgets: [{ "id": "sen_differentiation", "entry": "ui/dist/widget.html" }]`) + `window.mis.onInvalidated` live refresh
* Declarative Entity Deep-Linking (`<a data-mis-link="student" data-pupil-id="...">`) into the Keep-Alive LRU Pool
* Auto-Bound UK GDPR SAR Export & Retention Purge Hooks (`export_pupil_data` with `redactable_fields: ["iep_target", "classroom_adjustment", "review_notes"]` & `purge_pupil_data`)

---

## 1. Contract Surfaces That Worked Well (Validating `1A`–`1I` Enhancements)

1. **Asymmetric Role Scoping + Hierarchical Scope Subsumption (`1B` & `1D`)**
   * Splitting `log_iep_provision` (`dal.sen.read:all`, `mutates: true`) from `get_sen_differentiation_summary` (`dal.sen.read:own_groups`, `mutates: false`) in [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/senco_provision_plugin/manifest.json) cleanly modeled the real UK school division of responsibility between a **SENCO / HoD / DSL** and a **Classroom Teacher**:
     * `GET /api/system/ui-config` automatically filtered `permitted_methods` to `["get_sen_differentiation_summary", "log_iep_provision"]` for SENCO/HoD personas and `["get_sen_differentiation_summary"]` for Classroom Teachers.
     * Non-teaching support staff without `dal.sen.read` (`SUPPORT` / Attendance Officer) were automatically rejected at the RPC gateway with `403`.
     * Hierarchical scope subsumption (`1B`) allowed the SENCO/HoD (holding `:all`) to invoke the `:own_groups` read endpoint across any group or pupil in the school without needing a duplicate `:all` read endpoint.
2. **Host-Driven Widget Invalidation (`1H` `window.mis.onInvalidated`)**
   * When the SENCO logged IEP targets and TA allocations in [`ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/senco_provision_plugin/ui/dist/index.html) (`mutates: true`), [`ui/src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx#L615-L630) automatically broadcast `mis:plugin:invalidated` to the mounted `sen_differentiation` widget ([`ui/dist/widget.html`](file:///home/kieren/Documents/code/optimis/plugins/senco_provision_plugin/ui/dist/widget.html)), updating the classroom differentiation cards without polling or manual refresh clicks.
3. **Auto-Bound `gdpr_hooks` (`1G`) with Multi-Field Redaction**
   * Declaring `export_pupil_data` and `purge_pupil_data` solely under `gdpr_hooks` in [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/senco_provision_plugin/manifest.json) and returning `redactable_fields = { "iep_target", "classroom_adjustment", "review_notes" }` allowed the Admin SAR Redaction UI ([`ui/src/GdprSarPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/GdprSarPanel.tsx)) to scrub third-party peer-mentor names across both `classroom_adjustment` and `review_notes` before finalizing the SAR export.

---

## 2. Where the Contract Is Too Restrictive or Needs Adjustment

### 2.1 `data-mis-link="student"` Always Routes to the Global `entity_handler.student` Plugin (Cannot Target the Owning Plugin in the Keep-Alive LRU Pool)
* **Observed Friction:**
  Under Section 9.5 (9C) and [`ui/src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx#L545-L583) / [`internal/host/ui_sdk.go`](file:///home/kieren/Documents/code/optimis/internal/host/ui_sdk.go#L227-L231), clicking `<a data-mis-link="student" data-pupil-id="...">` inside *any* dashboard mini-widget always activates the single global `entity_handler.student` (`uk.org.optimis.test` by default in `__system_config`).
  * While `App.tsx` broadcasts `mis:init` with the new `pupil_id` to all mounted iframes in the Keep-Alive LRU Pool (so `uk.org.optimis.sen.provision` updates its active pupil state in the background), a teacher clicking a pupil's `E (ASD)` or `K (SPLD)` badge specifically inside the **Classroom SEN Differentiation Mini-Widget** often wants to open that pupil's full **SEN IEP & Provision Workspace** (`uk.org.optimis.sen.provision`) rather than the generic student profile handler.
  * Changing `entity_handler.student` globally in `__system_config` to `uk.org.optimis.sen.provision` would hijak student deep-links clicked inside Attendance, Behaviour, or Timetable widgets.
* **Recommended Contract Adjustment:**
  Support an optional **`data-mis-plugin`** attribute on `[data-mis-link]` anchors in [`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js) and [`ui/src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx):
  * If `data-mis-plugin` is omitted (default), `mis:link:navigate` routes to the global `entity_handler.<entity_type>` configured in `__system_config`.
  * If `data-mis-plugin="self"` (or an explicit plugin ID like `data-mis-plugin="uk.org.optimis.sen.provision"`, verified against active installed plugins), `App.tsx` wakes that specific plugin in the Keep-Alive LRU Pool and passes `{ pupil_id }`.

### 2.2 Core SEN Schema vs. Plugin DB Boundary (`core.pupil_sen_assistants` & Secondary SEN Needs Are Unreachable via `ctx.dal.sen`)
* **Observed Friction:**
  Inspecting [`schema/core/05_sen_medical_and_characteristics.sql`](file:///home/kieren/Documents/code/optimis/schema/core/05_sen_medical_and_characteristics.sql) shows that `core.db` already contains rich statutory SEN tables:
  * `pupil_sen_records` (`provision_code` `'N'|'K'|'E'`, `ehcp_start_date`, `next_annual_review_date`, `key_worker_staff_id`, `teaching_strategies`)
  * `pupil_sen_needs` (ranked primary *and* secondary `sen_need_code`s with `alert_level_id`)
  * `pupil_sen_assistants` (`pupil_id`, `staff_id`, `allocated_hours_per_week`, `support_focus`, `effective_from`, `effective_to`)
  However, [`CoreDAL.ListPupilSenAndMedicalBatch`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go#L298-L406) only projects `sen_provision_code`, `primary_sen_need_code` (`ranking = 1` only), and `teaching_strategies`. It omits secondary SEN needs (`ranking = 2`), `ehcp_start_date`, `next_annual_review_date`, `key_worker_staff_id`, and `pupil_sen_assistants`. Furthermore, there is no `dal.sen.write:all` capability on `ctx.dal.sen`.
  * Because `ctx.dal.sen` is read-only and omits `pupil_sen_assistants`, `senco_provision_plugin` had to store `ta_allocation_hours` and `assigned_ta_name` inside `plugin_sen_provision.db` (`sen_provisions`) rather than writing statutory TA allocations to `core.pupil_sen_assistants` linked to a validated `core.staff.id`.
* **Recommended Contract Adjustment:**
  1. **Expand `ctx.dal.sen` Read Projection:** Include `ehcp_start_date`, `next_annual_review_date`, `key_worker_staff_id`, all ranked `sen_needs` (`[{ ranking, sen_need_code, alert_level_id }]`), and active `sen_assistants` (`[{ staff_id, display_name, allocated_hours_per_week, support_focus }]`) in `ListPupilSenAndMedicalBatch`.
  2. **Add `dal.sen.write:all` Core DAL Methods for SENCOs:** Expose `ctx.dal.sen.upsert_sen_record(...)` and `ctx.dal.sen.upsert_ta_allocation(...)` so statutory SEN provision codes (`K`/`E`), annual review dates, and TA hours live in `core.db` (where the DfE Census engine `ctx.dal.census.get_statutory_rollup` can also read them), while `plugin_sen_provision.db` focuses on non-statutory IEP smart targets, classroom differentiation strategies, and review notes.

### 2.3 Missing Whole-School SEN Register Query & Per-Pupil SQL Loop in `ListPupilSenAndMedicalBatch`
* **Observed Friction:**
  1. In [`internal/dal/domains_pupils.go`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go#L311-L403), `CoreDAL.ListPupilSenAndMedicalBatch` iterates over `pupilIDs` in a Go `for` loop and executes **3 separate SQL queries per pupil** (`pupil_sen_records`, `pupil_sen_needs`, and `pupil_medical_conditions`). For a 30-pupil class roster, one Lua batch call executes 90 SQL queries; for a whole-school SEN register of 1,100 pupils, it executes 3,300 SQL queries.
  2. Additionally, there is no `ctx.dal.sen.list_sen_pupils(opts)` method to directly query only active `K` and `E` pupils across the school or year group when a SENCO opens the workspace without a `group_id`.
* **Recommended Contract Adjustment:**
  * Refactor `CoreDAL.ListPupilSenAndMedicalBatch` to execute 3 batched `WHERE pupil_id IN (?, ?, ...)` queries across the entire `pupilIDs` slice regardless of cohort size.
  * Add **`ctx.dal.sen.list_register(opts)`** (accepting optional `group_id`, `nc_year_group`, and `provision_codes = {"K", "E"}` filters) to return active SEN pupils in a single SQL `JOIN`.

### 2.4 Hardcoded Plugin ID Whitelist for HTTP `403` vs `400` Scope Rejections in `internal/host/server.go`
* **Observed Friction:**
  In [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L194-L197), when `ExecuteRPC` rejects a call with `"caller lacks required DAL scopes"`, the HTTP handler only returns `403 Forbidden` if `pluginID` is listed in a hardcoded `if (pluginID == ...)` condition (otherwise defaulting to `400 Bad Request`). We had to add `pluginID == "uk.org.optimis.sen.provision"` to that list in `server.go`.
* **Recommended Contract Adjustment:**
  Return a typed error or error classification from `ExecuteRPC` (e.g. `ErrUnauthorizedScope` $\rightarrow$ `403 Forbidden`, `ErrPluginDormant` $\rightarrow$ `423 Locked` / `403 Forbidden`, `ErrUndeclaredEndpoint` $\rightarrow$ `404 Not Found` / `400 Bad Request`) so HTTP status codes are consistent across all plugins without per-plugin whitelist checks in `server.go`.
