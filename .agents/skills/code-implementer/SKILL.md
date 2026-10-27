---
name: code-implementer
description: >-
  Test-driven implementation workflow and strict architectural guardrails for writing
  and modifying production code (Go Host, Core DAL, Lua plugins, and React UI) in the
  Optimis repository. Use this skill whenever implementing features, fixing bugs, or
  modifying production code against pre-written tests.
---

# Code Implementer Skill

Your primary mission is to **write, build/compile, and iterate on production code until the pre-written test passes**, strictly adhering to the guardrails below.

---

## 1. Core Guardrails & Non-Negotiable Rules

1. **MANDATORY Pre-Written E2E Test Gate (Do Not Proceed Without an E2E Test)**:
   - An E2E test must **always** be written first before implementing any production code.
   - Before writing or editing any production code, verify that a pre-written Playwright E2E test exists for the target behavior (following the `methodname_initialstate_expectedbehaviour_test()` convention and tapping both service and browser console logs).
   - **If no pre-written E2E test exists, STOP IMMEDIATELY.** Do not write production code. Inform the user (or delegate to `test-author` if instructed) so the E2E test is authored first.
   - Do **not** modify or weaken the pre-written test to make your implementation pass without explicit permission from the user.
2. **Build & Iterate Production Code Until the Test Passes**:
   - Write and modify the necessary production code (Go Host, Core DAL, SQL migrations, Lua plugins, or React UI) to satisfy all assertions in the pre-written E2E test (and any accompanying unit/DAL tests).
   - Always compile/build the modified production artifacts (e.g., `npm run build` when editing `/functions` or UI bundles, or Go compilation) before re-running the test.
   - Iterate on the production implementation and re-run the single test sequentially until it passes completely.
3. **Never Assume the Fault When Tests Fail — Verify via Secondary Means**:
   - When a test fails, **do not automatically assume** where the fault lies (whether in your implementation, the test itself, or local state).
   - Actively isolate and verify the root cause using **secondary verification tools** before jumping to conclusions:
     - Probe HTTP/RPC endpoints directly with `curl` to inspect raw status codes, headers, and response payloads.
     - Query local SQLite state (`sqlite3`) or inspect tapped service/browser logs to verify data persistence and migrations.
     - Write and run temporary **scratch tests or diagnostic scripts** (and clean them up afterwards) to isolate specific functions, Lua VM calls, or DOM states.
4. **NEVER Run Local Frontend or Backend Services**:
   - Always assume local services are already running and proceed immediately to building and executing the relevant local test.
   - If a test or secondary probe (`curl`) confirms that local frontend/backend processes, databases, or emulators are not running, **NEVER start them yourself**. Stop immediately and ask the user to start the services.
5. **Fail Fast / Zero Fallbacks & No Mock Data in Production**:
   - Never introduce silent default fallbacks for missing template files, prompt configurations, required environment variables, or authenticated states.
   - If a mandatory parameter is missing from an endpoint, state provider, or database lookup, throw an explicit error immediately so issues are caught at the boundary.
   - Never add mock/dummy data or mock account names (e.g., `@example.com`) into production code.
   - Store ports, addresses, and configuration in `.env` files (referring to `example.env`), keeping sensitive credentials strictly in `.env`.
6. **Sequential Local Test Verification via `npm` Only**:
   - Run **only LOCAL tests**, **one at a time, sequentially**, and **only tests relevant to your changes**.
   - Always use the workspace `npm` scripts (`npm test -- <test-name> local 1`, `npm run test:go -- <step-or-test-name>`, `npm run test:e2e`, `npm run test:emulator`) with `BypassSandbox: true`.
   - **Never** invoke `node scripts/run_tests.js` or `npx playwright` directly.
   - Every ten commands, check for and kill any leftover background tasks.
7. **Tooling & Documentation Boundaries**:
   - **Never** run `git` commands.
   - **Never** use the `gcloud` CLI or deploy anything directly (ask the user to deploy, or use MCP servers for cloud queries).
   - Preserve all existing comments and docstrings unrelated to your changes.
   - If you create a new `npm` command in `package.json`, **always document it** under `## Available NPM Commands` in [`.agents/AGENTS.md`](../../AGENTS.md).
   - Always update `<project-root>/GEMINI.md` after completing architectural changes or adding important files.

---

## 2. Step-by-Step Implementation Workflow

### Step 1: Verify Pre-Written E2E Test Exists
1. Locate the pre-written E2E test file and specific `methodname_initialstate_expectedbehaviour_test()` covering the requested change.
2. **Gate Check**:
   - If the pre-written E2E test is missing -> **Abort implementation immediately** and notify the user that a pre-written E2E test is required before `code-implementer` can proceed.
   - If the E2E test exists -> Read the test assertions, expected `data-testid` selectors, RPC method contracts, and error conditions carefully.

### Step 2: Run the Pre-Written Test (Red Phase)
1. Execute the single relevant local E2E test sequentially with `BypassSandbox: true`:
   - `npm test -- <test-name> local 1`
2. **Service Check**:
   - If the test output indicates connection refused / local services are down (verified via `curl` if ambiguous) -> **STOP and ask the user to start local services.** Never launch servers yourself.
   - Confirm the test fails for the expected unimplemented reason.

### Step 3: Write & Build Production Code to Pass the Test

Implement the production code required by the test while enforcing [GEMINI.md](../../GEMINI.md) architectural invariants:

#### A. Go Host Container & Core DAL (`core.db`)
- **Read/Write Pool Separation**: Route mutations (`mutates: true`) exclusively through `WriteDB` (`MaxOpenConns(1)`) and read queries through `ReadDB` (`MaxOpenConns(NumCPU)`).
- **No Open Write Locks During HTTP Egress**: Ensure any open SQLite write transaction is committed or halted before executing outbound `ctx.http` requests.
- **Temporal & Safeguarding Invariants**:
  - Enforce effective-dated filtering (`effective_from`, `effective_to`, `academic_year_id`) and resolve phases down to statutory `AM`/`PM` sessions.
  - Enforce Parental Responsibility (PR) and court-order blocks on all guardian/family queries.
  - Enforce method-level RBAC scopes (`:own_groups` vs `:all`) and envelope encryption for `dal.safeguarding.read`.
- **PII-Scrubbed Diagnostics**: Never log pupil IDs, local variables, or RPC payloads into the diagnostic ring buffer.

#### B. Lua Plugins & Manifest (`manifest.json`, `src/init.lua`, `migrations/*.sql`)
- **Default-Deny Contract**: Declare every endpoint, worker, DAL scope, and allowed HTTPS egress domain in `manifest.json`.
- **Expand-and-Contract Migrations**: Minor version SQL migrations (`migrations/*.sql`) must be non-destructive (no dropping or renaming columns).
- **UK GDPR Hooks**: Any plugin attaching `plugin_<name>.db` must implement `export_pupil_data(ctx, pupil_id)` and `purge_pupil_data(ctx, pupil_id)` leaving zero orphaned rows.

#### C. React Host UI Shell, Sandboxed Plugin UIs & Functions
- **Testability**: Implement the exact `data-testid` attributes and DOM behavior asserted by the pre-written E2E test.
- **Sandboxed Iframe Bridge**: Render plugin UIs inside `<iframe sandbox="allow-scripts">` with `connect-src 'none'` CSP, binding actions via `data-mis-rpc="<method_name>"` and `data-mis-args='<json>'`.
- **Build Artifacts**: Run required build steps (e.g., `npm run build` inside `/functions` or UI build scripts) so the compiled code is up to date before testing.

### Step 4: Verify, Diagnose via Secondary Means & Iterate Until Green
1. Re-run the single relevant local E2E test sequentially via `npm` (`BypassSandbox: true`).
2. **If the test does not pass, do not guess or assume the fault**:
   - Inspect the tapped browser console logs and backend service logs captured by the test.
   - Use secondary tools (`curl` requests against the endpoint, `sqlite3` queries, or temporary scratch scripts/tests) to verify whether the issue is in the backend handler, database state, UI rendering, or the test itself.
   - Clean up any temporary scratch scripts/files once the root cause is verified.
   - Refine the production code, rebuild, and re-run the test sequentially until it passes. (Only if secondary verification proves a bug in the pre-written test itself, present the evidence and ask the user for permission before touching the test).
3. If any new `npm` commands were added, document them in [`.agents/AGENTS.md`](../../AGENTS.md). If architectural changes or important files were added, update [GEMINI.md](../../GEMINI.md).

---

## 3. Verification Checklist Before Completing
- [ ] **Confirmed a pre-written E2E test existed before writing any production code.**
- [ ] **Built/compiled the production code and iterated until the pre-written E2E test passed.**
- [ ] **Used secondary verification (`curl`, `sqlite3`, scratch tests) to confirm root causes when debugging failures rather than assuming.**
- [ ] **Cleaned up any temporary scratch tests or diagnostic files.**
- [ ] **Never started local frontend/backend services** (asked user if services were down).
- [ ] All relevant pre-written local tests pass when run sequentially via `npm` with `BypassSandbox: true`.
- [ ] Zero silent fallbacks, default env vars, or `@example.com` mock data in production code.
- [ ] Pre-written tests were not altered without explicit user permission.
- [ ] Documented any newly created `npm` commands in [`.agents/AGENTS.md`](../../AGENTS.md) and updated [GEMINI.md](../../GEMINI.md) if architectural changes or important files were introduced.
