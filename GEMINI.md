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
**Core Data + Hot-Swappable Census Plugin ([`plugins/dfe_census_plugin`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/GEMINI.md), `uk.gov.dfe.census`).** Statutory data lives in `core.db` and is aggregated via [`ctx.dal.census.get_statutory_rollup`](file:///home/kieren/Documents/code/optimis/internal/dal/domains_census.go) (`dal.census.read:all`), while the termly DfE School Census / Workforce Census generators and validation rules are shipped as versioned Lua plugins backed by Host XML/XSD helpers ([`ctx.xml.build` & `ctx.xml.validate_xsd`](file:///home/kieren/Documents/code/optimis/internal/host/xml_xsd.go) validating `dfe_school_census_v1.xsd`) and a feature-flagged Mock DfE COLLECT service ([`ENABLE_MOCK_DFE_CENSUS_SERVICE`](file:///home/kieren/Documents/code/optimis/internal/host/dfe_census_service.go)), allowing rule updates without core container redeployment.

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
Released alongside the Base Platform (`cmd/mis-dev-server`), booting the real Go Host Container locally with an auto-seeded **Synthetic UK School `core.db`** (fake pupils, households, groups, and test personas for Teacher, HoD, DSL, Admin controllable via `TEST_DEFAULT_PERSONA` in `.env`, CLI flag `--persona`, or `POST /api/auth/persona` when `ENABLE_DEV_PERSONA_SWITCHER=true`—with zero URL query parameter or HTTP header caller overrides and zero in-UI persona switcher in production UI) plus hot-reload mounting for the plugin's `src/`, `migrations/`, and `ui/` folders.

### Automated Verification (8B) — `mis-verify` Gatekeeper & `mis-test` Runner
> [!NOTE]
> **Sequencing Gate Unlocked:** With 11 concrete production plugins and Host UI workflows implemented and verified, the SDK & Toolchain Architecture (Sections 8A–8C & 10A–10D) is now **Locked & Confirmed** for implementation.

`cmd/mis-verify` enforces 4 automated checks in CI before packaging into `.mispkg` (paired with `cmd/mis-test` for headless Lua test execution):
1. Cross-checking hand-written `manifest.json` <-> Lua functions <-> UI `data-mis-rpc` & `data-mis-icon` tags (including monochrome Lucide `icon` catalog validation `UNKNOWN_LUCIDE_ICON`).
2. CSP/Sandbox & Monochrome Icon linting (`connect-src 'none'` offline check and zero-emoji check `NON_MONOCHROME_ICON_EMOJI_VIOLATION`).
3. Automated UK GDPR test (running migrations on a temp SQLite DB and verifying `export_pupil_data` & `purge_pupil_data` leave zero orphaned pupil rows).
4. Lua test suite (`mis-test`) & SQLite lock-duration profiling.

### Manifest & Packaging (8C) — Hand-Written `manifest.json` + `mis-pack`
Developers explicitly author `manifest.json` as the contract source of truth (validated via `manifest.schema.json` for IDE autocomplete). The Go Host enforces a strict default-deny rule for any Lua function omitted from `manifest.json`, and `cmd/mis-pack` extracts the declared monochrome Lucide SVG icon (`stroke="currentColor" fill="none"`) and bundles the signed `.mispkg` archive.

### Day-2 Error Reporting (Gap 1) — Admin-Exported PII-Scrubbed Diagnostic Bundle
Zero automated outbound crash telemetry. The Go Host captures Lua stack traces (file, line number, error class, execution/lock wait time—automatically stripping all local variables, RPC request payloads, and pupil IDs) into a local ring buffer. When an issue occurs, the School Admin clicks **"Download Plugin Diagnostic Bundle"** and shares the sanitized `.json`/`.log` archive with the vendor.

### Live Rollbacks (Gap 2) — Non-Destructive Minor Migrations
`mis-verify` enforces Expand-and-Contract SQL rules on minor updates (v1.2 to v1.3 cannot drop/rename columns), allowing Admins to roll back plugin code to v1.2 mid-day without restoring an old DB snapshot or losing morning data.

### Offboarding (Gap 3) — Dormant Quarantine State
Uninstalling a plugin disables its UI, workers, and events immediately, but retains `plugin_<name>.db` and its read-only GDPR `export_pupil_data` / `purge_pupil_data` hooks in a dormant archive until statutory retention expires.

---

## 9. Plugin Contract Specification

The **Plugin Contract** serves as the strict boundary between the Go Host Container and any first-party or third-party `.mispkg` plugin. It consists of four non-negotiable surfaces: the **Package & Manifest Specification**, the **Injected Lua Runtime (`ctx`)**, the **Lifecycle & GDPR Hooks**, and the **Sandboxed UI Bridge**.

### 9.1 The `.mispkg` Bundle & Manifest Contract
Every plugin is distributed as a signed archive containing four static directories/files (`manifest.json`, `migrations/`, `src/init.lua`, and `ui/dist/`). The hand-written `manifest.json` acts as the **Default-Deny Routing & Permission Table**—if a method, scope, event, or domain is not explicitly declared here, the Go Host blocks it at runtime.

#### `id`, `version`, `core_dal_version`, `icon`
* **Contract Rule:** Reverse-DNS identifier (`uk.co.sparx.maths`), SemVer plugin version, compatible Core DAL SemVer range (`~1.4`), and monochrome Lucide icon name (`"icon": "calculator"` at the plugin root, with optional `"icon"` overrides on `views[]` and `widgets[]`).
* **Host Enforcement & DPO Visibility:** Host refuses to install or boot if the container's Core DAL does not satisfy `core_dal_version`, and `cmd/mis-verify` rejects unknown Lucide icon names (`UNKNOWN_LUCIDE_ICON`).

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

#### `views[]` (Strict URL Routing, Deep-Linking & Search Discovery Contract)
* **Contract Rule:** Required array of routable view patterns (`id`, `path`, `entry`, `title`, `allowed_query_params` / `query_params`, optional monochrome Lucide `icon`, and optional `keywords: string[]` containing synonyms/verbs so Global Search handles vocabulary mismatches). Every `views[].path` must start with `/` and **MUST NOT** equal `/ui` or start with `/ui/` (enforced by [`sdk/manifest.schema.json`](file:///home/kieren/Documents/code/optimis/sdk/manifest.schema.json), `cmd/mis-verify`, and the Go Host with `RESERVED_PLUGIN_VIEW_PATH` to prevent collisions with `/p/{plug_id}/ui/{asset_path...}`).
* **Host Enforcement & Opaque `<plugid>` Security:** On installation, the Host generates and persists an instance-local random 8-character lowercase alphanumeric `plug_id` (`^[a-z0-9]{8}$`) in `__system_config` (`plugin_instance_id.<manifest.id>`). Global reverse-DNS `manifest.id` strings (`uk.org.optimis.*`) are never exposed in browser URLs, DOM `data-testid` / `src` attributes, or `GET /api/system/ui-config`. Undeclared view paths (`404 UNDECLARED_PLUGIN_VIEW_PATH`), undeclared query parameters (`400 UNDECLARED_PLUGIN_VIEW_QUERY_PARAM`), or legacy `/plugins/*` & `/api/plugins/*` routes (`404 LEGACY_PLUGIN_ROUTE_DISABLED`) fail fast—serving `ui_dist/index.html` with the `400`/`404` HTTP status code when `Accept: text/html` so the React Host Shell boots and renders `data-testid="host-route-error-banner"`.

#### `widgets[]` (Preset-Size Dashboard Mini-Iframes & Search Discovery Contract)
* **Contract Rule:** Optional array of mini-iframe widget declarations (`id`, `title`, mandatory non-empty `description`, `entry`, required `"width": 1 | 2` and `"height": 1 | 2` preset dimensions, optional monochrome Lucide `icon`, and optional `keywords: string[]` for Global Search / Command Palette discovery). Because `widgets[].id` is scoped per plugin, both the Go DAL ([`internal/dal/dashboard_layouts.go`](file:///home/kieren/Documents/code/optimis/internal/dal/dashboard_layouts.go)) and the React Host UI Shell ([`getWidgetCompositeKey`](file:///home/kieren/Documents/code/optimis/ui/src/themeAndTypes.ts), [`useHostWorkspaceState.ts`](file:///home/kieren/Documents/code/optimis/ui/src/views/useHostWorkspaceState.ts), [`DashboardWidgetGridPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/DashboardWidgetGridPanel.tsx), [`GlobalSearchAndCommandPalette.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/components/GlobalSearchAndCommandPalette.tsx), and [`useHostIframeBridge.ts`](file:///home/kieren/Documents/code/optimis/ui/src/components/useHostIframeBridge.ts)) key widget placement, reordering, removal, and iframe reference maps by the composite key `plug_id + ":" + widget_id`.
* **Host Enforcement & DPO Visibility:** Enforced by [`sdk/manifest.schema.json`](file:///home/kieren/Documents/code/optimis/sdk/manifest.schema.json), [`internal/sdk/verify.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/verify.go) (`cmd/mis-verify`, failing fast with `MISSING_WIDGET_DESCRIPTION`), and [`internal/host/plugin_loader.go`](file:///home/kieren/Documents/code/optimis/internal/host/plugin_loader.go). Widgets cannot be arbitrarily resized by users; they render strictly at their declared `1x` or `2x` column width and `1x` or `2x` preset height in [`ui/src/DashboardWidgetGridPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/DashboardWidgetGridPanel.tsx), and are browsed/selected in Edit Mode via an enlarged (`1240px × 880px`) centered backdrop Widget Picker Modal (`data-testid="dashboard-widget-picker-backdrop"` / `data-testid="dashboard-widget-picker-modal"`, `data-modal-view-mode="gallery" | "split"`) supporting both **Gallery View** (`data-testid="widget-picker-gallery-view"`) and a two-column **Split Preview View** (`data-testid="widget-picker-split-view"`: left list `data-testid="widget-picker-split-list"` + right detail/live preview pane `data-testid="widget-picker-split-preview"` mounting a live sandboxed `<iframe sandbox="allow-scripts">` `data-testid="widget-picker-live-preview-iframe"` at exact preset dimensions `300×180`, `612×180`, `300×372`, or `612×372` alongside a mini 2D footprint diagram `data-testid="split-preview-footprint-diagram"`), plus size filter pills (`All Sizes`, `1×1`, `2×1`, `1×2`, `2×2`), text search (`data-testid="widget-picker-search-input"`), plugin dropdown (`data-testid="widget-picker-plugin-filter"`), and `Hide already placed` toggle (`data-testid="widget-picker-hide-placed-toggle"`).

#### `gdpr_hooks`
* **Contract Rule:** Required if `database.alias` is present. Must map `export_pupil_data` and `purge_pupil_data` to exported Lua functions.
* **Host Enforcement & DPO Visibility:** Verified by `mis-sdk verify` in CI and invoked by the Host for Subject Access Requests (SARs) and retention purges.

### 9.2 The Injected Lua Runtime Contract (`ctx`)
Plugins execute inside a stripped **Lua 5.1 (`gopher-lua`)** sandbox (`os`, `io`, `debug`, `package`, and `dofile` are removed). Every exported Lua function receives `(ctx, args)` and must return a serializable Lua table `{ ok = true, data = ... }` or throw a standard error string.

The `ctx` object injected by the Go Host exposes six capability-scoped namespaces:

#### `ctx.caller`
* **Provided Capabilities:** Read-only identity metadata: `type` (`"USER"` or `"SYSTEM"`), `user_id`, `staff_id`, `active_roles`, and `request_id`.
* **Contract Guardrails:** Immutable. Null/empty `staff_id` when invoked via a background worker or event listener.

#### `ctx.dal.<domain>`
* **Provided Capabilities:** Versioned, coarse-grained Core data methods (`pupils`, `groups`, `attendance`, `assessment`, `guardians`, `safeguarding`, `census`).
* **Contract Guardrails:** **Dynamic Row & Column Filtering:** Only domains declared in the current method's manifest scopes are populated on `ctx.dal`. If the method scope carries `:own_groups`, the Go DAL automatically restricts SQL results to the caller's active temporal teaching groups. Statutory census rollups (`ctx.dal.census.get_statutory_rollup`) strictly require `dal.census.read:all` (`ScopeAll`).

#### `ctx.plugin_db`
* **Provided Capabilities:** `.query(sql, params)` and `.exec(sql, params)` scoped to the plugin's attached `plugin_<name>.db`.
* **Contract Guardrails:** `.exec()` is only available if the method declares `mutates: true` (routing to the single-connection `WriteDB` pool). Hard execution timeout (e.g., 2,500ms) enforced via Go `context.Context` to prevent WAL lock starvation.

#### `ctx.events`
* **Provided Capabilities:** `.emit(event_name, payload)` for publishing to the Core Event Bus.
* **Contract Guardrails:** Messages are buffered in memory during the Lua call and **only dispatched after the SQLite transaction commits**. Hop count is incremented automatically to prevent infinite plugin loops.

#### `ctx.http`
* **Provided Capabilities:** `.get(url, headers)` and `.post(url, headers, body)` for outbound integration sync.
* **Contract Guardrails:** **Disallowed inside write transactions.** Host halts any open DB write transaction before executing outbound HTTP so slow external APIs never hold a local SQLite lock.

#### `ctx.xml`
* **Provided Capabilities:** `.build(root_node_spec)` (deterministic XML builder with sorted attributes and entity escaping) and `.validate_xsd(xml_string, xsd_schema_name)` (validating well-formedness and statutory schema rules such as `"dfe_school_census_v1.xsd"`).
* **Contract Guardrails:** Fail-fast Lua errors on missing root tags or unknown XSD schema names; returns `{ valid = true/false, errors = { ... } }` for statutory CBDS field/pattern checks (6-digit `URN`, `AUTUMN|SPRING|SUMMER` `Term`, 13-char `UPN`, 10-digit `ULN`, `SENProvision` `N|K|E`, `EnrolmentStatus` `C|M|S|F|O`, and non-negative attendance session counts).

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

* **Initialization Handshake (`mis:init`) & Two-Way Route Synchronization (`mis:route:push` / `mis:route:changed`):** When the iframe mounts (or when the caller session or browser URL changes), the React Host Shell sends a read-only initialization payload containing the school's theme tokens (CSS variables), current view context (extracted from both `/p/<plugid>/...` path parameters such as `:pupil_id`/`:group_id` and query parameters), `route` (`{ path, pattern, params, query }`), and the user's permitted UI methods for that plugin. Plugins call `window.mis.pushRoute({ path, query, replace })` (or declarative `[data-mis-route-path]` / `[data-mis-route-query]`) to update the parent browser URL `/p/<plugid><path>?<query>` and subscribe via `window.mis.onRouteChanged(cb)`.
* **Declarative Invocation (`data-mis-rpc`) & Monochrome Icons (`data-mis-icon`):** Elements tagged with `data-mis-rpc="<method_name>"` and `data-mis-args='<json>'` automatically send a `mis:rpc:request` message (`{ request_id, method, args }`) to the parent React Shell and dispatch `CustomEvent("mis:rpc:success")` or `CustomEvent("mis:rpc:error")` on the clicked element when the response returns. Elements tagged with `data-mis-icon="<lucide_name>"` are automatically hydrated by `mis-bridge.js` (`window.mis.renderIcons`) into monochrome `<svg fill="none" stroke="currentColor" data-lucide-icon="<lucide_name>">` icons that inherit `--mis-*` theme colors (`mis-theme.css`).
* **Host Verification & Response (`mis:rpc:response`):** The parent React Shell dispatches `POST /api/p/:plug_id/rpc/:method_name` using its own `SameSite=Strict` session cookie, and posts `{ request_id, ok, data, error_code }` back into the iframe.

### 9.5 Base UI Shell, Workspace & Deep-Linking Architecture (Locked & Confirmed)

| Area | Previous Blueprint (Section 9.1 / 9.5) | Locked Design | Required Contract / Schema Change |
| :--- | :--- | :--- | :--- |
| **1. Left Sidebar (`HostSidebar.tsx`)** | Dynamically lists all installed plugins from `manifest.json`. | Narrow Icon Rail containing only Home, Global Search, Pinned Recents, Settings, and Logout rendered with monochrome Lucide icons (with `<header data-testid="host-topbar">` and in-UI persona switcher completely removed). | Remove the full plugin list and topbar from the Host UI Shell; store user-pinned view routes (`/p/<plugid><path>`) + titles in `__system_config` or user preferences. |
| **2. Global Search & Discovery** | Relies on the left sidebar + `data-mis-link` entity handlers. | Landing Page Top-Half Search + Global Command Palette (`Ctrl+K`) indexing all `views[]`, `widgets[]` (with `"Add to Dashboard"` action), and pupils/groups. | Add `keywords: string[]` (synonyms/verbs) to `views[]` and `widgets[]` in `sdk/manifest.schema.json` so search handles vocabulary mismatches. |
| **3. Widget Declaration (`manifest.json`)** | `{ "title": "...", "entry": "ui/dist/widget.html" }` — freely resized by user in Edit Mode. | Strict Preset Sizes: Only `1x` or `2x` column widths and `1x` or `2x` preset heights on a 2D `(col, row)` coordinate grid. | Update `widgets[]` in `sdk/manifest.schema.json` to require `"width": 1 \| 2` and `"height": 1 \| 2`. |
| **4. Dashboard Persistence (`core.db`)** | 1D auto-flowing array (`DashboardWidgetGridPanel.tsx`). | Explicit 2D `(col, row)` Coordinate Grid + Role Defaults: Personal Home Dashboard edits only `scope_type = 'STAFF'`; Admin Dashboard Editor Plugin manages `ROLE_DEFAULT` and cross-staff layouts. | Store dashboard layout as a JSON array of `{ plug_id, widget_id, col, row }` objects + `column_count` in `core.db`, rendering revoked/unpermitted widgets with an inline permission error card. |

#### Styling & Theming Engine
**Tailwind CSS + CSS Custom Properties (`--mis-*`) + Strict Monochrome Lucide Icons (`stroke="currentColor"`, `fill="none"`).** Base React Shell and plugins use Tailwind (zero runtime JS styling cost) and monochrome Lucide SVG icons ([`MonochromeLucideIcon.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/components/MonochromeLucideIcon.tsx) in the Host Shell and `[data-mis-icon]` in [`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js)) with zero emojis or unicode star/pin glyphs. On iframe mount (`mis:init`), the Host injects school branding colors, surface tokens, and typography (`--mis-font-size-base`, `--mis-font-scale`, and `base_font_size`) as `:root` CSS variables and `data-mis-font-size` attributes inside both the Host Shell and child plugin/widget iframes. All Host UI Shell components (`App.tsx`, `HostSidebar.tsx`, `GlobalSearchAndCommandPalette.tsx`, `DashboardWidgetGridPanel.tsx`, `SafeguardingVaultPanel.tsx`, `GdprSarPanel.tsx`, `AdminDiagnosticsPanel.tsx`, `DashboardSummaryStrip.tsx`) and plugin/widget iframes (`scaleDocumentTypography` in [`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js) and [`sdk/ui/mis-theme.css`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-theme.css)) scale font sizes via `calc(<N>px * var(--mis-font-scale, 1))` so changing Base Font Size (`Small` `0.875`, `Medium` `1`, `Large` `1.125`, `Extra Large` `1.25`) immediately scales all rendered text.

#### Left Sidebar (`HostSidebar.tsx`) & Lucide Icons — Narrow Icon Rail, Exclusive Right-Hand Pane View Switching, Pinned Recents & Staff Settings (`staff_settings`)
Instead of dynamically listing all installed plugins in the left sidebar (`nav-item-plugin-*` is completely removed) or rendering a top header bar (`<header data-testid="host-topbar">` is completely removed), [`HostSidebar.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/components/HostSidebar.tsx) (`data-testid="host-sidebar"`, `data-rail-mode="narrow-icon-rail"`, constrained to the viewport via `position: sticky; top: 0; height: calc(100vh / var(--mis-font-scale, 1)); max-height: calc(100vh / var(--mis-font-scale, 1))` with `body { margin: 0 }` so it never stretches beyond `window.innerHeight` when `base_font_size` zoom is `large` or `extra-large` or when the main workspace scrolls) renders an `88px` **Narrow Icon Rail**. Clicking any sidebar navigation button replaces the content of the right-hand pane (`<main data-testid="host-dashboard-grid">`, tracked via `hostViewMode: 'home' | 'search' | 'settings' | 'plugin'` in [`useHostWorkspaceState.ts`](file:///home/kieren/Documents/code/optimis/ui/src/views/useHostWorkspaceState.ts) and [`routeUtils.ts`](file:///home/kieren/Documents/code/optimis/ui/src/components/routeUtils.ts)) without opening modal overlays:
* **Home (`/` — `data-testid="nav-rail-home"`, `<MonochromeLucideIcon name="house" />`):** Sets `hostViewMode = 'home'`, updates the browser URL to `/`, sets `aria-current="page"`, and renders [`DashboardSummaryStrip.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/views/DashboardSummaryStrip.tsx), [`LandingPageTopHalfSearch`](file:///home/kieren/Documents/code/optimis/ui/src/components/GlobalSearchAndCommandPalette.tsx), [`DashboardWidgetGridPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/DashboardWidgetGridPanel.tsx), and any permitted DSL/Admin panels (`SafeguardingVaultPanel`, `GdprSarPanel`, `AdminDiagnosticsPanel`) while hiding `KeepAlivePluginWorkspace` (`display: none` so the $N=5$ LRU pool stays warm in the DOM).
* **Global Search (`/search` — `data-testid="nav-rail-search"`, `<MonochromeLucideIcon name="search" />`):** Sets `hostViewMode = 'search'`, updates the browser URL to `/search` (served by [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go)), sets `aria-current="page"`, and replaces the right-hand pane with [`HostFullPageSearchView`](file:///home/kieren/Documents/code/optimis/ui/src/components/GlobalSearchAndCommandPalette.tsx) (`data-testid="host-search-view"`, `host-search-input`, `host-search-tab-*`, `host-search-results`, `host-search-result-view-*`, `host-search-result-widget-*`, `host-search-result-pupil-*`, `host-search-result-group-*`, `host-search-pin-view-btn-*`, `host-search-add-widget-btn-*`) without opening a modal.
* **Pinned Recents (`/p/<plugid>...` — `data-testid="nav-rail-pinned-recents"`):** Clicking a pinned view (`data-testid="pinned-recent-item-<idx>"`) or selecting a search/plugin result sets `hostViewMode = 'plugin'`, updates the browser URL to `/p/<plugid><path>?<query>`, and replaces the right-hand pane so only [`DashboardSummaryStrip.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/views/DashboardSummaryStrip.tsx) and the active plugin in [`KeepAlivePluginWorkspace.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/views/KeepAlivePluginWorkspace.tsx) are visible.
* **Settings (`/settings` — `data-testid="nav-rail-settings"`, `<MonochromeLucideIcon name="settings" />`):** Sets `hostViewMode = 'settings'`, updates the browser URL to `/settings` (served by [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go)), sets `aria-current="page"`, and replaces the right-hand pane with the full-page **Host Settings View** (`data-testid="host-settings-view"`, containing the `--mis-*` Light / WCAG AAA High-Contrast theme switcher `data-testid="host-settings-theme-toggle-btn"` and the 4-preset **Base Font Size** control `data-testid="host-settings-font-size-control"` with `Small` `14px` `host-settings-font-size-btn-small`, `Medium` `16px` `host-settings-font-size-btn-medium`, `Large` `18px` `host-settings-font-size-btn-large`, and `Extra Large` `20px` `host-settings-font-size-btn-extra-large`) without opening a modal.
* **Logout (`data-testid="nav-rail-logout"`, `<MonochromeLucideIcon name="log-out" />`):** Dispatches `POST /api/auth/logout` and renders `data-testid="host-logged-out-banner"`, alongside compact school identity (`data-testid="host-school-name"`).

Staff UI settings (`base_font_size`, `theme_mode`, and `pinned_recents_json`) are persisted per staff member in the dedicated `staff_settings` table in `core.db` ([`schema/core/00_system_and_academic_cycle.sql`](file:///home/kieren/Documents/code/optimis/schema/core/00_system_and_academic_cycle.sql), managed by [`internal/dal/staff_settings.go`](file:///home/kieren/Documents/code/optimis/internal/dal/staff_settings.go) and [`internal/host/pinned_recents.go`](file:///home/kieren/Documents/code/optimis/internal/host/pinned_recents.go) via `GET /api/system/ui-config`, `PUT /api/system/user-settings`, and `POST /api/system/pinned-recents`, enforcing cross-staff authorization `403 FORBIDDEN_STAFF_ID_MISMATCH` and fail-fast `400 INVALID_BASE_FONT_SIZE` / `INVALID_THEME_MODE` validation). User-pinned view routes (`/p/<plugid><path>?<query>`) and their titles are manually pinned/unpinned via the Pinned section in `HostSidebar.tsx` (`data-testid="topbar-pin-view-btn"` with `<MonochromeLucideIcon name="pin" />`) or Global Search (`data-testid="host-search-pin-view-btn-<plugId>-<viewId>"` / `data-testid="search-pin-view-btn-<plugId>-<viewId>"`), rendering each plugin's declared monochrome Lucide `icon` on `pinned-recent-item-<idx>` (without an `x` overlay button on the tile), and enforcing active plugin & `MatchPluginViewRoute` sub-path/query validation, caller permission checks, and automatic filtering of dormant/uninstalled/unpermitted routes. At package time, `mis-pack` extracts and embeds the single declared monochrome Lucide SVG into the `.mispkg` metadata so the Base App never ships unused icon bloat.

#### Global Search & Discovery — Landing Page Search + Full-Page Search View (`/search`) + Command Palette (`Ctrl+K`)
Plugin views, dashboard widgets, and school entities are discovered through the **Landing Page Top-Half Search** (`data-testid="landing-page-search-panel"` on the Home view with record-type filter tabs `data-testid="landing-search-filter-<id>"` for `All`, `Pages`, `Widgets`, `Pupils`, and `Groups` and inline view Pin buttons `data-testid="landing-search-pin-view-btn-<plugId>-<viewId>"`), the **Full-Page Search View** (`data-testid="host-search-view"` opened in the right-hand pane at `/search` via `data-testid="nav-rail-search"`), and the **Global Command Palette (`Ctrl+K`)** (opened via keyboard `Ctrl+K` or `data-testid="landing-open-command-palette-btn"` labeled `"Full list"`, rendering `data-testid="command-palette-modal"` with a conventional `<MonochromeLucideIcon name="x" size={14} />` close button `data-testid="command-palette-close-btn"` and category filter tabs `All`, `Pages`, `Widgets`, `Pupils`, and `Groups` via `data-testid="command-palette-tab-<id>"`). All three surfaces index permitted `views[]` (omitting parameterized routes such as `:pupil_id`, `:group_id`, or `:staff_id` when the required entity context ID is absent rather than rewriting to `'/'`), `widgets[]` (including an inline **"Add to Dashboard"** action), and caller-scoped pupils/groups (`GET /api/system/search` in [`internal/host/ui_sdk.go`](file:///home/kieren/Documents/code/optimis/internal/host/ui_sdk.go), with in-flight request cancellation via `AbortController` on rapid keystrokes). Both `views[]` and `widgets[]` in [`sdk/manifest.schema.json`](file:///home/kieren/Documents/code/optimis/sdk/manifest.schema.json) support `keywords: string[]` (synonyms and action verbs, validated by [`internal/host/plugin_loader.go`](file:///home/kieren/Documents/code/optimis/internal/host/plugin_loader.go) and [`internal/sdk/verify.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/verify.go)) so search handles vocabulary mismatches across plugins.

#### Workspace Window Manager (9A) — Cached Keep-Alive LRU Pool, Dynamic Vertical Sizing (`mis:resize`) & Opaque `/p/<plugid>` Routing
The Base App keeps the last $N$ (e.g., 5) visited full-page plugin iframes mounted in the DOM (`display: none` when inactive, `src="/p/<plugid>/ui/index.html"`) for instant, zero-reload tab switching during lessons, evicting the least-recently-used iframe when the cap is reached. Full-page plugin iframes render with a viewport-filling minimum height (`calc(100vh - 96px)`) and automatically expand vertically without clipping via `mis:resize` messages emitted by [`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js) (`MutationObserver`, `ResizeObserver`, `mis:init`, and `mis:rpc:response`) and applied by [`useHostIframeBridge.ts`](file:///home/kieren/Documents/code/optimis/ui/src/components/useHostIframeBridge.ts) and [`KeepAlivePluginWorkspace.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/views/KeepAlivePluginWorkspace.tsx). Every plugin view and its entity/tab state is reflected in the browser URL (`/p/<plugid>/...`, e.g., `/p/rolk9th3/student/4092?tab=iep`), remembering each plugin's last sub-route across navigation switches and restoring exact context when opened in a new browser tab or shared with a colleague on the same school instance.

#### Home Dashboard Grid & Persistence (9B) — Explicit 2D `(col, row)` Coordinate Grid (`300px` × `180px`), Strict Preset Sizes (`1x`/`2x`) & Add Widget Ghost Preview (`core.db`)
* **Fixed-Width Columns (`300px` × `180px`), Explicit 2D Coordinates (`col`, `row`) & Strict Preset Widget Sizes (`manifest.json`):** Each `1x` column has a fixed width of `300px` and each `1x` row has a base height of `180px` (default `4` columns, user-adjustable between `2` and `8` columns via `- Remove Column` / `+ Add Column` in Edit Mode and persisted per user in `core.db`; `- Remove Column` is disabled when `columnCount <= 2` or when any placed widget occupies the rightmost column). Each entry in `widgets[]` in [`sdk/manifest.schema.json`](file:///home/kieren/Documents/code/optimis/sdk/manifest.schema.json) requires `"width": 1 | 2` (1x or 2x column span) and `"height": 1 | 2` (1x or 2x row span) alongside `id`, `title`, `entry` (`ui/dist/widget.html`, served at `/p/<plugid>/ui/widget.html`), and optional `keywords: string[]` (validated fail-fast by [`internal/sdk/verify.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/verify.go) and [`internal/host/plugin_loader.go`](file:///home/kieren/Documents/code/optimis/internal/host/plugin_loader.go), and exposed via [`internal/host/ui_sdk.go`](file:///home/kieren/Documents/code/optimis/internal/host/ui_sdk.go)). Each placed widget in [`ui/src/DashboardWidgetGridPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/DashboardWidgetGridPanel.tsx) renders at its explicit 1-indexed 2D coordinate (`gridColumn: "${col} / span ${width}"`, `gridRow: "${row} / span ${height}"`, `data-grid-col`, `data-grid-row`) strictly at its declared `1x`/`2x` preset dimensions without arbitrary freeform resizing or unexpected auto-compaction.
* **Edit Mode Dotted 2D Cells, Explicit `+ Add Widget` Modal -> Live Ghost Preview, & 4-Way Movement Overlay (`Up`, `Down`, `Left`, `Right`) + `Delete`:** In **Edit Dashboard Grid** mode, the grid displays `1×1` dotted cell outlines (`data-testid="dashboard-empty-grid-cell-${col}-${row}"`) across all unoccupied coordinates plus one full trailing empty row at the bottom (with zero hover-only `+` buttons inside empty cells, preserving Daniel Jackson's visibility/affordance principles). Clicking the always-visible **`+ Add Widget`** button (`data-testid="dashboard-add-widget-btn"`) opens the **Widget Selection Modal** (`data-testid="dashboard-widget-picker-modal"`). Selecting a widget closes the modal and enters **Placement Preview Mode**, rendering a live `width × height` ghost preview (`data-testid="dashboard-widget-placement-preview"`, `data-preview-col`, `data-preview-row`) initialized at the first available non-overlapping `(col, row)` coordinate, moving to any hovered empty cell where its `width × height` footprint fits within bounds without overlapping existing widgets, and placing the widget on click. Placed widgets in Edit Mode are greyed out and non-interactive (`pointer-events: none` over the `<iframe sandbox="allow-scripts">`) with an overlay on top containing **Up**, **Down**, **Left**, **Right** (enabled only when stepping 1 cell into an unoccupied footprint within grid bounds), and **Delete** buttons.
* **2D Coordinate Array + Role Defaults (`core.db`) & Personal vs. School-Wide Editor Separation:** [`DashboardWidgetGridPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/DashboardWidgetGridPanel.tsx) persists the dashboard layout in `core.db` (`dashboard_layouts` table in [`schema/core/00_system_and_academic_cycle.sql`](file:///home/kieren/Documents/code/optimis/schema/core/00_system_and_academic_cycle.sql), backed by [`internal/dal/dashboard_layouts.go`](file:///home/kieren/Documents/code/optimis/internal/dal/dashboard_layouts.go) and [`internal/host/dashboard_layouts.go`](file:///home/kieren/Documents/code/optimis/internal/host/dashboard_layouts.go)) as a **JSON array of `{ plug_id, widget_id, col, row }` objects** (plus `column_count`). The **Edit Dashboard Grid** button on the Home view (`/`) strictly edits the logged-in user's own personal dashboard (`scope_type = 'STAFF'`, `GET`/`PUT /api/dashboard/layout`, documented in [`how-to-edit-your-personal-home-dashboard-grid.md`](file:///home/kieren/Documents/code/optimis/user-stories/core/how-to-edit-your-personal-home-dashboard-grid.md))—even when logged in as an `ADMIN`. School-wide dashboard governance—configuring default widget layouts per role (`ROLE_DEFAULT`), appending widgets across existing staff layouts, or editing/resetting other staff members' dashboards—is handled via a dedicated **Admin Dashboard Editor Plugin** (`uk.org.optimis.admin.dashboard_editor`, documented in [`user-stories/admin_dashboard_editor_plugin/`](file:///home/kieren/Documents/code/optimis/user-stories/admin_dashboard_editor_plugin/GEMINI.md)). If a widget in a user's saved layout belongs to a plugin or scope that has since been revoked, quarantined (`UNINSTALLED_DORMANT`), or is unpermitted for the active role, the dashboard renders an **inline permission error card** (`data-testid="widget-permission-error-card-<widget_id>"`) at its `(col, row)` coordinate.

#### Entity Deep-Linking (9C) — Configurable Entity Link Handler Plugins
The Base App maps entity types (e.g., `student`, `group`, `staff`) to a designated handler plugin in `__system_config` (resolved to its active opaque `plug_id` in `GET /api/system/ui-config`). Clicking `<a data-mis-link="student" data-pupil-id="104">` (or `data-mis-plugin="self"`) inside any plugin or widget resolves the target plugin's opaque `plug_id`, constructs a clean route path (`/student/104`) and query object without leaking the source plugin's query parameters, updates the browser URL to `/p/<targetPlugId>/student/104`, wakes the target plugin in the Keep-Alive pool, and dispatches `mis:init` and `mis:route:changed`.

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

### 10.4 Plugin Authoring SDK & Developer Toolchain (Locked & Confirmed — Implemented)

#### Toolchain Binaries (10A) — Separate Purpose-Built Go Binaries (`cmd/`)
Built from the Base Platform repository as independent, single-purpose tools backed by [`internal/sdk`](file:///home/kieren/Documents/code/optimis/internal/sdk/GEMINI.md):
* **[`mis-dev-server`](file:///home/kieren/Documents/code/optimis/cmd/mis-dev-server/main.go) (`cmd/mis-dev-server`):** Local host container + auto-seeded synthetic UK school DB + hot-reload mounting for `src/`, `migrations/`, and `ui/`.
* **[`mis-test`](file:///home/kieren/Documents/code/optimis/cmd/mis-test/main.go) (`cmd/mis-test`):** Headless Lua test runner executing plugin test suites (`tests/*_test.lua`) against an isolated temp SQLite school fixture.
* **[`mis-verify`](file:///home/kieren/Documents/code/optimis/cmd/mis-verify/main.go) (`cmd/mis-verify`):** CI contract & UK GDPR gatekeeper (cross-checking `manifest.json` $\leftrightarrow$ Lua functions $\leftrightarrow$ UI `data-mis-rpc` tags, offline CSP linting, Expand-and-Contract SQL migration checks, and zero-orphan GDPR `export_pupil_data` / `purge_pupil_data` verification).
* **[`mis-pack`](file:///home/kieren/Documents/code/optimis/cmd/mis-pack/main.go) (`cmd/mis-pack`):** Lucide SVG icon extractor (`icon.svg`) + deterministic `.mispkg` archive bundler with `SIGNATURE.json`.

#### VS Code Integration (10B) — Zero-Config `.vscode/` Workspace
Plugin repositories include [`.vscode/settings.json`](file:///home/kieren/Documents/code/optimis/.vscode/settings.json) pre-binding the VS Code Lua extension (`Lua 5.1`, sandboxed globals) to [`.mis-types/dal.d.lua`](file:///home/kieren/Documents/code/optimis/.mis-types/dal.d.lua) (synced with [`sdk/lua/ctx_types.lua`](file:///home/kieren/Documents/code/optimis/sdk/lua/ctx_types.lua)) and binding `manifest.json` to [`sdk/manifest.schema.json`](file:///home/kieren/Documents/code/optimis/sdk/manifest.schema.json) for instant IDE autocomplete and inline linting.

#### Frontend Bridge (10C) — Untyped JSON RPC + Declarative Tags
Lightweight `@optimis/bridge` ([`sdk/packages/bridge`](file:///home/kieren/Documents/code/optimis/sdk/packages/bridge/GEMINI.md), wrapping [`sdk/ui/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js), auto-injecting `--mis-*` CSS variables, binding declarative `data-mis-rpc` / `data-mis-link` attributes, and exposing a `useMisRpc` hook) and `@optimis/tailwind-preset` ([`sdk/packages/tailwind-preset`](file:///home/kieren/Documents/code/optimis/sdk/packages/tailwind-preset/GEMINI.md), wrapping [`sdk/ui/mis-theme.css`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-theme.css)). Zero schema/codegen bloat.

#### Observability in Dev (10D) — Agent-First Structured Stdout Logs
Zero visual inspector HUD. `mis-dev-server` and `mis-test` output single-line structured logs (`type=rpc`, `type=lua_err`, `sqlite_write_hold_ms=`, `type=event_emit`, file/line stack traces) directly to `stdout`, with persona selection controlled via `TEST_DEFAULT_PERSONA` in `.env`, CLI flag (`--persona`), or `POST /api/auth/persona` (`mis_session` cookie) when `ENABLE_DEV_PERSONA_SWITCHER=true` (with zero URL query parameter or HTTP header caller identity overrides). Standalone plugin repositories (populated via [`scripts/populate_plugin_template.mjs`](file:///home/kieren/Documents/code/optimis/scripts/populate_plugin_template.mjs)) bundle pre-compiled `-trimpath` CLI binaries (`./bin/`), `.vscode/`, `.mis-types/`, `sdk/`, and an editable `./schema/core/*.sql` directory (resolved first by [`internal/dal/dal.go`](file:///home/kieren/Documents/code/optimis/internal/dal/dal.go)) so external plugin developers can run `npm run verify`, `npm test`, `npm run dev`, and `npm run pack` locally with zero external dependencies.

---

## 11. Repository Map & Module Index

Each directory maintains its own `GEMINI.md` describing its internal files and responsibilities:

| Module | Directory | Docs Index | Purpose |
| :--- | :--- | :--- | :--- |
| **Local Ops & Quick Start** | [`.`](file:///home/kieren/Documents/code/optimis) | [`README.md`](file:///home/kieren/Documents/code/optimis/README.md) | Local human testing steps, `npm run seed`, plugin preloading, log tailing, and persona cheat sheet. |
| **Agent Rules & Skills** | [`.agents/`](file:///home/kieren/Documents/code/optimis/.agents) | [`.agents/AGENTS.md`](file:///home/kieren/Documents/code/optimis/.agents/AGENTS.md) | Workspace rules, available `npm` commands, and [`skills/`](file:///home/kieren/Documents/code/optimis/.agents/skills). |
| **Server, QA & CLI Binaries** | [`cmd/`](file:///home/kieren/Documents/code/optimis/cmd) | [`cmd/GEMINI.md`](file:///home/kieren/Documents/code/optimis/cmd/GEMINI.md) | Entrypoints for `mis-server`, `mis-seed-qa`, `mis-dev-server`, `mis-test`, `mis-verify`, and `mis-pack`. |
| **Core SQLite Schema** | [`schema/core/`](file:///home/kieren/Documents/code/optimis/schema/core) | [`schema/core/GEMINI.md`](file:///home/kieren/Documents/code/optimis/schema/core/GEMINI.md) | 46-table effective-dated Core DDL (`00`–`06`) and [`SCHEMA_MAP.md`](file:///home/kieren/Documents/code/optimis/schema/core/SCHEMA_MAP.md). |
| **Core Data Access Layer** | [`internal/dal/`](file:///home/kieren/Documents/code/optimis/internal/dal) | [`internal/dal/GEMINI.md`](file:///home/kieren/Documents/code/optimis/internal/dal/GEMINI.md) | `WriteDB`/`ReadDB` WAL pools, `ATTACH` lifecycle, `:own_groups` vs `:all` domain queries, safeguarding vault, and GDPR SAR/purge. |
| **Go Host Container** | [`internal/host/`](file:///home/kieren/Documents/code/optimis/internal/host) | [`internal/host/GEMINI.md`](file:///home/kieren/Documents/code/optimis/internal/host/GEMINI.md) | HTTP/RPC gateway, `gopher-lua` sandbox, opaque `plug_id` routing, Event Bus, egress, pinned recents, and 1D dashboard layouts. |
| **SDK Toolchain Engine** | [`internal/sdk/`](file:///home/kieren/Documents/code/optimis/internal/sdk) | [`internal/sdk/GEMINI.md`](file:///home/kieren/Documents/code/optimis/internal/sdk/GEMINI.md) | Implementation of `mis-verify`, `mis-test`, and `mis-pack`. |
| **Plugin SDK, Schemas & Types** | [`sdk/`](file:///home/kieren/Documents/code/optimis/sdk) | [`sdk/GEMINI.md`](file:///home/kieren/Documents/code/optimis/sdk/GEMINI.md) | `manifest.schema.json`, `mis-bridge.js`, `mis-theme.css`, `@optimis/*` packages, and [`.mis-types/`](file:///home/kieren/Documents/code/optimis/.mis-types/GEMINI.md). |
| **React Host UI Shell** | [`ui/`](file:///home/kieren/Documents/code/optimis/ui) | [`ui/GEMINI.md`](file:///home/kieren/Documents/code/optimis/ui/GEMINI.md) | Embedded React SPA ([`ui/src/`](file:///home/kieren/Documents/code/optimis/ui/src/GEMINI.md), [`ui/src/components/`](file:///home/kieren/Documents/code/optimis/ui/src/components/GEMINI.md), [`ui/src/views/`](file:///home/kieren/Documents/code/optimis/ui/src/views/GEMINI.md)). |
| **Plugins & Fixtures** | [`plugins/`](file:///home/kieren/Documents/code/optimis/plugins) | `plugins/*/GEMINI.md` | First-party, partner, and test fixture `.mispkg` plugin folders including [`plugins/widget_gallery_plugin`](file:///home/kieren/Documents/code/optimis/plugins/widget_gallery_plugin/GEMINI.md) (12 multi-size `1x1`/`2x1`/`1x2`/`2x2` widgets). |
| **Playwright E2E Suites** | [`tests/e2e/`](file:///home/kieren/Documents/code/optimis/tests/e2e) | [`tests/e2e/GEMINI.md`](file:///home/kieren/Documents/code/optimis/tests/e2e/GEMINI.md) | Sequential full-stack Playwright E2E test suites. |
| **Dev & Test Scripts** | [`scripts/`](file:///home/kieren/Documents/code/optimis/scripts) | [`scripts/GEMINI.md`](file:///home/kieren/Documents/code/optimis/scripts/GEMINI.md) | Hot-reload watcher, runtime plugin manager, live log/diagnostic tailing, and standalone plugin template populator (`populate_plugin_template.mjs`). |
| **Reference Docs & Feedback** | [`docs/`](file:///home/kieren/Documents/code/optimis/docs) | [`docs/GEMINI.md`](file:///home/kieren/Documents/code/optimis/docs/GEMINI.md) | Schema blueprints and vertical plugin design feedback ([`docs/plugin_feedback/`](file:///home/kieren/Documents/code/optimis/docs/plugin_feedback/GEMINI.md)). |
| **User Stories Catalog** | [`user-stories/`](file:///home/kieren/Documents/code/optimis/user-stories) | [`user-stories/GEMINI.md`](file:///home/kieren/Documents/code/optimis/user-stories/GEMINI.md) | Domain-partitioned user stories and acceptance criteria across `core/` and `.mispkg` plugin boundaries ([`user-stories/INDEX.md`](file:///home/kieren/Documents/code/optimis/user-stories/INDEX.md)). |
