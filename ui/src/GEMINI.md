# React Host UI Shell Source (`ui/src`)

This directory contains the TypeScript + React source files for the embedded Optimis Host UI Shell (Sections 5B, 5C, 6C, 7B, 9.4 & 9.5).

## Key Files

* [`App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx):
  * **Default Dashboard Layout & Theme Engine (`--mis-*`)**: Renders the Left Navigation Sidebar (`data-testid="host-sidebar"`), Top Workspace Bar (`data-testid="host-topbar"` with `data-testid="theme-mode-toggle-btn"` and `data-testid="active-theme-mode-indicator"`), and Responsive Default Dashboard Grid (`data-testid="host-dashboard-grid"` & `data-testid="dashboard-summary-strip"`). Applies `--mis-*` Light and WCAG AAA High-Contrast CSS Custom Properties to `:root` and propagates live theme updates to child plugin iframes.
  * **Sandboxed Plugin Bridge & Entity Deep-Linking (`window.postMessage`)**: Unconditionally responds to `mis:ready` with `mis:init` (`theme_mode`, `theme_tokens`, `context`, `permitted_methods`), routes `mis:rpc:request` calls to `POST /api/plugins/:plugin_id/rpc/:method`, and handles `mis:link:navigate` entity deep-links (`student`, `group`, `staff`) by waking the designated handler plugin and updating its context.
  * **Host Compliance Consoles**: Renders the DSL Envelope-Encrypted Safeguarding Vault (`data-testid="dsl-safeguarding-vault-panel"`) and Admin UK GDPR SAR Redaction & Retention Purge UI (`data-testid="sar-redaction-panel"`).
* [`main.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/main.tsx): Fail-fast React DOM entrypoint mounting `<App />` into `#root`.
