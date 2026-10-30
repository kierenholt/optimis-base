# Optimis Plugin SDK & Toolchain (`sdk`)

This directory houses the official client-side SDK assets, Lua type stubs, JSON schemas, and workspace packages provided by the Optimis Base Platform for sandboxed `.mispkg` plugins (Sections 6B, 6C, 8A–8C, 9.4, 9.5 & 10.4 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md)).

## Locked & Confirmed SDK Architecture (Sections 10A–10D)

* **Toolchain Binaries (10A):** Separate purpose-built Go binaries in [`cmd/`](file:///home/kieren/Documents/code/optimis/cmd/GEMINI.md) backed by [`internal/sdk`](file:///home/kieren/Documents/code/optimis/internal/sdk/GEMINI.md): [`cmd/mis-dev-server`](file:///home/kieren/Documents/code/optimis/cmd/mis-dev-server/main.go) (local host + synthetic school DB + hot-reload), [`cmd/mis-test`](file:///home/kieren/Documents/code/optimis/cmd/mis-test/main.go) (headless Lua test runner), [`cmd/mis-verify`](file:///home/kieren/Documents/code/optimis/cmd/mis-verify/main.go) (CI contract & GDPR gatekeeper), and [`cmd/mis-pack`](file:///home/kieren/Documents/code/optimis/cmd/mis-pack/main.go) (Lucide SVG extractor + `.mispkg` bundler).
* **VS Code Integration (10B):** Zero-config [`.vscode/settings.json`](file:///home/kieren/Documents/code/optimis/.vscode/settings.json) pre-binding the VS Code Lua extension (`Lua 5.1`, sandboxed globals) to [`.mis-types/dal.d.lua`](file:///home/kieren/Documents/code/optimis/.mis-types/dal.d.lua) and binding `manifest.json` to [`manifest.schema.json`](file:///home/kieren/Documents/code/optimis/sdk/manifest.schema.json) for instant IDE autocomplete and inline linting.
* **Frontend Bridge (10C):** Untyped JSON RPC + Declarative Tags via lightweight `@ukmis/bridge` ([`packages/bridge/`](file:///home/kieren/Documents/code/optimis/sdk/packages/bridge/GEMINI.md), auto-injecting `--mis-*` CSS variables and binding `data-mis-rpc` / `data-mis-link` attributes + `useMisRpc` hook) and `@ukmis/tailwind-preset` ([`packages/tailwind-preset/`](file:///home/kieren/Documents/code/optimis/sdk/packages/tailwind-preset/GEMINI.md)) with zero schema/codegen bloat.
* **Observability in Dev (10D):** Agent-first single-line structured `stdout` logs (`type=rpc`, `type=lua_err`, `sqlite_write_hold_ms=`, `type=event_emit`, file/line stack traces) from `mis-dev-server` and `mis-test`, with persona switching controlled via CLI flag (`--persona`), URL param (`?__persona=`), or HTTP header (`X-MIS-Persona` / `X-MIS-Staff-ID`).

## Files & Subdirectories

* [`manifest.schema.json`](file:///home/kieren/Documents/code/optimis/sdk/manifest.schema.json): JSON Schema Draft-07 specification for `manifest.json` (Section 10B).
* [`lua/`](file:///home/kieren/Documents/code/optimis/sdk/lua/GEMINI.md):
  * [`ctx_types.lua`](file:///home/kieren/Documents/code/optimis/sdk/lua/ctx_types.lua): Authoritative EmmyLua / LuaLS static type annotations (synced with [`.mis-types/dal.d.lua`](file:///home/kieren/Documents/code/optimis/.mis-types/dal.d.lua)) for the injected `ctx` runtime (`ctx.caller`, `ctx.school`, `ctx.dal.*`, `ctx.plugin_db`, `ctx.crypto`, `ctx.clock`, `ctx.json`, `ctx.http`, `ctx.events`, and `ctx.xml`).
* [`ui/`](file:///home/kieren/Documents/code/optimis/sdk/ui/GEMINI.md):
  * [`mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js): Official sandboxed `<iframe sandbox="allow-scripts">` UI runtime (`window.mis.rpc()` Promise API, `window.mis.useMisRpc()` hook, `window.mis.onPersonaChanged()`, declarative `[data-mis-rpc]` delegation, declarative `[data-mis-link]` Entity Deep-Link emitter, and `mis:init` / `mis:theme:update` `--mis-*` CSS Custom Property synchronization).
  * [`mis-theme.css`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-theme.css): Canonical `--mis-*` Design Token stylesheet supporting Default School Branding (`light`) and WCAG AAA Classroom Projector High-Contrast Mode (`high-contrast`).
* [`packages/`](file:///home/kieren/Documents/code/optimis/sdk/packages/GEMINI.md):
  * [`packages/bridge/`](file:///home/kieren/Documents/code/optimis/sdk/packages/bridge/GEMINI.md): `@ukmis/bridge` ESM/CJS package (`rpc`, `useMisRpc`, `onPersonaChanged`, `getContext`).
  * [`packages/tailwind-preset/`](file:///home/kieren/Documents/code/optimis/sdk/packages/tailwind-preset/GEMINI.md): `@ukmis/tailwind-preset` Tailwind CSS preset mapping `--mis-*` tokens.
