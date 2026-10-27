---
name: test-author
description: >-
  Comprehensive workflow and standards for authoring unit, integration, Lua plugin,
  and Playwright E2E tests in the Optimis repository. Use this skill whenever the
  user asks to write new tests, update existing test suites, or add test coverage.
---

# Test Author Skill

Follow this workflow and set of strict guardrails whenever authoring or updating tests in this repository.

---

## 1. Core Guardrails & Non-Negotiable Rules

1. **Always Write an E2E Test First**:
   For any feature, user story, or bugfix, **always author a Playwright E2E test first** before any production code is implemented. Additional Go DAL or Lua unit tests may be authored alongside the E2E test, but the E2E test is mandatory as the primary specification.
2. **Test Naming Convention**:
   All test functions/blocks MUST follow the pattern:
   `methodname_initialstate_expectedbehaviour_test()`
   *(e.g., `exportPupilData_withBehaviourRecords_returnsStructuredSections_test`, `TestResolveSessions_MidYearTransfer_ReturnsActiveAMPM_test`)*.
3. **NEVER Touch Production Code (Strict Read-Only Boundary)**:
   You may **only** create or modify test files and test-only fixtures/helpers. **Never** create, edit, or delete production source files (including adding `data-testid` attributes, fixing bugs, or altering schemas/configs). If a test requires a missing `data-testid` selector or exposes a production issue, define the expected `data-testid` in the E2E test and report the contract for `code-implementer` to build.
4. **Do NOT Run the Tests**:
   Your role as `test-author` is strictly to design and write the test code and fixtures. **Do not execute test commands yourself** unless explicitly instructed to do so in the prompt.
5. **Fail Fast / Zero Fallbacks**:
   Never introduce silent default fallbacks for missing template files, prompt configurations, environment variables, or authenticated states in tests or test helpers. Throw an explicit error immediately if any required fixture, env var, or parameter is missing.
6. **Never Use the Browser Tool to Inspect Layout**:
   Do **not** open a browser tool to investigate page layout before writing E2E tests. Inspect the React/UI source code directly (or include a Playwright `await page.content()` dump step when diagnosing unknown DOM structures).
7. **Selector Standard**:
   Assert existence and interact with page elements using `data-testid` attributes wherever possible (`page.getByTestId(...)`), and `data-mis-rpc` / `data-mis-args` when verifying plugin iframe bindings.
8. **Always Tap Service Logs and Browser Console Logs**:
   Every authored test MUST actively tap and capture logs from both the running services and the browser console:
   - **Browser Console & Page Errors**: Attach `page.on('console', ...)` and `page.on('pageerror', ...)` listeners in every Playwright test `beforeEach` to record all browser console output and uncaught page exceptions, surfacing them immediately in test output and asserting zero unexpected errors.
   - **Service Logs**: Tap the backend/host service logs (e.g., tailing the local service log file/stream or querying the Go Host diagnostic log buffer / test log endpoint before and after each test) so server-side stack traces, SQLite lock warnings, and RPC errors are captured alongside test assertions.
9. **Sequential Design & Self-Contained Test Data Lifecycle**:
   - Design all tests to run **sequentially** (never rely on parallel execution state).
   - If a test relies on mock or seeded data, create/insert it explicitly during test setup (`beforeEach` / `t.Cleanup`) and **clean it up completely at the end** of the test so no orphaned state leaks across runs.
   - **Never** run `git` commands.

---

## 2. Step-by-Step Test Authoring Workflow

### Step 1: Investigate & Plan
1. Read the target requirements, user stories, or existing implementation files to understand inputs, state transitions, error conditions, and DAL scope requirements.
2. For UI/E2E tests, inspect the component source code for existing `data-testid` attributes and `data-mis-rpc` bindings (or define the target `data-testid` contract for new UI elements).
3. Identify the required test cases using the `methodname_initialstate_expectedbehaviour_test` naming scheme:
   - Happy path with valid state and scopes.
   - Boundary / temporal edge cases (`effective_from`, `effective_to`, `academic_year_id`, AM/PM statutory session resolution).
   - Security / RBAC / Safeguarding invariants (e.g., `:own_groups` vs `:all`, PR/court-order blocks, DSL vault access).
   - Fail-fast error paths (missing required arguments, unauthorized scopes, undeclared manifest methods/domains).

### Step 2: Author the Tests (Playwright E2E First)

#### A. Playwright E2E & UI Tests (Mandatory First Step)
- Always author the Playwright E2E test first so the end-to-end user journey and DOM/RPC contracts are locked in before implementation begins.
- Configure test blocks to run **sequentially** (`test.describe.configure({ mode: 'serial' })`).
- **Tap Browser & Service Logs in Setup/Teardown**:
  - In `beforeEach`, attach `page.on('console', ...)` and `page.on('pageerror', ...)` and start tapping the local backend service logs (via log file offset or diagnostic log endpoint).
  - In `afterEach`, flush/print captured browser console logs and service logs if the test failed, and assert no unexpected server errors or uncaught browser errors occurred.
- Seed required test fixtures in `beforeEach` and purge them in `afterEach` / `afterAll`.
- Prefer `page.getByTestId('...')` for all element assertions and interactions.
- When authoring tests for Sandboxed Plugin UIs (`<iframe sandbox="allow-scripts">`):
  - Verify the `mis:init` handshake payload and `mis:rpc:request` / `mis:rpc:response` round-trip over `window.postMessage`.
  - Assert that `connect-src 'none'` CSP blocks direct external network requests from inside the iframe.

#### B. Go Host & Core DAL Tests
- Write tests targeting isolated SQLite databases (`WriteDB` with `MaxOpenConns(1)` and `ReadDB` with WAL mode enabled).
- Tap the Go Host structured logger / diagnostic ring buffer in the test harness so all service-level log entries emitted during the test are captured and inspectable via `t.Log`.
- Verify Core DAL invariants:
  - Temporal queries respect `effective_from`, `effective_to`, and `academic_year_id`.
  - Guardian lookups strictly enforce Parental Responsibility (PR) and court-order safeguarding blocks.
  - Interactive RPC calls enforce user scopes; worker calls enforce plugin service principal scopes.
  - Outbound `ctx.http` calls fail fast if invoked inside an open SQLite write transaction or against an undeclared domain.

#### C. Lua Plugin & UK GDPR Compliance Tests
- Ensure every exported function in `src/init.lua` declared in `manifest.json` is tested for `{ ok = true, data = ... }` and explicit error handling.
- For plugins with `database.alias`:
  - Write tests covering both `export_pupil_data(ctx, pupil_id)` and `purge_pupil_data(ctx, pupil_id)`.
  - Verify `export_pupil_data` returns structured section tables (`title`, `records`, `redactable_fields`).
  - Verify `purge_pupil_data` removes 100% of rows tied to `pupil_id` across all `plugin_<name>.db` tables, leaving zero orphaned records.

---

## 3. Verification Checklist Before Completing
- [ ] **A Playwright E2E test was written first before any code implementation.**
- [ ] **Zero production code files were modified.**
- [ ] **No test commands were executed.**
- [ ] **Tests tap both service logs and browser console logs (`console` & `pageerror`).**
- [ ] All test names match `methodname_initialstate_expectedbehaviour_test()`.
- [ ] All E2E suites are configured for sequential execution (`mode: 'serial'`).
- [ ] All DOM assertions use `data-testid` wherever possible.
- [ ] Mock/seeded data is created in setup and cleaned up in teardown.
- [ ] No silent fallbacks were introduced in test helpers or fixtures.

---

## 4. Reference Templates

See [test_templates.md](./examples/test_templates.md) for ready-to-use boilerplate patterns for:
- Playwright E2E tests (serial execution, browser console & service log tapping, `data-testid` selectors, setup/teardown, and HTML dump debugging)
- Go Host & Core DAL tests (isolated SQLite WAL DB, service log/ring-buffer tapping, and PR/court-order safeguarding assertions)
- Lua Plugin GDPR `export_pupil_data` & `purge_pupil_data` verification tests

