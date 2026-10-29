# Repository Rules

* Do not run git commands. Never touch git.
* NO FALLBACKS / FAIL FAST: Never introduce silent default fallbacks for missing template files, prompt configurations, required environment variables, or authenticated states. If any required file or parameter is missing, throw an explicit error immediately.
* Always write an E2E test first before implementing the code.
* Always use `npm` commands (e.g., `npm test -- <test-name> local 1`, `npm run test:go -- [step-or-test-name]`) to run tests rather than invoking `node scripts/...` or `npx playwright` directly.
* Always execute tests outside of the sandbox (`BypassSandbox: true`), as Playwright E2E and integration tests require network access to local emulators, microservices, and browser binaries.
* If you create or modify an `npm` script/command in `package.json`, you MUST document it under `## Available NPM Commands` in `.agents/AGENTS.md`.
* Always create or update a `GEMINI.md` in any new or heavily updated folders to describe the files and architecture inside it, in addition to updating `<project-root>/GEMINI.md`.

## Available NPM Commands

### Root Workspace (`package.json`)
* `npm run dev`: Build and run the Go Host server and React UI in hot-reload watch mode (`node scripts/dev_watch.mjs`).
* `npm run dev:all-plugins`: Build and run in hot-reload watch mode with all production plugins in `PLUGINS_DIR` preloaded (`PRELOAD_PLUGINS=all node scripts/dev_watch.mjs --all-plugins`).
* `npm run dev:no-plugins`: Build and run in hot-reload watch mode with zero plugins preloaded (`PRELOAD_PLUGINS=none node scripts/dev_watch.mjs --no-plugins`).
* `npm start`: Start the compiled Go Host server (`./bin/mis-server`).
* `npm run start:all-plugins`: Start the compiled Go Host server with all production plugins preloaded (`PRELOAD_PLUGINS=all ./bin/mis-server --all-plugins`).
* `npm run start:no-plugins`: Start the compiled Go Host server with zero plugins preloaded (`PRELOAD_PLUGINS=none ./bin/mis-server --no-plugins`).
* `npm run build`: Build the embedded React UI (`build:ui`) and compile the Go Host binary (`build:go`).
* `npm run build:ui`: Build the Vite + React Host UI Shell into `internal/host/ui_dist`.
* `npm run build:go`: Compile the Go Host server binary into `bin/mis-server`.
* `npm run seed`: Build and populate the 45-table synthetic UK All-Through school QA database (`go run ./cmd/mis-seed-qa`). Does not read from `.env`; interactively prompts the user for the target database path if none is passed (defaulting to `.data/core.db`), or accepts `-- -db <path>` / `-- <path>` / `-- --overwrite-dev-db`.
* `npm run seed:qa`: Build and populate the synthetic QA database non-interactively at `.data/qa_core.db` (`go run ./cmd/mis-seed-qa -db .data/qa_core.db`).
* `npm run seed:dev`: Build and populate the synthetic QA database directly into `.data/core.db` (`go run ./cmd/mis-seed-qa --overwrite-dev-db`).
* `npm run plugins:all`: Hot-install and activate all production plugins in `PLUGINS_DIR` against the running local server (`node scripts/manage_plugins.mjs all`).
* `npm run plugins:none`: Remove all active plugins from the running local server (`node scripts/manage_plugins.mjs none`).
* `npm run plugins:load -- <folder...>`: Hot-install and activate specific plugin folder(s) on the running local server (`node scripts/manage_plugins.mjs load`).
* `npm run plugins:status`: List currently installed plugins and their lifecycle states on the running local server (`node scripts/manage_plugins.mjs status`).
* `npm run logs:tail`: Stream live Host service logs (`/api/test/logs`) and PII-scrubbed Lua diagnostics (`/api/admin/diagnostics/bundle`) in real time (`node scripts/tail_logs.mjs`).
* `npm run logs:tail:file`: Tail the local `<DATA_DIR_PATH>/mis-server.log` file directly (`node scripts/tail_logs.mjs --file`).
* `npm test -- <test-name> local 1`: Run a single local test sequentially.
* `npm run test:go -- [step-or-test-name]`: Run Go Host / Core DAL unit and integration tests.
* `npm run test:e2e`: Run Playwright E2E test suite.
* `npm run test:emulator`: Run local emulator integration tests.
* `npm run test:live`: Run live integration tests (only when explicitly requested).

