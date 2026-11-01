# Go Internal Packages (`internal`)

This directory houses the private Go packages forming the Optimis Host Container and Core Data Access Layer (DAL).

## Subpackages

* [`internal/dal`](file:///home/kieren/Documents/code/optimis/internal/dal/GEMINI.md): Versioned Core Data Access Layer (`core.db` in WAL mode with separated `WriteDB` and `ReadDB` connection pools, `ConnectHook` per-plugin database attachment, effective-dated temporal filtering, `:own_groups` vs `:all` RBAC scoping, Parental Responsibility & court-order safeguarding enforcement, AES-256-GCM safeguarding vault encryption, statutory DfE School Census rollups, ordered 1D dashboard layout persistence, and UK GDPR SAR export & retention purge queries).
* [`internal/host`](file:///home/kieren/Documents/code/optimis/internal/host/GEMINI.md): Go Host Runtime executing the HTTP/RPC gateway, capability-sandboxed `gopher-lua` VM pool, per-plugin SQLite provisioning and snapshot-protected migrations, deterministic XML/XSD builder & validator, feature-flagged external sync & Mock DfE COLLECT services, `ctx.http` Egress Engine, Official Sandboxed UI SDK & Theme Token server, opaque `plug_id` routing, per-persona `pinned_recents` & 1D dashboard layout endpoints, session/persona resolution, DSL safeguarding vault endpoints, UK GDPR SAR & purge orchestration, and in-memory post-commit Event Bus.
* [`internal/sdk`](file:///home/kieren/Documents/code/optimis/internal/sdk/GEMINI.md): Section 10A Plugin Authoring SDK & Developer Toolchain engine powering `mis-verify`, `mis-test`, and `mis-pack`.
