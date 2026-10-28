# Official Plugin UI SDK & Design Token System (`sdk/ui`)

This directory contains the canonical client-side Plugin UI SDK (`mis-bridge.js`) and CSS Custom Property Design Token stylesheet (`mis-theme.css`) implementing Sections 6B, 6C, 9.4, and 9.5 of the Optimis Architecture Blueprint.

## Key Files & Responsibilities

1. **[`mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js) — Sandboxed Plugin UI SDK (`window.mis`)**:
   - **Initialization & Theme Handshake (`mis:ready` -> `mis:init` / `mis:theme:update`)**:
     - Automatically dispatches `{ type: 'mis:ready' }` to `window.parent` on load.
     - Injects all `--mis-*` CSS Custom Properties onto `document.documentElement` (`:root`) and synchronizes `data-mis-theme="light" | "high-contrast"`.
     - Exposes `window.mis.context`, `window.mis.theme`, `window.mis.themeMode`, `window.mis.permittedMethods`, `window.mis.ready()`, and `window.mis.onInit(callback)`.
   - **Programmatic RPC Promise API (`window.mis.rpc(method, args)`)**:
     - Validates `method` (non-empty string) and `args` (plain JSON object), failing fast by rejecting the returned `Promise` immediately on malformed inputs (`MISSING_RPC_METHOD`, `INVALID_RPC_ARGS`).
     - Posts `{ type: 'mis:rpc:request', request_id, method, args }` to `window.parent` and resolves with `msg.data` (`ok: true`) or rejects with a structured `Error` (`err.code = msg.error_code`).
   - **Declarative DOM RPC Delegation (`[data-mis-rpc]`)**:
     - Intercepts clicks on elements with `data-mis-rpc="<method_name>"` and `data-mis-args='<json>'`, manages `data-mis-loading="true"`, and dispatches bubbling `mis:rpc:success` and `mis:rpc:error` `CustomEvent`s.
   - **Declarative Entity Deep-Linking (`[data-mis-link]`, Section 9.5 / 9C)**:
     - Intercepts clicks on `<a data-mis-link="student|group|staff" ...>`, prevents default iframe navigation, validates the target entity ID (`data-pupil-id`, `data-group-id`, `data-staff-id`, or `data-entity-id`), and emits `{ type: 'mis:link:navigate', entity_type, entity_id, ... }` to the parent Host Shell (also callable programmatically via `window.mis.navigate(entityType, params)`).

2. **[`mis-theme.css`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-theme.css) — CSS Custom Property Design Token Stylesheet (`--mis-*`)**:
   - Defines school branding, surface, typography, border, focus ring, and semantic status tokens (`--mis-brand-primary`, `--mis-bg-canvas`, `--mis-bg-surface`, `--mis-text-primary`, `--mis-border-subtle`, `--mis-focus-ring`, `--mis-status-*`) for both Default Light Mode (`:root, [data-mis-theme="light"]`) and WCAG AAA Classroom High-Contrast Mode (`[data-mis-theme="high-contrast"]`).
