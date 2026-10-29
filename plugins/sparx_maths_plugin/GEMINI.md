# Sparx Maths Partner Sync Plugin (`plugins/sparx_maths_plugin`)

This directory contains the **Sparx Maths Third-Party EdTech Partner Plugin** (`uk.co.sparx.maths`), implementing the **100% Zero-Fee Lua Plugin SDK Model** (Sections 4, 8 Gap 1, 9, and 10 of the Architecture Blueprint) with an attached SQLite database (`plugin_sparx.db`), explicit HTTPS egress whitelisting (`https://sync.sparxmaths.uk`), Host-managed AES-256-GCM secret envelope encryption (`ctx.crypto.seal_secret` / `ctx.crypto.open_secret`), Admin/HoD-scoped vendor configuration (`plugin.config.write` + `require_roles: ["ADMIN", "HOD"]`), and the **Two-Phase Read/Egress $\rightarrow$ WriteDB** synchronization pattern.

---

## 1. Key Files & Responsibilities

* [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/sparx_maths_plugin/manifest.json):
  * Declares plugin ID `uk.co.sparx.maths`, `database.alias`: `plugin_sparx`, `egress.allowed_domains`: `["https://sync.sparxmaths.uk"]`, `gdpr_hooks` (`export_pupil_data` & `purge_pupil_data`), `configure_vendor_credentials` gated by `scopes: ["plugin.config.write"]` and `require_roles: ["ADMIN", "HOD"]`, and operational endpoints (`get_vendor_config`, `sync_group_homework`, `list_homework_scores`) scoped to `dal.groups.read:own_groups` and `dal.pupils.read:own_groups`.
* [`migrations/001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/sparx_maths_plugin/migrations/001_init.sql):
  * Provisions `sparx_vendor_config` (storing the AES-256-GCM sealed `enc:v1:...` partner API key ciphertext and masked `key_preview`), `sparx_homework_scores` (persisting per-pupil homework completion rate, XP earned, bookwork accuracy, and redactable `teacher_note`), and `sparx_sync_runs` inside `plugin_sparx.db`.
* [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/sparx_maths_plugin/src/init.lua):
  * `configure_vendor_credentials(ctx, args)`: Encrypts the vendor API key via `ctx.crypto.seal_secret(api_key)` (`enc:v1:<base64>`) and commits the sealed envelope into `plugin_sparx.sparx_vendor_config` prior to any outbound sync.
  * `sync_group_homework(ctx, args)`: Implements the **Two-Phase Read/Egress $\rightarrow$ WriteDB** pattern:
    1. **Phase 1 (Read & Egress — Zero Open WriteDB Transaction):** Reads the sealed vendor API key from `plugin_sparx.db` via `ReadDB`, decrypts it via `ctx.crypto.open_secret` (which automatically registers the plaintext secret in the Host's per-invocation `extraSecrets` diagnostic scrubber), verifies `:own_groups` assignment via `ctx.dal.groups.assert_caller_assigned`, queries the teaching group roster via `ctx.dal.pupils.list_by_group`, and executes `ctx.http.post("https://sync.sparxmaths.uk/v1/homework/sync", ...)` while holding zero SQLite write locks.
    2. **Phase 2 (WriteDB Persistence):** Opens a `WriteDB` transaction only after the HTTP response returns, batch-inserts returned homework completion scores into `plugin_sparx.sparx_homework_scores` via `ctx.plugin_db.exec_batch`, and records `READ_EGRESS_THEN_WRITEDB_OK` in `plugin_sparx.sparx_sync_runs`.
  * `export_pupil_data(ctx, args)` & `purge_pupil_data(ctx, args)`: UK GDPR Subject Access Request (SAR) export (`redactable_fields = { "teacher_note" }`) and zero-orphan retention purge hooks.
* [`ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/sparx_maths_plugin/ui/dist/index.html):
  * Offline-bundled sandboxed UI (`<iframe sandbox="allow-scripts">`, `connect-src 'none'` CSP) with declarative `data-mis-rpc="configure_vendor_credentials"` and `data-mis-rpc="sync_group_homework"` bindings and `data-mis-link="student"` entity deep-links.
