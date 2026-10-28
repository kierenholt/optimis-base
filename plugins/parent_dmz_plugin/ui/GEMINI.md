# Parent Portal DMZ Sync Sandboxed UI (`plugins/parent_dmz_plugin/ui`)

This directory contains the offline-bundled micro-frontend for `uk.org.optimis.parent.dmz`, rendered inside `<iframe sandbox="allow-scripts">` with a strict `connect-src 'none'` Content Security Policy.

## Files

* [`dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/ui/dist/index.html):
  * Uses `/sdk/mis-theme.css` (`--mis-*` design tokens) and `/sdk/mis-bridge.js` (`window.mis.onInit` and declarative `data-mis-rpc` attributes).
  * Exposes `data-testid="pull-dmz-inbox-btn"` (`data-mis-rpc="pull_dmz_inbox"`) and `data-testid="push-dmz-projection-btn"` (`data-mis-rpc="push_sanitized_projection"`).
  * Renders accepted vs safeguarding-blocked submission badges (`dmz-submission-status-<guardian_person_id>`), sanitized read-projection summary cards (`dmz-projection-card`, `dmz-projection-upn-badge`, `dmz-projection-redacted-addresses`), JSON payload preview (`dmz-projection-json-preview`), and fail-fast error banners (`dmz-error-banner`).
