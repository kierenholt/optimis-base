# React Host UI Shell (`ui`)

This directory contains the Vite + React Single-Page Application (SPA) compiled into [`internal/host/ui_dist`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L18-L31) and embedded into the Go Host binary via `embed.FS`.

## Architectural Responsibilities

1. **Sandboxed Plugin `<iframe sandbox="allow-scripts">` Coordinator ([`src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx))**:
   - Mounts plugin micro-frontends inside `<iframe sandbox="allow-scripts">` (without `allow-same-origin`).
   - Listens for `mis:ready` from the child iframe and responds with `mis:init` containing school theme tokens (`--mis-*`), contextual route parameters (`pupil_id`, `group_id`, `other_group_id`, `reference_date`, `academic_year_id`, `staff_id`), and `permitted_methods`.
   - Intercepts `mis:rpc:request` (`{ request_id, method, args }`) over `window.postMessage`, forwards authenticated requests to `POST /api/plugins/:plugin_id/rpc/:method`, and posts `mis:rpc:response` (`{ request_id, ok, data, error_code }`) back to the iframe.
2. **DSL Safeguarding Vault & OAuth Cloud Key Unlock Panel ([`src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx))**:
   - Renders the Host-level DSL Safeguarding Vault (`data-testid="dsl-safeguarding-vault-panel"`) exclusively when the active session holds `dal.safeguarding.read`.
   - Lists envelope-encrypted safeguarding note metadata without exposing plaintext until unlocked with a valid short-lived signed OAuth Cloud Key (`POST /api/safeguarding/cloud-key/renew` and `POST /api/safeguarding/notes/:note_id/decrypt`).
3. **Admin UK GDPR SAR Redaction UI & Statutory Retention Purge Console ([`src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx))**:
   - Renders the integrated Admin SAR Redaction & Retention Purge panel (`data-testid="sar-redaction-panel"`, Sections 5C, 7B & 9.3).
   - Loads cross-Core and cross-Plugin SAR section tables (`POST /api/admin/gdpr/sar/export`), renders interactive redaction inputs and `[REDACTED]` mask buttons for every column declared in `redactable_fields`, finalizes the sanitized JSON export bundle (`POST /api/admin/gdpr/sar/finalize`), and executes statutory retention purges (`POST /api/admin/gdpr/purge`).

## Key Files

* [`src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx): Root Host Shell component managing the `window.postMessage` RPC coordinator, `<iframe sandbox="allow-scripts">` container, DSL Safeguarding Vault UI, and Admin UK GDPR SAR Redaction & Retention Purge UI.
* [`src/main.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/main.tsx): React DOM entrypoint mounting `<App />`.
* [`vite.config.ts`](file:///home/kieren/Documents/code/optimis/ui/vite.config.ts): Vite build configuration outputting static assets into `../internal/host/ui_dist`.
