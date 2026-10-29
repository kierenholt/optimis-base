# Plugin Contract Feedback — Vertical 5: Assessment Markbook & MAT Attainment Rollup (`uk.org.optimis.assessment.markbook`)

**Plugin Path:** [`plugins/assessment_markbook_plugin/`](file:///home/kieren/Documents/code/optimis/plugins/assessment_markbook_plugin/GEMINI.md)
**Plugin ID:** `uk.org.optimis.assessment.markbook`
**Attached Database:** `plugin_markbook` (`plugin_markbook.db`)
**Personas Tested:** Classroom Teacher (`TEACHER`, `:own_groups`), Head of Department (`HOD`, `:all`), System Admin / DPO (`ADMIN`, `:all` + GDPR SAR/Purge)

---

## 1. Executive Summary: Is the Plugin Contract Too Restrictive?

Overall, **the Plugin Contract is well-calibrated and not overly restrictive** for building a stateful, multi-persona school assessment and markbook vertical. In particular, the platform enhancements delivered in [`1_4_response.md`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/1_4_response.md) (`1A`–`1I`) eliminated almost all of the boilerplate that affected Verticals 1–4:

* **Hierarchical Scope Subsumption (`1B`)** allowed the Head of Department (`HOD` holding `dal.pupils.read:all` and `dal.groups.read:all`) to seamlessly call `:own_groups` endpoints (`get_markbook_context`, `create_markbook_column`, `save_pupil_grades`) alongside the `:all`-only `get_cross_set_attainment_rollup` endpoint without needing duplicate endpoint definitions in [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/assessment_markbook_plugin/manifest.json).
* **Auto-Bound `gdpr_hooks` (`1G`)** allowed `export_pupil_data` (`redactable_fields: ["teacher_comment"]`) and `purge_pupil_data` to be declared once in `gdpr_hooks` without polluting the interactive `endpoints` table.
* **Single-Transaction `WriteDB` Atomicity** ensured that saving a batch of pupil grades and formative comments (`save_pupil_grades`) either commits all grades atomically or rolls back cleanly if any grade violates the MAT grading scale (`INVALID_MAT_GRADE_VALUE`) or group roster membership (`PUPIL_NOT_IN_GROUP_ROSTER`).

However, implementing Vertical 5 surfaced **four specific contract adjustments** (one of which required a small Host extension during implementation, and three of which are recommended refinements before locking `cmd/mis-sdk`).

---

## 2. Contract Friction Points & Recommended Adjustments

### 2A. Exposing MAT Synced Policies & Grading Schemas to Sandboxed Plugins (`ctx.school`)
* **What Happened:**
  * Vertical 5 requires teachers to enter assessment grades against a MAT-standardized grading scale (`9–1` or `WT,EXS,GDS`) pulled into `core.db`'s `__system_config` table via `POST /api/sync/mat/pull-policies`.
  * Because Lua plugins cannot query `core.db.__system_config` directly (by design), [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/assessment_markbook_plugin/src/init.lua) initially had no way to check whether MAT policies had been synced or which `mat_grading_schema_version` was active.
  * **Adjustment Made During Implementation:** We extended `ctx.school` in [`internal/host/host.go`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L963-L976) to expose `mat_id` and `mat_grading_schema_version` from `__system_config`, allowing the plugin to fail fast with `MAT_GRADING_SCHEMA_NOT_SYNCED` when unsynced.
* **Further Contract Adjustment Needed:**
  * While `ctx.school.mat_grading_schema_version` exposes the version identifier (`"MAT-GCSE-9-1-v2026.1"`), the actual grade tokens and numeric point equivalencies (`9–1` $\rightarrow$ `9..1, U`; `WT,EXS,GDS` $\rightarrow$ `GDS=3, EXS=2, WT=1`) still had to be defined in Lua (`SCALE_DEFINITIONS` in [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/assessment_markbook_plugin/src/init.lua#L10-L24)).
  * **Recommendation:** Populate a structured, read-only `ctx.school.mat_grading_schema` table (decoded from `__system_config` when MAT policies are pulled) so assessment plugins can dynamically discover MAT-defined grade tokens, point values, and key stage mappings without hardcoding scale tables in Lua.

### 2B. First-Class `:own_groups` Write-Guard Helper for Plugin-Owned Tables (`ctx.plugin_db`)
* **What Happened:**
  * Earlier platform update `1C` intentionally relaxed `ctx.dal.pupils.list_by_group(group_id)` and `ctx.dal.groups.list_group_pupils(group_id)` so Classroom Teachers holding `:own_groups` can read pupil rosters across the school for cover/pastoral workflows.
  * However, in a markbook plugin, mutations write to **plugin-owned tables** (`markbook_columns` and `markbook_grades` in `plugin_markbook.db`), not to Core DAL write methods. Because the Go Host cannot automatically infer that `args.group_id` in a `ctx.plugin_db.exec()` call requires `:own_groups` teacher assignment, the Lua plugin must explicitly verify that the caller is assigned to `args.group_id`.
  * In [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/assessment_markbook_plugin/src/init.lua#L96-L137), we wrote a custom `ensure_caller_can_access_group(ctx, group_id)` helper that inspects `ctx.caller.active_roles` for `:all` personas (`HOD`, `ADMIN`, `DSL`, `SUPPORT`) and otherwise cross-checks `group_id` against `ctx.dal.groups.list_my_groups()`.
* **Risk & Recommended Contract Adjustment:**
  * Relying on every plugin author to hand-roll `ensure_caller_can_access_group` (and to remember *not* to use `list_by_group` as an authorization check after `1C`) is error-prone.
  * **Recommendation:** Add a dedicated capability helper on `ctx.dal.groups`:
    * `ctx.dal.groups.assert_caller_assigned(group_id, opts)` (or `ctx.dal.groups.can_mutate_group(group_id, opts)`): returns `true` if the caller's effective scope is `:all` or if `ctx.caller.staff_id` has an active temporal assignment in `group_staff_assignments` for `group_id`; otherwise raises a standard `GROUP_NOT_IN_OWN_GROUPS` error.

### 2C. Standardizing `403 Forbidden` & Structured Domain Error Codes Across the Host RPC Gateway
* **What Happened:**
  * When a Classroom Teacher (`:own_groups`) invokes `get_cross_set_attainment_rollup` (which requires `dal.groups.read:all`), the Host blocks the call before Lua execution with `"caller lacks required DAL scopes"`. However, [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L192-L203) previously gated `http.StatusForbidden` (`403`) behind a plugin-ID allowlist, requiring us to add `uk.org.optimis.assessment.markbook` to that condition.
  * Additionally, when a Lua endpoint throws `error("GROUP_NOT_IN_OWN_GROUPS: ...")`, the Host wraps the error in `{"error_code": "PLUGIN_RPC_FAILED", "message": "..."}` with HTTP `400`.
* **Recommended Contract Adjustment:**
  1. Remove the plugin-ID check in [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L192-L203) (or unify with existing test expectations) so any `"caller lacks required DAL scopes"` failure consistently returns `403 Forbidden` with `error_code: "INSUFFICIENT_DAL_SCOPES"`.
  2. Support structured failure returns from Lua endpoints—e.g. `return { ok = false, error_code = "GROUP_NOT_IN_OWN_GROUPS", error = "caller is not assigned to group" }`—so the Host RPC gateway can surface the plugin's exact `error_code` directly in the JSON response instead of collapsing all Lua errors into `PLUGIN_RPC_FAILED`.

### 2D. Prepared Batch Execution (`ctx.plugin_db.exec_batch`) for Multi-Pupil Markbook Saves
* **What Happened:**
  * Saving grades for a class in `save_pupil_grades` iterates over `args.grades` in Lua and issues an `INSERT` or `UPDATE` per pupil via `ctx.plugin_db.exec(...)`. Because the entire `mutates: true` RPC runs inside a single SQLite `BEGIN IMMEDIATE` transaction on `WriteDB`, saving 30 pupils is fast (~1ms) and atomic.
  * However, if a Head of Department imports mock exam results for a full year group (e.g. 240 pupils $\times$ 3 papers = 720 grade cells), executing 720 individual `ctx.plugin_db.query` + `ctx.plugin_db.exec` calls across the Go-Lua boundary adds avoidable statement-preparation overhead.
* **Recommended Contract Adjustment:**
  * Add `ctx.plugin_db.exec_batch(sql, batch_params_table)` (which prepares `sql` once inside the active `writeTx` and executes it for each parameter row) and encourage `INSERT INTO ... ON CONFLICT(column_id, pupil_id) DO UPDATE SET ...` upserts in plugin migrations/queries.

---

## 3. Summary Verdict Table

| Surface | Verdict | Notes |
| :--- | :--- | :--- |
| **`manifest.json` Scopes (`:own_groups` vs `:all`)** | **Fits Well** | Cleanly separated Teacher set entry from HoD cross-set rollup; `1B` scope subsumption worked out-of-the-box. |
| **`ctx.school` MAT Policy Metadata** | **Adjusted / Expand** | Added `mat_id` & `mat_grading_schema_version` to `ctx.school`; consider exposing full MAT scale token definitions next. |
| **`:own_groups` Guard on `ctx.plugin_db` Writes** | **Needs Helper** | Add `ctx.dal.groups.assert_caller_assigned(group_id)` so plugins don't hand-roll `:own_groups` vs `:all` checks after `1C`. |
| **`gdpr_hooks` (`export_pupil_data` / `purge_pupil_data`)** | **Fits Well** | `1G` auto-binding and `redactable_fields: ["teacher_comment"]` integrated seamlessly with the Host SAR Redaction UI. |
| **Sandboxed UI Bridge (`mis-bridge.js` & `onInvalidated`)** | **Fits Well** | Topbar Persona Switcher (`TEACHER` $\rightarrow$ `HOD`) and `window.mis.rpc` / `data-mis-rpc` worked without friction. |
