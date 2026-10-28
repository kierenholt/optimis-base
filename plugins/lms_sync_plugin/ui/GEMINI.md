# Google Classroom LMS Sync Micro-Frontend UI (`plugins/lms_sync_plugin/ui`)

This directory contains [`dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/lms_sync_plugin/ui/dist/index.html), the offline-bundled micro-frontend for `uk.org.optimis.lms.classroom` served inside `<iframe sandbox="allow-scripts">` with a strict `connect-src 'none'` Content Security Policy.

---

## Key UI Elements & Bridge Bindings

* Loads [`/sdk/mis-theme.css`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-theme.css) and [`/sdk/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js).
* `data-testid="transfer-pupil-set-btn"` (`data-mis-rpc="upsert_pupil_membership"`): Transfers the active pupil into the target group/set and displays confirmation in `data-testid="transfer-membership-result"`.
* `data-testid="refresh-lms-sync-logs-btn"` (`data-mis-rpc="list_sync_logs"`): Queries `plugin_lms_sync.lms_sync_logs` and updates `data-testid="lms-sync-status-badge"` (`SYNCED` / `FAILED`) and `data-testid="lms-sync-logs-json"`.
