# Parent Portal DMZ Sync Plugin (`plugins/parent_dmz_plugin`)

This directory contains the first-party **Parent Portal DMZ Sync Plugin** (`uk.org.optimis.parent.dmz`), implementing the 100% Outbound School-Initiated Pull/Push contract between the private enclave School Container and the Public Cloud DMZ Parent Container (Sections 1, 3, 5, 6 & 9 of the Architecture Blueprint).

---

## 1. Architectural Responsibilities

* **Outbound DMZ Write-Inbox Pull (`pull_dmz_inbox`)**:
  * Initiates an outbound `ctx.http.post` request to `https://dmz.parent.optimis.org.uk/v1/inbox/pull` (outside any open SQLite write transaction) to drain queued parent absence submissions.
  * Verifies each submission against `ctx.dal.pupils.get_by_id` and `ctx.dal.guardians.list_for_pupil` (`require_parental_responsibility = true`).
  * Accepts submissions (`ACCEPTED`) only from guardians holding active Parental Responsibility (`has_parental_responsibility == true`) without court-order safeguarding blocks, and rejects unauthorized/court-order-blocked submissions with `REJECTED_SAFEGUARDING_BLOCK`.
  * Persists all processed inbox decisions into `plugin_parent_dmz.dmz_inbox_submissions`.
* **Outbound Sanitized Read-Projection Push (`push_sanitized_projection`)**:
  * Queries pupil identity (`ctx.dal.pupils.get_by_id`), guardian links (`ctx.dal.guardians.list_for_pupil` with `require_parental_responsibility = false`), and statutory AM/PM attendance session summaries (`ctx.dal.attendance.get_pupil_summary`).
  * Strips the statutory `UPN`, court-order safeguarding notes, and confidential refuge household addresses (`is_confidential_address_redacted`), retaining only authorized PR guardian IDs and a `redacted_confidential_address_count` counter.
  * Pushes the sanitized JSON projection via `ctx.http.post` to `https://dmz.parent.optimis.org.uk/v1/projections/push` before recording an audit snapshot in `plugin_parent_dmz.dmz_projection_snapshots`.
* **Method-Level Scoped RBAC**:
  * Interactive endpoints require `dal.pupils.read:all`, `dal.guardians.read:all`, and `dal.attendance.read:all` (held by Attendance Officers under the `SUPPORT` role and System Admins, while rejecting Classroom Teachers holding only `:own_groups` scopes with `403`).
* **UK GDPR SAR & Retention Hooks**:
  * Implements mandatory `export_pupil_data` (returning structured sections for `"Parent DMZ Write-Inbox Submissions"` with `redactable_fields = { "reason" }` and `"Parent DMZ Read-Projection Snapshots"` with `redactable_fields = { "notes" }`) and `purge_pupil_data` (deleting all pupil rows across both tables).

---

## 2. Directory Structure

* [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/manifest.json): Default-deny permission manifest declaring `id: "uk.org.optimis.parent.dmz"`, `database.alias: "plugin_parent_dmz"`, `egress.allowed_domains: ["https://dmz.parent.optimis.org.uk"]`, `gdpr_hooks`, and interactive endpoints.
* [`migrations/`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/migrations/GEMINI.md): Expand-and-Contract SQLite DDL migrations for `plugin_parent_dmz.db`.
* [`src/`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/src/GEMINI.md): Pure-Go Lua 5.1 (`gopher-lua`) implementation (`init.lua`).
* [`ui/`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/ui/GEMINI.md): Offline-bundled sandboxed micro-frontend (`ui/dist/index.html`) running inside `<iframe sandbox="allow-scripts">` with `connect-src 'none'` CSP.
