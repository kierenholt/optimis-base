# Plugin Contract Feedback — Vertical 10: Sparx Maths Third-Party Partner Sync (`uk.co.sparx.maths`)

This document captures engineering feedback on the **Optimis Plugin Contract** (Sections 4, 5, 8 Gap 1, 9, and 10 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md)) gathered while implementing [`plugins/sparx_maths_plugin`](file:///home/kieren/Documents/code/optimis/plugins/sparx_maths_plugin/GEMINI.md) and its verification suites ([`tests/e2e/sparx_maths_plugin_vertical.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/sparx_maths_plugin_vertical.spec.ts) and [`internal/host/sparx_maths_plugin_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/sparx_maths_plugin_test.go)).

---

## 1. Executive Summary: Is the Contract Too Restrictive?

**Overall Verdict: The core architectural invariants are NOT too restrictive—they fit the Third-Party EdTech Partner use case cleanly—but 3 targeted adjustments around vendor credential encryption, role-separated admin vs. teacher endpoints, and multi-row SQL batch inserts will make the contract significantly safer and more ergonomic for third-party vendors.**

Specifically:
* The **Two-Phase Read/Egress $\rightarrow$ WriteDB** invariant (`ctx.http` disallowed inside open SQLite write transactions) was straightforward to implement and prevented external partner API latency from holding a `WriteDB` lock.
* The **`1A`–`1I` platform enhancements** (`ctx.json.encode`/`decode`, `ctx.clock.now_utc()`, `ctx.crypto.uuid()`, `ctx.school`, and auto-bound `gdpr_hooks`) eliminated all of the boilerplate friction seen in earlier verticals.
* However, storing third-party vendor API keys in `plugin_<name>.db` exposed an important **Defense-in-Depth gap** around secret storage and diagnostic scrubbing (which required extending [`diagnostics.go`](file:///home/kieren/Documents/code/optimis/internal/host/diagnostics.go#L58-L77) during this implementation).

---

## 2. What Worked Exceptionally Well (Keep As-Is)

1. **Two-Phase Read/Egress $\rightarrow$ WriteDB Pattern (§9.2 & §10.1)**:
   * Splitting [`sync_group_homework`](file:///home/kieren/Documents/code/optimis/plugins/sparx_maths_plugin/src/init.lua#L89-L304) into **Phase 1 (Read & Egress)** (`ctx.plugin_db.query` on `ReadDB` + `ctx.dal.pupils.list_by_group` + `ctx.http.post`) and **Phase 2 (WriteDB Persistence)** (`ctx.plugin_db.exec` to insert returned homework completion scores and sync run metadata) required zero workarounds.
   * When we simulated a single-phase violation (`WRITE_TX_BEFORE_EGRESS`), the Go Host immediately rolled back the uncommitted write transaction, released the lock tracker, and recorded `EGRESS_WRITE_TX_HALT` in the diagnostic ring buffer.
2. **Scoped `:own_groups` Roster Resolution with `:all` Subsumption (`1B`)**:
   * Declaring `dal.groups.read:own_groups` and `dal.pupils.read:own_groups` on `sync_group_homework` automatically restricted classroom teachers to syncing only their assigned teaching sets (excluding mid-year transferred pupils and other teachers' sets) while still allowing Heads of Department (`HOD`) or System Admins (`ADMIN`) holding `:all` to trigger syncs across the department.
3. **`ctx.json`, `ctx.clock`, `ctx.crypto`, and Auto-Bound `gdpr_hooks` (`1A`, `1F`, `1G`)**:
   * Encoding the outbound roster payload and decoding the partner's nested `scores` JSON array via `ctx.json.encode` / `ctx.json.decode` took 4 lines of clean Lua.
   * Auto-binding `gdpr_hooks` (`export_pupil_data` and `purge_pupil_data`) without duplicating them in `manifest.json`'s `endpoints` map kept the manifest concise.

---

## 3. Where the Contract Needs Adjustment

### A. Plaintext Vendor API Keys in `plugin_<name>.db` & Diagnostic Scrubbing (§4 & §8 Gap 1)
* **The Friction / Risk**:
  * Section 4 states: *"Plugins manage their own vendor credentials within their isolated `plugin_<name>.db` state."*
  * When a plugin stores its vendor API key in a regular SQLite table (`plugin_sparx.sparx_vendor_config`) and later loads that key via `ctx.plugin_db.query` inside `sync_group_homework`, the Go Host has **no intrinsic knowledge** that that specific SQLite column is a sensitive credential.
  * Initially, [`collectArgSecretStrings`](file:///home/kieren/Documents/code/optimis/internal/host/diagnostics.go#L79-L109) only knew about secrets passed directly in the current RPC's `args` map. When `sync_group_homework` loaded the stored API key from `plugin_sparx.db` (not from `args`) and threw a simulated fault containing the key, the ring buffer relied on regex pattern matching (`vendorSecretRegex` in [`diagnostics.go`](file:///home/kieren/Documents/code/optimis/internal/host/diagnostics.go#L61)) to strip `Bearer ...` and `spx_...` tokens.
  * Relying purely on regexes for third-party vendor keys is fragile because vendors use arbitrary token formats (e.g., high-entropy base64 or hex strings without a known prefix). Furthermore, the API key sits in plaintext on disk inside `plugin_sparx.db`.
* **Recommended Contract Adjustment**:
  * **Add Host-Managed Secret Seal/Open Helpers (`ctx.crypto.seal_secret` / `ctx.crypto.open_secret`) or a Dedicated `ctx.secrets` KV Store:**
    1. Allow plugins to store encrypted vendor credentials via `ctx.crypto.seal_secret(plaintext)` (encrypted using a per-plugin key derived from `HOST_KMS_KEY_HANDLE` + `plugin_id`) and decrypt them in memory via `ctx.crypto.open_secret(ciphertext)`.
    2. Whenever `ctx.crypto.open_secret` is called during a Lua invocation, the Go Host automatically appends that decrypted secret string to the invocation's `extraSecrets` scrub list in `ExecuteRPC` / `ExecuteWorker`. That guarantees **100% deterministic scrubbing** of any decrypted vendor credential from the diagnostic ring buffer and service logs—regardless of the vendor's token format—and ensures API keys are encrypted at rest in `plugin_<name>.db`.

### B. Admin-Only Configuration Endpoints vs. Teacher Operational Endpoints (§5A & §9.1)
* **The Friction**:
  * In a real school deployment, **configuring the school-wide Sparx Maths vendor API key** (`configure_vendor_credentials`) is an **IT Admin / Head of Maths** setup task, whereas **syncing a class's homework scores** (`sync_group_homework`) is a daily **Classroom Teacher** workflow.
  * Currently, `manifest.json` `endpoints.<method>.scopes` only checks **Core DAL scopes** (`dal.<domain>.<action>:<modifier>`). Because `configure_vendor_credentials` only writes to `plugin_sparx.db` and does not touch `core.db`, there is no dedicated scope for *"plugin configuration / admin settings"* other than requiring a `:all` Core DAL scope (e.g. `dal.groups.read:all`) or checking `ctx.caller.active_roles` manually in Lua.
  * Conversely, if a developer puts `dal.groups.read:own_groups` on `configure_vendor_credentials` so a teacher can test it in a single-persona flow, any classroom teacher can overwrite the school's vendor API key.
* **Recommended Contract Adjustment**:
  * Support an optional `"roles"` or `"plugin_admin": true` attribute on `manifest.json` endpoints (or a standard capability scope `plugin.config.write` granted to `ADMIN` and `HOD` roles), and allow `configure_vendor_credentials` to be gated cleanly at the Host RBAC layer without conflating plugin settings with Core DAL data scopes.

### C. Per-Row `ctx.plugin_db.exec` Loop Overhead When Persisting Batch Partner Scores (§9.2)
* **The Friction**:
  * When `https://sync.sparxmaths.uk` returns homework completion scores for a class of 30 pupils (or a whole-year cohort of 240 pupils), [`sync_group_homework`](file:///home/kieren/Documents/code/optimis/plugins/sparx_maths_plugin/src/init.lua#L202-L261) must iterate `for _, item in ipairs(raw_scores) do` and invoke `ctx.plugin_db.exec(...)` 30–240 times across the Go-Lua boundary.
  * While all `.exec()` calls within a single RPC share the same underlying `WriteDB` `*sql.Tx` (so SQLite only commits once at the end of the RPC), each call still parses the SQL string, runs `validatePluginDBSQLScope` regex checks, and allocates Lua return tables.
* **Recommended Contract Adjustment**:
  * Add `ctx.plugin_db.exec_batch(sql_query, array_of_param_tables)` to `ctx.plugin_db`, which prepares the SQL statement once on `*writeTx`, validates the schema prefix once, and executes all parameter rows in a tight Go loop before returning `{ rows_affected = total }`.

---

## 4. Summary Table of Recommended Adjustments

| # | Surface | Current Contract Behavior | Proposed Adjustment | Priority |
| :--- | :--- | :--- | :--- | :--- |
| **1** | **Vendor Secrets & Diagnostics (`ctx.crypto` / `diagnostics.go`)** | Vendor API keys stored in plaintext in `plugin_<name>.db`; diagnostic scrubbing of DB-loaded secrets relies on regex patterns (`vendorSecretRegex`). | Add `ctx.crypto.seal_secret(str)` & `ctx.crypto.open_secret(cipher)` (using `HOST_KMS_KEY_HANDLE`), and auto-register any opened secret into the per-invocation `extraSecrets` scrubber list. | **High** *(Security / Defense-in-Depth)* |
| **2** | **Batch Plugin DB Writes (`ctx.plugin_db`)** | `ctx.plugin_db.exec(sql, params)` executes one row per Lua-to-Go call. | Add `ctx.plugin_db.exec_batch(sql, params_batch)` using a single prepared statement inside `*writeTx`. | **Medium** *(Performance / Ergonomics)* |
| **3** | **Plugin Admin Scope (`manifest.json` `endpoints`)** | Endpoints that only mutate `plugin_<name>.db` settings must borrow a Core DAL scope (`dal.groups.read:...`) to satisfy RBAC. | Introduce a standard `plugin.config.write` scope (or `require_roles` manifest field) for vendor credential & settings endpoints. | **Low** *(RBAC Clarity)* |
