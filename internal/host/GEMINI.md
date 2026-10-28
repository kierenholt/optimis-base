# Go Host Runtime, Session/Persona Resolution & RPC Gateway (`internal/host`)

This package implements the **Optimis Go Host Container** (Sections 5, 7, 8, 9, and 10 of the Architecture Blueprint), responsible for authentication/persona session resolution, HTTP/RPC routing, capability-sandboxed `gopher-lua` execution, per-plugin SQLite lifecycle/migration management, post-commit Event Bus dispatch, and PII-scrubbed diagnostic capture.

---

## 1. Key Files & Responsibilities

* [`host.go`](file:///home/kieren/Documents/code/optimis/internal/host/host.go):
  * [`NewHost`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L111): Initializes the Host container, mounts `core.db` via [`dal.NewCoreDAL`](file:///home/kieren/Documents/code/optimis/internal/dal/dal.go#L97), and initializes the in-memory Event Bus and PII-scrubbed diagnostic ring buffer.
  * [`LoadPlugin`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L504) & [`GetLoadedPlugin`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L527): Validates `manifest.json` and `src/init.lua` from disk (with automatic mtime-based hot-reload), takes file-level `.snapshot` backups of `plugin_<alias>.db`, runs pending `migrations/*.sql` inside `BEGIN IMMEDIATE` with automatic snapshot rollback on failure, and attaches the plugin database across `WriteDB` and `ReadDB`.
  * [`ExecuteRPC`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L554) & [`ExecuteWorker`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go): Enforces default-deny `manifest.json` endpoint/worker declarations and RBAC scopes before executing inside a stripped Lua 5.1 VM (`io`, `os`, `debug`, `package`, `dofile`, `loadfile`, `require` removed).
* [`persona_auth.go`](file:///home/kieren/Documents/code/optimis/internal/host/persona_auth.go):
  * **Environment Flag Gate (`ENABLE_DEV_PERSONA_SWITCHER`)**:
    * [`ConfigureAuthMode`](file:///home/kieren/Documents/code/optimis/internal/host/persona_auth.go#L72-L89) configures whether the Host operates in Dev Persona Switcher mode (`true`) or Real Federated SSO mode (`false`).
    * **Fail-Fast Real SSO Invariant**: When `ENABLE_DEV_PERSONA_SWITCHER=false`, [`loadRealSSOProvider`](file:///home/kieren/Documents/code/optimis/internal/host/persona_auth.go#L57-L70) queries `__system_config` for `sso_oidc_issuer_url` and `sso_oidc_client_id` and fails fast immediately with `REAL_SSO_NOT_CONFIGURED` if real OIDC SSO is unconfigured or unavailable, while blocking all `/api/auth/personas` and `/api/auth/persona` requests (`403 DEV_PERSONA_SWITCHER_DISABLED`).
  * **Session & Persona Resolution**:
    * [`ResolveRequestCaller`](file:///home/kieren/Documents/code/optimis/internal/host/persona_auth.go#L99-L199) resolves the [`CallerContext`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L79-L86) for every incoming RPC call by inspecting the `SameSite=Strict; HttpOnly` `mis_session` cookie (or active dev persona selection) and querying [`CoreDAL.ResolveStaffPersonaByID`](file:///home/kieren/Documents/code/optimis/internal/dal/persona_auth.go#L200-L263) against `core.db`.
  * **Auth & Persona Endpoints ([`RegisterAuthAndPersonaRoutes`](file:///home/kieren/Documents/code/optimis/internal/host/persona_auth.go#L201-L361))**:
    * `GET /api/auth/personas`: Lists active staff personas and their resolved roles/scopes from `core.db` (returns `403` when `ENABLE_DEV_PERSONA_SWITCHER=false`).
    * `POST /api/auth/persona`: Validates `staff_id` against `core.db`, issues a `SameSite=Strict; HttpOnly` `mis_session` cookie, and returns the resolved [`CallerContext`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L79-L86).
    * `GET /api/auth/session`: Returns the active session's resolved [`CallerContext`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L79-L86).
    * `GET /api/auth/sso/login`: Initiates Federated SSO login (fails fast with `500 REAL_SSO_NOT_CONFIGURED` when `ENABLE_DEV_PERSONA_SWITCHER=false` and SSO is not configured).
* [`dal_bindings.go`](file:///home/kieren/Documents/code/optimis/internal/host/dal_bindings.go):
  * Binds capability-scoped `ctx.dal.<domain>` Lua namespaces (`pupils`, `groups`, `attendance`, `guardians`, `safeguarding`), dynamically omitting any domain not declared in the current endpoint/worker's `manifest.json` scopes and enforcing `:own_groups` vs `:all` row-level filtering.
* [`server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go):
  * [`NewHTTPHandler`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L19): Mounts the embedded React UI (`ui_dist`), serves sandboxed plugin assets (`GET /plugins/{plugin_id}/ui/{asset_path...}`) with `Content-Security-Policy: default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'none';`, registers auth/persona routes, and dispatches `POST /api/plugins/{plugin_id}/rpc/{method}` using [`ResolveRequestCaller`](file:///home/kieren/Documents/code/optimis/internal/host/persona_auth.go#L99-L199).

---

## 2. Test Suites (`internal/host`)

* [`persona_session_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/persona_session_test.go): Verifies fail-fast behavior when `ENABLE_DEV_PERSONA_SWITCHER=false` (`REAL_SSO_NOT_CONFIGURED`), `core.db` staff/role/scope resolution, terminated staff rejection, and `mis_session` cookie RPC scope enforcement.
* [`dal_scoping_host_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/dal_scoping_host_test.go): Verifies dynamic `ctx.dal` namespace population, `:own_groups` Lua RPC scoping, `mutates: false` write-protection, write-only `ctx.dal.safeguarding.create_postbox`, background worker `SYSTEM` principal execution, and PII-scrubbed diagnostic ring buffer entries.
* [`plugin_db_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/plugin_db_test.go): Verifies per-plugin attached SQLite databases (`plugin_<alias>.db`), `PRAGMA max_page_count`, cross-schema SQL isolation, and automatic file-snapshot rollback on failed migrations.
* [`mvp_host_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/mvp_host_test.go): Verifies end-to-end Lua RPC execution, post-commit Event Bus emission, default-deny scope checks, and disk hot-reloading.
