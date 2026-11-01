# Widget Gallery & Layout Bench Plugin (`plugins/widget_gallery_plugin`)

This directory contains the **Widget Gallery & Layout Bench** test plugin (`uk.org.optimis.widget.gallery`), designed to test and showcase a rich variety of Home Dashboard mini-iframe widgets across all four strict preset 2D grid footprints (`1x1`, `2x1`, `1x2`, and `2x2`).

## 1. Plugin Contract & Architecture

* **Manifest ([`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/manifest.json))**:
  * **ID**: `uk.org.optimis.widget.gallery`
  * **Title**: `"Widget Gallery & Layout Bench"`
  * **Icon**: `"layout-grid"` (monochrome Lucide icon)
  * **Stateless Core DAL Design**: Operates purely against `core.db` via `ctx.dal.*` without requiring an attached `plugin_*.db` SQLite file.
  * **Interactive RPC Endpoints (`endpoints`)**:
    * `get_widget_gallery_summary` (`mutates: false`, scopes: `dal.pupils.read:own_groups`, `dal.groups.read:own_groups`, `dal.sen.read:own_groups`, `dal.medical.read:own_groups`, `dal.attendance.read:own_groups`): Verifies `:own_groups` assignment via `ctx.dal.groups.assert_caller_assigned`, resolves group metadata via `ctx.dal.groups.list_for_staff`, fetches the active temporal roster via `ctx.dal.pupils.list_by_group`, and aggregates SEN (`E`/`K`) and Medical (`IHCP`) alert counts via `ctx.dal.pupils.list_sen_and_medical_batch`.
    * `get_pupil_quick_card` (`mutates: false`, scopes: `dal.pupils.read:own_groups`): Fetches a single pupil summary via `ctx.dal.pupils.get_by_id` enforcing `:own_groups` isolation.

## 2. Preset-Sized Dashboard Widgets (`widgets[]`)

Exposes **12 mini-iframe widgets** (3 in each preset footprint, each with a mandatory `description`, monochrome Lucide `icon`, and search `keywords` rendered in the centered Widget Picker Modal):

### `1x1` Compact Widgets (`300px` × `180px`)
1. `wg_1x1_attendance_counter` ([`ui/dist/widget_1x1_attendance_counter.html`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/ui/dist/widget_1x1_attendance_counter.html)) — Session Attendance Counter (`icon: "clipboard-check"`).
2. `wg_1x1_sen_badge` ([`ui/dist/widget_1x1_sen_badge.html`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/ui/dist/widget_1x1_sen_badge.html)) — SEN & Medical Quick Badge (`icon: "heart-pulse"`).
3. `wg_1x1_group_size_stat` ([`ui/dist/widget_1x1_group_size_stat.html`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/ui/dist/widget_1x1_group_size_stat.html)) — Active Group Size Stat (`icon: "users"`).

### `2x1` Wide Banner Widgets (`2 columns` × `1 row`)
4. `wg_2x1_attendance_pulse_bar` ([`ui/dist/widget_2x1_attendance_pulse_bar.html`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/ui/dist/widget_2x1_attendance_pulse_bar.html)) — AM/PM Attendance Pulse Bar (`icon: "activity"`).
5. `wg_2x1_pupil_quick_lookup` ([`ui/dist/widget_2x1_pupil_quick_lookup.html`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/ui/dist/widget_2x1_pupil_quick_lookup.html)) — Pupil Quick Lookup & Deep-Link (`icon: "search"`).
6. `wg_2x1_next_lesson_banner` ([`ui/dist/widget_2x1_next_lesson_banner.html`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/ui/dist/widget_2x1_next_lesson_banner.html)) — Next Timetable Lesson Banner (`icon: "calendar"`).

### `1x2` Tall Vertical List Widgets (`1 column` × `2 rows`)
7. `wg_1x2_roster_ticker` ([`ui/dist/widget_1x2_roster_ticker.html`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/ui/dist/widget_1x2_roster_ticker.html)) — Class Roster Vertical Ticker (`icon: "user-check"`).
8. `wg_1x2_sen_medical_watch_list` ([`ui/dist/widget_1x2_sen_medical_watch_list.html`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/ui/dist/widget_1x2_sen_medical_watch_list.html)) — SEN & IHCP Watch List (`icon: "stethoscope"`).
9. `wg_1x2_period_timeline` ([`ui/dist/widget_1x2_period_timeline.html`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/ui/dist/widget_1x2_period_timeline.html)) — Daily Period Timeline (`icon: "book-open"`).

### `2x2` Large Overview & Interactive Widgets (`2 columns` × `2 rows`)
10. `wg_2x2_class_overview_matrix` ([`ui/dist/widget_2x2_class_overview_matrix.html`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/ui/dist/widget_2x2_class_overview_matrix.html)) — Class Overview & RPC Matrix (`icon: "layout-grid"`).
11. `wg_2x2_attendance_breakdown_grid` ([`ui/dist/widget_2x2_attendance_breakdown_grid.html`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/ui/dist/widget_2x2_attendance_breakdown_grid.html)) — Session Attendance Breakdown Grid (`icon: "file-check"`).
12. `wg_2x2_interactive_sandbox_bench` ([`ui/dist/widget_2x2_interactive_sandbox_bench.html`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/ui/dist/widget_2x2_interactive_sandbox_bench.html)) — Interactive Widget RPC & Link Bench (`icon: "award"`).

## 3. Files

* [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/manifest.json): Plugin contract declaring endpoints, view, and 12 preset-sized widgets.
* [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/src/init.lua): Sandboxed Lua 5.1 implementation of `get_widget_gallery_summary` and `get_pupil_quick_card`.
* [`ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/ui/dist/index.html): Minimal full-page status header view.
* [`tests/widget_gallery_test.lua`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/tests/widget_gallery_test.lua): Headless Lua unit tests executed by `bin/mis-test`.
