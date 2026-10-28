# End-to-End Playwright Verification Suites (`tests/e2e`)

This directory contains the sequential Playwright E2E test suites verifying the Optimis Host UI Shell, Sandboxed `<iframe sandbox="allow-scripts">` Plugin Bridge, Core DAL domain scoping, and Backend Session/Persona Resolution.

---

## 1. Non-Negotiable E2E Test Standards

* **Sequential Execution**: Every spec file configures `test.describe.configure({ mode: 'serial' })` and runs with `--workers=1` via `npm test -- <test-name> local 1`.
* **Test Naming Convention**: Every test follows `methodname_initialstate_expectedbehaviour_test`.
* **Dual Log Tapping**: Every `beforeEach` / `afterEach` taps both:
  1. Browser console (`page.on('console')`) and uncaught page exceptions (`page.on('pageerror')`).
  2. Backend Go Host service logs (`GET /api/test/logs/cursor` and `GET /api/test/logs?since=...`).
* **Self-Contained Fixture Lifecycle**: Every test seeds its own isolated SQLite records in `beforeEach` and deletes them in `afterEach` so zero test state leaks across runs.

---

## 2. E2E Spec Files

* [`backend_session_persona_resolution.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/backend_session_persona_resolution.spec.ts):
  * Verifies `GET /api/auth/personas`, `POST /api/auth/persona` (`SameSite=Strict` `mis_session` cookie issuance), and `GET /api/auth/session` against seeded `ADMIN`, `DSL`, `TEACHER`, and terminated staff in `core.db`.
  * Verifies that switching to a persona without required endpoint scopes causes `POST /api/plugins/:id/rpc/:method` to fail fast (`caller lacks required DAL scopes`).
  * Verifies that setting `ENABLE_DEV_PERSONA_SWITCHER=false` fails fast with `REAL_SSO_NOT_CONFIGURED` when loading real SSO (`/api/auth/sso/login`) and blocks dev persona endpoints with `403 DEV_PERSONA_SWITCHER_DISABLED`.
* [`milestone1_core_dal_scoping.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/milestone1_core_dal_scoping.spec.ts):
  * Verifies coarse-grained Core DAL methods (`get_group_roster_summary`, `get_pupil_guardians`, `record_group_am_attendance`, `submit_safeguarding_postbox`), `:own_groups` temporal isolation, court-order/confidential refuge address redaction, and write-only safeguarding postbox submission.
* [`plugin_attached_db.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/plugin_attached_db.spec.ts):
  * Verifies per-plugin attached SQLite persistence (`plugin_test.db`), `PRAGMA max_page_count` enforcement, and UK GDPR `export_pupil_data` / `purge_pupil_data` hooks.
* [`core_schema_functional.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/core_schema_functional.spec.ts):
  * Verifies end-to-end 45-table Core Schema invariants through the Host UI Shell and sandboxed plugin iframe.
* [`mvp_plugin.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/mvp_plugin.spec.ts):
  * Verifies the baseline `mis:init` and `mis:rpc:request` / `mis:rpc:response` `window.postMessage` bridge between [`App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx) and `uk.org.optimis.test`.
