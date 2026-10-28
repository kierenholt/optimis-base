# Plugins Workspace (`plugins`)

This directory contains unpacked plugin bundles mounted by the Go Host Container during local development and automated test verification (`PLUGINS_DIR`).

## Subdirectories

* [`plugins/test_plugin`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/GEMINI.md): Reference synthetic fixture plugin (`uk.org.optimis.test`) exercising the full Plugin Contract (Section 9):
  * [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/manifest.json): Default-deny routing, `database.alias` (`plugin_test`), `gdpr_hooks`, `egress.allowed_domains` & `egress.timeout_ms`, and scoped endpoints.
  * [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/GEMINI.md): Sandboxed Lua 5.1 endpoint and GDPR hook implementations (`ctx.dal`, `ctx.plugin_db`, `ctx.events`, `ctx.http`).
  * [`migrations/`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/migrations/GEMINI.md): Non-destructive SQLite schema migrations (`001_init.sql`).
  * [`ui/`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/ui/GEMINI.md): Offline-bundled micro-frontend (`ui/dist/index.html`) running inside `<iframe sandbox="allow-scripts">` with `connect-src 'none'` CSP.
