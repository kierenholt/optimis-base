# Go Host Runtime, Session/Persona Resolution & RPC Gateway (`internal/host`)

This package implements the **Optimis Go Host Container** (Sections 5, 7, 8, 9, and 10 of the Architecture Blueprint), responsible for authentication/persona session resolution, HTTP/RPC routing, capability-sandboxed `gopher-lua` execution, per-plugin SQLite lifecycle/migration management, UK GDPR SAR export & retention purge orchestration, post-commit Event Bus dispatch, and PII-scrubbed diagnostic capture.

---

## 1. Key Files & Responsibilities

* [`host.go`](file:///home/kieren/Documents/code/optimis/internal/host/host.go):
  * [`NewHost`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L130): Initializes the Host container, mounts `core.db` via [`dal.NewCoreDAL`](file:///home/kieren/Documents/code/optimis/internal/dal/dal.go#L97), and initializes the in-memory Event Bus and PII-scrubbed diagnostic ring buffer.
  * [`LoadPlugin`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L363) & [`GetLoadedPlugin`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L388): Validates `manifest.json` and `src/init.lua` from disk (failing fast if `database.alias` is declared without mandatory `gdpr_hooks`), delegates SQLite provisioning/migrations to [`provisionAndMigratePluginDB`](file:///home/kieren/Documents/code/optimis/internal/host/plugin_migration.go#L52), and starts per-plugin worker pools.
  * [`ExecuteRPC`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L417): Enforces `PluginStateActive` (blocking `UNINSTALLED_DORMANT` plugins with `PLUGIN_DORMANT_QUARANTINE`), default-deny `manifest.json` endpoint declarations, and RBAC scopes before executing inside a stripped Lua 5.1 VM (`io`, `os`, `debug`, `package`, `dofile`, `loadfile`, `require` removed).
* [`plugin_migration.go`](file:///home/kieren/Documents/code/optimis/internal/host/plugin_migration.go):
  * [`provisionAndMigratePluginDB`](file:///home/kieren/Documents/code/optimis/internal/host/plugin_migration.go#L52) & [`runMigrationsInImmediateTx`](file:///home/kieren/Documents/code/optimis/internal/host/plugin_migration.go#L181): Provisions `plugin_<alias>.db`, configures WAL pragmas and `PRAGMA max_page_count`, creates automatic `.snapshot` file backups prior to running pending `migrations/*.sql` inside `BEGIN IMMEDIATE`, and restores the file snapshot on failure before attaching across `WriteDB` and `ReadDB`.
* [`gdpr.go`](file:///home/kieren/Documents/code/optimis/internal/host/gdpr.go):
  * [`SetPluginState`](file:///home/kieren/Documents/code/optimis/internal/host/gdpr.go#L63), [`ExportPupilSAR`](file:///home/kieren/Documents/code/optimis/internal/host/gdpr.go#L255), [`FinalizePupilSAR`](file:///home/kieren/Documents/code/optimis/internal/host/gdpr.go#L305), [`PurgePupilData`](file:///home/kieren/Documents/code/optimis/internal/host/gdpr.go#L334), & [`RegisterGDPRRoutes`](file:///home/kieren/Documents/code/optimis/internal/host/gdpr.go#L392): Manages `ACTIVE` / `UNINSTALLED_DORMANT` plugin states, aggregates Core + Plugin SAR sections, and enforces KCSIE retention locks before executing zero-orphan statutory purges.
* [`persona_auth.go`](file:///home/kieren/Documents/code/optimis/internal/host/persona_auth.go):
  * [`ConfigureAuthMode`](file:///home/kieren/Documents/code/optimis/internal/host/persona_auth.go#L72) & [`ResolveRequestCaller`](file:///home/kieren/Documents/code/optimis/internal/host/persona_auth.go#L99): Enforces `ENABLE_DEV_PERSONA_SWITCHER` (`true`/`false`), failing fast with `REAL_SSO_NOT_CONFIGURED` when `false`, and resolves `SameSite=Strict; HttpOnly` `mis_session` cookies against `core.db`.
* [`safeguarding_vault.go`](file:///home/kieren/Documents/code/optimis/internal/host/safeguarding_vault.go):
  * [`RegisterSafeguardingRoutes`](file:///home/kieren/Documents/code/optimis/internal/host/safeguarding_vault.go#L22): Exposes Section 5B endpoints (`POST /api/safeguarding/postbox`, `GET /api/safeguarding/notes`, `POST /api/safeguarding/cloud-key/renew`, and `POST /api/safeguarding/notes/{note_id}/decrypt`).
* [`dal_bindings.go`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go):
  * [`buildLuaDALTable`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L69) & [`ExecuteWorker`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go#L595): Binds capability-scoped `ctx.dal.<domain>` Lua namespaces (`pupils`, `groups`, `attendance`, `guardians`, `safeguarding`) and executes background workers under the `SYSTEM` service principal.
* [`egress.go`](file:///home/kieren/Documents/code/optimis/internal/host/egress.go):
  * [`buildLuaHTTPEgressTable`](file:///home/kieren/Documents/code/optimis/internal/host/egress.go#L204) & [`ConfigureEgressMode`](file:///home/kieren/Documents/code/optimis/internal/host/egress.go#L61): Implements `ctx.http.get` & `ctx.http.post` gated by `ENABLE_REAL_HTTP_EGRESS`, enforcing open-write-tx rollback (`EGRESS_WRITE_TX_HALT`), origin & redirect domain whitelist checks (`EGRESS_REDIRECT_BLOCKED`), and `egress.timeout_ms`.
* [`event_bus.go`](file:///home/kieren/Documents/code/optimis/internal/host/event_bus.go):
  * [`validateManifestWorkersAndLuaSource`](file:///home/kieren/Documents/code/optimis/internal/host/event_bus.go#L41) & [`startOrReplacePluginWorkerPool`](file:///home/kieren/Documents/code/optimis/internal/host/event_bus.go#L118): Allocates per-plugin buffered `chan Event` worker pools (`max_concurrency`), enforces `Hop > 3` loop protection (`events_dropped_total`), and exposes `GET /api/events/catalog`.
* [`ui_sdk.go`](file:///home/kieren/Documents/code/optimis/internal/host/ui_sdk.go):
  * [`RegisterUISDKAndConfigRoutes`](file:///home/kieren/Documents/code/optimis/internal/host/ui_sdk.go#L97): Serves `/sdk/mis-bridge.js` and `/sdk/mis-theme.css` with `connect-src 'none'` CSP headers and exposes `GET /api/system/ui-config`.
* [`server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go):
  * [`NewHTTPHandler`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L19): Mounts the embedded React UI (`ui_dist`), sandboxed plugin UI assets, and all Host HTTP/RPC endpoints.

---

## 2. Test Suites (`internal/host`)

* [`event_bus_router_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/event_bus_router_test.go), [`egress_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/egress_test.go), [`ui_sdk_and_theme_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/ui_sdk_and_theme_test.go), [`gdpr_sar_purge_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/gdpr_sar_purge_test.go), [`persona_session_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/persona_session_test.go), [`dal_scoping_host_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/dal_scoping_host_test.go), [`plugin_db_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/plugin_db_test.go), & [`mvp_host_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/mvp_host_test.go).
