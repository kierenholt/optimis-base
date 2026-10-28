# Optimis School MIS — Architecture Blueprint

## 1. Scope & Tenancy Model

### Target Market
UK All-Through (Primary, Secondary, All-Through K-12), serving both Multi-Academy Trusts (MATs) and standalone/Local Authority schools.

### Tenancy & Compute
Siloed Single-Tenant Containers (1 App Container + 1 Isolated DB per School) with Scale-to-Zero & Predictive Autoscaling (maintaining a Scale-to-Minimal floor to keep the warm SQLite WAL lock on local disk rather than scaling to zero replicas).

### Network Enclave
* **School Container:** Hosted on LGfL private WAN / dark fibre (zero public internet ingress). Remote staff access strictly via VPN into the private enclave. Evening maintenance window for updates/backups.
* **Parent Container:** Public Cloud DMZ hosting sanitized read-projections and an untrusted write-inbox.

### External Sync
100% Outbound School-Initiated Pull/Push. The School Container pulls MAT policies (taxonomies, grading schemas) and pulls queued parent submissions from the DMZ inbox. It pushes parent read-projections, MAT analytics, and DfE returns outbound.

### Authentication
Federated SSO via the school's native Microsoft Entra ID or Google Workspace.

---

## 2. Core Architecture & Data Model

### Database Engine
In-Process SQLite with `PRAGMA journal_mode=WAL;`, `synchronous=NORMAL;`, and `busy_timeout=5000;` mounted on persistent local block storage.

### Plugin Runtime & Governance
In-Process Capability-Sandboxed **Lua VM** (`io`/`os`/network stripped). Rather than rigid instruction-count termination, the host Data Access Layer (DAL) monitors lock contention and execution duration per plugin—alerting admins and circuit-breaking (disabling) plugins that repeatedly hold up the database.

### Data Access Layer (DAL)
Scoped, Versioned DAL injected as a host object into the Lua environment. Minor DB migrations maintain backwards compatibility via Expand-and-Contract; major breaking changes bump the DAL major version.

### Plugin Storage
Isolated per-plugin `.sqlite` files mounted dynamically via `ATTACH DATABASE 'plugin_<name>.db' AS plugin_<name>;`.

### Temporal Model
Effective-Dated Core Schema (`effective_from`, `effective_to`, `academic_year_id`). Complexity is abstracted away by simplified query methods in the DAL.

---

## 3. Core Domain Modules & Boundaries

### Core Boundary (`core.db`)
Holds all **Statutory**, **Safeguarding**, and **Common Non-Proprietary** school data:
* Pupils (UPN/ULN)
* Staff
* Family/Guardian relationships
* Safeguarding records & court-order blocks
* Statutory Attendance (AM/PM sessions)
* Groups/Timetable Spine
* SEN/Medical flags
* DfE School Census engine

### Plugin Boundary (`plugin_*.db`)
Holds proprietary or phase-specific feature workflows (e.g., custom Assessment/Gradebook models, Behaviour point schemes, Catering, Trips, Exam board entries).

### Deferred Low-Level Models
Exact SQL table schemas for the Family/Household graph and Group/Timetable spine are deferred to Low-Level Design, with a mandatory Core DAL invariant: the DAL contract must enforce **Parental Responsibility (PR) / court-order safeguarding flags** and resolve all phases down to **AM/PM statutory sessions**.

### Event Bus (Pub/Sub)
Host-managed, asynchronous (post-commit) Event Bus with hop-count limits (preventing infinite loops). The **Core defines the standard domain event catalog**; plugins primarily subscribe to Core events and only emit versioned custom events when opting to share proprietary state.

---

## 4. Integration & Interoperability

### Third-Party EdTech Strategy
**100% Zero-Fee Lua Plugin Model.** No third-party data aggregators (e.g., Wonde) or external payment middleware (e.g., ParentPay). Approved partners (e.g., Sparx) build lightweight Lua integration plugins with £0 connection/SDK fees.

### LMS & Directory Sync
Implemented as first-party Lua plugins/workers (`plugin_google_classroom`, `plugin_ms_teams`) listening to Core Event Bus membership changes.

### DfE Statutory Census
**Core Data + Hot-Swappable Census Plugin.** Statutory data lives in `core.db`, while the termly DfE School Census / Workforce Census generators and validation rules are shipped as versioned Lua plugins (backed by host XML/XSD helpers), allowing rule updates without core container redeployment.

### Outbound Network & Secrets
Plugins manage their own vendor credentials within their isolated `plugin_<name>.db` state. For outbound communication, plugins declare required DAL scopes and target HTTPS domains in their manifest, which the Host Container routes and audits.

---

## 5. Security, Privacy & Compliance

### Access Control Model (5A) — Method-Level Scoped RBAC
Interactive plugin endpoints and background worker methods declare required DAL scopes (`:own_groups` vs `:all`). Interactive calls verify the **Logged-In User's scopes**; background workers verify the **Admin-approved Plugin Manifest scopes** displayed at installation.

### Safeguarding Isolation (5B) — Envelope-Encrypted DSL Vault + OAuth Cloud Key Renewal
All staff hold `dal.safeguarding.create` (write-only postbox). Reading (`dal.safeguarding.read`) is restricted to Designated Safeguarding Leads (DSLs) and protected by **envelope encryption** in `core.db`. DSLs unlock the vault's Master Data Key using a **short-lived, cryptographically signed key** issued by a **Cloud Key Generator** linked to their OAuth account, requiring renewal every $X$ days. If lost or expired, generating a new cloud key seamlessly restores access without data loss.

### UK GDPR SAR & Retention (5C) — Mandatory Plugin Hooks + Admin Redaction UI
Every plugin attaching a `plugin_<name>.db` file implements `export_pupil_data(pupil_id)` and `purge_pupil_data(pupil_id)`. Exports are initiated by a System Admin/DPO and loaded into an **integrated Redaction UI** so third-party names and sensitive notes can be reviewed and redacted before export.

---

## 6. Plugin Packaging, Distribution & UI Architecture

### Distribution & Upgrade (6A) — Self-Contained `.mispkg` Bundles
Manual file upload by the School Admin is the primary installation workflow, with an optional **MAT Sync Feed** allowing MAT-linked school containers to pull pre-approved `.mispkg` bundles outbound (subject to Admin scope approval if permissions change).

### Migration Safety
Prior to applying any `.mispkg` SQL migration scripts (`migrations/*.sql`), the Host Container takes an automatic file-level snapshot of `plugin_<name>.db` and executes DDL inside an immediate transaction for zero-downtime automatic rollback on failure.

### UI Rendering Model (6B) — Sandboxed `<iframe>` Micro-Frontends
Plugin UIs run inside `<iframe sandbox="allow-scripts">` (without `allow-same-origin`) and a strict `connect-src 'none'` Content Security Policy. All HTML/JS/CSS/font assets must be bundled offline inside the `.mispkg` archive.

### UI-to-Lua Bridge (6C) — Declarative Tag Binding -> `postMessage` -> Host RPC Gateway
Plugin UI code binds actions via declarative tags/attributes matching the Lua method names in `manifest.json`. The bundled Host JS shim intercepts these events, sends a `window.postMessage` payload to the parent Host UI Shell, and the Host executes the authenticated `POST /api/plugins/:id/rpc/:method` call against the Lua VM after verifying the user's method-level scopes.

---

## 7. Host Runtime & Infrastructure Stack

### Host Language & Runtime (7A)
**Go (Golang)** compiled as a single static binary (`CGO_ENABLED=0` where possible). Executes the HTTP/RPC gateway, async Event Bus, background sync workers, and Lua VM pool natively.

### Lua & SQLite Engine Integration
**Pure-Go Lua (`yuin/gopher-lua`)** targeting Lua 5.1 syntax to eliminate cgo FFI boundary overhead on DAL calls, paired with **coarse-grained batch DAL methods**. SQLite uses two separate Go `sql.DB` pools:
* **WriteDB (`MaxOpenConns(1)`):** Serializes writes without lock contention.
* **ReadDB (`MaxOpenConns(NumCPU)`):** Serves concurrent WAL reads, using a `ConnectHook` to keep `ATTACH DATABASE` state synchronized across all pooled connections.

### Host UI Shell (7B)
**Embedded React SPA** compiled into the Go binary via `embed.FS`. Renders the global MIS navigation chrome, authentication flows, DPO plugin-install scope approvals, the Admin SAR Redaction UI, and the `window.postMessage` RPC coordinator for child plugin iframes.

### Configuration & Break-Glass (7C)
**Minimal Bootstrap + Self-Contained SQLite Config.** Only 3 immutable bootstrap variables (`DATA_DIR_PATH`, `SCHOOL_URN`, `HOST_KMS_KEY_HANDLE`) are injected via environment variables; all runtime configuration (OAuth client IDs, MAT sync endpoints, plugin manifests) lives in `core.db`'s `__system_config` table for single-folder disaster recovery. The Go binary includes local CLI break-glass subcommands (`./mis-server config ...`) to repair misconfigured SSO settings without web UI access.

---

## 8. Repository Topology, SDK & Day-2 Operations

### Repository Topology
**Strict Core vs. Plugin Separation.** The Base Platform repository contains only the Go Host Container, React Shell, Core DAL, and synthetic fixture/test plugins. All production plugins (both first-party and third-party) live in separate repositories.

### Local Testbed (8A) — Distributable `mis-dev-server` Binary
Released alongside the Base Platform, booting the real Go Host Container locally with an auto-seeded **Synthetic UK School `core.db`** (fake pupils, households, groups, and an in-UI **Persona Switcher** for Teacher, HoD, DSL, Admin) plus hot-reload mounting for the plugin's `src/`, `migrations/`, and `ui/` folders.

### Automated Verification (8B) — `mis-sdk verify` Gatekeeper
Enforces 4 automated checks in CI before packaging into `.mispkg`:
1. Cross-checking hand-written `manifest.json` <-> Lua functions <-> UI `data-mis-rpc` tags.
2. CSP/Sandbox linting (`connect-src 'none'` offline check).
3. Automated UK GDPR test (running migrations on a temp SQLite DB and verifying `export_pupil_data` & `purge_pupil_data` leave zero orphaned pupil rows).
4. Lua test suite & SQLite lock-duration profiling.

### Manifest & Packaging (8C) — Hand-Written `manifest.json` + `mis-sdk pack`
Developers explicitly author `manifest.json` as the contract source of truth (validated via a published JSON Schema for IDE autocomplete). The Go Host enforces a strict default-deny rule for any Lua function omitted from `manifest.json`.

### Day-2 Error Reporting (Gap 1) — Admin-Exported PII-Scrubbed Diagnostic Bundle
Zero automated outbound crash telemetry. The Go Host captures Lua stack traces (file, line number, error class, execution/lock wait time—automatically stripping all local variables, RPC request payloads, and pupil IDs) into a local ring buffer. When an issue occurs, the School Admin clicks **"Download Plugin Diagnostic Bundle"** and shares the sanitized `.json`/`.log` archive with the vendor.

### Live Rollbacks (Gap 2) — Non-Destructive Minor Migrations
`mis-sdk verify` enforces Expand-and-Contract SQL rules on minor updates (v1.2 to v1.3 cannot drop/rename columns), allowing Admins to roll back plugin code to v1.2 mid-day without restoring an old DB snapshot or losing morning data.

### Offboarding (Gap 3) — Dormant Quarantine State
Uninstalling a plugin disables its UI, workers, and events immediately, but retains `plugin_<name>.db` and its read-only GDPR `export_pupil_data` / `purge_pupil_data` hooks in a dormant archive until statutory retention expires.

---

## 9. Plugin Contract Specification

The **Plugin Contract** serves as the strict boundary between the Go Host Container and any first-party or third-party `.mispkg` plugin. It consists of four non-negotiable surfaces: the **Package & Manifest Specification**, the **Injected Lua Runtime (`ctx`)**, the **Lifecycle & GDPR Hooks**, and the **Sandboxed UI Bridge**.

### 9.1 The `.mispkg` Bundle & Manifest Contract
Every plugin is distributed as a signed archive containing four static directories/files (`manifest.json`, `migrations/`, `src/init.lua`, and `ui/dist/`). The hand-written `manifest.json` acts as the **Default-Deny Routing & Permission Table**—if a method, scope, event, or domain is not explicitly declared here, the Go Host blocks it at runtime.

#### `id`, `version`, `core_dal_version`
* **Contract Rule:** Reverse-DNS identifier (`uk.co.sparx.maths`), SemVer plugin version, and compatible Core DAL SemVer range (`~1.4`).
* **Host Enforcement & DPO Visibility:** Host refuses to install or boot if the container's Core DAL does not satisfy `core_dal_version`.

#### `database.alias`
* **Contract Rule:** Optional string (e.g., `plugin_sparx`). Must match `^plugin_[a-z0-9_]+$`.
* **Host Enforcement & DPO Visibility:** Host provisions `<alias>.db`, attaches it via `ConnectHook`, and enforces Expand-and-Contract rules on minor migrations.

#### `egress.allowed_domains`
* **Contract Rule:** Array of explicit HTTPS origins (e.g., `["https://sync.sparxmaths.uk"]`).
* **Host Enforcement & DPO Visibility:** Shown to DPO on install. Host blocks any `ctx.http` call to undeclared domains.

#### `endpoints.<method_name>`
* **Contract Rule:** Maps an interactive Lua function name to required DAL scopes (`dal.<domain>.<action>:<modifier>`) and `mutates: true/false`.
* **Host Enforcement & DPO Visibility:** Checked against the **Logged-In User's scopes** on every RPC request. Determines whether the call uses `ReadDB` or `WriteDB`.

#### `workers.<method_name>`
* **Contract Rule:** Maps a background Lua function to a Cron schedule or Core Event subscription, plus required DAL scopes.
* **Host Enforcement & DPO Visibility:** Checked against the **Plugin Service Principal** approved by the Admin at installation.

#### `gdpr_hooks`
* **Contract Rule:** Required if `database.alias` is present. Must map `export_pupil_data` and `purge_pupil_data` to exported Lua functions.
* **Host Enforcement & DPO Visibility:** Verified by `mis-sdk verify` in CI and invoked by the Host for Subject Access Requests (SARs) and retention purges.

### 9.2 The Injected Lua Runtime Contract (`ctx`)
Plugins execute inside a stripped **Lua 5.1 (`gopher-lua`)** sandbox (`os`, `io`, `debug`, `package`, and `dofile` are removed). Every exported Lua function receives `(ctx, args)` and must return a serializable Lua table `{ ok = true, data = ... }` or throw a standard error string.

The `ctx` object injected by the Go Host exposes five capability-scoped namespaces:

#### `ctx.caller`
* **Provided Capabilities:** Read-only identity metadata: `type` (`"USER"` or `"SYSTEM"`), `user_id`, `staff_id`, `active_roles`, and `request_id`.
* **Contract Guardrails:** Immutable. Null/empty `staff_id` when invoked via a background worker or event listener.

#### `ctx.dal.<domain>`
* **Provided Capabilities:** Versioned, coarse-grained Core data methods (`pupils`, `groups`, `attendance`, `assessment`, `guardians`, `safeguarding`).
* **Contract Guardrails:** **Dynamic Row & Column Filtering:** Only domains declared in the current method's manifest scopes are populated on `ctx.dal`. If the method scope carries `:own_groups`, the Go DAL automatically restricts SQL results to the caller's active temporal teaching groups.

#### `ctx.plugin_db`
* **Provided Capabilities:** `.query(sql, params)` and `.exec(sql, params)` scoped to the plugin's attached `plugin_<name>.db`.
* **Contract Guardrails:** `.exec()` is only available if the method declares `mutates: true` (routing to the single-connection `WriteDB` pool). Hard execution timeout (e.g., 2,500ms) enforced via Go `context.Context` to prevent WAL lock starvation.

#### `ctx.events`
* **Provided Capabilities:** `.emit(event_name, payload)` for publishing to the Core Event Bus.
* **Contract Guardrails:** Messages are buffered in memory during the Lua call and **only dispatched after the SQLite transaction commits**. Hop count is incremented automatically to prevent infinite plugin loops.

#### `ctx.http`
* **Provided Capabilities:** `.get(url, headers)` and `.post(url, headers, body)` for outbound integration sync.
* **Contract Guardrails:** **Disallowed inside write transactions.** Host halts any open DB write transaction before executing outbound HTTP so slow external APIs never hold a local SQLite lock.

### 9.3 Lifecycle & UK GDPR State Contract
A plugin moves through five deterministic states managed by the Go Host Container:

1. **Pre-Flight & Migration (`INSTALLING` / `UPGRADING`):**
   * Host verifies `.mispkg` signature and checks if any requested scopes changed since the previous version (pausing for Admin/DPO approval if scopes expanded).
   * Host takes an instant file copy of `plugin_<name>.db`, runs pending `migrations/*.sql` inside `BEGIN IMMEDIATE`, and rolls back the file snapshot automatically if any migration fails.
2. **Active Execution (`ACTIVE`):**
   * Endpoints, workers, and event listeners are live.
   * **Circuit Breaker Contract:** If a plugin method exceeds the CPU instruction limit or causes repeated `SQLITE_BUSY` timeouts, the Host trips the circuit breaker on that specific method and writes a PII-scrubbed stack trace (file, line number, error class, duration—zero variable state or pupil IDs) to the **Local Diagnostic Ring Buffer** for Admin export.
3. **Subject Access Request (`SAR_EXPORT`):**
   * When an Admin initiates a SAR for `pupil_id`, the Host invokes `gdpr_hooks.export_pupil_data(ctx, pupil_id)`.
   * **Return Contract:** The function must return a structured array of section tables (`title`, `records`, and `redactable_fields` indicating free-text columns that may contain third-party pupil names) so the **Admin SAR Redaction UI** can render them uniformly.
4. **Dormant Quarantine (`UNINSTALLED_DORMANT`):**
   * When uninstalled by an Admin, the UI, endpoints, and workers are disabled immediately.
   * `plugin_<name>.db` and the Lua GDPR hooks remain mounted in read-only/purge mode so the school remains compliant with SARs and statutory retention schedules until expiry.
5. **Retention Purge (`PURGED`):**
   * Invoking `gdpr_hooks.purge_pupil_data(ctx, pupil_id)` permanently removes all rows tied to that pupil inside `plugin_<name>.db`.

### 9.4 The Sandboxed UI & RPC Bridge Contract
Plugin frontends run inside `<iframe sandbox="allow-scripts">` with a strict `connect-src 'none'` Content Security Policy. The UI communicates with the Host Container exclusively through declarative HTML attributes or the bundled `mis-bridge.js` helper over `window.postMessage`:

* **Initialization Handshake (`mis:init`):** When the iframe mounts, the React Host Shell sends a read-only initialization payload containing the school's theme tokens (CSS variables), current view context (e.g., currently selected `pupil_id` or `group_id` if opened as a contextual tab), and the user's permitted UI methods for that plugin.
* **Declarative Invocation (`data-mis-rpc`):** Elements tagged with `data-mis-rpc="<method_name>"` and `data-mis-args='<json>'` automatically send a `mis:rpc:request` message (`{ request_id, method, args }`) to the parent React Shell.
* **Host Verification & Response (`mis:rpc:response`):** The parent React Shell dispatches `POST /api/plugins/:plugin_id/rpc/:method_name` using its own `SameSite=Strict` session cookie, and posts `{ request_id, ok, data, error_code }` back into the iframe.

---

## 10. Runtime & Resource Contract

### 10.1 Author-Declared Resource & Concurrency Envelope (`manifest.json`)
The plugin author declares the operational footprint their plugin (or specific method) requires inside `manifest.json`, with sensible host fallbacks if omitted. Because it is in the manifest, the School Admin sees the exact resource profile on the installation screen, and the Go Host sets up the `gopher-lua` context and worker pools to match.

*(Structural Safety Invariant: The Host always enforces **no open SQLite write transactions during an outbound `ctx.http` fetch**, so external network latency never locks the local database).*

#### `max_concurrency`
* **Scope:** Plugin or per-method.
* **What It Controls:** Sizes the Go semaphore/worker pool for this plugin (e.g., max 4 concurrent executions of `sparx_sync`).
* **Host Fallback (If Omitted):** `4` (Interactive), `1` (Workers).

#### `timeout_ms`
* **Scope:** Per-method.
* **What It Controls:** Passed directly to `context.WithTimeout` on the `gopher-lua` state (`L.SetContext`).
* **Host Fallback (If Omitted):** `2500` ms (Endpoints), `30000` ms (Workers).

#### `max_memory_mb`
* **Scope:** Plugin-wide.
* **What It Controls:** Sets the `gopher-lua` call-stack and registry allocation cap per VM instance.
* **Host Fallback (If Omitted):** `32` MB.

#### `max_db_size_mb`
* **Scope:** `database` block.
* **What It Controls:** Translated to SQLite `PRAGMA plugin_<alias>.max_page_count`.
* **Host Fallback (If Omitted):** `500` MB.

#### `egress.allowed_domains`
* **Scope:** `egress` block.
* **What It Controls:** Explicit whitelist of HTTPS domains (including redirect targets). Anything else is rejected.
* **Host Fallback (If Omitted):** `[]` (Zero outbound access).

#### `egress.timeout_ms`
* **Scope:** `egress` block.
* **What It Controls:** Timeout applied to the Go `http.Client` for `ctx.http` calls.
* **Host Fallback (If Omitted):** `8000` ms.

### 10.2 Native In-Memory Event Bus (Go Channels)
Instead of backing the Event Bus with SQLite tables and Dead-Letter Queues, the Host runs a **native Go in-memory pub/sub router** built on buffered Go channels and worker goroutines:

* **Per-Plugin Buffered Channel:** When a plugin subscribes to events in `manifest.json`, the Go Host allocates a dedicated buffered `chan Event` for that plugin and spawns $N$ worker goroutines matching the plugin's declared `max_concurrency`.
* **Non-Blocking Dispatch:** After a Core or Plugin DB transaction commits, the Host router fans the event out to subscriber channels. If Plugin B is slow or hits its concurrency limit, its channel buffers the work in memory without blocking the caller or slowing down Plugin A.
* **Hop-Count Header:** The Go `Event` struct carries a simple `Hop uint8` counter incremented on re-emission (dropping events at `Hop > 3` to prevent infinite plugin loops).

### 10.3 Raw Base Metrics (No Synthesized Health State)
The Go Host does not compute traffic-light states or run automatic quarantine timers. Instead, the RPC Gateway, Event Router, and SQLite/HTTP wrappers simply record **raw atomic counters and latency histograms** per plugin and per method:

* `invocations_total` (by method & status: `ok`, `error`, `timeout`)
* `duration_ms` (execution time)
* `sqlite_write_lock_wait_ms` & `sqlite_write_hold_ms`
* `http_egress_requests_total` & `http_egress_duration_ms`
* `event_channel_depth` & `events_dropped_total`
* **Diagnostic Ring Buffer:** Last $N$ PII-scrubbed Lua stack traces (file, line number, error string) ready for Admin export.

Any UI dashboards, alerts, or circuit-breaking policies can be derived directly from these base metrics.

---

## 11. Important Repository Files & Agent Customizations

* [`.agents/AGENTS.md`](file:///home/kieren/Documents/code/optimis/.agents/AGENTS.md): Workspace-level repository rules, fail-fast requirements, and `npm` test execution conventions.
* [`.agents/skills/test-author/SKILL.md`](file:///home/kieren/Documents/code/optimis/.agents/skills/test-author/SKILL.md): On-demand agent skill defining the workflow and guardrails for authoring Playwright E2E, Go Host/DAL, and Lua GDPR compliance tests (strictly without modifying production code or executing tests).
* [`.agents/skills/test-author/examples/test_templates.md`](file:///home/kieren/Documents/code/optimis/.agents/skills/test-author/examples/test_templates.md): Reference test templates for sequential Playwright E2E tests, isolated SQLite Go DAL tests, and Lua GDPR hook verification.
* [`.agents/skills/code-implementer/SKILL.md`](file:///home/kieren/Documents/code/optimis/.agents/skills/code-implementer/SKILL.md): On-demand agent skill for test-driven production code implementation (requires a pre-written test before proceeding, never starts local services, and enforces fail-fast and Optimis architectural invariants).
* [`cmd/mis-server/main.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-server/main.go): Go Host server binary entrypoint loading `.env` with fail-fast environment checks and booting `core.db` and `test_plugin`.
* [`internal/dal/dal.go`](file:///home/kieren/Documents/code/optimis/internal/dal/dal.go): Core DAL managing separated `WriteDB` (`MaxOpenConns(1)`) and `ReadDB` (`MaxOpenConns(NumCPU)`) SQLite WAL pools, `__system_config`, and effective-dated `pupils` table operations.
* [`internal/host/host.go`](file:///home/kieren/Documents/code/optimis/internal/host/host.go): Capability-sandboxed `gopher-lua` plugin runtime injecting `ctx.caller`, `ctx.dal`, `ctx.plugin_db`, `ctx.events`, and `ctx.http`, enforcing default-deny `manifest.json` scopes, post-commit Event Bus dispatch, and PII-scrubbed diagnostic ring buffer logging.
* [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go): HTTP/RPC gateway serving the embedded React SPA (`embed.FS`), sandboxed plugin UI assets with `connect-src 'none'` CSP, `POST /api/plugins/:id/rpc/:method`, and test log/seed endpoints.
* [`ui/src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx): Embedded Vite + React Host UI Shell rendering `<iframe sandbox="allow-scripts">` and coordinating `mis:init` and `mis:rpc:request` / `mis:rpc:response` messages over `window.postMessage`.
* [`plugins/test_plugin/manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/manifest.json), [`plugins/test_plugin/src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/init.lua), [`plugins/test_plugin/ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/ui/dist/index.html): Unpacked MVP test plugin declaring `get_pupil_record` (`dal.pupils.read:all`) and rendering a declarative `data-mis-rpc="get_pupil_record"` button that prints the returned pupil record to the console.
* [`internal/host/mvp_host_test.go`](file:///home/kieren/Documents/code/optimis/internal/host/mvp_host_test.go), [`internal/dal/core_schema_functional_test.go`](file:///home/kieren/Documents/code/optimis/internal/dal/core_schema_functional_test.go), [`tests/e2e/mvp_plugin.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/mvp_plugin.spec.ts), & [`tests/e2e/core_schema_functional.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/core_schema_functional.spec.ts): Pre-written Go Host/DAL, 45-table Core Schema functional, and Playwright E2E test suites.
* [`scripts/dev_watch.mjs`](file:///home/kieren/Documents/code/optimis/scripts/dev_watch.mjs): Full-stack hot-reload orchestrator (`npm run dev`) running `vite build --watch` for the React Host UI Shell and watching `cmd/` & `internal/` to recompile and restart `./bin/mis-server` on `.go` file edits.
* [`.vscode/launch.json`](file:///home/kieren/Documents/code/optimis/.vscode/launch.json): IDE Run & Debug launch configuration (`Build & Start Optimis MIS Server (Hot Reload)`) executing `npm run dev`.
* [`docs/schema-blueprints/`](file:///home/kieren/Documents/code/optimis/docs/schema-blueprints): Reference SQL schema blueprints downloaded from open-source SIS/MIS standards ([`gibbon.sql`](file:///home/kieren/Documents/code/optimis/docs/schema-blueprints/gibbon.sql), [`rosariosis.sql`](file:///home/kieren/Documents/code/optimis/docs/schema-blueprints/rosariosis.sql), [`ceds-ids.sql`](file:///home/kieren/Documents/code/optimis/docs/schema-blueprints/ceds-ids.sql), [`edfi-ods-tables.sql`](file:///home/kieren/Documents/code/optimis/docs/schema-blueprints/edfi-ods-tables.sql), [`edfi-ods-foreign-keys.sql`](file:///home/kieren/Documents/code/optimis/docs/schema-blueprints/edfi-ods-foreign-keys.sql), and Ed-Fi OneRoster 6.1.0 SQL projections).
* [`docs/schema-blueprints/gibbon-domains/00_TABLE_INDEX.md`](file:///home/kieren/Documents/code/optimis/docs/schema-blueprints/gibbon-domains/00_TABLE_INDEX.md): Agent-navigable domain index and triage map (`CORE`, `PLUGIN`, `DROP`) decomposing [`gibbon.sql`](file:///home/kieren/Documents/code/optimis/docs/schema-blueprints/gibbon.sql)'s 199 tables into 9 small domain SQL files (`01`–`09`, each under 800 lines, stripped of `INSERT` seed noise).
* [`schema/core/SCHEMA_MAP.md`](file:///home/kieren/Documents/code/optimis/schema/core/SCHEMA_MAP.md): Master agent navigation map for the production Optimis `core.db` SQLite schema (45 tables, 75 indexed foreign keys across 7 modular domain SQL files `00`–`06` in [`schema/core/`](file:///home/kieren/Documents/code/optimis/schema/core), each 77–228 lines).
* [`cmd/mis-seed-qa/main.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/main.go), [`cmd/mis-seed-qa/catalog_and_names.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/catalog_and_names.go), [`cmd/mis-seed-qa/seed_domains_00_to_03.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/seed_domains_00_to_03.go), [`cmd/mis-seed-qa/seed_pupils_and_timetable.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/seed_pupils_and_timetable.go), & [`cmd/mis-seed-qa/seed_domains_02_04_05_06.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/seed_domains_02_04_05_06.go): Standalone deterministic PRNG + pinned-anchor QA database generator (`npm run seed:qa`) populating `<DATA_DIR_PATH>/qa_core.db` across all 45 core tables (1,108 pupils across 16 year groups `N1`–`Y13` and 42 registration forms, 95 staff, 889 households, 137 groups, 2-week rolling timetable, 44,320 statutory `AM`/`PM` marks, SEN/EHCP, medical IHCP, and AES-256-GCM envelope-encrypted safeguarding records).


