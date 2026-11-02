# Plugin SDK & Developer Toolchain Engine (`internal/sdk`)

This package implements the core verification, headless Lua test execution, synthetic school testbed cohort seeding, and `.mispkg` packaging logic backing the four `10A` CLI binaries (`cmd/mis-verify`, `cmd/mis-test`, `cmd/mis-pack`, and `cmd/mis-dev-server`).

## Files

* [`verify.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/verify.go):
  * [`VerifyPluginDir`](file:///home/kieren/Documents/code/optimis/internal/sdk/verify.go): Orchestrates the 4 automated CI gates before packaging into `.mispkg` (Sections 8B & 10A).
  * Offline CSP Gate (`CSP_OFFLINE_OK`): Scans `ui/` HTML/JS/CSS assets for external `http://` or `https://` URLs violating offline `connect-src 'none'` sandbox rules, and rejects non-monochrome emoji/dingbat glyphs (`NON_MONOCHROME_ICON_EMOJI_VIOLATION`).
  * Migrations Gate (`MIGRATIONS_OK`): Enforces Expand-and-Contract SQL migration rules (forbidding `DROP COLUMN`, `RENAME COLUMN`, and `DROP TABLE`) and executes all `migrations/*.sql` inside an immediate SQLite transaction.
* [`verify_manifest.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/verify_manifest.go):
  * [`ReadAndValidateManifest`](file:///home/kieren/Documents/code/optimis/internal/sdk/verify_manifest.go): Validates `manifest.json` schema, reverse-DNS ID, SemVer, endpoints, monochrome Lucide `icon` catalog validation via `IsValidLucideIcon` (rejecting unknown icons with `UNKNOWN_LUCIDE_ICON`), `views[]` and `widgets[]` `keywords: string[]` validation, and strict `widgets[]` preset `"width": 1 | 2` and `"height": 1 | 2` enforcement.
  * Manifest, Lua & UI Gate (`MANIFEST_LUA_UI_OK`): Initializes a sandboxed Lua 5.1 state (`newSandboxedLuaState`), verifies every declared `endpoints`, `workers`, and `gdpr_hooks` method is a function in `src/init.lua`, cross-checks all `data-mis-rpc` attributes in `ui/` against `manifest.json` `endpoints`, and validates all `data-mis-icon` attributes in `ui/` against `IsValidLucideIcon`.
* [`verify_gdpr.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/verify_gdpr.go):
  * UK GDPR Zero-Orphan Gate (`GDPR_ZERO_ORPHANS_OK`): Provisions a temporary headless `Host`, discovers all attached plugin tables containing a `pupil_id` column, inspects `PRAGMA <alias>.foreign_key_list(<table>)` (`inspectTableForeignKeys`) to recursively seed synthetic parent rows (`ensureTableSeeded`) before inserting synthetic `pupil_id` rows (`insertSyntheticPupilRow`), invokes `gdpr_hooks.export_pupil_data` & `gdpr_hooks.purge_pupil_data`, and verifies `0` orphaned rows remain.
* [`dev_cohort.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/dev_cohort.go):
  * [`SeedSyntheticDevCohort`](file:///home/kieren/Documents/code/optimis/internal/sdk/dev_cohort.go): Seeds deterministic synthetic staff personas (`stf_admin_01`, `stf_dsl_01`, `stf_senco_01`, `stf_hod_maths`, `stf_tch_7a`, `stf_tch_7b`, `stf_support_01`), teaching/registration groups (`grp_reg_7a`, `grp_reg_7b`, `grp_mat_7x1`), and pupils (`pup_yr7_alice`, `pup_yr7_noah`, `pup_yr7_zara`, `pup_pri_maya`).
* [`test_runner.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/test_runner.go):
  * [`RunPluginTests`](file:///home/kieren/Documents/code/optimis/internal/sdk/test_runner.go): Runs `tests/*_test.lua` suites headlessly against an isolated synthetic `core.db` and attached `plugin_<alias>.db`, emitting single-line structured `stdout` logs (`type=rpc`, `type=lua_err`, `type=event_emit`, `sqlite_write_hold_ms=`).
* [`pack.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/pack.go):
  * [`PackPlugin`](file:///home/kieren/Documents/code/optimis/internal/sdk/pack.go) & [`IsValidLucideIcon`](file:///home/kieren/Documents/code/optimis/internal/sdk/pack.go): Runs `VerifyPluginDir`, validates and extracts the declared monochrome Lucide SVG icon (`stroke="currentColor" fill="none" data-lucide-icon="<name>"`) into `icon.svg`, computes SHA-256 digests into `SIGNATURE.json`, and writes a deterministic `.mispkg` ZIP bundle.
* [`sdk_test.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/sdk_test.go): Unit tests verifying `VerifyPluginDir`, `RunPluginTests`, and `PackPlugin`.
