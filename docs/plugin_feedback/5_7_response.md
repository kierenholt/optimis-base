# Engineering Response: Plugin Contract Feedback (Verticals 5–7)

**Reviewed Feedback Documents:**
1. [`docs/plugin_feedback/5_assessment.md`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/5_assessment.md) — Assessment & Gradebook Plugin (`uk.org.optimis.assessment.gradebook`)
2. [`docs/plugin_feedback/6_trip_consent.md`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/6_trip_consent.md) — Trips, Consent & Clubs Plugin (`uk.org.optimis.ops.trips`)
3. [`docs/plugin_feedback/7_senco_provision.md`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/7_senco_provision.md) — SENCO Provision Mapping & Pupil Passport Plugin (`uk.org.optimis.senco.provision`)

---

## 1. Implemented Platform Enhancements (`1A` – `1H`)

All items in this section have been **implemented in the Core DAL, Go Host, QA Seeder, and UI SDK** and are ready for plugin authors to adopt directly.

---

### 1A. Expose Structured `ctx.school.mat_grading_schema` from `__system_config`
* **Source:** Assessment & Gradebook (`5_assessment.md` §1.1 & §5.1)
* **Problem:** `ctx.school` previously injected `mat_grading_schema_version` (`"v1.0"` or `"v2.0"`) from `__system_config`, but did not expose the parsed scale definitions (`9-1` vs `WT,EXS,GDS`) or numeric point mappings, forcing plugins to hardcode MAT schema tables in Lua.
* **Implemented Solution:**
  * [`MATPolicyBundle`](file:///home/kieren/Documents/code/optimis/internal/host/external_sync_and_kms.go#L46-L51) and [`handleMATPolicyPull`](file:///home/kieren/Documents/code/optimis/internal/host/external_sync_and_kms.go#L350-L390) now persist `mat_grading_schema_json` in `__system_config` alongside `mat_grading_schema_version` (using [`defaultMATGradingSchemaForVersion`](file:///home/kieren/Documents/code/optimis/internal/host/external_sync_and_kms.go#L53-L82) when the MAT payload omits an explicit schema object).
  * [`injectPluginRuntimeContext`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L1020-L1035) now populates `ctx.school.mat_grading_schema` as a native Lua table whenever `mat_grading_schema_version` has been synced:
    ```lua
    local schema = ctx.school.mat_grading_schema
    -- schema.version == "v2.0"
    -- schema.scales["9-1"].allowed_grades == { "9", "8", "7", "6", "5", "4", "3", "2", "1", "U" }
    -- schema.scales["9-1"].points["9"] == 9
    -- schema.scales["WT,EXS,GDS"].allowed_grades == { "GDS", "EXS", "WT", "PKF" }
    -- schema.scales["WT,EXS,GDS"].points["GDS"] == 3
    ```

---

### 1B. First-Class `:own_groups` Write-Guard Helpers (`ctx.dal.groups.assert_caller_assigned` & `is_caller_assigned`)
* **Source:** Assessment & Gradebook (`5_assessment.md` §1.2, §4.2 & §5.2) & Trips, Consent & Clubs (`6_trip_consent.md` §3.1)
* **Problem:** Following `1_4_response.md` Item `1B`, `ctx.dal.groups.list_roster(group_id)` defaults to school-wide read visibility for teachers (`ScopeAll`). Consequently, plugins that relied on `list_roster(group_id)` throwing an error to guard mutations into `plugin_*.db` needed an explicit, ergonomic `:own_groups` ownership guard.
* **Implemented Solution:**
  * Added [`CoreDAL.EnsureCallerAssignedToGroupScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go#L114-L120) and [`CoreDAL.IsCallerAssignedToGroupScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go#L122-L132) in [`internal/dal/domains.go`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go).
  * Exposed [`ctx.dal.groups.assert_caller_assigned(group_id, opts)`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L510-L547) (and alias `ctx.dal.groups.can_mutate_group`) plus [`ctx.dal.groups.is_caller_assigned(group_id, opts)`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L549-L586) in [`internal/host/dal_bindings.go`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go):
    * If the caller only holds `:own_groups` (and not `:all` / `ADMIN` / `SLT`), `assert_caller_assigned(group_id)` verifies active `group_staff_assignments` membership and raises an explicit `GROUP_NOT_IN_OWN_GROUPS: access denied under own_groups scope: ...` Lua error on failure.
    * `is_caller_assigned(group_id)` returns `true` or `false` without throwing.

---

### 1C. Prepared Batch Execution (`ctx.plugin_db.exec_batch`)
* **Source:** Assessment & Gradebook (`5_assessment.md` §3.2 & §5.4) & Trips, Consent & Clubs (`6_trip_consent.md` §2.2)
* **Problem:** Bulk markbook entry and trip cohort seeding previously executed $N$ individual `ctx.plugin_db.exec(...)` calls from Lua inside a loop, crossing the Go/Lua boundary and re-preparing the SQL statement for every pupil.
* **Implemented Solution:**
  * Added [`ctx.plugin_db.exec_batch(sql, batch_params_table)`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L913-L970) in [`internal/host/host.go`](file:///home/kieren/Documents/code/optimis/internal/host/host.go).
  * Prepares `sql` once inside the active `WriteDB` transaction (`writeTx.PrepareContext`) and executes all parameter rows in a single Go loop, returning `{ rows_affected = totalRowsAffected, batch_size = count }`.
  * Available exclusively when the endpoint or worker declares `mutates: true`.

---

### 1D. Standardize `403 Forbidden` Scope Rejections & Support Structured `{ ok = false, error_code = "..." }` Lua Returns
* **Source:** Assessment (`5_assessment.md` §4.2), Trips (`6_trip_consent.md` §4.1), and SENCO Provision (`7_senco_provision.md` §3.2 & §5.3)
* **Problem:**
  1. In [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go), HTTP `403 Forbidden` on missing manifest scopes was previously gated to an explicit allowlist of plugin IDs instead of applying globally across all feature plugins.
  2. When a Lua endpoint returned `{ ok = false, error_code = "GROUP_NOT_IN_OWN_GROUPS", error = "..." }` instead of calling `error(...)`, [`ExecuteRPC`](file:///home/kieren/Documents/code/optimis/internal/host/host.go) previously treated it as `ok: true` and committed `writeTx`.
* **Implemented Solution:**
  1. Updated [`handlePluginRPC`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L194-L197) in [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go) so any plugin (other than the legacy `uk.org.optimis.test` fixture) automatically receives HTTP `403 Forbidden` on `"caller lacks required DAL scopes"`.
  2. Updated [`ExecuteRPC`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L603-L633) in [`internal/host/host.go`](file:///home/kieren/Documents/code/optimis/internal/host/host.go) so returning `{ ok = false, error_code = "...", error = "..." }` from Lua automatically rolls back any open `writeTx`, preserves the explicit `error_code` and `error` message, and returns a structured failure response without requiring `error(...)`.

---

### 1E. Relax Read Scoping on SEN, Medical & PR/Court-Order-Filtered Guardians for Teachers (+ Baseline `TEACHER` Scope)
* **Source:** Trips, Consent & Clubs (`6_trip_consent.md` §1.1, §3.1, §5.1 & §5.2) & SENCO Provision (`7_senco_provision.md` §1.2)
* **Problem:**
  1. Standard `TEACHER` personas lacked `dal.guardians.read:own_groups` in their default role scope map, causing `403 Forbidden` when teachers opened the Trip Leader pack or verified guardian emergency contacts.
  2. Under our school-wide teacher read principle (established in `1_4_response.md` Item `1B`), teachers organizing cross-group school trips or providing cover need to read pupil SEN/medical summaries and PR/court-order-filtered guardian contacts across cohorts, while keeping mutations strictly scoped.
* **Implemented Solution:**
  1. Added `"dal.guardians.read:own_groups"` to the default `TEACHER` role scopes in [`internal/dal/persona_auth.go`](file:///home/kieren/Documents/code/optimis/internal/dal/persona_auth.go#L559) and [`cmd/mis-seed-qa/seed_domains_00_to_03.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/seed_domains_00_to_03.go#L195).
  2. Updated [`ctx.dal.guardians.list_for_pupil(pupil_id, opts)`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L1138-L1163) and [`ctx.dal.guardians.list_for_pupils_batch(pupil_ids, opts)`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L1165-L1190) (alias `list_batch`) so callers with `dal.guardians.read:own_groups` default to school-wide read visibility (`dal.ScopeAll`) unless `opts = { strict_own_groups = true }` is explicitly passed.
  3. **Non-Negotiable Safeguarding Invariant Preserved:** `court_order_restricted = 0`, `has_parental_responsibility = 1` (when `require_parental_responsibility = true`), and `is_confidential_address` redaction remain unconditionally enforced inside the Go DAL SQL query regardless of read scope.

---

### 1F. Core Event `core.parent_dmz.submission_received` on Verified DMZ Inbox Pull
* **Source:** Trips, Consent & Clubs (`6_trip_consent.md` §1.3 & §5.3)
* **Problem:** When the Host Container pulled a verified parent submission via `POST /api/sync/parent-dmz/pull-inbox` (`submission_type: "TRIP_CONSENT"`), it updated `parent_dmz_inbox_submissions` in `core.db` but did not emit a Core Event Bus event, requiring plugins to expose manual RPC bridge endpoints.
* **Implemented Solution:**
  * Added [`EventCoreParentDMZSubmissionReceived = "core.parent_dmz.submission_received"`](file:///home/kieren/Documents/code/optimis/internal/dal/events.go#L18) to the canonical Core Event Catalog in [`internal/dal/events.go`](file:///home/kieren/Documents/code/optimis/internal/dal/events.go#L16-L35).
  * Updated [`handleParentDMZPullInbox`](file:///home/kieren/Documents/code/optimis/internal/host/external_sync_and_kms.go#L571-L586) in [`internal/host/external_sync_and_kms.go`](file:///home/kieren/Documents/code/optimis/internal/host/external_sync_and_kms.go) so every submission that passes PR and court-order verification emits `core.parent_dmz.submission_received` post-commit with `{ submission_id, pupil_id, guardian_id, submission_type, payload_json, status = "ACCEPTED" }`.

---

### 1G. Batched SQL `IN (...)` Refactor for `ListPupilSenAndMedicalBatch` + `ctx.dal.sen.list_register(opts)` & Richer SEN Read Fields
* **Source:** SENCO Provision (`7_senco_provision.md` §1.1, §1.2, §2.2, §5.1 & §5.2)
* **Problem:**
  1. `CoreDAL.ListPupilSenAndMedicalBatch` previously looped over `pupilIDs` in Go and executed 3 separate SQL queries per pupil ($3N$ queries total).
  2. `pupil_sen_records` columns (`secondary_sen_need_code`, `pupil_targets`, `senco_notes`, `ehcp_start_date`, `ehcp_review_due_date`) and child tables (`pupil_sen_needs`, `pupil_sen_assistants`) were not projected to Lua.
  3. Whole-school SENCO dashboards had to enumerate all teaching groups and deduplicate rosters in Lua just to list the school's active SEN register.
* **Implemented Solution:**
  1. **True Batched SQL (`4` queries total instead of $3N$):** Refactored [`CoreDAL.ListPupilSenAndMedicalBatch`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go#L341-L558) in [`internal/dal/domains_pupils.go`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go) to execute 4 batched `WHERE pupil_id IN (?, ...)` queries across `pupil_sen_records`, `pupil_sen_needs`, `pupil_sen_assistants`, and `pupil_medical_conditions`.
  2. **Expanded SEN Projection:** Updated [`PupilSenAndMedicalSummary`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go#L35-L67) and [`senAndMedicalSummaryToLuaTable`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L390-L435) so every summary row includes:
     * `secondary_sen_need_code`, `pupil_targets`, `senco_notes`, `ehcp_start_date`, `ehcp_review_due_date` (plus alias `next_annual_review_date`)
     * `sen_needs` (`{ { sen_need_id, need_Category_code, need_ranking, description, effective_from, effective_to }, ... }`)
     * `sen_assistants` (`{ { sen_assistant_id, staff_id, support_hours_per_week, funding_band, notes, effective_from, effective_to }, ... }`)
  3. **Direct SEN Register Query (`ctx.dal.sen.list_register(opts)`):** Added [`CoreDAL.ListSenRegisterScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go#L560-L699) and exposed [`ctx.dal.sen.list_register(opts)`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L495-L535) (plus alias `ctx.dal.pupils.list_sen_register(opts)`) for callers holding `dal.sen.read:*`. Supports optional filters `{ sen_status = "K" | "E", year_group_code = "07", has_ehcp = true }` and returns the full joined pupil + SEN + medical summaries in a single call.

---

### 1H. Optional `data-mis-plugin` Target Override on `[data-mis-link]` Anchors
* **Source:** SENCO Provision (`7_senco_provision.md` §3.1 & §5.4)
* **Problem:** Clicking `<a data-mis-link="student" data-pupil-id="...">` inside a plugin's own roster table always navigated away to the global `entity_link_handler_student` plugin configured in `__system_config`, even when the author wanted to update the active `pupil_id` context while staying inside the current plugin (e.g., switching from the SEN Register tab to the Pupil Passport tab inside `uk.org.optimis.senco.provision`).
* **Implemented Solution:**
  * Updated [`navigate()`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js#L140-L176) and the [`[data-mis-link]` click listener](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js#L290-L315) in [`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js) to read `data-mis-plugin="<plugin_id | self>"` (or `params.target_plugin` / `params.plugin_id`) and include `target_plugin` in the `mis:link:navigate` `postMessage` payload.
  * Updated the [`mis:link:navigate` handler](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx#L554-L576) in [`ui/src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx) so `data-mis-plugin="self"` keeps the source iframe's plugin active in the Keep-Alive LRU Pool (or activates the specified installed `plugin_id`) while broadcasting the updated `pupil_id` / `group_id` / `staff_id` context to all mounted iframes via `mis:init`.

---

## 2. Rejected Proposals (`2A` – `2C`)

The following items from Verticals 5–7 feedback are **rejected** to preserve Core vs. Plugin boundary invariants, SQLite migration safety, and host security guardrails.

---

### 2A. Moving Gradebook Marksheets & Mark Entries into `core.db` (`ctx.dal.assessment.*` Write Methods)
* **Source:** Assessment & Gradebook (`5_assessment.md` §1.1 — observation that `ctx.dal.assessment` has no core SQL tables)
* **Decision:** **Rejected.**
* **Architectural Rationale:** Section 3 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md) (*Core vs. Plugin Boundary*) explicitly defines custom Assessment/Gradebook models, subject marksheets, and formative/summative grade entry workflows as **Plugin Boundary (`plugin_*.db`)** responsibilities. Schools and MATs use widely varying assessment frameworks (KS1/KS2 `WT/EXS/GDS`, KS4 `9-1`, BTEC, IB, custom rubric matrices). Keeping marksheets and raw mark entries inside `plugin_assessment_gradebook.db` while injecting the MAT policy schema via `ctx.school.mat_grading_schema` (Item `1A`) and sharing published attainment snapshots via `ctx.events.emit("plugin.uk.org.optimis.assessment.gradebook.marksheet_published", ...)` preserves schema agility without polluting `core.db`.

---

### 2B. Allowing `CREATE INDEX` / DDL Mutations on `core.db` Tables from Plugin Migrations
* **Source:** Trips, Consent & Clubs (`6_trip_consent.md` §3.1 — query performance notes on `pupil_guardian_relationships`)
* **Decision:** **Rejected.**
* **Architectural Rationale:** Plugin SQL migrations (`migrations/*.sql`) execute strictly against the plugin's isolated `plugin_<name>.db` file and are snapshotted for zero-downtime rollback (Section 6A). Allowing a plugin migration to execute DDL or index creation against `core.db` would violate tenant DB isolation and risk locking `core.db` during mid-day plugin upgrades. Core table indexing is managed exclusively by Core DAL migrations (`internal/dal/migrations/`).

---

### 2C. Allowing Live `ctx.http` Calls Directly Inside `mutates: true` Write Transactions
* **Source:** Trips, Consent & Clubs (`6_trip_consent.md` §1.3 — outbound payment/consent sync flow)
* **Decision:** **Rejected.**
* **Architectural Rationale:** Section 9.2 and Section 10.1 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md) enforce a non-negotiable structural safety invariant: **no open SQLite write transactions during an outbound `ctx.http` fetch**. Because SQLite serializes writes through `WriteDB` (`MaxOpenConns(1)`), holding an open write transaction while waiting on an external HTTPS endpoint would stall the entire school container's database for all staff. Plugins must continue to split outbound HTTP sync and local DB persistence across two phase steps or background worker invocations.
