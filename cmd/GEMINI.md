# Go Command Line & Server Binaries (`cmd`)

This directory contains the Go `main` package entrypoints for the Optimis platform and the Section 10A Plugin Developer Toolchain.

## Implemented Binaries

* [`cmd/mis-server`](file:///home/kieren/Documents/code/optimis/cmd/mis-server/GEMINI.md): Main Go Host Container binary (`./bin/mis-server`) enforcing fail-fast bootstrap validation via single `DATABASE_PATH` (with `--db` flag support), loading all plugins from `PLUGINS_DIR` (or 0 plugins if blank in `.env`) with optional CLI overrides (`--no-plugins` / `--plugins`), providing the Section 7C Local Break-Glass CLI subcommands (`./bin/mis-server config list`, `config get <key>`, `config set <key> <value>`, and `config repair-sso`), and serving the embedded React Host UI Shell, Lua RPC gateway, and Core DAL.
* [`cmd/mis-seed-qa`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/GEMINI.md): Deterministic synthetic UK All-Through school QA database generator (`npm run seed` with interactive path prompt or positional argument) populating all 45 core tables without reading from `.env`.

## Section 10A SDK Toolchain Binaries (Locked & Confirmed)

* `cmd/mis-dev-server`: Standalone local host + auto-seeded synthetic school DB + hot-reload mounting for plugin `src/`, `migrations/`, and `ui/`, emitting single-line structured `stdout` logs (10D) and supporting persona switching via CLI flag, `?__persona=` URL param, or HTTP header.
* `cmd/mis-test`: Headless Lua test runner executing plugin test suites against an isolated synthetic school SQLite fixture with structured single-line `stdout` logs (`rpc`, `lua_err`, `sqlite_write_hold_ms`, `event_emit`, file/line stack traces).
* `cmd/mis-verify`: CI contract & UK GDPR gatekeeper enforcing manifest/Lua/UI `data-mis-rpc` cross-checks, offline CSP linting (`connect-src 'none'`), Expand-and-Contract SQL migration rules, and headless GDPR `export_pupil_data` & `purge_pupil_data` zero-orphan verification.
* `cmd/mis-pack`: Deterministic `.mispkg` archive bundler and Lucide SVG icon extractor.
