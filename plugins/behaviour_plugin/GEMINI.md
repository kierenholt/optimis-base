# Behaviour & Pastoral Points Plugin (`plugins/behaviour_plugin`)

This directory contains the first-party Behaviour & Pastoral Points Lua plugin (`uk.org.optimis.behaviour`), verifying per-plugin SQLite storage (`plugin_behaviour.db`), `:own_groups` temporal pupil scoping (`dal.pupils.read:own_groups`), the Home Dashboard Mini-Iframe Widget Grid (`widgets: [{ "id": "behaviour_summary", "entry": "ui/dist/widget.html" }]`) with declarative `data-mis-link="student"` entity deep-linking, and UK GDPR SAR export & redaction (`redactable_fields: ["incident_notes"]`) + zero-orphan retention purge.

---

## 1. Files & Contract Surface

* [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/manifest.json):
  * **Plugin ID**: `uk.org.optimis.behaviour` (`version: "0.1.0"`, `core_dal_version: "~1.0"`).
  * **Attached SQLite Database**: `database.alias: "plugin_behaviour"` (`max_db_size_mb: 64`).
  * **UK GDPR Hooks**: `export_pupil_data` & `purge_pupil_data`.
  * **Interactive Endpoints**:
    * `log_behaviour_event`: `scopes: ["dal.pupils.read:own_groups"]`, `mutates: true`.
    * `get_pupil_behaviour_summary`: `scopes: ["dal.pupils.read:own_groups"]`, `mutates: false`.
    * `export_pupil_data`: `scopes: ["dal.pupils.read:all"]`, `mutates: false`.
    * `purge_pupil_data`: `scopes: ["dal.pupils.read:all"]`, `mutates: true`.
  * **Home Dashboard Mini-Iframe Widget**:
    * `id: "behaviour_summary"`, `title: "Behaviour & Merits Summary"`, `entry: "ui/dist/widget.html"`.
* [`migrations/001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/migrations/001_init.sql):
  * Provisions `behaviour_events` (`id`, `pupil_id`, `academic_year_id`, `event_date`, `event_type` `MERIT|INCIDENT`, `category`, `points`, `incident_notes`, `logged_by_staff_id`, `created_at`) and indexes on `pupil_id` and `event_date`.
* [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/src/init.lua):
  * `log_behaviour_event(ctx, args)`: Verifies the pupil is in the caller's active temporal `:own_groups` teaching/registration group via `ctx.dal.pupils.get_by_id(args.pupil_id, opts)`, inserts the merit/incident into `plugin_behaviour.behaviour_events`, emits `behaviour.event_logged` post-commit, and returns updated merit/demerit/net point rollups.
  * `get_pupil_behaviour_summary(ctx, args)`: Enforces `:own_groups` access via `ctx.dal.pupils.get_by_id` and computes live `total_merits`, `total_demerits`, `net_points`, and ordered event history.
  * `export_pupil_data(ctx, args)`: Returns structured UK GDPR SAR section `"Plugin Behaviour & Incident Records"` with `redactable_fields = { "incident_notes" }` so third-party pupil names in free-text incident notes can be redacted in the Admin SAR Redaction UI.
  * `purge_pupil_data(ctx, args)`: Deletes all `behaviour_events` rows for `pupil_id`, leaving zero orphaned records.
* [`ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/ui/dist/index.html):
  * Sandboxed `<iframe sandbox="allow-scripts">` workspace with `connect-src 'none'` CSP and declarative `data-mis-rpc="log_behaviour_event"` binding.
* [`ui/dist/widget.html`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/ui/dist/widget.html):
  * Home Dashboard Mini-Iframe Widget (`behaviour_summary`) displaying live merit/demerit/net points via `window.mis.rpc('get_pupil_behaviour_summary', ...)` and deep-linking to the designated student handler plugin in the Keep-Alive LRU Pool via `<a data-mis-link="student" data-pupil-id="...">`.
