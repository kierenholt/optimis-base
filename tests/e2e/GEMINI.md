# End-to-End Playwright Verification Suites (`tests/e2e`)

This directory contains the sequential Playwright E2E test suites verifying the Optimis Host UI Shell, Sandboxed `<iframe sandbox="allow-scripts">` Plugin Bridge, Core DAL domain scoping, Backend Session/Persona Resolution, Safeguarding Vault Envelope Encryption, and UK GDPR SAR Redaction & Retention Purge.

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

* [`event_bus_worker_subscriptions.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/event_bus_worker_subscriptions.spec.ts):
  * Verifies the Native In-Memory Event Bus Router, Manifest Worker Subscriptions, and Standard Core Domain Event Catalog (`GET /api/events/catalog`, Sections 3, 9.1 & 10.2): post-commit emission of `core.attendance.session_recorded` and `core.safeguarding.postbox_created` (envelope metadata only, zero plaintext leak), asynchronous `SYSTEM` worker execution (`on_core_event_audit`) into `plugin_test.db`, and recursive worker re-emission (`on_loop_relay`) enforcing `Hop <= 3` and dropping `Hop = 4` (`events_dropped_total`).
* [`http_egress_engine.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/http_egress_engine.spec.ts):
  * Verifies the `ctx.http` (`ctx.http.get` & `ctx.http.post`) Egress Engine (Sections 4, 9.2 & 10.1): approved HTTPS domain requests with custom headers/body, `CheckRedirect` following approved domain redirects vs rejecting redirects to unapproved domains (`egress redirect target domain not permitted by manifest.json`), prefix-spoofed domain rejection, `egress.timeout_ms` deadline enforcement (`EGRESS_TIMEOUT`), and open-write-transaction halt/rollback (`disallowed inside an open SQLite write transaction`).
* [`plugin_ui_sdk_and_theming.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/plugin_ui_sdk_and_theming.spec.ts):
  * Verifies the Official Plugin UI SDK ([`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js)), Design Token & Theming System ([`sdk/ui/mis-theme.css`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-theme.css), `--mis-*` Light & High-Contrast live toggling), Default Dashboard Layout chrome (`host-sidebar`, `host-topbar`, `host-dashboard-grid`, `dashboard-summary-strip`), `window.mis.rpc()` Promise API resolution and fail-fast rejection, declarative `data-mis-link` Entity Deep-Linking (`student`, `group`), `connect-src 'none'` CSP enforcement inside `<iframe sandbox="allow-scripts">`, and caller-scoped `permitted_methods` filtering across `GET /api/system/ui-config` and `mis:init` (`window.mis.permittedMethods`).
* [`gdpr_sar_redaction_and_purge.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/gdpr_sar_redaction_and_purge.spec.ts):
  * Verifies UK GDPR Subject Access Request (SAR) aggregation across `core.db` and attached `plugin_<name>.db` instances (both `ACTIVE` and `UNINSTALLED_DORMANT`), interactive redaction of `redactable_fields` in the React Host Shell Admin SAR Redaction UI (`App.tsx`), KCSIE safeguarding retention lock protection (`KCSIE_RETENTION_LOCK`), and zero-orphan statutory retention purges.
* [`safeguarding_vault_dsl_unlock.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/safeguarding_vault_dsl_unlock.spec.ts):
  * Verifies Section 5B Safeguarding Isolation: write-only postbox creation by Teachers (`dal.safeguarding.create`), 403 rejection of non-DSLs on vault read/key-renewal/decrypt, envelope-encrypted DSL Vault UI unlock via short-lived signed OAuth Cloud Key (`POST /api/safeguarding/cloud-key/renew` & `POST /api/safeguarding/notes/:id/decrypt`), and strict write-only Lua sandbox isolation.
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
