# Development & Build Orchestration Scripts (`scripts`)

This directory contains workspace automation scripts for local development.

## Key Files

* [`dev_watch.mjs`](file:///home/kieren/Documents/code/optimis/scripts/dev_watch.mjs): Full-stack hot-reload orchestrator invoked via `npm run dev`, `npm run dev:all-plugins`, and `npm run dev:no-plugins`. Runs `npm run build` initially, spawns `vite build --watch` for the React Host UI Shell (`ui/`), forwards startup CLI flags to `./bin/mis-server`, and watches `cmd/` and `internal/` for `.go` source modifications to automatically recompile and restart `./bin/mis-server`.
* [`manage_plugins.mjs`](file:///home/kieren/Documents/code/optimis/scripts/manage_plugins.mjs): Runtime plugin management helper invoked via `npm run plugins:all`, `npm run plugins:none`, `npm run plugins:load -- <folder...>`, and `npm run plugins:status` to hot-install, activate, remove, or inspect plugins on a running local Host server via `/api/admin/plugins/*`.
* [`tail_logs.mjs`](file:///home/kieren/Documents/code/optimis/scripts/tail_logs.mjs): Real-time local log tailer invoked via `npm run logs:tail` and `npm run logs:tail:file`, streaming color-coded Host service logs (`/api/test/logs`), PII-scrubbed Lua diagnostic ring buffer entries (`/api/admin/diagnostics/bundle`), and `<DATA_DIR_PATH>/mis-server.log` with optional `--new-only`, `--file`, and `--filter <pattern>` flags.
