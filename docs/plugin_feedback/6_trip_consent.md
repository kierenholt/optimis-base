# Plugin Contract Feedback — Vertical 7: Educational Visits, PR Consent & Trip Emergency Pack (`uk.org.optimis.ops.trips`)

**Plugin Path:** [`plugins/trips_consent_plugin`](file:///home/kieren/Documents/code/optimis/plugins/trips_consent_plugin/GEMINI.md)  
**Plugin ID:** `uk.org.optimis.ops.trips`  
**Attached Database:** `plugin_trips` (`plugin_trips.db`)  
**E2E Verification Suite:** [`tests/e2e/trips_consent_plugin_vertical.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/trips_consent_plugin_vertical.spec.ts)

---

## 1. Executive Summary: Is the Contract Too Restrictive?

The Core Plugin Contract (Sections 9 & 10 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md))—especially with the `1A`–`1I` platform improvements from [`1_4_response.md`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/1_4_response.md)—handled 90% of the Educational Visits & Trip Emergency Pack vertical cleanly:

1. **Core Safeguarding Invariants Held Without Leakage:** Calling `ctx.dal.guardians.list_for_pupil(pupil_id, { require_parental_responsibility = true })` allowed the plugin to enforce UK statutory Parental Responsibility (`PR`) and court-order blocks (`Prohibited Steps Order`) in 10 lines of Lua, with zero exposure of `court_order_notes` or confidential refuge addresses (`10 Safe Haven Way`) to the Lua VM or iframe DOM.
2. **Coarse-Grained Batch Emergency Pack Generation:** Combining `ctx.dal.pupils.list_by_group` and `ctx.dal.pupils.list_sen_and_medical_batch` allowed `generate_trip_emergency_pack` to assemble a full off-site emergency manifest (SEN `E`/`K` provisions, primary need codes, `IHCP` flags, and life-safety emergency response plans like `Administer EpiPen immediately and call 999`) without $N+1$ query loops.
3. **Seamless UK GDPR SAR Redaction & Retention Purge:** Auto-bound `gdpr_hooks` (`1G`) and `redactable_fields = { "dietary_medical_notes" }` integrated directly into the Host Admin SAR Redaction UI, allowing third-party pupil names in free-text consent notes to be redacted prior to export.

However, implementing this vertical exposed **three concrete areas where the current RBAC scope model and cross-plugin boundaries are either too restrictive or need targeted adjustment**.

---

## 2. Concrete Findings & Recommended Contract Adjustments

### 2A. `TEACHER` Role Default Scopes Omitted `dal.guardians.read:own_groups` (High Priority — Fixed in Milestone 1 Fixtures)

* **What Happened:**
  * A Trip Leader is typically a Classroom Teacher (`TEACHER` role) organizing an educational visit for their class.
  * To verify parental consent (`ctx.dal.guardians.list_for_pupil`) and include emergency contact numbers on the Trip Emergency Pack, the plugin endpoints `record_trip_consent` and `generate_trip_emergency_pack` must declare `"dal.guardians.read:own_groups"`.
  * While the Go Host (`internal/host/dal_bindings.go`) and Core DAL (`ListGuardiansForPupilScoped` in `internal/dal/domains_guardians_safeguarding.go`) explicitly implement `dal.guardians.read:own_groups` (`ScopeOwnGroups`), the `TEACHER` role's `default_scopes_json` in the database seeders only granted `dal.pupils.read:own_groups`, `dal.groups.read:own_groups`, `dal.attendance.*:own_groups`, `dal.sen.read:own_groups`, and `dal.medical.read:own_groups`.
  * Consequently, the Host RPC gateway initially rejected the Trip Leader's call to `record_trip_consent` with `400 caller lacks required DAL scopes for uk.org.optimis.ops.trips.record_trip_consent`.
* **Adjustment Needed:**
  * Ensure `"dal.guardians.read:own_groups"` is part of the canonical baseline `TEACHER` role scopes (updated in [`internal/dal/test_fixtures.go`](file:///home/kieren/Documents/code/optimis/internal/dal/test_fixtures.go#L288)).
  * **Optional Granularity Refinement:** If a school DPO wants Classroom Teachers to verify `PR` status and emergency phone numbers for `:own_groups` pupils *without* exposing full household address lines during routine lessons, consider splitting `dal.guardians.read:own_groups` or adding an `include_address = false` projection default when called under `:own_groups`.

---

### 2B. `:own_groups` Restriction Blocks Whole-Year or Cross-Curricular Educational Visits (Medium-High Priority)

* **Why the Current Contract is Too Restrictive for Real-World School Trips:**
  * In `1C` ([`1_4_response.md`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/1_4_response.md)), core pupil and group roster reads (`ctx.dal.pupils.list_by_group`, `ctx.dal.groups.list_group_pupils`) were relaxed so a teacher can view rosters across the school.
  * **However**, `ctx.dal.guardians.list_for_pupil`, `ctx.dal.pupils.list_sen_and_medical_batch`, `ctx.dal.sen.list_batch`, and `ctx.dal.medical.list_batch` still strictly enforce `ensurePupilInOwnGroups` (`group_staff_assignments` for the calling `staff_id`) whenever the caller holds `:own_groups` scopes.
  * In UK schools, trips are frequently **not** restricted to a single timetable class taught by one teacher—for example:
    * A Year 8 Geography Field Trip taking all 120 pupils across 4 form groups (`8A`, `8B`, `8C`, `8D`), led by one Geography Teacher (`TEACHER`).
    * A Duke of Edinburgh, Ski Trip, or Orchestra Visit combining pupils from Years 9–11.
  * Under the current contract, if a `TEACHER` is the Trip Leader for a cross-class visit, calling `ctx.dal.guardians.list_for_pupil` or `ctx.dal.pupils.list_sen_and_medical_batch` for a pupil on the trip whom they do not personally teach fails fast with `access denied: pupil is not in caller's own_groups`. The only workaround today is elevating the Trip Leader to `HOD`/`ADMIN` (`:all`).
* **Recommended Contract Adjustment:**
  * **Temporal `TRIP_COHORT` Group Assignment in `core.db`:** Model educational visit cohorts as temporal groups in `core.db` (`groups.group_type = 'TRIP_COHORT'` or `'EVENT_GROUP'`) with `effective_from` and `effective_to` bounding the trip dates, and assign the Trip Leader + accompanying staff via `group_staff_assignments`.
  * Because `:own_groups` in [`internal/dal/domains.go`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go) resolves any active `group_staff_assignments` + `group_pupil_memberships` for `reference_date`, assigning the Trip Leader to a temporal `TRIP_COHORT` group automatically grants `:own_groups` access to `dal.guardians.read:own_groups`, `dal.sen.read:own_groups`, and `dal.medical.read:own_groups` **strictly for the pupils on that trip and strictly for the dates of the trip**, without granting permanent `:all` access across the school.

---

### 2C. Outbound/Inbound Parent DMZ Bridge for Parental Trip Consents (Medium Priority)

* **Observation:**
  * In Vertical 7, parental consent (`record_trip_consent`) is recorded interactively via the plugin RPC gateway and validated against `ctx.dal.guardians.list_for_pupil(pupil_id, { require_parental_responsibility = true })`.
  * According to Section 1 of the Architecture Blueprint (*Parent Container: Public Cloud DMZ hosting sanitized read-projections and an untrusted write-inbox*), parents in production do not log into the School Container directly; they submit consent forms into the **Parent DMZ Write-Inbox**, and the School Container pulls those submissions outbound.
  * Currently, plugins cannot subscribe to verified DMZ write-inbox items via the Core Event Bus (the Standard Core Event Catalog in [`internal/dal/events.go`](file:///home/kieren/Documents/code/optimis/internal/dal/events.go) only defines `core.pupil.enrolled`, `core.group.membership_changed`, `core.attendance.session_recorded`, `core.safeguarding.postbox_created`, and `core.gdpr.pupil_purged`).
* **Recommended Contract Adjustment:**
  * Add a standard Core domain event—e.g. `core.parent_dmz.submission_received` (or allow `uk.org.optimis.parent.dmz` to emit a versioned custom event as envisioned in Section 3 *Event Bus*)—after the Host/DMZ pull verifies `has_parental_responsibility = 1` and `court_order_restricted = 0`.
  * Feature plugins like `uk.org.optimis.ops.trips` can then subscribe a background worker (`workers.on_parent_consent_submission`) to automatically ingest verified parental trip consents from the DMZ inbox into `plugin_trips.db`.

---

### 2D. Opaque Rejection Reason (`has_parental_responsibility = 0` vs. `court_order_restricted = 1`) — Keep As-Is

* **Observation:**
  * Because `ctx.dal.guardians.list_for_pupil(pupil_id, { require_parental_responsibility = true })` filters out both non-PR guardians (`GRANDPARENT`, `has_parental_responsibility = 0`) and court-order-restricted guardians (`FATHER`, `court_order_restricted = 1`) inside SQLite before returning the slice to Lua, the plugin cannot distinguish *why* a given `guardian_person_id` was excluded and returns a unified `GUARDIAN_NOT_AUTHORIZED_FOR_CONSENT`.
* **Verdict — Do Not Change:**
  * This is a **feature, not a bug**. Disclosing to a general classroom teacher that a specific parent is blocked due to a `Prohibited Steps Order` (`court_order_restricted = 1`) vs. lacking PR (`has_parental_responsibility = 0`) can inadvertently leak sensitive family court safeguarding status to non-DSL staff. Returning a uniform `GUARDIAN_NOT_AUTHORIZED_FOR_CONSENT` error code preserves strict KCSIE need-to-know isolation.

---

## 3. Summary Matrix

| Surface Evaluated | Verdict | Notes / Action |
| :--- | :--- | :--- |
| `ctx.dal.guardians.list_for_pupil` (`require_parental_responsibility = true`) | **Excellent** | Enforces PR (`has_parental_responsibility = 1`), strips `court_order_restricted = 1`, and redacts `is_confidential_address = 1` automatically. |
| `ctx.dal.pupils.list_by_group` & `list_sen_and_medical_batch` | **Excellent** | Zero $N+1$ queries when building the Trip Emergency Manifest with SEN (`E`/`K`) and Medical (`IHCP` + emergency response plan) alerts. |
| `TEACHER` Default Scopes (`dal.guardians.read:own_groups`) | **Adjusted** | Added `dal.guardians.read:own_groups` to Milestone 1 `TEACHER` role fixtures so Trip Leaders can verify PR consent for their groups. |
| Cross-Group Trips under `:own_groups` | **Needs Adjustment** | Recommend temporal `TRIP_COHORT` group assignments so Trip Leaders get time-bounded `:own_groups` access to medical/guardian data for cross-class trip cohorts without requiring `:all`. |
| UK GDPR `export_pupil_data` & `purge_pupil_data` | **Excellent** | Auto-bound `gdpr_hooks` (`1G`) and `redactable_fields` (`dietary_medical_notes`) worked out-of-the-box with the Host SAR Redaction UI. |
