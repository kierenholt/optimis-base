# Plugin Contract Feedback — Vertical 6: Medical Room & First Aid (`uk.org.optimis.medical.firstaid`)

**Plugin Path:** [`plugins/first_aid_medical_plugin`](file:///home/kieren/Documents/code/optimis/plugins/first_aid_medical_plugin/GEMINI.md)  
**Plugin ID:** `uk.org.optimis.medical.firstaid`  
**Attached Database:** `plugin_first_aid` (`plugin_first_aid.db`)  
**Verification Suites:** [`tests/e2e/first_aid_medical_plugin_vertical.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/first_aid_medical_plugin_vertical.spec.ts) & [`internal/host/first_aid_medical_plugin_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/first_aid_medical_plugin_test.go)

---

## 1. Executive Summary: Is the Contract Too Restrictive?

Overall, the **Plugin Contract (Sections 9 & 10 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md))** proved well-suited for the Medical Room & First Aid vertical:

1. **Zero-Leakage Safeguarding & Refuge Redaction (`ctx.dal.guardians.list_for_pupil`):** Calling `ctx.dal.guardians.list_for_pupil(pupil_id, opts)` (`dal.guardians.read:all`) enforced both court-order blocks and confidential refuge redaction inside the Go Core DAL ([`ListGuardiansForPupilScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_guardians_safeguarding.go#L41)) before data ever reached the Lua VM. The court-order-blocked father (`George Byron`, `Prohibited Steps Order`) was completely excluded from the returned slice, while the refuge guardian (`Judith Milbanke`, emergency contact #2) remained callable by phone (`07700900333`) with `is_confidential_address_redacted = true` and all refuge address fields (`10 Safe Haven Way`, `Confidential Refuge`, `SW1A 1AA`) blanked out.
2. **Coarse-Grained Life-Safety Medical Lookup (`ctx.dal.pupils.list_sen_and_medical_batch`):** Fetching `CRITICAL` anaphylaxis/IHCP alerts (`alert_level_id = 'CRITICAL'`, `has_ihcp = true`, `emergency_response_plan = 'Administer EpiPen immediately and call 999'`) required a single DAL call without $N+1$ queries.
3. **Post-Commit Custom Event Bus (`ctx.events.emit` + `workers`):** Emitting `plugin.uk.org.optimis.medical.firstaid.logged` inside `log_first_aid_treatment` buffered the event in memory during the `WriteDB` transaction and dispatched it to `workers.on_first_aid_logged` strictly after commit, preventing event-driven side effects from holding a SQLite write lock.
4. **UK GDPR SAR & Retention Hooks:** Auto-bound `export_pupil_data` and `purge_pupil_data` hooks cleanly exported `redactable_fields` (`injury_summary`, `treatment_given`) for third-party name redaction and purged 100% of pupil rows across both `first_aid_treatments` and `first_aid_event_logs`.

However, implementing this vertical surfaced **five concrete areas** where the contract or Host bindings were either too restrictive or required adjustment.

---

## 2. Concrete Findings & Recommended Contract Adjustments

### 2A. `ctx.dal.pupils.list_sen_and_medical_batch` & `ctx.dal.medical.list_batch` Omit Critical Medication & Trigger Columns (High Priority)

* **Why the Current Contract is Too Restrictive:**
  * In [`schema/core/03_sen_medical_safeguarding.sql`](file:///home/kieren/Documents/code/optimis/schema/core/03_sen_medical_safeguarding.sql), the `pupil_medical_conditions` table stores rich clinical fields essential for a Medical Room / First Aid workflow:
    * `triggers` (e.g., peanuts, tree nuts, wasp venom)
    * `reaction_symptoms` (e.g., airway constriction, hives)
    * `medication_name_and_dose` (e.g., *EpiPen 0.3mg IM* / *Salbutamol 100mcg inhaler*)
    * `medication_location` (e.g., *Medical Room Red Cabinet #2 & Pupil Blazer Pocket*)
    * `medication_expiry_date`
    * `parental_consent_given`
  * However, [`ListPupilSenAndMedicalBatch`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go#L368) and [`internal/host/dal_bindings.go`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L401) currently project only 6 columns (`id`, `condition_name`, `condition_category`, `alert_level_id`, `has_ihcp`, `emergency_response_plan`).
  * While a minimal summary is appropriate for a Classroom Teacher's markbook badge (`dal.medical.read:own_groups`), a School Nurse or First Aider holding `dal.medical.read:all` in the Medical Room **must** see where the pupil's auto-injector/inhaler is stored (`medication_location`), its dose (`medication_name_and_dose`), its expiry date (`medication_expiry_date`), and whether `parental_consent_given` is on file before administering medication.
* **Recommended Contract Adjustment:**
  * Expand `PupilMedicalConditionAlert` in [`internal/dal/domains_pupils.go`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go) and [`internal/host/dal_bindings.go`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go) to project `triggers`, `reaction_symptoms`, `medication_name_and_dose`, `medication_location`, `medication_expiry_date`, and `parental_consent_given` (at minimum whenever the caller holds `dal.medical.read:all`, or across both `:own_groups` and `:all`).

---

### 2B. `SUPPORT` Persona Role Scopes Omitted `dal.medical.read:all` (Fixed During Implementation)

* **What Happened:**
  * In UK schools, First Aiders, Medical Room Officers, and Welfare Assistants sit under the `SUPPORT` staff category (`School Office & Pastoral Support`), rather than `HOD` or `DSL`.
  * Prior to Vertical 6, the `SUPPORT` role in [`internal/dal/persona_auth.go`](file:///home/kieren/Documents/code/optimis/internal/dal/persona_auth.go) and [`cmd/mis-seed-qa/seed_domains_00_to_03.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/seed_domains_00_to_03.go) granted `dal.pupils.read:all`, `dal.attendance.read:all`, `dal.attendance.write:all`, `dal.guardians.read:all`, and `dal.safeguarding.create`, but omitted `dal.medical.read:all`.
* **Adjustment Applied & Future Recommendation:**
  * We added `"dal.medical.read:all"` to `SUPPORT` in [`internal/dal/persona_auth.go`](file:///home/kieren/Documents/code/optimis/internal/dal/persona_auth.go#L561) and [`cmd/mis-seed-qa/seed_domains_00_to_03.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/seed_domains_00_to_03.go#L197) and exposed `school_nurse_staff_id` on `PersonaTestFixtures`.
  * **Future RBAC Refinement:** If a larger secondary school wants to separate general reception/office staff from qualified First Aiders / School Nurses, add a dedicated `MEDICAL_OFFICER` (or `FIRST_AIDER`) role in `roles` holding `["dal.pupils.read:all","dal.medical.read:all","dal.guardians.read:all","dal.safeguarding.create"]` without granting `dal.attendance.write:all`.

---

### 2C. Undocumented Lua Table Field Names on `ctx.dal.guardians.list_for_pupil` (`authorised_to_collect` & `contact_priority`) (Medium Priority)

* **What Happened:**
  * When writing [`plugins/first_aid_medical_plugin/src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/first_aid_medical_plugin/src/init.lua), we initially read `g.can_collect_child` / `g.can_collect_pupil` and `g.emergency_contact_priority` from the guardian rows returned by `ctx.dal.guardians.list_for_pupil`.
  * Because Lua tables evaluate missing keys to `nil` (falsy) without raising an error, unrestricted Mother `Anne Byron` was silently marked as `RESTRICTED_COLLECTION` until caught by [`internal/host/first_aid_medical_plugin_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/first_aid_medical_plugin_test.go).
  * Inspecting [`guardianToLuaTable`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L160-L182) showed the actual bound keys are `authorised_to_collect`, `is_emergency_contact`, and `contact_priority`.
* **Recommended Contract Adjustment:**
  * Publish an explicit **Lua DAL Field Reference** in Section 9.2 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md) (or ship a `sdk/lua/ctx_types.lua` EmmyLua/LuaLS annotation file for IDE autocomplete) listing the exact field names returned by `pupilToLuaTable`, `guardianToLuaTable`, `staffToLuaTable`, and `list_sen_and_medical_batch`.

---

### 2D. Hardcoded Plugin ID Allowlist for HTTP `403 Forbidden` Scope Errors in `internal/host/server.go` (Medium Priority)

* **What Happened:**
  * In [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L195), when `h.ExecuteRPC` returns an error containing `"caller lacks required DAL scopes"`, the HTTP gateway only mapped that error to `http.StatusForbidden` (`403`) if `pluginID` matched a hardcoded allowlist (`uk.gov.dfe.census`, `uk.org.optimis.parent.dmz`, `uk.org.optimis.assessment.markbook`, `uk.org.optimis.ops.cover`), returning `400 Bad Request` for all other plugins.
  * We added `uk.org.optimis.medical.firstaid` to that check in [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L195), but hardcoding third-party or first-party plugin IDs inside the Host HTTP router violates strict Core-vs-Plugin separation (Section 8).
* **Recommended Contract Adjustment:**
  * Standardize the Host RPC Gateway Contract so **any** `ExecuteRPC` failure caused by `caller lacks required DAL scopes` (ideally via a typed `ErrUnauthorizedScope` sentinel error in `internal/host`) returns `403 Forbidden` across all plugins (or `403` for all plugins except legacy `uk.org.optimis.test` if backward compatibility with a specific MVP test is required).

---

### 2E.Sandboxed UI Bridge Event Naming (`document` `mis:rpc:success` / `mis:rpc:error` vs `window` `mis:rpc:response`) (Low-Medium Priority)

* **What Happened:**
  * Section 9.4 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md) describes the `window.postMessage` wire protocol message as `mis:rpc:response`, whereas [`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js) intercepts that `postMessage` and dispatches custom DOM events `mis:rpc:success` and `mis:rpc:error` on `document` (in addition to resolving `window.mis.rpc()` Promises).
* **Recommended Contract Adjustment:**
  * Clarify in Section 9.4 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md) that declarative `data-mis-rpc` consumers in plugin HTML should listen to `document.addEventListener('mis:rpc:success', ...)` and `document.addEventListener('mis:rpc:error', ...)` (while imperative JS callers use `await window.mis.rpc(method, args)`).

---

## 3. Summary Matrix

| Surface Evaluated | Verdict | Notes / Action |
| :--- | :--- | :--- |
| `ctx.dal.guardians.list_for_pupil` (`dal.guardians.read:all`) | **Excellent** | Automatically excludes `court_order_restricted = 1` father (`George Byron`) and redacts `is_confidential_address = 1` refuge address (`Judith Milbanke`) while preserving emergency phone numbers. |
| `ctx.dal.pupils.list_sen_and_medical_batch` (`dal.medical.read:all`) | **Needs Expansion** | Returns `CRITICAL` / `IHCP` / `emergency_response_plan` cleanly, but should also project `medication_name_and_dose`, `medication_location`, `medication_expiry_date`, `triggers`, `reaction_symptoms`, and `parental_consent_given` from `pupil_medical_conditions`. |
| `ctx.events.emit("plugin.uk.org.optimis.medical.firstaid.logged")` | **Excellent** | Post-commit custom event emission and background worker subscription (`workers.on_first_aid_logged`) worked out-of-the-box without holding SQLite write locks. |
| `SUPPORT` Role Default Scopes | **Adjusted** | Added `dal.medical.read:all` to `SUPPORT` (`SchoolNurseStaffID`) in `persona_auth.go` and QA seeder; consider a dedicated `FIRST_AIDER` / `MEDICAL_OFFICER` role for finer least-privilege separation. |
| UK GDPR `export_pupil_data` & `purge_pupil_data` | **Excellent** | Exported `redactable_fields` (`injury_summary`, `treatment_given`) and purged 100% of pupil treatment and event audit rows cleanly. |
