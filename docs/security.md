# OpenMIS UK: Security & Attack Surface Engineering Analysis

> **Status:** Active Reference Specification  
> **Target Audience:** Security Auditors, AI Agents, Backend Engineers  
> **Related Specifications:** [Architecture](./architecture.md) | [Security](./security.md) | [Performance](./performance.md)

---

## Table of Contents

- [Executive Summary](#executive-summary)
- [1. Browser & Frontend Surfaces](#1-browser--frontend-surfaces)
  - [A. PostMessage RPC Channel](#a-postmessage-rpc-channel)
  - [B. Host DOM Integration Points](#b-host-dom-integration-points)
  - [C. Plugin UI Assets](#c-plugin-ui-assets)
- [2. Network & API Perimeter](#2-network--api-perimeter)
  - [A. Fastify API Gateway Endpoints](#a-fastify-api-gateway-endpoints)
  - [B. Token & Signature Headers](#b-token--signature-headers)
  - [C. Network Egress Broker (context.fetch)](#c-network-egress-broker-contextfetch)
- [3. Execution Environments & Sandboxes](#3-execution-environments--sandboxes)
  - [A. Worker Runtime Boundary (Persistent Daemon & Unix Socket)](#a-worker-runtime-boundary-persistent-daemon--unix-socket)
  - [B. Injected Host Capabilities & Session-Gated DB Access](#b-injected-host-capabilities--session-gated-db-access)
  - [C. Statutory XML Parsers](#c-statutory-xml-parsers)
  - [D. Document Generation Engines](#d-document-generation-engines)
- [4. Database Plane & School Boundary Enforcement](#4-database-plane--school-boundary-enforcement)
  - [A. Database Schema Boundary](#a-database-schema-boundary)
  - [B. Guarded Core DbContext & Zero-Trust RLS](#b-guarded-core-dbcontext--zero-trust-rls)
  - [C. Database Connection Pools & Per-Plugin Role Quotas](#c-database-connection-pools--per-plugin-role-quotas)
  - [D. Object Storage Broker](#d-object-storage-broker)
- [5. Asynchronous Processing & Inter-Process Pipelines](#5-asynchronous-processing--inter-process-pipelines)
  - [A. Task Broker & Queue (pg-boss)](#a-task-broker--queue-pg-boss)
  - [B. Lifecycle & Provisioning Automation (openmis_migrator)](#b-lifecycle--provisioning-automation-openmis_migrator)
- [6. Consolidated Technical Demand Matrix](#6-consolidated-technical-demand-matrix)
- [7. The Unified Security & Execution Model](#7-the-unified-security--execution-model)

---

## Executive Summary

A comprehensive security and structural analysis across the OpenMIS UK single-tenant deployment architecture (dedicated Fastify container and dedicated logical PostgreSQL database per school). This document specifies threat vectors, attack mechanisms, concrete defensive mitigations, and implementation complexity ratings across all platform boundaries.

---

## 1. Browser & Frontend Surfaces

### A. PostMessage RPC Channel

* **Mode of Attack:**
  * **Origin / Handle Spoofing:** An unauthorized iframe or nested window dispatches forged JSON-RPC payloads directly to the host message listener.
  * **Event Flooding (DoS):** An untrusted iframe rapidly dispatches high-frequency messages to saturate the main thread UI event loop, freezing user interactions during time-critical morning attendance roll-calls.
  * **Payload Injection:** Supplying unexpected object trees, prototype-polluting keys (`__proto__`, `constructor`), or malformed arguments inside message payloads to corrupt client application state.
* **Can We Stop It:** Yes.
  * Validate that both `event.origin === 'https://plugins.openmis-static.net'` and `event.source === activeIframe.contentWindow` before handling any message.
  * Enforce strict runtime schema validation using Zod on all incoming message payloads.
  * Apply client-side message rate limiting, debounce resize events to 60fps, and drop bursts exceeding 100 requests/second per iframe.
* **Technical Demand:** Low. Managed via standard browser APIs, origin checks, and lightweight Zod validation schemas.

### B. Host DOM Integration Points

* **Mode of Attack:**
  * **Manifest Metadata Injection (XSS):** Injecting malicious script tags, arbitrary URLs (`javascript:`), or HTML inside UI labels declared in `manifest.json` that execute when rendered natively by React and Tailwind.
  * **UI Redressing / Deceptive Routing:** Registering deceptive sidebar tabs or navigation targets that mimic statutory views to phish credentials or capture sensitive inputs.
* **Can We Stop It:** Yes.
  * Treat all manifest strings strictly as raw text rather than executable markup (interpolate values as standard React children; ban `dangerouslySetInnerHTML`).
  * Validate manifests strictly at installation against an audited Zod schema that permits only safe string patterns and layout enums.
  * Visually badge third-party plugin components to distinguish them clearly from core statutory screens.
* **Technical Demand:** Low. Relies on declarative install-time validation and standard React safe-rendering defaults.

### C. Plugin UI Assets

* **Mode of Attack:**
  * **Client-Side Supply Chain Compromise:** Injecting malicious scripts into the plugin frontend bundle to execute arbitrary JavaScript within the framed sandbox.
  * **Local Asset Tampering:** Modifying bundle assets on host storage or in transit to serve untrusted client code.
* **Can We Stop It:** Yes.
  * Serve framed assets exclusively from an isolated static domain (`plugins.openmis-static.net`).
  * Configure iframe sandboxing with `sandbox="allow-scripts allow-forms"` (omitting `allow-same-origin`, `allow-top-navigation`, and `allow-modals`).
  * Enforce strict Content Security Policy (CSP) headers, pinning `connect-src` exclusively to the designated backend API prefix (`https://api.openmis.uk/api/plugins/<id>/`) and disallowing external domains.
  * Verify content digests (`PackageDigest`) before saving or serving static bundle assets.
* **Technical Demand:** Low. Involves static file hosting, standard CSP header configuration, and SHA-256 integrity verification.

---

## 2. Network & API Perimeter

### A. Fastify API Gateway Endpoints

* **Mode of Attack:**
  * **SSO Callback Exploitation:** State tampering, replay attacks, or redirect URI manipulation across OAuth 2.0 / OIDC integrations (Microsoft Entra ID, Google Workspace) to hijack staff sessions.
  * **Statutory Ingestion Flooding / Malformed Input:** Submitting oversized, deeply nested, or corrupt DfE CTF XML / Census payloads to trigger memory exhaustion or parser deadlocks.
  * **Plugin Route Traversal & Queue Saturation:** Dispatching high-volume mutation bursts to plugin action routes to exhaust local container memory or saturating background task workers.
* **Can We Stop It:** Yes.
  * Enforce PKCE and cryptographic state/nonce verification on all SSO callbacks; restrict redirect targets to hardcoded, immutable allowlists.
  * Configure route-scoped `bodyLimit` thresholds in Fastify combined with native Ajv JSON schema validation.
  * Apply local in-memory rate limiting via `@fastify/rate-limit`.
  * Validate incoming request tokens against the container's static identity (`process.env.SCHOOL_ID`), dropping mismatched requests immediately at the boundary.
* **Technical Demand:** Low. Handled via standard Fastify ecosystem plugins and static environment checks in a single-school container.

### B. Token & Signature Headers

* **Mode of Attack:**
  * **Signature Bypass & Algorithm Confusion:** Supplying tokens signed with `alg: "none"` or exploiting symmetric/asymmetric key confusion to forge arbitrary authentication claims.
  * **Token Replay & Scope Escalation:** Using a captured Plugin Access Token (PAT) outside its designated audience, school identity, or granted permission scopes.
  * **Digest Mismatch Exploitation:** Tampering with or omitting the bundle digest claim to invoke actions against incompatible or unverified worker versions.
* **Can We Stop It:** Yes.
  * Enforce explicit signature algorithms (EdDSA or ES256) and reject unsigned or algorithm-mismatched tokens.
  * Strictly validate claims: assert `token.school_id === process.env.SCHOOL_ID`, verify `token.plugin_id` matches the route parameter, and check expirations against tight clock tolerances.
  * Verify that `token.pkg_digest` matches the recorded hash in `public.active_plugins`.
* **Technical Demand:** Low. Standard cryptographic JWT verification using audited libraries within Fastify pre-handler hooks.

### C. Network Egress Broker (`context.fetch`)

* **Mode of Attack:**
  * **Server-Side Request Forgery (SSRF):** Dispatching HTTP requests toward private cloud infrastructure, link-local metadata endpoints (`169.254.169.254`), or internal container/database interfaces.
  * **DNS Rebinding:** Fast-flux DNS changes between initial hostname checks and socket establishment to evade domain allowlists.
  * **Data Exfiltration:** Exfiltrating sensitive student records or directories via outbound HTTP requests to undeclared third-party servers.
* **Can We Stop It:** Yes.
  * Validate destination URLs against the auditable outbound allowlist declared in `manifest.json`.
  * Intercept socket connection resolution via Node's native HTTP dispatcher (`undici.Agent` `connect.lookup`):
    1. Resolve the hostname once via `dns.resolve`.
    2. Parse the resulting IP and reject non-unicast CIDRs, private RFC-1918/RFC-4193 subnets, loopback addresses, and `169.254.169.254`.
    3. Pin the outbound socket directly to the validated IP address to neutralize DNS rebinding.
  * Enforce HTTPS strictly and disable automatic redirect following (`redirect: 'manual'`).
  * Restrict cloud hypervisor metadata access (e.g., set the AWS ECS metadata hop limit to 1).
* **Technical Demand:** Low. Managed entirely in pure TypeScript using native `undici.Agent` dispatcher hooks without requiring C++ compilation or dedicated forward proxy sidecars.

---

## 3. Execution Environments & Sandboxes

### A. Worker Runtime Boundary (Persistent Daemon & Unix Socket)

* **Mode of Attack:**
  * **Resource Starvation (Compute/Memory DoS):** Running unbounded recursion, CPU loops, or high-allocation arrays inside the plugin process to exhaust host memory or monopolize CPU cores.
  * **Process Crash Loops (Poison Pills):** Submitting tasks that trigger fatal unhandled exceptions, process exits, or out-of-memory errors to repeatedly crash worker runners.
  * **OS Inspection & Secret Leaking:** Reading `process.env` to harvest host secrets, database superuser credentials, or accessing sensitive files outside the plugin's working directory.
* **Can We Stop It:** Yes.
  * Execute workers as persistent child process daemons communicating over local Unix Domain Sockets (`/run/openmis/plugin-<id>.sock`) with explicit Node.js resource constraints (`--max-old-space-size=64`).
  * Strip ambient environment variables at spawn time: pass an explicit dictionary containing only non-sensitive execution parameters (`PLUGIN_ID`, `SCHOOL_ID`), preventing host secret exposure.
  * Enforce hard execution deadlines (500ms CPU budget per socket payload, 60s for background queues) enforced by the parent supervisor process using kernel-level `child.kill('SIGKILL')`.
  * Enforce the automated "Three-Strikes" circuit breaker in the school database: suspend plugins that trigger repeated abnormal exits before retry storms accumulate.
* **Technical Demand:** Low. Standard built-in Node.js runtime APIs without native C++ compilation or fragile in-process memory bridges.

### B. Injected Host Capabilities & Session-Gated DB Access

* **Mode of Attack:**
  * **Cross-Schema SQL Breakout / Unauthorized Table Mutation:** Plugins attempting to read from or write to statutory core schemas (`public.pupils`, `public.attendance_marks`, `public.pupil_safeguarding_notes`).
  * **Scope & Projection Overreach:** Plugins attempting to query expanded student profiles beyond minimal directory fields.
* **Can We Stop It:** Yes.
  * **Dynamic PostgreSQL Role Boundary:** The daemon connects directly to PostgreSQL using a dedicated, unprivileged role (`role_plugin_<id>`). This role has all permissions on statutory core tables revoked, possessing column-restricted `SELECT` solely on authorized core columns (`id`, `upn`, `first_name`, `last_name`, `year_group`).
  * **Pre-Registered Session Table & RLS:** The host Fastify gateway registers authenticated user sessions into `auth.active_plugin_sessions` (TTL 5 min). Database queries execute against Row-Level Security policies calling `auth.is_session_valid()`, validating scopes at PostgreSQL query plan-time.
  * **Safeguarding Quarantine:** Child protection records (`pupil_safeguarding_notes`) are unmapped in application facades and structurally blocked by PostgreSQL role permissions, making safeguarding data unreachable.
* **Technical Demand:** Low. Leverages standard PostgreSQL role grants, UNLOGGED session registries, and native RLS policies.

### C. Statutory XML Parsers

* **Mode of Attack:**
  * **XML External Entity (XXE) & DTD Attacks:** Processing DfE Common Transfer Files (CTF) or Census returns containing external DTD references to read local system files or initiate SSRF requests.
  * **Recursive Entity Expansion (Billion Laughs):** Submitting nested entity references to trigger exponential memory allocation during parsing.
  * **Parser Deserialization & Crash Bugs:** Submitting malformed XML structures designed to crash native parser implementations.
* **Can We Stop It:** Yes.
  * Disable external entity resolution (`resolveEntities: false`), DTD processing, and network lookups in `fast-xml-parser` and `libxml-wasm`.
  * Enforce strict input byte limits on incoming statutory XML files prior to parsing.
  * Validate XML files against offline DfE XSD schemas using pure WebAssembly runtimes before domain processing.
* **Technical Demand:** Low. Standard configuration of modern JavaScript/WASM XML parsers without native C++ compilation toolchains.

### D. Document Generation Engines

* **Mode of Attack:**
  * **PDF Resource Exhaustion:** Supplying deeply nested layouts, infinite-page structures, or unbounded tabular data to trigger memory spikes during in-memory rendering.
  * **Local Resource / File Inclusion:** Exploiting asset loaders inside `@react-pdf/renderer` or Typst to access host file paths or private network resources.
* **Can We Stop It:** Yes.
  * Enforce strict pagination, row count, and data size limits on inputs before initiating rendering.
  * Restrict asset embedding: disable external network resolution for fonts and images; bundle all required styling assets statically inside the container image.
  * Offload heavy document rendering to lower-priority background workers governed by memory caps.
* **Technical Demand:** Low. Requires configuring isolated document asset loaders and enforcing pagination constraints on input data.

---

## 4. Database Plane & School Boundary Enforcement

### A. Database Schema Boundary

* **Mode of Attack:**
  * **Dynamic Identifier Injection:** Injecting malformed table or column names via client payloads to trick query builders into escaping plugin schemas into core statutory schemas (`public.pupils`, `public.attendance_marks`).
  * **Table Shadowing:** A plugin declaring a table identical to a core entity to exploit unqualified PostgreSQL search paths.
* **Can We Stop It:** Yes.
  * Enforce declarative table and column whitelisting validated against the plugin's static manifest via Zod.
  * Reject reserved PostgreSQL system keywords and internal identifiers during manifest registration.
  * Explicitly qualify every generated SQL statement with the schema prefix (`plugin_<id>.<table_name>`) rather than relying on session-level search paths.
* **Technical Demand:** Low. Managed via declarative schema validation and explicit table qualification in Drizzle ORM.

### B. Guarded Core DbContext & Zero-Trust RLS

* **Mode of Attack:**
  * **Scope Spoofing:** Supplying unverified capability scopes (e.g., claiming `safeguarding:read:dsl`) to read restricted child protection records.
  * **Perimeter Omission:** An LLM- or developer-generated Fastify route omitting perimeter middleware (`requireScopes`), leaving statutory methods exposed to unauthenticated callers.
  * **Privilege Creep:** Exploiting background workers or internal saga workflows executing without user context to trigger sensitive database mutations.
* **Can We Stop It:** Yes.
  * Eliminate ambient/global `DbContext` instances; require an authenticated, verified `CallerContext` to construct `GuardedCoreDbContext`.
  * Enforce capability checks directly in front of database operations (`GuardedCoreDbContext.assertScope`) so queries fail closed even if HTTP perimeter hooks are omitted.
  * Structurally unmap child protection tables (`pupil_safeguarding_notes`) from plugin interfaces.
  * Apply the effective scope resolution formula:

$$\text{Effective Scopes} = \text{Manifest Scopes} \cap \text{User Capabilities}$$

* **Technical Demand:** Low. Relies on architectural discipline and unit-tested wrapper classes wrapping Drizzle database instances.

### C. Database Connection Pools & Per-Plugin Role Quotas

* **Mode of Attack:**
  * **Session State Poisoning:** A plugin execution issuing `SET search_path` or modifying transaction settings, leaving the connection in an altered state for subsequent queries.
  * **Advisory Lock / Connection Starvation:** Leaving uncommitted transactions or dangling locks to starve critical attendance marks or safeguarding writes.
* **Can We Stop It:** Yes.
  * Bind engine-level `CONNECTION LIMIT <n>` (e.g., 2–5) directly to each dynamic role (`role_plugin_<id>`).
  * Execute `DISCARD ALL` (or pool connection reset hooks) whenever a connection returns to the pool to clear temporary tables and session variables.
  * Enforce strict client query timeouts (`statement_timeout` and `idle_in_transaction_session_timeout`) on all PostgreSQL roles.
* **Technical Demand:** Low. Standard database connection pool configuration and explicit schema-qualified queries.

### D. Object Storage Broker

* **Mode of Attack:**
  * **Path Traversal Key Injection:** Supplying directory traversal sequences (`../../`) inside file path parameters to generate presigned upload/download URLs targeting host assets or other schools.
  * **Stale Presigned Access:** Reusing unexpired presigned upload/download URLs after a school plugin has been uninstalled.
  * **Storage Quota Flooding:** Bypassing size limits by uploading unbounded binary payloads via unconstrained presigned upload URLs.
* **Can We Stop It:** Yes.
  * Isolate plugin paths under cryptographically secure GUID prefixes resolved strictly server-side:

```
s3://openmis-storage/schools/{school_id}/plugins/{plugin_id}-{guid}/
```

  * Sanitize target file names by stripping path separators (`/` and `\`) before concatenating with the prefix.
  * Issue short-lived presigned URLs (e.g., 5-minute expiry) and enforce strict HTTP `Content-Length` bounds inside storage policy conditions.
  * Execute atomic recursive prefix purges on teardown before dropping database schemas.
* **Technical Demand:** Low. Standard cloud storage SDK configuration and path sanitization.

---

## 5. Asynchronous Processing & Inter-Process Pipelines

### A. Task Broker & Queue (`pg-boss`)

* **Mode of Attack:**
  * **Queue Starvation & Resource Exhaustion:** Flooding plugin worker queues with high-volume tasks to exhaust PostgreSQL connection pools or cause lock contention on shared job tables.
  * **Cross-Queue / Action Tampering:** Forging job payloads or exploiting enqueue parameters to publish tasks into queues outside the plugin's assigned namespace.
  * **Poison Pill & Retry Storms:** Enqueuing malformed or crashing payloads that exhaust retry quotas repeatedly and tie up background execution threads.
* **Can We Stop It:** Yes.
  * Enforce queue segregation: allocate dedicated, priority-bounded queues for statutory workloads with independent worker pools and timeouts.
  * Strictly derive enqueue targets on the gateway using verified token identity (`plugin:${token.plugin_id}`) rather than accepting arbitrary queue names from the caller.
  * Enforce hard task concurrency caps, school backlog limits (e.g., maximum 500 active tasks per school plugin), and set strict execution deadlines (60 seconds) with dead-letter queue routing for repeated failures.
* **Technical Demand:** Low. Configurable via standard `pg-boss` queue settings, partition options, and gateway enqueue validation.

### B. Lifecycle & Provisioning Automation (`openmis_migrator`)

* **Mode of Attack:**
  * **DDL Injection via Manifest Metadata:** Supplying crafted table names, column names, or default constraints in `manifest.json` that escape intended DDL generation to execute arbitrary SQL commands during schema creation.
  * **Privilege Escalation via Migrator Role:** Exploiting the migrator worker (which possesses elevated DDL capabilities) to modify core public tables, grant unauthorized database permissions, or alter statutory audit logs.
  * **Orphaned Storage / Resource Leakage:** Triggering interrupted deprovisioning cycles where database schemas are dropped but associated S3 object prefixes or background tasks persist, leaking storage or leaving dangling references.
* **Can We Stop It:** Yes.
  * Enforce strict regex validation via Zod on manifests: restrict table and column names to alphanumeric snake_case (`^[a-z0-9_]{1,63}$`), strictly rejecting SQL keywords, punctuation, and system identifiers.
  * Use parameterized Drizzle DDL constructs with hardcoded double-quoted identifiers rather than raw string concatenation.
  * Restrict `openmis_migrator`'s database privileges to schema-level creation (`CREATE ON DATABASE school_<id>`) and explicit ownership of `plugin_<id>`, denying write access to core public statutory schemas.
  * Execute deprovisioning as a two-phase saga: complete object storage recursive prefix purging before running `DROP SCHEMA plugin_<id> CASCADE; DROP ROLE role_plugin_<id>;`.
* **Technical Demand:** Low. Eliminates custom migration diff engines in favor of declarative regex validation, standard Drizzle DDL primitives, and sequential teardown sagas.

---

## 6. Consolidated Technical Demand Matrix

| Section | Boundary / Attack Surface | Primary Mitigation | Technical Demand |
| --- | --- | --- | --- |
| **1. Frontend** | PostMessage RPC Channel | Origin + window verification; Zod schema validation | **Low** |
|  | Host DOM Integration | Strict manifest validation; text interpolation (no raw HTML) | **Low** |
|  | Plugin UI Assets | Sandboxed iframes; dedicated origin; strict CSP | **Low** |
| **2. Perimeter** | Fastify API Gateway | Route-scoped body limits; static school ID checks | **Low** |
|  | Token & Signature Headers | Cryptographic EdDSA/ES256 verification; digest binding | **Low** |
|  | Network Egress Broker | Native `undici.Agent` IP lookup pinning; SSRF blocking | **Low** |
| **3. Sandboxes** | Worker Runtime Boundary | Persistent Unix socket daemon; stripped env; SIGKILL timeouts | **Low** |
|  | Injected Capabilities | Restricted role (`role_plugin_<id>`); pre-registered session RLS | **Low** |
|  | Statutory XML Parsers | Disabled DTD/XXE in `fast-xml-parser` and `libxml-wasm` | **Low** |
|  | Document Generation | Input pagination caps; static asset bundling | **Low** |
| **4. Database** | Database Schema Boundary | Zod manifest validation; schema-qualified Drizzle queries | **Low** |
|  | Guarded Core DbContext | DbContext-level capability assertion; unmapped safeguarding | **Low** |
|  | Connection Pools | Engine `CONNECTION LIMIT`; `DISCARD ALL` on checkout | **Low** |
|  | Object Storage Broker | Randomized GUID prefixes; short-lived presigned URLs | **Low** |
| **5. Pipelines** | Task Broker (`pg-boss`) | Queue segregation; token-bound enqueue; concurrency caps | **Low** |
|  | Migrator Automation | Regex-validated identifiers; scoped DDL grants; teardown sagas | **Low** |

---

## 7. The Unified Security & Execution Model

The entire architecture rests on a single, coherent security and execution model:

* **Approve at Gateway, Enforce at Database:** Fastify performs JWT/PAT verification, identity pinning (`process.env.SCHOOL_ID`), and rate limiting at the perimeter. PostgreSQL acts as the unbypassable gatekeeper at the bottom: if code attempts to read or mutate a column outside its granted scopes or session context, the database engine aborts the query immediately.
* **Column-Level Scopes Mapped Directly to PostgreSQL Grants & RLS:**
  * `core:pupil:identity:read` -> `GRANT SELECT (id, upn, first_name, last_name, year_group) ON public.pupils TO role_plugin_<id>;`
  * `safeguarding:alert:create` -> Core Fastify host handles statutory safeguarding writes directly (`public.pupil_safeguarding_notes`), bypassing third-party roles entirely.
  * Destructive rights (`UPDATE`, `DELETE`, `TRUNCATE`) on statutory logs remain permanently revoked, providing append-only immutability out of the box.
* **Zero-Overhead Runtime:**
  * No C++ isolate compilation dependencies (`isolated-vm` dropped).
  * Persistent Unix socket IPC eliminates per-request startup latency.
  * Session validation performed via fast UNLOGGED table lookups (`<0.05 ms`).
  * No proxy sidecar containers (SSRF/DNS rebinding handled natively via `undici.Agent` `connect.lookup`).
* **Pluggable Statutory Workflows:** Core state tracks `public.workflow_bindings`. School admins select the provider plugin in settings, and the gateway delegates the task to standard `pg-boss` queues or daemon child processes using pure Drizzle SQL queries.
