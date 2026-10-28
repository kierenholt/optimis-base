# Test Plugin Sandboxed Micro-Frontend UI (`plugins/test_plugin/ui`)

This directory contains [`dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/ui/dist/index.html), the offline-bundled micro-frontend for `uk.org.optimis.test` served by [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L142-L177) inside `<iframe sandbox="allow-scripts">` with a strict `connect-src 'none'` Content Security Policy.

---

## Declarative `data-mis-rpc` Bridge & Selectors

1. **Handshake (`mis:ready` -> `mis:init`)**:
   * On load, posts `{ type: 'mis:ready' }` to the parent React Host Shell ([`ui/src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx)).
   * Receives `mis:init` containing `context` (`pupil_id`, `group_id`, `other_group_id`, `reference_date`, `academic_year_id`, `staff_id`) and populates `data-mis-args` attributes on declarative action buttons.
2. **Declarative RPC Buttons (`data-testid` / `data-mis-rpc`)**:
   * `print-record-btn` (`data-mis-rpc="get_pupil_record"`): Fetches the core pupil record and prints `[plugin:uk.org.optimis.test] ...` to the console.
   * `save-plugin-note-btn` (`data-mis-rpc="save_pupil_note"`): Persists a pupil note to `plugin_test.db` via `WriteDB` and updates `data-testid="plugin-note-status"`.
   * `load-plugin-notes-btn` (`data-mis-rpc="list_pupil_notes"`): Queries `plugin_test.db` via `ReadDB` and logs `[plugin:uk.org.optimis.test:plugin_db] ...` with `max_page_count`.
   * `load-own-group-roster-btn`, `load-other-group-roster-btn`, `load-pupil-guardians-btn`, `record-am-attendance-btn`, and `submit-safeguarding-postbox-btn`: Trigger `:own_groups` Core DAL domain endpoints and render results or `data-testid="rpc-error-banner"`.
