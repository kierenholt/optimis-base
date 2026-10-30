# Plugin SDK & Developer Toolchain Engine (`internal/sdk`)

This package implements the core verification, headless Lua test execution, synthetic school testbed cohort seeding, and `.mispkg` packaging logic backing the four `10A` CLI binaries (`cmd/mis-verify`, `cmd/mis-test`, `cmd/mis-pack`, and `cmd/mis-dev-server`).

## Files

* [`verify.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/verify.go):
  * [`VerifyPluginDir`](file:///home/kieren/Documents/code/optimis/internal/sdk/verify.go): Executes the 4 automated CI gates before packaging into `.mispkg` (Sections 8B & 10A):
    1. `MANIFEST_LUA_UI_OK`: Validates `manifest.json`, verifies every declared `endpoints`, `workers`, and `gdpr_hooks` method is a function in `src/init.lua`, and cross-checks all `data-mis-rpc` attributes in `ui/` against `manifest.json` `endpoints`.
    2. `CSP_OFFLINE_OK`: Scans `ui/` HTML/JS/CSS assets for external `http://` or `https://` URLs violating offline `connect-src 'none'` sandbox rules.
    3. `MIGRATIONS_OK`: Enforces Expand-and-Contract SQL migration rules (forbidding `DROP COLUMN`, `RENAME COLUMN`, and `DROP TABLE`) and executes all `migrations/*.sql` inside an immediate SQLite transaction.
    4. `GDPR_ZERO_ORPHANS_OK`: Provisions a temporary headless `Host`, seeds synthetic rows across every attached plugin table containing a `pupil_id` column, invokes `gdpr_hooks.export_pupil_data` & `gdpr_hooks.purge_pupil_data`, and verifies `0` orphaned rows remain.
* [`test_runner.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/test_runner.go):
  * [`SeedSyntheticDevCohort`](file:///home/kieren/Documents/code/optimis/internal/sdk/test_runner.go): Seeds deterministic synthetic staff personas (`stf_admin_01`, `stf_dsl_01`, `stf_senco_01`, `stf_hod_maths`, `stf_tch_7a`, `stf_tch_7b`, `stf_support_01`), teaching/registration groups (`grp_reg_7a`, `grp_reg_7b`, `grp_mat_7x1`), and pupils (`pup_yr7_alice`, `pup_yr7_noah`, `pup_yr7_zara`, `pup_pri_maya`).
  * [`RunPluginTests`](file:///home/kieren/Documents/code/optimis/internal/sdk/test_runner.go): Runs `tests/*_test.lua` suites headlessly against an isolated synthetic `core.db` and attached `plugin_<alias>.db`, emitting single-line structured `stdout` logs (`type=rpc`, `type=lua_err`, `type=event_emit`, `sqlite_write_hold_ms=`).
* [`pack.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/pack.go):
  * [`PackPlugin`](file:///home/kieren/Documents/code/optimis/internal/sdk/pack.go): Runs `VerifyPluginDir`, extracts the declared Lucide SVG icon into `icon.svg`, computes SHA-256 digests into `SIGNATURE.json`, and writes a deterministic `.mispkg` ZIP bundle.
* [`sdk_test.go`](file:///home/kieren/Documents/code/optimis/internal/sdk/sdk_test.go): Unit tests verifying `VerifyPluginDir`, `RunPluginTests`, and `PackPlugin`.
