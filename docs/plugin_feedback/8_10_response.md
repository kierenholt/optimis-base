# Engineering Response: Plugin Contract Feedback (Verticals 8–10)

**Reviewed Feedback Documents:**
1. [`docs/plugin_feedback/8_staff_cover.md`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/8_staff_cover.md) — Cover Manager & Supply Teacher (`uk.org.optimis.ops.cover`)
2. [`docs/plugin_feedback/9_sparx.md`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/9_sparx.md) — Sparx Maths Third-Party Partner Sync (`uk.co.sparx.maths`)
3. [`docs/plugin_feedback/10_first_aid.md`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/10_first_aid.md) — Medical Room & First Aid (`uk.org.optimis.medical.firstaid`)

---

## 0. Previously Resolved in Earlier Platform Releases (`1_4_response.md` & `5_7_response.md`)

* **Prepared Batch Execution (`ctx.plugin_db.exec_batch`)** (*`9_sparx.md` §3C*): Implemented in [`5_7_response.md` Item `1C`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/5_7_response.md) ([`internal/host/host.go`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L913-L970)). Prepares `sql` once inside `writeTx` and executes an array of parameter rows in a single Go loop.
* **Global `403 Forbidden` Scope Error Mapping** (*`8_staff_cover.md` §2.5 & `10_first_aid.md` §2D*): Implemented in [`5_7_response.md` Item `1D`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/5_7_response.md) ([`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L194-L197)). All feature plugins receive HTTP `403 Forbidden` when `"caller lacks required DAL scopes"`.
* **Persona Switch Broadcast to Mounted Keep-Alive Iframes** (*`8_staff_cover.md` §2.3*): Implemented in [`1_4_response.md` Item `1H`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/1_4_response.md) ([`handlePersonaSwitch`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx#L501-L517) in [`ui/src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx)) and extended in `1B` below with `window.mis.onPersonaChanged(callback)` in [`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js).
* **`SUPPORT` Role `dal.medical.read:all` Scope Coverage** (*`10_first_aid.md` §2B*): Active in [`internal/dal/persona_auth.go`](file:///home/kieren/Documents/code/optimis/internal/dal/persona_auth.go#L561) and [`cmd/mis-seed-qa/seed_domains_00_to_03.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/seed_domains_00_to_03.go#L197).

---

## 1. Implemented Platform Enhancements (`1A` – `1F`)

All items in this section have been **implemented in the Core Schema, Core DAL, Go Host, QA Seeder, and SDK** and are ready for plugin authors to adopt directly.

---

### 1A. `'COVER_TEACHER'` Role & Assignment Revocation Helper (`ctx.dal.groups.revoke_staff_assignment`)
* **Source:** Cover Manager & Supply Teacher (`8_staff_cover.md` §2.1)
* **Implemented Solution:**
  1. Expanded the `CHECK` constraint on `group_staff_assignments.role` in [`schema/core/03_groups_and_timetable.sql`](file:///home/kieren/Documents/code/optimis/schema/core/03_groups_and_timetable.sql#L131) and [`CoreDAL.UpsertGroupStaffAssignmentScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_groups_attendance.go#L567-L570) to accept `'COVER_TEACHER'` alongside `'LEAD_TEACHER'`, `'CO_TEACHER'`, `'FORM_TUTOR'`, and `'TEACHING_ASSISTANT'`.
  2. Added [`CoreDAL.RevokeGroupStaffAssignmentScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_groups_attendance.go#L650-L720) in [`internal/dal/domains_groups_attendance.go`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_groups_attendance.go) and exposed [`ctx.dal.groups.revoke_staff_assignment(input, opts)`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L1014-L1059) (and alias `ctx.dal.groups.delete_staff_assignment`) in [`internal/host/dal_bindings.go`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go) (`dal.groups.write:*`, `mutates: true`), returning `{ revoked = true/false, rows_affected = count }`.

---

### 1B. Coarse-Grained Timetable Availability Queries (`get_staff_timetable_slots_batch` & `list_free_staff_for_period`) + `window.mis.onPersonaChanged`
* **Source:** Cover Manager & Supply Teacher (`8_staff_cover.md` §2.3 & §2.4)
* **Implemented Solution:**
  1. Added [`CoreDAL.GetStaffTimetableSlotsBatchScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_groups_attendance.go#L259-L353) and [`CoreDAL.ListFreeStaffForPeriodScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_groups_attendance.go#L355-L426) in [`internal/dal/domains_groups_attendance.go`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_groups_attendance.go), and exposed:
     * [`ctx.dal.groups.get_staff_timetable_slots_batch(staff_ids, cycle_week, day_of_week_id, opts)`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L801-L859) — returns a Lua map `staff_id -> TimetableSlot[]` in a single `WHERE ts.staff_id IN (...)` SQL query.
     * [`ctx.dal.groups.list_free_staff_for_period(cycle_week, day_of_week_id, period_id_or_code, opts)`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L861-L914) (also accepts a table `{ cycle_week = 1, day_of_week_id = 1, period_id = "..." }`) — returns all active `StaffSummary[]` who have no scheduled `timetable_slots` row during that period.
  2. Added [`window.mis.onPersonaChanged(callback)`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js#L396-L401) and a DOM `mis:persona:changed` `CustomEvent` in [`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js#L190-L227) whenever `staff_id` changes across `mis:init` payloads.

---

### 1C. Host-Managed Secret Envelope Helpers (`ctx.crypto.seal_secret` & `ctx.crypto.open_secret`) with Automatic Diagnostic Scrubber Registration
* **Source:** Sparx Maths Partner Sync (`9_sparx.md` §3A)
* **Implemented Solution:**
  * Added [`ctx.crypto.seal_secret(plaintext)`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L1056-L1072) and [`ctx.crypto.open_secret(ciphertext)`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L1073-L1089) in [`internal/host/host.go`](file:///home/kieren/Documents/code/optimis/internal/host/host.go).
  * Secrets are encrypted with AES-256-GCM (`enc:v1:<base64>`) using a per-plugin key derived from `HOST_KMS_KEY_HANDLE` + `plugin_id` (`derivePluginSecretKey`) and authenticated with `plugin_id` as AAD (so one plugin cannot decrypt another plugin's ciphertext).
  * Whenever `seal_secret` or `open_secret` is called during an RPC or worker invocation, the plaintext secret is automatically appended to the invocation's `extraSecrets` scrubber list, guaranteeing deterministic redaction from the PII-scrubbed diagnostic ring buffer (`[REDACTED_SECRET]`) regardless of the vendor's token format.

---

### 1D. Manifest Endpoint `require_roles` & `plugin.config.write` Admin Gating for Plugin Settings Endpoints
* **Source:** Sparx Maths Partner Sync (`9_sparx.md` §3B)
* **Implemented Solution:**
  1. Added `"plugin.config.write"` to the default scopes of `ADMIN` and `HOD` in [`internal/dal/persona_auth.go`](file:///home/kieren/Documents/code/optimis/internal/dal/persona_auth.go#L547-L555) and [`cmd/mis-seed-qa/seed_domains_00_to_03.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/seed_domains_00_to_03.go#L191-L194).
  2. Added optional `require_roles: ["ADMIN", "HOD", ...]` to [`EndpointSpec`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L45) and enforced [`hasRequiredEndpointRoles`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L1176-L1192) in both [`ExecuteRPC`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L526) and [`GET /api/system/ui-config` (`permitted_methods`)](file:///home/kieren/Documents/code/optimis/internal/host/ui_sdk.go#L277).

---

### 1E. Project Full Clinical Medication & Trigger Columns in `ListPupilSenAndMedicalBatch`
* **Source:** Medical Room & First Aid (`10_first_aid.md` §2A)
* **Implemented Solution:**
  * Expanded [`PupilMedicalConditionAlert`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go#L11-L25), [`CoreDAL.ListPupilSenAndMedicalBatch`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go#L520-L569), and [`senMedicalBatchFn`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L430-L448) so every item in `medical_conditions` now includes:
    * `triggers`, `reaction_symptoms`, `emergency_response_plan`
    * `medication_name_and_dose`, `medication_location`, `parental_consent_given` (`boolean`)
    * `effective_from`, `effective_to`, and alias `medication_expiry_date`

---

### 1F. Lua DAL Field Reference & EmmyLua Type Stubs (`sdk/lua/ctx_types.lua`) + Section 9.4 DOM Event Documentation
* **Source:** Medical Room & First Aid (`10_first_aid.md` §2C & §2E)
* **Implemented Solution:**
  1. Created [`sdk/lua/ctx_types.lua`](file:///home/kieren/Documents/code/optimis/sdk/lua/ctx_types.lua) and [`sdk/lua/GEMINI.md`](file:///home/kieren/Documents/code/optimis/sdk/lua/GEMINI.md) with complete EmmyLua / LuaLS `@class` and `@field` annotations for `ctx` and all returned DAL record tables (`OptimisGuardianRecord` with `authorised_to_collect`, `is_emergency_contact`, `contact_priority`, `OptimisMedicalConditionAlert`, `OptimisSenAndMedicalSummary`, `OptimisPupilRecord`, `OptimisStaffRecord`, `OptimisGroupSummary`, `OptimisTimetableSlot`).
  2. Updated Section 9.4 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md) to document both `await window.mis.rpc(method, args)` and the bubbling DOM events `mis:rpc:success` / `mis:rpc:error` dispatched on `document` by `mis-bridge.js`.

---

## 2. Rejected Proposals (`2A` – `2B`)

### 2A. Inverting `ctx.dal.pupils.list_by_group` Back to `ScopeOwnGroups` by Default
* **Source:** Cover Manager & Supply Teacher (`8_staff_cover.md` §2.2)
* **Decision:** **Rejected.**
* **Architectural Rationale:** Per [`1_4_response.md` Item `1B`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/1_4_response.md), Core read queries intentionally default to school-wide visibility (`ScopeAll`) so any teacher can view any pupil's core profile, class roster, SEN/medical alerts, and PR/court-order-filtered guardian contacts across the school without friction. Endpoints that require strict `:own_groups` enforcement on writes or roster guards can call [`ctx.dal.groups.assert_caller_assigned(group_id)`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L565-L598) (`5_7_response.md` Item `1B`) or pass `{ strict_own_groups = true }`.

---

### 2B. Replacing Calendar-Day `effective_from` / `effective_to` on `group_staff_assignments` with Period-Level (`slot_id`) Temporal Keys in `core.db`
* **Source:** Cover Manager & Supply Teacher (`8_staff_cover.md` §2.1)
* **Decision:** **Rejected.**
* **Architectural Rationale:** Section 2 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md) defines the Core Temporal Model around calendar-day effective dating (`effective_from`, `effective_to`, `academic_year_id`). Coupling `group_staff_assignments` to timetable period slots (`slot_id`) would break non-timetabled groups (registration forms, intervention cohorts, clubs) and complicate `:own_groups` SQL joins across every domain. Granting a cover teacher `:own_groups` access to the covered group for `cover_date` (with `role = 'COVER_TEACHER'` and instant revocation via `ctx.dal.groups.revoke_staff_assignment` in `1A`), while tracking period-level cover slots inside `plugin_cover.db`, maintains a clean Core vs. Plugin boundary.
