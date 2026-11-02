# Host Runtime & Integration Test Suites (`internal/host/tests`)

This directory consolidates all unit, integration, security, and sample plugin test suites for the Optimis Go Host Container.

## Test Suite Catalog

### Core Host & Subsystem Tests
* [`mvp_host_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/mvp_host_test.go): Shared [`setupIsolatedTestHost`](file:///home/kieren/Documents/code/optimis/internal/host/tests/mvp_host_test.go#L15) helper, initial boot lifecycle, hot-reload, and basic RPC execution.
* [`plugin_runtime_lifecycle_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/plugin_runtime_lifecycle_test.go): Plugin installation, activation, removal, quarantine states (`ACTIVE`, `UNINSTALLED_DORMANT`), and Admin lifecycle endpoints (`GET /api/admin/plugins`, `POST /api/admin/plugins/install`, `POST /api/admin/plugins/{id}/activate`, `DELETE /api/admin/plugins/{id}`).
* [`plugin_db_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/plugin_db_test.go): SQLite attachment (`plugin_<alias>.db`), snapshot rollbacks on migration failure, and SQL scope validation.
* [`persona_session_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/persona_session_test.go): Dev persona switcher gates (`ENABLE_DEV_PERSONA_SWITCHER`), real SSO OIDC validation, session cookie resolution, and cross-staff authorization checks.
* [`dal_scoping_host_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/dal_scoping_host_test.go): Method-level scoped RBAC (`:own_groups` vs `:all`), caller group assignment assertion, and hierarchical role subsumption.
* [`dal_bindings_host_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/dal_bindings_host_test.go): Host DAL Lua binding execution across production Core DAL domains (academic, dashboard, pupils, sen, medical, guardians, staff, groups, attendance, census).
* [`event_bus_router_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/event_bus_router_test.go): In-memory post-commit event routing, buffered worker pools, loop protection (`Hop > 3`), and worker concurrency limits.
* [`egress_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/egress_test.go): Outbound HTTPS whitelist validation, open-write-tx rollback halt, redirect blocks, and mock routing.
* [`dashboard_layouts_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/dashboard_layouts_test.go): 2D Home dashboard layout persistence, role defaults, staff layout resolution, and inline permission error card verification.
* [`ui_sdk_and_theme_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/ui_sdk_and_theme_test.go): UI bridge scripts (`/sdk/mis-bridge.js`, `/sdk/mis-theme.css`), UI config (`GET /api/system/ui-config`), global search (`GET /api/system/search`), opaque `plug_id` routing, and pinned recents.
* [`gdpr_sar_purge_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/gdpr_sar_purge_test.go): UK GDPR Subject Access Request (SAR) aggregation, redaction UI endpoints, KCSIE retention lock verification, and zero-orphan statutory purges.
* [`diagnostic_bundle_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/diagnostic_bundle_test.go): PII-scrubbed diagnostic ring buffer capture, stack trace sanitization, and bundle export (`/api/admin/diagnostics/bundle`).
* [`cli_config_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/cli_config_test.go): Section 7C Break-Glass CLI (`config list`, `get`, `set`, `repair-sso`).
* [`external_sync_and_kms_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/external_sync_and_kms_test.go): Cloud KMS mode gates, mock Parent DMZ push/pull synchronization, and mock MAT sync policy feed.

### Sample Plugin Integration Suites
* [`assessment_markbook_plugin_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/assessment_markbook_plugin_test.go): Full integration suite for `uk.org.optimis.assessment.markbook`.
* [`dfe_census_plugin_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/dfe_census_plugin_test.go): Full integration suite for `uk.gov.dfe.census`.
* [`first_aid_medical_plugin_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/first_aid_medical_plugin_test.go): Full integration suite for `uk.org.optimis.medical.first_aid`.
* [`lms_sync_plugin_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/lms_sync_plugin_test.go): Full integration suite for `uk.org.optimis.lms.sync`.
* [`sparx_maths_plugin_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/tests/sparx_maths_plugin_test.go): Full integration suite for `uk.co.sparx.maths`.
