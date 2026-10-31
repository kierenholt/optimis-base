# Host UI Shell Views & State Hooks (`ui/src/views`)

This folder contains the primary workspace views and state hooks for the Optimis Host UI Shell.

## Key Files

* [`DashboardSummaryStrip.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/views/DashboardSummaryStrip.tsx): Renders the route validation error banner (`data-testid="host-route-error-banner"`) and the Section 9.5 Default Dashboard Statutory Summary Strip (`data-testid="dashboard-summary-strip"`).
* [`KeepAlivePluginWorkspace.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/views/KeepAlivePluginWorkspace.tsx): Renders the Section 9A Cached Keep-Alive LRU Plugin Workspace Pool (`data-testid="dashboard-workspace-panel"` and `data-testid="keep-alive-lru-pool"`), keeping up to $N = 5$ (`MAX_KEEP_ALIVE_POOL_SIZE`) sandboxed `<iframe sandbox="allow-scripts">` instances mounted (`display: block` when active, `display: none` when inactive).
* [`useHostWorkspaceState.ts`](file:///home/kieren/Documents/code/optimis/ui/src/views/useHostWorkspaceState.ts): React hook managing school branding & `--mis-*` theme state (`GET /api/system/ui-config`), Keep-Alive LRU Pool ordering and eviction (`activatePluginInKeepAlivePool`), opaque `/p/<plugid>` browser history synchronization (`syncBrowserRouteUrl`), and Editable $N \times M$ Widget Grid placement/resizing handlers.
* [`useHostAdminAndDslState.ts`](file:///home/kieren/Documents/code/optimis/ui/src/views/useHostAdminAndDslState.ts): React hook managing Section 8A Dev Persona Switcher session state (`GET /api/auth/personas`, `GET /api/auth/session`, `POST /api/auth/persona`), Section 5B DSL Safeguarding Vault envelope loading & OAuth Cloud Key decryption (`/api/safeguarding/*`), Section 5C Admin/DPO UK GDPR SAR Redaction & Retention Purge (`/api/admin/gdpr/*`), and Section 8 Gap 1 Admin PII-Scrubbed Diagnostic Bundle export (`/api/admin/diagnostics/bundle`).
