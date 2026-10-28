# React Host UI Shell (`ui`)

This directory contains the Vite + React Single-Page Application (SPA) compiled into [`internal/host/ui_dist`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L18-L31) and embedded into the Go Host binary via `embed.FS`.

## Architectural Responsibilities

1. **Default School MIS Dashboard Layout & `--mis-*` Theme Engine ([`src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx))**:
   - Renders the default UK All-Through School MIS workspace chrome: Left Navigation Sidebar (`data-testid="host-sidebar"` with school identity `data-testid="host-school-name"`, academic cycle badge, plugin navigation links, and `data-testid="active-entity-context-pill"`), Top Workspace Header (`data-testid="host-topbar"` with `data-testid="active-theme-mode-indicator"`, `data-testid="theme-mode-toggle-btn"`, and `data-testid="deep-link-navigation-status"`), and the Responsive Default Dashboard Grid (`data-testid="host-dashboard-grid"` & `data-testid="dashboard-summary-strip"`).
   - Fetches school branding, `--mis-*` theme catalogs (`light` & `high-contrast`), active plugin & widget descriptors, and entity link handler mappings from `GET /api/system/ui-config`.
   - Applies `--mis-*` CSS Custom Properties and `data-mis-theme` onto `document.documentElement` (`:root`) and propagates live theme switches into all mounted child plugin and widget iframes without reloading.
2. **Cached Keep-Alive LRU Pool (Section 9A, [`src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx))**:
   - Keeps the last $N = 5$ (`MAX_KEEP_ALIVE_POOL_SIZE`) visited full-page plugin `<iframe sandbox="allow-scripts">` instances mounted inside `data-testid="keep-alive-lru-pool"` (`display: none` when inactive, `display: block` when active) in stable DOM insertion order so tab switches never reload iframe state, and evicts the Least-Recently-Used (`lruPluginOrder[0]`) iframe when a 6th distinct plugin is visited.
3. **Editable $N \times M$ Mini-Iframe Widget Grid (Section 9B, [`src/DashboardWidgetGridPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/DashboardWidgetGridPanel.tsx))**:
   - Renders mini-iframe widgets declared in `manifest.json` (`widgets: [{ id, title, entry }]`) inside a responsive $N \times M$ CSS grid (`data-testid="dashboard-widget-grid"`).
   - Provides **Dashboard Edit Mode** (`data-testid="dashboard-edit-mode-toggle-btn"`) allowing staff to place widgets from the manifest catalog (`data-testid="dashboard-widget-catalog"`), resize column and row spans (`colSpan` $\times$ `rowSpan`), and remove widgets.
4. **Source-Aware Sandboxed `<iframe sandbox="allow-scripts">` RPC & Deep-Link Coordinator ([`src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx))**:
   - Maps `event.source` across all mounted Keep-Alive plugin iframes and mini-iframe widgets back to the owning `pluginId`.
   - Responds to `mis:ready` with `mis:init` containing `theme_mode`, `theme_tokens` (`--mis-*`), contextual route/entity parameters, and caller-scoped `permitted_methods`.
   - Intercepts `mis:link:navigate` emitted by `[data-mis-link]` anchors inside any plugin or widget iframe (Section 9C), wakes the designated handler plugin in the Keep-Alive LRU pool, and broadcasts the updated `mis:init` context.
   - Intercepts `mis:rpc:request` (`{ request_id, method, args }`), forwards authenticated requests to `POST /api/plugins/:plugin_id/rpc/:method`, and posts `mis:rpc:response` back to the originating `event.source` window.
5. **In-UI Topbar Persona Switcher (Section 8A, [`src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx))**:
   - Queries `GET /api/auth/personas` and `GET /api/auth/session` on mount. When `ENABLE_DEV_PERSONA_SWITCHER=true`, renders the interactive Topbar Persona Switcher (`data-testid="persona-switcher-container"`, `data-testid="persona-switcher-select"`, and `data-testid="active-persona-badge"`).
   - Selecting a persona (`Teacher`, `HoD`, `DSL`, `Admin/DPO`) dispatches `POST /api/auth/persona`, refreshes `GET /api/system/ui-config` to update sidebar plugins (`data-testid="nav-item-plugin-methods-count-<id>"`), toggles the DSL Safeguarding Vault and Admin/DPO SAR Redaction consoles live, and broadcasts updated `mis:init` payloads (`permitted_methods`, `scopes`, `active_roles`, `context.staff_id`) to all mounted plugin and widget iframes.
6. **DSL Safeguarding Vault & OAuth Cloud Key Unlock Panel ([`src/SafeguardingVaultPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/SafeguardingVaultPanel.tsx))**:
   - Renders the Host-level DSL Safeguarding Vault (`data-testid="dsl-safeguarding-vault-panel"`) exclusively when the active session holds `dal.safeguarding.read`.
7. **Admin UK GDPR SAR Redaction UI & Statutory Retention Purge Console ([`src/GdprSarPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/GdprSarPanel.tsx))**:
   - Renders the integrated Admin SAR Redaction & Retention Purge panel (`data-testid="sar-redaction-panel"`, Sections 5C, 7B & 9.3) exclusively when the active session holds `ADMIN`/`DPO` role or `dal.staff.write:admin`.
8. **Admin-Exported PII-Scrubbed Plugin Diagnostic Bundle Console ([`src/AdminDiagnosticsPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/AdminDiagnosticsPanel.tsx))**:
   - Renders the Day-2 Operations diagnostic bundle export panel (`data-testid="admin-diagnostics-panel"`, Section 8 Gap 1, Section 9.3 & Section 10.3) with the `"Download Plugin Diagnostic Bundle"` control (`data-testid="download-plugin-diagnostic-bundle-btn"`) exclusively when the active session holds `ADMIN`/`DPO` role or `dal.staff.write:admin`.

## Key Files

* [`src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx): Root Host Shell component managing the Default Dashboard Layout, Topbar Persona Switcher (Section 8A), Keep-Alive LRU Pool ($N = 5$), Editable $N \times M$ Widget Grid state, `--mis-*` theme switcher, source-aware `window.postMessage` RPC & deep-link coordinator, DSL Safeguarding Vault UI, Admin UK GDPR SAR Redaction & Retention Purge UI, and Admin Plugin Diagnostic Bundle export UI.
* [`src/AdminDiagnosticsPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/AdminDiagnosticsPanel.tsx): Admin-Exported PII-Scrubbed Diagnostic Bundle panel (`GET /api/admin/diagnostics/bundle`).
* [`src/DashboardWidgetGridPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/DashboardWidgetGridPanel.tsx): Editable $N \times M$ Mini-Iframe Widget Grid component (Section 9B).
* [`src/main.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/main.tsx): React DOM entrypoint mounting `<App />`.
* [`vite.config.ts`](file:///home/kieren/Documents/code/optimis/ui/vite.config.ts): Vite build configuration outputting static assets into `../internal/host/ui_dist`.
