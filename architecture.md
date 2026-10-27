This setup works remarkably well, and yes—it mirrors two established software engineering practices: **Architecture Decision Records (ADRs)** and **Design Reviews (or Red Teaming)**.

In industry, documenting the context, the options considered, the final decision, and the trade-offs is the exact structure of an **ADR** (popularized by Michael Nygard). Splitting the collaborator role into **Explorer** (divergent thinking: surfacing patterns, edge cases, and options) and **Challenger** (convergent thinking: stress-testing assumptions, pointing out hidden costs, compliance traps, and failure modes) prevents groupthink and catches architectural dead-ends early.

Here is how we will run the loop for each stage of the School Management Information System (MIS):

1. **The Explorer Phase:** I will lay out the architectural or product options for the current topic (e.g., tenancy model, data schema, integrations, security/safeguarding), highlighting the pros and cons of each.

2. **Your Call:** You propose a direction or ask questions.

3. **The Challenger Phase:** Before we lock it in, I will push back on your choice—pointing out potential bottlenecks, edge cases (like mid-year timetable changes or complex custody arrangements), or operational headaches to make sure you are comfortable with the trade-offs.

4. **The ADR Summary:** Once you confirm the decision, I will generate a clean **Architecture Decision Record** logging the options, the chosen path, and the rationale, then introduce the next design pillar.

### **Proposed Design Pillars (Our Roadmap)**

Unless you want to change the order, a logical sequence for a School MIS high-level design is:

1. **Scope & Tenancy Model:** Target market (single school, Multi-Academy Trusts / districts, K-12 vs. Higher Ed) and SaaS multi-tenancy vs. isolated instances.

2. **Core Architecture & Data Model:** Monolith vs. microservices, database choices, and handling temporal data (academic years, historical rollbacks, and audit trails).

3. **Core Domain Modules:** Student records, complex family/guardian structures, timetabling, attendance, assessment, and behaviour/safeguarding.

4. **Integration & Interoperability:** Third-party ecosystem (Wonde, Clever, OneRoster), LMS sync, cashless catering, and statutory government reporting (e.g., DfE School Census).

5. **Security, Privacy & Compliance:** Role-Based Access Control (RBAC), GDPR/data residency, safeguarding lockdowns, and parent/student portal isolation.

| Field | Detail |
| :---- | :---- |
| **Status** | Locked & Confirmed |
| **Target Market** | UK All-Through (Primary, Secondary, All-Through K-12), serving both MATs and standalone/Local Authority schools. |
| **Tenancy & Compute** | Siloed Single-Tenant Containers (1 App Container \+ 1 Isolated DB per School) with Scale-to-Zero & Predictive Autoscaling. |
| **Network Enclave** | **School Container:** Hosted on LGfL private WAN / dark fibre (zero public internet ingress). Remote staff access strictly via VPN into the private enclave. Evening maintenance window for updates/backups. **Parent Container:** Public Cloud DMZ hosting sanitized read-projections and an untrusted write-inbox. |
| **External Sync** | 100% Outbound School-Initiated Pull/Push. The School Container pulls MAT policies (taxonomies, grading schemas) and pulls queued parent submissions from the DMZ inbox. It pushes parent read-projections, MAT analytics, and DfE returns outbound. |
| **Authentication** | Federated SSO via the school's native Microsoft Entra ID or Google Workspace. |

| Field | Detail |
| :---- | :---- |
| **Status** | Locked & Confirmed |
| **Database Engine** | In-Process SQLite with PRAGMA journal\_mode=WAL;, synchronous=NORMAL;, and busy\_timeout=5000; mounted on persistent local block storage. |
| **Compute Baseline** | Predictive Autoscaling with a **Scale-to-Minimal** floor (maintaining the warm SQLite WAL lock on local disk rather than scaling to zero replicas). |
| **Plugin Runtime & Governance** | In-Process Capability-Sandboxed **Lua VM** (io/os/network stripped). Rather than rigid instruction-count termination, the host DAL monitors lock contention and execution duration per plugin—alerting admins and circuit-breaking (disabling) plugins that repeatedly hold up the database. |
| **Data Access Layer (DAL)** | Scoped, Versioned DAL injected as a host object into the Lua environment. Minor DB migrations maintain backwards compatibility via Expand-and-Contract; major breaking changes bump the DAL major version. |
| **Plugin Storage** | Isolated per-plugin .sqlite files mounted dynamically via ATTACH DATABASE 'plugin\_\<name\>.db' AS plugin\_\<name\>;. |
| **Temporal Model** | Effective-Dated Core Schema (effective\_from, effective\_to, academic\_year\_id). Complexity is abstracted away by simplified query methods in the DAL. |

| Field | Detail |
| :---- | :---- |
| **Status** | Locked & Confirmed |
| **Core Boundary (core.db)** | Holds all **Statutory**, **Safeguarding**, and **Common Non-Proprietary** school data: Pupils (UPN/ULN), Staff, Family/Guardian relationships, Safeguarding records & court-order blocks, Statutory Attendance (AM/PM sessions), Groups/Timetable Spine, SEN/Medical flags, and the DfE School Census engine. |
| **Plugin Boundary (plugin\_\*.db)** | Holds proprietary or phase-specific feature workflows (e.g., custom Assessment/Gradebook models, Behaviour point schemes, Catering, Trips, Exam board entries). |
| **Deferred Low-Level Models** | Exact SQL table schemas for the Family/Household graph and Group/Timetable spine are deferred to Low-Level Design, with a mandatory Core DAL invariant: the DAL contract must enforce **Parental Responsibility (PR) / court-order safeguarding flags** and resolve all phases down to **AM/PM statutory sessions**. |
| **Event Bus (Pub/Sub)** | Host-managed, asynchronous (post-commit) Event Bus with hop-count limits (preventing infinite loops). The **Core defines the standard domain event catalog**; plugins primarily subscribe to Core events and only emit versioned custom events when opting to share proprietary state. |

| Field | Detail |
| :---- | :---- |
| **Status** | Locked & Confirmed |
| **Third-Party EdTech Strategy** | **100% Zero-Fee Lua Plugin Model.** No third-party data aggregators (e.g., Wonde) or external payment middleware (e.g., ParentPay). Approved partners (e.g., Sparx) build lightweight Lua integration plugins with £0 connection/SDK fees. |
| **LMS & Directory Sync** | Implemented as first-party Lua plugins/workers (plugin\_google\_classroom, plugin\_ms\_teams) listening to Core Event Bus membership changes. |
| **DfE Statutory Census** | **Core Data \+ Hot-Swappable Census Plugin.** Statutory data lives in core.db, while the termly DfE School Census / Workforce Census generators and validation rules are shipped as versioned Lua plugins (backed by host XML/XSD helpers), allowing rule updates without core container redeployment. |
| **Outbound Network & Secrets** | Plugins manage their own vendor credentials within their isolated plugin\_\<name\>.db state. For outbound communication, plugins declare required DAL scopes and target HTTPS domains in their manifest, which the Host Container routes and audits. |

| Field | Detail |
| :---- | :---- |
| **Status** | Locked & Confirmed |
| **Access Control Model (5A)** | **Method-Level Scoped RBAC.** Interactive plugin endpoints and background worker methods declare required DAL scopes (:own\_groups vs :all). Interactive calls verify the **Logged-In User's scopes**; background workers verify the **Admin-approved Plugin Manifest scopes** displayed at installation. |
| **Safeguarding Isolation (5B)** | **Envelope-Encrypted DSL Vault \+ OAuth Cloud Key Renewal.** All staff hold dal.safeguarding.create (write-only postbox). Reading (dal.safeguarding.read) is restricted to DSLs and protected by **envelope encryption** in core.db. DSLs unlock the vault's Master Data Key using a **short-lived, cryptographically signed key** issued by a **Cloud Key Generator** linked to their OAuth account, requiring renewal every $X$ days. If lost or expired, generating a new cloud key seamlessly restores access without data loss. |
| **UK GDPR SAR & Retention (5C)** | **Mandatory Plugin Hooks \+ Admin Redaction UI.** Every plugin attaching a plugin\_\<name\>.db file implements export\_pupil\_data(pupil\_id) and purge\_pupil\_data(pupil\_id). Exports are initiated by a System Admin/DPO and loaded into an **integrated Redaction UI** so third-party names and sensitive notes can be reviewed and redacted before export. |

| Field | Detail |
| :---- | :---- |
| **Status** | Locked & Confirmed |
| **Distribution & Upgrade (6A)** | **Self-Contained .mispkg Bundles.** Manual file upload by the School Admin is the primary installation workflow, with an optional **MAT Sync Feed** allowing MAT-linked school containers to pull pre-approved .mispkg bundles outbound (subject to Admin scope approval if permissions change). |
| **Migration Safety** | Prior to applying any .mispkg SQL migration scripts (migrations/\*.sql), the Host Container takes an automatic file-level snapshot of plugin\_\<name\>.db and executes DDL inside an immediate transaction for zero-downtime automatic rollback on failure. |
| **UI Rendering Model (6B)** | **Sandboxed \<iframe\> Micro-Frontends.** Plugin UIs run inside \<iframe sandbox="allow-scripts"\> (without allow-same-origin) and a strict connect-src 'none' Content Security Policy. All HTML/JS/CSS/font assets must be bundled offline inside the .mispkg archive. |
| **UI-to-Lua Bridge (6C)** | **Declarative Tag Binding \-\> postMessage \-\> Host RPC Gateway.** Plugin UI code binds actions via declarative tags/attributes matching the Lua method names in manifest.json. The bundled Host JS shim intercepts these events, sends a window.postMessage payload to the parent Host UI Shell, and the Host executes the authenticated POST /api/plugins/:id/rpc/:method call against the Lua VM after verifying the user's method-level scopes. |

| Field | Detail |
| :---- | :---- |
| **Status** | Locked & Confirmed |
| **Host Language & Runtime (7A)** | **Go (Golang)** compiled as a single static binary (CGO\_ENABLED=0 where possible). Executes the HTTP/RPC gateway, async Event Bus, background sync workers, and Lua VM pool natively. |
| **Lua & SQLite Engine Integration** | **Pure-Go Lua (yuin/gopher-lua)** targeting Lua 5.1 syntax to eliminate cgo FFI boundary overhead on DAL calls, paired with **coarse-grained batch DAL methods**. SQLite uses two separate Go sql.DB pools: **WriteDB (MaxOpenConns(1))** to serialize writes without lock contention, and **ReadDB (MaxOpenConns(NumCPU))** for concurrent WAL reads, using a ConnectHook to keep ATTACH DATABASE state synchronized across all pooled connections. |
| **Host UI Shell (7B)** | **Embedded React SPA** compiled into the Go binary via embed.FS. Renders the global MIS navigation chrome, authentication flows, DPO plugin-install scope approvals, the Admin SAR Redaction UI, and the window.postMessage RPC coordinator for child plugin iframes. |
| **Configuration & Break-Glass (7C)** | **Minimal Bootstrap \+ Self-Contained SQLite Config.** Only 3 immutable bootstrap variables (DATA\_DIR\_PATH, SCHOOL\_URN, HOST\_KMS\_KEY\_HANDLE) are injected via environment variables; all runtime configuration (OAuth client IDs, MAT sync endpoints, plugin manifests) lives in core.db's \_\_system\_config table for single-folder disaster recovery. The Go binary includes local CLI break-glass subcommands (./mis-server config ...) to repair misconfigured SSO settings without web UI access. |

| Field | Detail |
| :---- | :---- |
| **Status** | Locked & Confirmed |
| **Repository Topology** | **Strict Core vs. Plugin Separation.** The Base Platform repository contains only the Go Host Container, React Shell, Core DAL, and synthetic fixture/test plugins. All production plugins (both first-party and third-party) live in separate repositories. |
| **Local Testbed (8A)** | **Distributable mis-dev-server Binary.** Released alongside the Base Platform, booting the real Go Host Container locally with an auto-seeded **Synthetic UK School core.db** (fake pupils, households, groups, and an in-UI **Persona Switcher** for Teacher, HoD, DSL, Admin) plus hot-reload mounting for the plugin's src/, migrations/, and ui/ folders. |
| **Automated Verification (8B)** | **mis-sdk verify Gatekeeper.** Enforces 4 automated checks in CI before packaging: (1) Cross-checking hand-written manifest.json \<-\> Lua functions \<-\> UI data-mis-rpc tags; (2) CSP/Sandbox linting (connect-src 'none' offline check); (3) Automated UK GDPR test (running migrations on a temp SQLite DB and verifying export\_pupil\_data & purge\_pupil\_data leave zero orphaned pupil rows); (4) Lua test suite & SQLite lock-duration profiling. |
| **Manifest & Packaging (8C)** | **Hand-Written manifest.json (Backed by JSON Schema) \+ mis-sdk pack.** Developers explicitly author manifest.json as the contract source of truth (validated via a published JSON Schema for IDE autocomplete). The Go Host enforces a strict default-deny rule for any Lua function omitted from manifest.json. |

| Field | Detail |
| :---- | :---- |
| **Status** | Locked & Confirmed |
| **Repository Topology** | **Strict Core vs. Plugin Separation.** Base Platform repo holds the Go Host, React Shell, Core DAL, and synthetic fixture plugins. All production plugins live in separate repositories. |
| **Local Testbed (8A)** | **Distributable mis-dev-server Binary.** Boots the real Go Host locally with an auto-seeded **Synthetic UK School core.db** and an in-UI **Persona Switcher** (Teacher, HoD, DSL, Admin) plus hot-reload mounting for src/, migrations/, and ui/. |
| **CI Verification & Packaging (8B/8C)** | **Hand-Written manifest.json \+ mis-sdk verify.** Enforces 4 automated CI gates before producing .mispkg: (1) Cross-checking manifest.json \<-\> Lua functions \<-\> UI data-mis-rpc tags; (2) CSP offline check (connect-src 'none'); (3) Automated UK GDPR test (verifying export\_pupil\_data & purge\_pupil\_data leave zero orphaned rows); (4) Lua test suite & SQLite lock-duration profiling. |
| **Day-2 Error Reporting (Gap 1\)** | **Admin-Exported PII-Scrubbed Diagnostic Bundle.** Zero automated outbound crash telemetry. The Go Host captures Lua stack traces (file, line number, error class, execution/lock wait time—automatically stripping all local variables, RPC request payloads, and pupil IDs) into a local ring buffer. When an issue occurs, the School Admin clicks **"Download Plugin Diagnostic Bundle"** and shares the sanitized .json/.log archive with the vendor. |
| **Live Rollbacks (Gap 2\)** | **Non-Destructive Minor Migrations.** mis-sdk verify enforces Expand-and-Contract SQL rules on minor updates (v1.2 to v1.3 cannot drop/rename columns), allowing Admins to roll back plugin code to v1.2 mid-day without restoring an old DB snapshot or losing morning data. |
| **Offboarding (Gap 3\)** | **Dormant Quarantine State.** Uninstalling a plugin disables its UI, workers, and events immediately, but retains plugin\_\<name\>.db and its read-only GDPR export\_pupil\_data / purge\_pupil\_data hooks in a dormant archive until statutory retention expires. |

PLUGIN CONTRACT

The **Plugin Contract** serves as the strict boundary between the Go Host Container and any first-party or third-party .mispkg plugin. It consists of four non-negotiable surfaces: the **Package & Manifest Specification**, the **Injected Lua Runtime (ctx)**, the **Lifecycle & GDPR Hooks**, and the **Sandboxed UI Bridge**.

## **1\. The .mispkg Bundle & Manifest Contract**

Every plugin is distributed as a signed archive containing four static directories/files (manifest.json, migrations/, src/init.lua, and ui/dist/). The hand-written manifest.json acts as the **Default-Deny Routing & Permission Table**—if a method, scope, event, or domain is not explicitly declared here, the Go Host blocks it at runtime.

| Manifest Section | Contract Rule | Host Enforcement & DPO Visibility |
| :---- | :---- | :---- |
| **id, version, core\_dal\_version** | Reverse-DNS identifier (uk.co.sparx.maths), SemVer plugin version, and compatible Core DAL SemVer range (\~1.4). | Host refuses to install or boot if the container's Core DAL does not satisfy core\_dal\_version. |
| **database.alias** | Optional string (e.g., plugin\_sparx). Must match ^plugin\_\[a-z0-9\_\]+\$. | Host provisions \<alias\>.db, attaches it via ConnectHook, and enforces Expand-and-Contract rules on minor migrations. |
| **egress.allowed\_domains** | Array of explicit HTTPS origins (e.g., \["\[https\://sync.sparxmaths.uk\](https\://sync.sparxmaths.uk)"\]). | Shown to DPO on install. Host blocks any ctx.http call to undeclared domains. |
| **endpoints.\<method\_name\>** | Maps an interactive Lua function name to required DAL scopes (dal.\<domain\>.\<action\>:\<modifier\>) and mutates: true/false. | Checked against the **Logged-In User's scopes** on every RPC request. Determines whether the call uses ReadDB or WriteDB. |
| **workers.\<method\_name\>** | Maps a background Lua function to a Cron schedule or Core Event subscription, plus required DAL scopes. | Checked against the **Plugin Service Principal** approved by the Admin at installation. |
| **gdpr\_hooks** | Required if database.alias is present. Must map export\_pupil\_data and purge\_pupil\_data to exported Lua functions. | Verified by mis-sdk verify in CI and invoked by the Host for Subject Access Requests (SARs) and retention purges. |

## **2\. The Injected Lua Runtime Contract (ctx)**

Plugins execute inside a stripped **Lua 5.1 (gopher-lua)** sandbox (os, io, debug, package, and dofile are removed). Every exported Lua function receives (ctx, args) and must return a serializable Lua table { ok \= true, data \= ... } or throw a standard error string.

The ctx object injected by the Go Host exposes five capability-scoped namespaces:

| Namespace | Provided Capabilities | Contract Guardrails |
| :---- | :---- | :---- |
| **ctx.caller** | Read-only identity metadata: type ("USER" or "SYSTEM"), user\_id, staff\_id, active\_roles, and request\_id. | Immutable. Null/empty staff\_id when invoked via a background worker or event listener. |
| **ctx.dal.\<domain\>** | Versioned, coarse-grained Core data methods (pupils, groups, attendance, assessment, guardians, safeguarding). | **Dynamic Row & Column Filtering:** Only domains declared in the current method's manifest scopes are populated on ctx.dal. If the method scope carries :own\_groups, the Go DAL automatically restricts SQL results to the caller's active temporal teaching groups. |
| **ctx.plugin\_db** | .query(sql, params) and .exec(sql, params) scoped to the plugin's attached plugin\_\<name\>.db. | .exec() is only available if the method declares mutates: true (routing to the single-connection WriteDB pool). Hard execution timeout (e.g., 2,500ms) enforced via Go context.Context to prevent WAL lock starvation. |
| **ctx.events** | .emit(event\_name, payload) for publishing to the Core Event Bus. | Messages are buffered in memory during the Lua call and **only dispatched after the SQLite transaction commits**. Hop count is incremented automatically to prevent infinite plugin loops. |
| **ctx.http** | .get(url, headers) and .post(url, headers, body) for outbound integration sync. | **Disallowed inside write transactions.** Host halts any open DB write transaction before executing outbound HTTP so slow external APIs never hold a local SQLite lock. |

## **3\. Lifecycle & UK GDPR State Contract**

A plugin moves through five deterministic states managed by the Go Host Container:

1. **Pre-Flight & Migration (INSTALLING / UPGRADING):**

   * Host verifies .mispkg signature and checks if any requested scopes changed since the previous version (pausing for Admin/DPO approval if scopes expanded).

   * Host takes an instant file copy of plugin\_\<name\>.db, runs pending migrations/\*.sql inside BEGIN IMMEDIATE, and rolls back the file snapshot automatically if any migration fails.

2. **Active Execution (ACTIVE):**

   * Endpoints, workers, and event listeners are live.

   * **Circuit Breaker Contract:** If a plugin method exceeds the CPU instruction limit or causes repeated SQLITE\_BUSY timeouts, the Host trips the circuit breaker on that specific method and writes a PII-scrubbed stack trace (file, line number, error class, duration—zero variable state or pupil IDs) to the **Local Diagnostic Ring Buffer** for Admin export.

3. **Subject Access Request (SAR\_EXPORT):**

   * When an Admin initiates a SAR for pupil\_id, the Host invokes gdpr\_hooks.export\_pupil\_data(ctx, pupil\_id).

   * **Return Contract:** The function must return a structured array of section tables (title, records, and redactable\_fields indicating free-text columns that may contain third-party pupil names) so the **Admin SAR Redaction UI** can render them uniformly.

4. **Dormant Quarantine (UNINSTALLED\_DORMANT):**

   * When uninstalled by an Admin, the UI, endpoints, and workers are disabled immediately.

   * plugin\_\<name\>.db and the Lua GDPR hooks remain mounted in read-only/purge mode so the school remains compliant with SARs and statutory retention schedules until expiry.

5. **Retention Purge (PURGED):**

   * Invoking gdpr\_hooks.purge\_pupil\_data(ctx, pupil\_id) permanently removes all rows tied to that pupil inside plugin\_\<name\>.db.

## **4\. The Sandboxed UI & RPC Bridge Contract**

Plugin frontends run inside \<iframe sandbox="allow-scripts"\> with a strict connect-src 'none' Content Security Policy. The UI communicates with the Host Container exclusively through declarative HTML attributes or the bundled mis-bridge.js helper over window.postMessage:

* **Initialization Handshake (mis:init):** When the iframe mounts, the React Host Shell sends a read-only initialization payload containing the school's theme tokens (CSS variables), current view context (e.g., currently selected pupil\_id or group\_id if opened as a contextual tab), and the user's permitted UI methods for that plugin.

* **Declarative Invocation (data-mis-rpc):** Elements tagged with data-mis-rpc="\<method\_name\>" and data-mis-args='\<json\>' automatically send a mis:rpc:request message ({ request\_id, method, args }) to the parent React Shell.

* **Host Verification & Response (mis:rpc:response):** The parent React Shell dispatches POST /api/plugins/:plugin\_id/rpc/:method\_name using its own SameSite=Strict session cookie, and posts { request\_id, ok, data, error\_code } back into the iframe.

Moving execution limits into manifest.json makes way more sense than hardcoding arbitrary host ceilings—a DfE School Census plugin crunching 1,500 student histories needs a completely different timeout and memory profile than a simple Behaviour button. Likewise, using native Go channels for the queue and emitting raw base metrics instead of inventing a rigid health state machine keeps the Go host lean.

Here is the stripped-back, streamlined **Runtime & Resource Contract**:

## **1\. Author-Declared Resource & Concurrency Envelope (manifest.json)**

The plugin author declares the operational footprint their plugin (or specific method) requires inside manifest.json, with sensible host fallbacks if omitted. Because it is in the manifest, the School Admin sees the exact resource profile on the installation screen, and the Go Host sets up the gopher-lua context and worker pools to match.

| Manifest Field | Scope | What It Controls | Host Fallback (If Omitted) |
| :---- | :---- | :---- | :---- |
| **max\_concurrency** | Plugin or per-method | Sizes the Go semaphore/worker pool for this plugin (e.g., max 4 concurrent executions of sparx\_sync). | 4 (Interactive), 1 (Workers) |
| **timeout\_ms** | Per-method | Passed directly to context.WithTimeout on the gopher-lua state (L.SetContext). | 2500 (Endpoints), 30000 (Workers) |
| **max\_memory\_mb** | Plugin-wide | Sets the gopher-lua call-stack and registry allocation cap per VM instance. | 32 MB |
| **max\_db\_size\_mb** | database block | Translated to SQLite PRAGMA plugin\_\<alias\>.max\_page\_count. | 500 MB |
| **egress.allowed\_domains** | egress block | Explicit whitelist of HTTPS domains (including redirect targets). Anything else is rejected. | \[\] (Zero outbound access) |
| **egress.timeout\_ms** | egress block | Timeout applied to the Go http.Client for ctx.http calls. | 8000 ms |

*(Note: The single structural safety rule the Host always enforces is **no open SQLite write transactions during an outbound ctx.http fetch**, so external network latency never locks the local database).*

## **2\. Native In-Memory Event Bus (Go Channels)**

Instead of backing the Event Bus with SQLite tables and Dead-Letter Queues, the Host runs a **native Go in-memory pub/sub router** built on buffered Go channels and worker goroutines:

* **Per-Plugin Buffered Channel:** When a plugin subscribes to events in manifest.json, the Go Host allocates a dedicated buffered chan Event for that plugin and spawns N worker goroutines matching the plugin's declared max\_concurrency.

* **Non-Blocking Dispatch:** After a Core or Plugin DB transaction commits, the Host router fans the event out to subscriber channels. If Plugin B is slow or hits its concurrency limit, its channel buffers the work in memory without blocking the caller or slowing down Plugin A.

* **Hop-Count Header:** The Go Event struct carries a simple Hop uint8 counter incremented on re-emission (dropping events at Hop \> 3 to prevent infinite plugin loops).

## **3\. Raw Base Metrics (No Synthesized Health State)**

The Go Host does not compute traffic-light states or run automatic quarantine timers. Instead, the RPC Gateway, Event Router, and SQLite/HTTP wrappers simply record **raw atomic counters and latency histograms** per plugin and per method:

* invocations\_total (by method & status: ok, error, timeout)

* duration\_ms (execution time)

* sqlite\_write\_lock\_wait\_ms & sqlite\_write\_hold\_ms

* http\_egress\_requests\_total & http\_egress\_duration\_ms

* event\_channel\_depth & events\_dropped\_total

* **Diagnostic Ring Buffer:** Last $N$ PII-scrubbed Lua stack traces (file, line number, error string) ready for Admin export.

Any UI dashboards, alerts, or circuit-breaking policies can be derived later directly from these base metrics.  
