# Go Internal Packages (`internal`)

This directory houses the private Go packages forming the Optimis Host Container and Core Data Access Layer (DAL).

## Subpackages

* [`internal/dal`](file:///home/kieren/Documents/code/optimis/internal/dal/GEMINI.md): Versioned Core Data Access Layer (`core.db` in WAL mode with separated `WriteDB` and `ReadDB` connection pools, `ConnectHook` per-plugin database attachment, effective-dated temporal filtering, `:own_groups` vs `:all` RBAC scoping, Parental Responsibility & court-order safeguarding enforcement, AES-256-GCM safeguarding vault encryption, and UK GDPR SAR export & retention purge queries).
* [`internal/host`](file:///home/kieren/Documents/code/optimis/internal/host/GEMINI.md): Go Host Runtime executing the HTTP/RPC gateway, capability-sandboxed `gopher-lua` VM pool, per-plugin SQLite provisioning and snapshot-protected migrations, `ctx.http` Egress Engine (`egress.go`), Official Sandboxed UI SDK & Theme Token server (`ui_sdk.go`), session/persona resolution (`persona_auth.go`), DSL safeguarding vault endpoints (`safeguarding_vault.go`), UK GDPR SAR & purge orchestration (`gdpr.go`), and in-memory post-commit Event Bus.
