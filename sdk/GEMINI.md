# Optimis Plugin SDK (`sdk`)

This directory houses the official client-side SDK assets and design token stylesheets provided by the Optimis Base Platform for sandboxed `.mispkg` plugins (Sections 6B, 6C, 9.4 & 9.5 of the Architecture Blueprint).

## Subdirectories

* [`ui/`](file:///home/kieren/Documents/code/optimis/sdk/ui/GEMINI.md):
  * [`mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js): Official sandboxed `<iframe sandbox="allow-scripts">` UI runtime (`window.mis.rpc()` Promise API, declarative `[data-mis-rpc]` delegation, declarative `[data-mis-link]` Entity Deep-Link emitter, and `mis:init` / `mis:theme:update` `--mis-*` CSS Custom Property synchronization).
  * [`mis-theme.css`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-theme.css): Canonical `--mis-*` Design Token stylesheet supporting Default School Branding (`light`) and WCAG AAA Classroom Projector High-Contrast Mode (`high-contrast`).
