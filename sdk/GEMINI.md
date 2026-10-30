# Optimis Plugin SDK & Toolchain (`sdk`)

This directory houses the official client-side SDK assets, Lua type stubs, JSON schemas, and workspace templates provided by the Optimis Base Platform for sandboxed `.mispkg` plugins (Sections 6B, 6C, 8A–8C, 9.4, 9.5 & 10.4 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md)).

## Locked & Confirmed SDK Architecture (Sections 10A–10D)

* **Toolchain Binaries (10A):** Separate purpose-built Go binaries in [`cmd/`](file:///home/kieren/Documents/code/optimis/cmd/GEMINI.md): `mis-dev-server` (local host + synthetic school DB + hot-reload), `mis-test` (headless Lua test runner), `mis-verify` (CI contract & GDPR gatekeeper), and `mis-pack` (Lucide SVG extractor + `.mispkg` bundler).
* **VS Code Integration (10B):** Zero-config `.vscode/settings.json` pre-binding the VS Code Lua extension (`Lua 5.1`, sandboxed globals) to `.mis-types/dal.d.lua` and binding `manifest.json` to `manifest.schema.json` for instant IDE autocomplete and inline linting.
* **Frontend Bridge (10C):** Untyped JSON RPC + Declarative Tags via lightweight `@ukmis/bridge` (auto-injecting `--mis-*` CSS variables and binding `data-mis-rpc` / `data-mis-link` attributes + `useMisRpc` hook) and `@ukmis/tailwind-preset` with zero schema/codegen bloat.
* **Observability in Dev (10D):** Agent-first single-line structured `stdout` logs (`rpc`, `lua_err`, `sqlite_write_hold_ms`, `event_emit`, file/line stack traces) from `mis-dev-server` and `mis-test`, with persona switching controlled via CLI flag, URL param (`?__persona=`), or HTTP header.

## Subdirectories

* [`lua/`](file:///home/kieren/Documents/code/optimis/sdk/lua/GEMINI.md):
  * [`ctx_types.lua`](file:///home/kieren/Documents/code/optimis/sdk/lua/ctx_types.lua): Authoritative EmmyLua / LuaLS static type annotations (`.mis-types/dal.d.lua` source) for the injected `ctx` runtime (`ctx.caller`, `ctx.school`, `ctx.dal.*`, `ctx.plugin_db`, `ctx.crypto`, `ctx.clock`, `ctx.json`, `ctx.http`, `ctx.events`, and `ctx.xml`).
* [`ui/`](file:///home/kieren/Documents/code/optimis/sdk/ui/GEMINI.md):
  * [`mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js): Official sandboxed `<iframe sandbox="allow-scripts">` UI runtime (`window.mis.rpc()` Promise API, declarative `[data-mis-rpc]` delegation, declarative `[data-mis-link]` Entity Deep-Link emitter, and `mis:init` / `mis:theme:update` `--mis-*` CSS Custom Property synchronization).
  * [`mis-theme.css`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-theme.css): Canonical `--mis-*` Design Token stylesheet supporting Default School Branding (`light`) and WCAG AAA Classroom Projector High-Contrast Mode (`high-contrast`).
