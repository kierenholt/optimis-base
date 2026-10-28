# Plugin Contract Feedback — Vertical 4: Parent Portal DMZ Sync (`uk.org.optimis.parent.dmz`)

This document captures engineering feedback on the **Optimis Plugin Contract** (Sections 1, 3, 5, 6, 9, and 10 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md)) gathered while implementing [`plugins/parent_dmz_plugin`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/GEMINI.md) and its full-stack verification suite ([`tests/e2e/parent_dmz_plugin_vertical.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/parent_dmz_plugin_vertical.spec.ts)).

---

## 1. What Worked Exceptionally Well (Keep As-Is)

* **Structural Safety Invariant (`ctx.http` Disallowed Inside Open SQLite Write Transactions, §9.2 & §10.1)**:
  * In [`pull_dmz_inbox`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/src/init.lua#L38-L135) and [`push_sanitized_projection`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/src/init.lua#L137-L246), enforcing that [`ctx.http.post`](file:///home/kieren/Documents/code/optimis/internal/host/egress.go#L219) executes before opening any `WriteDB` transaction led to a clean 3-stage pipeline:
    1. Pull queued submissions via `ctx.http.post("https://dmz.parent.optimis.org.uk/v1/inbox/pull", ...)` or read pupil/guardian/attendance records via `ctx.dal.*` (`ReadDB`).
    2. Validate Parental Responsibility (`has_parental_responsibility`) and strip confidential refuge addresses (`is_confidential_address_redacted`) and statutory `UPN` in Lua memory.
    3. Persist audit rows into `plugin_parent_dmz.db` (`WriteDB`) via `ctx.plugin_db.exec(...)`.
  * This guarantees that Public Cloud DMZ network latency can never hold a local SQLite WAL write lock.
* **Core DAL Safeguarding & Parental Responsibility Enforcement (`ctx.dal.guardians.list_for_pupil`, §3 & §9.2)**:
  * Having [`ListGuardiansForPupilScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_guardians_safeguarding.go) enforce `court_order_restricted = 0`, `require_parental_responsibility`, and automatic confidential household address redaction (`is_confidential_address_redacted = true`) at the Go DAL layer meant the Lua plugin never had raw access to the confidential refuge address (`10 Safe Haven Way`) or court-order notes (`Prohibited Steps Order`).
* **Mandatory UK GDPR Hooks (`export_pupil_data` & `purge_pupil_data`, §5C & §9.3)**:
  * Coupling `database.alias` (`plugin_parent_dmz`) to mandatory `gdpr_hooks` ensured both `dmz_inbox_submissions` (`redactable_fields = { "reason" }`) and `dmz_projection_snapshots` (`redactable_fields = { "notes" }`) were indexed by `pupil_id` and verified for zero-orphan purges from day one.

---

## 2. Where the Contract Was Too Restrictive or Caused Friction

### A. Missing Native JSON Encode/Decode in the Stripped Lua 5.1 Sandbox (`ctx.json`)
* **The Problem**: Because `os`, `io`, `package`, `require`, and `dofile` are stripped from the `gopher-lua` VM ([`host.go`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L460)), plugins have no built-in JSON parser or serializer.
* **Impact in `parent_dmz_plugin`**:
  * In [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/src/init.lua#L1-L36), `pull_dmz_inbox` had to parse the JSON array returned by `https://dmz.parent.optimis.org.uk/v1/inbox/pull` using Lua pattern matching (`string.gmatch(arr_str, "{([^{}]+)}")` and `string.match`), while `push_sanitized_projection` had to hand-roll `json_escape` and `string.format` to serialize `sanitized_payload_json` and `authorized_guardian_ids_json`.
  * Pattern-based JSON parsing in Lua 5.1 breaks if a parent's free-text absence `reason` contains curly braces `{}` or escaped quotes `\"`.
* **Recommendation**: Inject a host-backed **`ctx.json`** capability namespace (`ctx.json.encode(lua_table)` and `ctx.json.decode(json_string)`) backed by Go's `encoding/json` alongside [`ctx.xml`](file:///home/kieren/Documents/code/optimis/internal/host/xml_xsd.go).

### B. Coarse-Grained Batch DAL Lookups for Multi-Pupil Inbox Pulls
* **The Problem**: `ctx.dal.pupils.get_by_id` and `ctx.dal.guardians.list_for_pupil` only accept a single `pupil_id`.
* **Impact in `parent_dmz_plugin`**:
  * In [`pull_dmz_inbox`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/src/init.lua#L55-L80), draining a morning batch of $N$ queued parent absence notices from the DMZ write-inbox requires $2N$ sequential Go-to-Lua DAL queries (`pcall(ctx.dal.pupils.get_by_id, ...)` + `ctx.dal.guardians.list_for_pupil(...)` per submission).
* **Recommendation**: Add batch equivalents to `ctx.dal.pupils` and `ctx.dal.guardians`—specifically `ctx.dal.pupils.get_by_ids_batch(pupil_ids, opts)` and `ctx.dal.guardians.list_for_pupils_batch(pupil_ids, opts)` (mirroring [`ctx.dal.pupils.list_sen_and_medical_batch`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_pupils.go))—so morning inbox pulls execute in $O(1)$ SQL queries.

### C. Host-Level Statutory Identifier (`UPN` / `ULN`) Guard on Untrusted DMZ Egress
* **The Problem**: Because the Attendance Officer holds `dal.pupils.read:all`, `ctx.dal.pupils.get_by_id` returns `pupil.upn`. In [`push_sanitized_projection`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/src/init.lua#L179-L190), stripping `pupil.upn` before posting to `https://dmz.parent.optimis.org.uk/v1/projections/push` relies on the Lua plugin author remembering to omit `upn` from the outbound payload.
* **Recommendation**: Allow `manifest.json` `egress` rules to declare `"disallow_statutory_identifiers": true` (or provide a `:public_safe` modifier on `dal.pupils.read`) so the Go Host `ctx.http` gateway automatically inspects and blocks any outbound DMZ request body containing a 13-character `UPN` or 10-digit `ULN`.

### D. Lazy `WriteDB` Transaction Lifetime vs. Multi-Step Sync State (`ctx.plugin_db`)
* **The Problem**: Calling `ctx.plugin_db.exec(...)` opens an uncommitted `WriteDB` transaction that stays open until the Lua function returns, and any subsequent `ctx.http` call triggers `EGRESS_WRITE_TX_HALT` ([`egress.go`](file:///home/kieren/Documents/code/optimis/internal/host/egress.go)).
* **Impact in `parent_dmz_plugin`**:
  * A plugin cannot persist a `'PULLING'` or `'PENDING_PUSH'` state to `plugin_parent_dmz.db`, perform the outbound `ctx.http.post` call, and then update the row to `'PUSHED'` within the same RPC call.
* **Recommendation**: Auto-commit single `ctx.plugin_db.exec(...)` statements on `WriteDB` by default and provide `ctx.plugin_db.tx(function() ... end)` for explicit multi-statement transactions (keeping `ctx.http` strictly banned inside `ctx.plugin_db.tx`).

---

## 3. Summary Checklist of Recommended Contract Adjustments

| Surface | Current Contract | Recommended Adjustment |
| :--- | :--- | :--- |
| **Lua Runtime (`ctx`)** | `ctx.xml` only; no JSON serializer/parser in stripped Lua 5.1 VM | Add `ctx.json.encode(table)` and `ctx.json.decode(string)` in Go Host. |
| **Core DAL (`ctx.dal`)** | Single-pupil `pupils.get_by_id` and `guardians.list_for_pupil` | Add `pupils.get_by_ids_batch` and `guardians.list_for_pupils_batch` for bulk inbox pulls. |
| **HTTP Egress (`ctx.http`, §9.2)** | UPN stripping is enforced manually by plugin Lua code | Add optional Host egress guard (`disallow_statutory_identifiers: true`) for untrusted DMZ domains. |
| **Plugin DB (`ctx.plugin_db`, §9.2)** | First `.exec()` holds `WriteDB` tx open until RPC exit, blocking later `ctx.http` calls | Auto-commit single `.exec()` calls or add `ctx.plugin_db.tx(fn)` / `ctx.plugin_db.commit()`. |
