# React Host UI Shell (`ui`)

This directory contains the Vite + React Single-Page Application (SPA) compiled into [`internal/host/ui_dist`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L18-L31) and embedded into the Go Host binary via `embed.FS`.

## Architectural Responsibilities

1. **Sandboxed Plugin `<iframe sandbox="allow-scripts">` Coordinator ([`src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx))**:
   - Mounts plugin micro-frontends inside `<iframe sandbox="allow-scripts">` (without `allow-same-origin`).
   - Listens for `mis:ready` from the child iframe and responds with `mis:init` containing school theme tokens (`--mis-*`), contextual route parameters (`pupil_id`, `group_id`, `other_group_id`, `reference_date`, `academic_year_id`, `staff_id`), and `permitted_methods`.
   - Intercepts `mis:rpc:request` (`{ request_id, method, args }`) over `window.postMessage`, forwards authenticated requests to `POST /api/plugins/:plugin_id/rpc/:method`, and posts `mis:rpc:response` (`{ request_id, ok, data, error_code }`) back to the iframe.

## Key Files

* [`src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx): Root Host Shell component managing the `window.postMessage` RPC coordinator and `<iframe sandbox="allow-scripts">` container.
* [`src/main.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/main.tsx): React DOM entrypoint mounting `<App />`.
* [`vite.config.ts`](file:///home/kieren/Documents/code/optimis/ui/vite.config.ts): Vite build configuration outputting static assets into `../internal/host/ui_dist`.
