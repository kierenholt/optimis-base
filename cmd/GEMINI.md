# Go Command Line & Server Binaries (`cmd`)

This directory contains the Go `main` package entrypoints for the Optimis platform.

## Subdirectories

* [`cmd/mis-server`](file:///home/kieren/Documents/code/optimis/cmd/mis-server/GEMINI.md): Main Go Host Container binary (`./bin/mis-server`) enforcing fail-fast bootstrap validation via single `DATABASE_PATH` (with `--db` flag support), supporting `PRELOAD_PLUGINS` (single folder, parent directory, comma-separated folder list, `all`, or `none`) and `--all-plugins` / `--no-plugins` / `--plugins` flags, providing the Section 7C Local Break-Glass CLI subcommands (`./bin/mis-server config list`, `config get <key>`, `config set <key> <value>`, and `config repair-sso`), and serving the embedded React Host UI Shell, Lua RPC gateway, and Core DAL.
* [`cmd/mis-seed-qa`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/GEMINI.md): Deterministic synthetic UK All-Through school QA database generator (`npm run seed` with interactive path prompt, `npm run seed:qa` for `.data/qa_core.db`, or `npm run seed:dev` / `--overwrite-dev-db` for `.data/core.db`) populating all 45 core tables without reading from `.env`.
