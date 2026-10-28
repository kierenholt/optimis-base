# Test Plugin Sandboxed Micro-Frontend UI (`plugins/test_plugin/ui`)

This directory contains [`dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/ui/dist/index.html), the offline-bundled micro-frontend for `uk.org.optimis.test` served by [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go) inside `<iframe sandbox="allow-scripts">` with a strict `connect-src 'none'` Content Security Policy.

---

## Official SDK Integration (`mis-bridge.js` & `mis-theme.css`)

1. **Theme & Initialization (`window.mis.onInit`)**:
   * Loads [`/sdk/mis-theme.css`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-theme.css) and [`/sdk/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js).
   * Receives `mis:init` and `mis:theme:update` events via `window.mis.onInit(snapshot)`, rendering active theme state in `data-testid="sdk-theme-status"` (`THEME:light` / `THEME:high-contrast` + `--mis-brand-primary`) and active entity context in `data-testid="sdk-context-status"`.
2. **Programmatic Promise RPC API (`window.mis.rpc`)**:
   * `sdk-promise-rpc-btn`: Calls `window.mis.rpc('get_pupil_record', { pupil_id })` and writes `PROMISE_OK:<name>:<upn>` to `data-testid="sdk-promise-rpc-result"`.
   * `sdk-promise-rpc-error-btn`: Verifies fail-fast Promise rejection (`PROMISE_ERR:<error_code>`).
3. **Declarative Entity Deep-Linking (`data-mis-link`)**:
   * `deep-link-student-anchor` (`data-mis-link="student"`) and `deep-link-group-anchor` (`data-mis-link="group"`): Emit `mis:link:navigate` to the parent Host Shell via `mis-bridge.js`.
4. **Declarative RPC Buttons (`data-testid` / `data-mis-rpc`)**:
   * `print-record-btn` (`data-mis-rpc="get_pupil_record"`), `save-plugin-note-btn` (`data-mis-rpc="save_pupil_note"`), `load-plugin-notes-btn` (`data-mis-rpc="list_pupil_notes"`), `load-own-group-roster-btn`, `load-other-group-roster-btn`, `load-pupil-guardians-btn`, `record-am-attendance-btn`, `submit-safeguarding-postbox-btn`, `egress-sync-get-btn` (`data-mis-rpc="sync_external_sparx_get"`), and `egress-sync-post-btn` (`data-mis-rpc="push_external_sparx_post"`), rendering egress payloads into `data-testid="egress-sync-result"`.
5. **CSP Verification (`connect-src 'none'`)**:
   * `sdk-csp-fetch-test-btn`: Attempts a direct external `fetch()` inside the sandboxed iframe and records `CSP_BLOCKED` in `data-testid="sdk-csp-fetch-status"`.

