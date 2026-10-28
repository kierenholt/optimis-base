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
* `npm start`: Start the compiled Go Host server (`./bin/mis-server`).
* `npm run build`: Build the embedded React UI (`build:ui`) and compile the Go Host binary (`build:go`).
* `npm run build:ui`: Build the Vite + React Host UI Shell into `internal/host/ui_dist`.
* `npm run build:go`: Compile the Go Host server binary into `bin/mis-server`.
* `npm run seed:qa`: Build and populate a comprehensive 45-table synthetic UK All-Through school QA database (`<DATA_DIR_PATH>/qa_core.db` by default, or `--overwrite-dev-db` / `--db <path>`) with ~1,108 pupils across 42 registration forms, 95 staff, 889 households, a 2-week rolling timetable, and 20 days of statutory `AM`/`PM` & lesson attendance (`go run ./cmd/mis-seed-qa`).
* `npm test -- <test-name> local 1`: Run a single local test sequentially.
* `npm run test:go -- [step-or-test-name]`: Run Go Host / Core DAL unit and integration tests.
* `npm run test:e2e`: Run Playwright E2E test suite.
* `npm run test:emulator`: Run local emulator integration tests.
* `npm run test:live`: Run live integration tests (only when explicitly requested).

