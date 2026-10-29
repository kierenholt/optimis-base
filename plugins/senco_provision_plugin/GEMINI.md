# SENCO & Classroom Differentiation Plugin (`plugins/senco_provision_plugin`)

This directory contains the first-party SENCO & Classroom Differentiation Lua plugin (`uk.org.optimis.sen.provision`), enabling SENCOs (`dal.sen.read:all`) to manage Individual Education Plan (IEP) targets, classroom differentiation adjustments, and Teaching Assistant (TA) support allocations for `K` (SEN Support) and `E` (EHCP) pupils in `plugin_sen_provision.db`, while surfacing a Classroom SEN Differentiation Mini-Widget (`ui/dist/widget.html`) on the Home Dashboard for Classroom Teachers (`dal.sen.read:own_groups`) with `data-mis-link="student"` entity deep-linking into the Keep-Alive LRU Pool.

---

## 1. Files & Contract Surface

* [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/senco_provision_plugin/manifest.json):
  * **Plugin ID**: `uk.org.optimis.sen.provision` (`version: "0.1.0"`, `core_dal_version: "~1.0"`).
  * **Attached SQLite Database**: `database.alias: "plugin_sen_provision"` (`max_db_size_mb: 64`).
  * **UK GDPR Hooks (`1G` Auto-Bound)**: `export_pupil_data` & `purge_pupil_data`.
  * **Interactive Endpoints**:
    * `log_iep_provision`: `scopes: ["dal.pupils.read:all", "dal.sen.read:all"]`, `mutates: true`.
    * `get_sen_differentiation_summary`: `scopes: ["dal.pupils.read:own_groups", "dal.sen.read:own_groups", "dal.groups.read:own_groups"]`, `mutates: false`.
  * **Home Dashboard Mini-Iframe Widget**:
    * `id: "sen_differentiation"`, `title: "Classroom SEN Differentiation"`, `entry: "ui/dist/widget.html"`.
* [`migrations/001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/senco_provision_plugin/migrations/001_init.sql):
  * Provisions `sen_provisions` (`id`, `pupil_id`, `academic_year_id`, `sen_provision_code` `CHECK (sen_provision_code IN ('K', 'E'))`, `primary_sen_need_code`, `iep_target`, `classroom_adjustment`, `ta_allocation_hours`, `assigned_ta_name`, `review_notes`, `logged_by_staff_id`, `effective_date`, `created_at`) and index `idx_sen_provisions_pupil_year`.
* [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/senco_provision_plugin/src/init.lua):
  * `log_iep_provision(ctx, args)`: Verifies pupil enrolment and active `K` or `E` SEN provision via `ctx.dal.pupils.get_by_id` and `ctx.dal.pupils.list_sen_and_medical_batch` (failing fast if `sen_provision_code` is `'N'`), generates UUID v4 primary key (`iep_<uuid>`) via `ctx.crypto.uuid()` and UTC timestamp via `ctx.clock.now_utc()`, inserts into `plugin_sen_provision.sen_provisions`, emits `sen.provision_logged` post-commit, and returns the updated pupil & class SEN differentiation rollup.
  * `get_sen_differentiation_summary(ctx, args)`: Hydrates individual pupils (`pupil_id`), class rosters (`group_id` via `ctx.dal.sen.list_register`), whole-school/scoped SEN registers (when `pupil_id` and `group_id` are omitted via `ctx.dal.sen.list_register`, `1G`), or batch pupil lists (`pupil_ids` via `ctx.dal.pupils.get_by_ids_batch`) with Core SEN & Medical summaries (`ehcp_start_date`, `ehcp_review_due_date`, `next_annual_review_date`, `secondary_sen_need_code`, `sen_needs`, `sen_assistants`, `medical_conditions`) and logged IEP targets/TA allocations from `plugin_sen_provision.sen_provisions`.
  * `export_pupil_data(ctx, args)`: Returns structured UK GDPR SAR section `"Plugin SEN IEP & TA Provision Records"` with `redactable_fields = { "iep_target", "classroom_adjustment", "review_notes" }`.
  * `purge_pupil_data(ctx, args)`: Deletes all `sen_provisions` rows for `pupil_id`, leaving zero orphaned records.
* [`ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/senco_provision_plugin/ui/dist/index.html):
  * Sandboxed `<iframe sandbox="allow-scripts">` SENCO & Teacher workspace with `connect-src 'none'` CSP, pupil selector (`sen-pupil-select`), Core SEN badge & teaching strategies display, and declarative `data-mis-rpc="log_iep_provision"` binding.
* [`ui/dist/widget.html`](file:///home/kieren/Documents/code/optimis/plugins/senco_provision_plugin/ui/dist/widget.html):
  * Home Dashboard Mini-Iframe Widget (`sen_differentiation`) displaying all `K` (SEN Support) and `E` (EHCP) pupils in the classroom, their primary SEN need badges, logged IEP classroom adjustments, and TA support allocations (`hrs/wk` + assigned TA), live-refreshing on `window.mis.onInvalidated` and deep-linking each pupil badge via `<a data-mis-link="student" data-mis-plugin="self" data-pupil-id="...">` (`1H`) to directly wake `uk.org.optimis.sen.provision` in the Keep-Alive LRU Pool for that pupil.
