# Behaviour & Pastoral Points Plugin (`plugins/behaviour_plugin`)

This directory contains the first-party Behaviour & Pastoral Points Lua plugin (`uk.org.optimis.behaviour`), verifying per-plugin SQLite storage (`plugin_behaviour.db`), `:own_groups` temporal pupil scoping & `:all` hierarchical scope subsumption (`dal.pupils.read:own_groups`), coarse-grained batch pupil hydration (`ctx.dal.pupils.get_by_ids_batch`), host UUID & UTC clock helpers (`ctx.crypto.uuid()`, `ctx.clock.now_utc()`, `ctx.school`), the Home Dashboard Mini-Iframe Widget Grid (`widgets: [{ "id": "behaviour_summary", "entry": "ui/dist/widget.html" }]`) with automatic `window.mis.onInvalidated` live refresh and declarative `data-mis-link="student"` entity deep-linking, and auto-bound UK GDPR SAR export & redaction (`redactable_fields: ["incident_notes"]`) + zero-orphan retention purge.

---

## 1. Files & Contract Surface

* [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/manifest.json):
  * **Plugin ID**: `uk.org.optimis.behaviour` (`version: "0.2.0"`, `core_dal_version: "~1.0"`).
  * **Attached SQLite Database**: `database.alias: "plugin_behaviour"` (`max_db_size_mb: 64`).
  * **UK GDPR Hooks (`1G` Auto-Bound)**: `export_pupil_data` & `purge_pupil_data` (declared solely in `gdpr_hooks` without duplicating in `endpoints`).
  * **Interactive Endpoints**:
    * `log_behaviour_event`: `scopes: ["dal.pupils.read:own_groups"]`, `mutates: true`.
    * `get_pupil_behaviour_summary`: `scopes: ["dal.pupils.read:own_groups"]`, `mutates: false`.
  * **Home Dashboard Mini-Iframe Widget**:
    * `id: "behaviour_summary"`, `title: "Behaviour & Merits Summary"`, `entry: "ui/dist/widget.html"`.
* [`migrations/001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/migrations/001_init.sql):
  * Provisions `behaviour_events` (`id`, `pupil_id`, `academic_year_id`, `event_date`, `event_type` `MERIT|INCIDENT`, `category`, `points`, `incident_notes`, `logged_by_staff_id`, `created_at`) and indexes on `pupil_id` and `event_date`.
* [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/src/init.lua):
  * `log_behaviour_event(ctx, args)`: Verifies pupil enrolment (and enforces `:own_groups` when `strict_own_groups = true`, with `:all` subsumption for HoD/Admin/DSL callers via `1B`/`1C`), generates `event_id` via `ctx.crypto.uuid()` and `created_at` via `ctx.clock.now_utc()` (`1F`), inserts the merit/incident into `plugin_behaviour.behaviour_events`, emits `behaviour.event_logged` post-commit, and returns updated merit/demerit/net point rollups with `preferred_first_name`, `preferred_last_name`, and `nc_year_group` (`1E`).
  * `get_pupil_behaviour_summary(ctx, args)`: Supports single-pupil lookup via `ctx.dal.pupils.get_by_id(args.pupil_id, opts)` and batch lookup via `ctx.dal.pupils.get_by_ids_batch(args.pupil_ids, opts)` (`1E`), computing live `total_merits`, `total_demerits`, `net_points`, and ordered event history.
  * `export_pupil_data(ctx, args)`: Returns structured UK GDPR SAR section `"Plugin Behaviour & Incident Records"` with `redactable_fields = { "incident_notes" }` so third-party pupil names in free-text incident notes can be redacted in the Admin SAR Redaction UI.
  * `purge_pupil_data(ctx, args)`: Deletes all `behaviour_events` rows for `pupil_id`, leaving zero orphaned records.
* [`ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/ui/dist/index.html):
  * Sandboxed `<iframe sandbox="allow-scripts">` workspace with `connect-src 'none'` CSP and declarative `data-mis-rpc="log_behaviour_event"` binding (relying on the Host UI Shell's `X-MIS-Staff-ID` propagation rather than passing `staff_id` in RPC `args`, `1H`).
* [`ui/dist/widget.html`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/ui/dist/widget.html):
  * Home Dashboard Mini-Iframe Widget (`behaviour_summary`) displaying live merit/demerit/net points via `window.mis.rpc('get_pupil_behaviour_summary', ...)`, automatically re-fetching on `window.mis.onInvalidated(...)` when `log_behaviour_event` mutates state (`1H`), and deep-linking to the designated student handler plugin in the Keep-Alive LRU Pool via `<a data-mis-link="student" data-pupil-id="...">`.
