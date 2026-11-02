# Core DAL & Host Engine E2E Verification Suites (`tests/e2e/core`)

This directory contains sequential Playwright E2E integration suites for the Optimis Host Core SQLite schema, Data Access Layer (DAL) method scoping, asynchronous Event Bus, and outbound HTTP Egress engine.

---

## E2E Spec Files

* [`core_schema_functional.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/core/core_schema_functional.spec.ts):
  * Verifies end-to-end 46-table Core Schema invariants through the Host UI Shell and sandboxed plugin iframe.
* [`milestone1_core_dal_scoping.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/core/milestone1_core_dal_scoping.spec.ts):
  * Verifies coarse-grained Core DAL methods (`get_group_roster_summary`, `get_pupil_guardians`, `record_group_am_attendance`, `submit_safeguarding_postbox`), `:own_groups` temporal isolation, court-order/confidential refuge address redaction, and write-only safeguarding postbox submission.
* [`event_bus_worker_subscriptions.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/core/event_bus_worker_subscriptions.spec.ts):
  * Verifies the Native In-Memory Event Bus Router, Manifest Worker Subscriptions, and Standard Core Domain Event Catalog (`GET /api/events/catalog`, Sections 3, 9.1 & 10.2): post-commit emission of `core.attendance.session_recorded` and `core.safeguarding.postbox_created` (envelope metadata only, zero plaintext leak), asynchronous `SYSTEM` worker execution (`on_core_event_audit`) into `plugin_test.db`, and recursive worker re-emission (`on_loop_relay`) enforcing `Hop <= 3` and dropping `Hop = 4` (`events_dropped_total`).
* [`http_egress_engine.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/core/http_egress_engine.spec.ts):
  * Verifies the `ctx.http` (`ctx.http.get` & `ctx.http.post`) Egress Engine (Sections 4, 9.2 & 10.1): approved HTTPS domain requests with custom headers/body, `CheckRedirect` following approved domain redirects vs rejecting redirects to unapproved domains (`egress redirect target domain not permitted by manifest.json`), prefix-spoofed domain rejection, `egress.timeout_ms` deadline enforcement (`EGRESS_TIMEOUT`), and open-write-transaction halt/rollback (`disallowed inside an open SQLite write transaction`).
