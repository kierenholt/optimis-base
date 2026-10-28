# Timetable & Statutory Session Attendance Plugin (`plugins/timetable_attendance_plugin`)

This directory contains **Vertical 1 (`uk.org.optimis.timetable.attendance`)**, a pure Core DAL Lua plugin (`0` attached SQLite databases) that implements the teacher rolling timetable lookup, statutory `AM`/`PM` session resolution, class roster view with batch SEN/Medical alerts, and batch statutory session attendance submission under `:own_groups` RBAC scoping.

## Key Files

* [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/timetable_attendance_plugin/manifest.json): Declares `id: "uk.org.optimis.timetable.attendance"`, zero attached databases (omits `database` and `gdpr_hooks`), zero outbound egress domains, and three interactive RPC endpoints scoped strictly to `:own_groups`:
  * `load_staff_timetable` (`dal.groups.read:own_groups`, `mutates: false`)
  * `load_slot_roster_with_flags` (`dal.groups.read:own_groups`, `dal.pupils.read:own_groups`, `dal.attendance.read:own_groups`, `mutates: false`)
  * `submit_session_attendance_batch` (`dal.groups.read:own_groups`, `dal.pupils.read:own_groups`, `dal.attendance.read:own_groups`, `dal.attendance.write:own_groups`, `mutates: true`)
* [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/timetable_attendance_plugin/src/init.lua): Implements the vertical Lua workflow:
  * `load_staff_timetable(ctx, args)`: Queries [`ctx.dal.groups.get_staff_timetable_slots`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L247-L287) for the caller's `staff_id`, `cycle_week`, and `day_of_week_id`, and resolves the statutory `AM`/`PM` session from `maps_to_statutory_session`.
  * `load_slot_roster_with_flags(ctx, args)`: Queries [`ctx.dal.pupils.list_by_group`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L113-L140), enriches active pupils via [`ctx.dal.pupils.list_sen_and_medical_batch`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L142-L195) (`E`/`K` SEN provision & primary need codes plus IHCP medical conditions), and attaches any existing statutory session marks via [`ctx.dal.attendance.get_group_session`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L353-L390).
  * `submit_session_attendance_batch(ctx, args)`: Validates statutory session (`AM` or `PM`), persists batch attendance marks via [`ctx.dal.attendance.upsert_session_batch`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L395-L501), and reads back persisted session marks.
* [`ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/timetable_attendance_plugin/ui/dist/index.html): Sandboxed micro-frontend (`connect-src 'none'`) using `/sdk/mis-theme.css` and `/sdk/mis-bridge.js` declarative `data-mis-rpc` tags (`load-timetable-btn`, `load-roster-btn`, `load-other-group-roster-btn`, `submit-attendance-batch-btn`).
