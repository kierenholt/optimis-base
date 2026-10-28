# Development & Build Orchestration Scripts (`scripts`)

This directory contains workspace automation scripts for local development.

## Key Files

* [`dev_watch.mjs`](file:///home/kieren/Documents/code/optimis/scripts/dev_watch.mjs): Full-stack hot-reload orchestrator invoked via `npm run dev`. Runs `npm run build` initially, spawns `vite build --watch` for the React Host UI Shell (`ui/`), and watches `cmd/` and `internal/` for `.go` source modifications to automatically recompile and restart `./bin/mis-server`.
