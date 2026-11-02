/**
 * Documentation & Agent Guidance Templates for Optimis Plugin Starter
 */

export const DOC_TEMPLATES = {
  readmeMd: () => `# Optimis School MIS — Plugin Starter Template

A zero-install starter repository for building, testing, verifying, and packaging capability-sandboxed **\`.mispkg\`** plugins for the **Optimis School MIS** platform.

Everything you need is included directly in this repository:
* **Pre-compiled Toolchain Binaries (\`./bin/\`)** for Linux (\`amd64\`), macOS (\`arm64\` & \`amd64\`), and Windows (\`amd64\`) — no Go compiler or external server installation required.
* **Editable Local Core SQL Schema (\`./schema/core/\`)** — inspect or experiment with the 46-table UK All-Through School SQLite schema locally.
* **Instant VS Code Autocomplete (\`.vscode/\` + \`.mis-types/dal.d.lua\` + \`sdk/manifest.schema.json\`)** — full inline type checking for the Lua 5.1 \`ctx\` sandbox and \`manifest.json\`.
* **Working Starter Plugin** — complete with \`manifest.json\`, Expand-and-Contract SQLite migration (\`migrations/001_init.sql\`), Lua 5.1 endpoints & UK GDPR SAR/Purge hooks (\`src/init.lua\`), sandboxed UI & 1x1 dashboard widget (\`ui/dist/\`), and a headless Lua test suite (\`tests/init_test.lua\`).

---

## Quick Start (4 Local Commands)

Run all verification, testing, local development, and packaging locally from your terminal:

\`\`\`bash
# 1. Run the 4-gate contract, CSP, icon, migration & UK GDPR zero-orphan verifier
npm run verify

# 2. Run the headless Lua test suite against an isolated synthetic UK school DB
npm test

# 3. Boot the local synthetic UK school MIS with hot-reload on http://127.0.0.1:18080
npm run dev

# 4. Bundle a signed .mispkg archive ready for School Admin upload
npm run pack
\`\`\`

*(Prefer calling the binaries directly without \`npm\`? Run \`./bin/mis-verify .\`, \`./bin/mis-test .\`, \`./bin/mis-dev-server --plugin .\`, or \`./bin/mis-pack .\`)*

---

## Repository Structure

\`\`\`text
.
├── bin/                         # Pre-compiled CLI binaries (Linux, macOS, Windows)
│   ├── mis-dev-server           # Local Host + auto-seeded synthetic UK school DB + hot-reload
│   ├── mis-test                 # Headless Lua test runner (executes tests/*_test.lua)
│   ├── mis-verify               # 4-gate contract, CSP, migration & UK GDPR gatekeeper
│   └── mis-pack                 # Monochrome Lucide SVG extractor + .mispkg bundler
├── .vscode/
│   ├── extensions.json          # Recommends sumneko.lua (Lua Language Server)
│   └── settings.json            # Enforces Lua 5.1 sandbox globals + binds manifest.schema.json
├── .mis-types/
│   ├── dal.d.lua                # EmmyLua type definitions for ctx.caller, ctx.dal.*, ctx.plugin_db, etc.
│   └── records.d.lua            # EmmyLua record shapes for pupils, staff, guardians, timetable, etc.
├── schema/core/                 # Editable 46-table Core SQLite DDL used by mis-dev-server & mis-test
│   ├── 00_system_and_academic_cycle.sql ... 06_statutory_dfe_census.sql
│   └── SCHEMA_MAP.md            # Complete table & column reference for core.db
├── sdk/
│   ├── manifest.schema.json     # JSON Schema Draft-07 contract for manifest.json
│   ├── ui/
│   │   ├── mis-bridge.js        # Sandboxed iframe postMessage RPC bridge & [data-mis-icon] hydrator
│   │   └── mis-theme.css        # --mis-* light & WCAG AAA high-contrast CSS tokens
│   └── packages/                # Optional @optimis/bridge & @optimis/tailwind-preset packages
├── migrations/
│   └── 001_init.sql             # Plugin SQLite schema mounted as plugin_<alias>.db
├── src/
│   └── init.lua                 # Sandboxed Lua 5.1 endpoints, workers, and UK GDPR hooks
├── ui/dist/
│   ├── index.html               # Full-page plugin view (/p/<plugid>/)
│   └── widget.html              # Preset-size (1x1) Home Dashboard mini-widget
├── tests/
│   └── init_test.lua            # Headless Lua tests run via \`npm test\`
└── manifest.json                # Default-deny routing, scope, view, widget & resource contract
\`\`\`

---

## Synthetic UK School Cohort (Pre-Seeded in \`mis-dev-server\` & \`mis-test\`)

When you run \`npm run dev\` or \`npm test\`, the toolchain automatically provisions a temporary \`core.db\` from \`./schema/core/*.sql\` and seeds these synthetic staff personas, groups, and pupils:

### Staff Personas (pass via \`--persona <staff_id>\`)
| Persona ID | Name | Role | Key DAL Scopes |
| :--- | :--- | :--- | :--- |
| \`stf_tch_7a\` *(default)* | Ms Hannah Abbott | \`TEACHER\` (Form Tutor 7A) | \`:own_groups\` (\`grp_reg_7a\`, \`grp_mat_7x1\`) |
| \`stf_tch_7b\` | Mr Neville Longbottom | \`TEACHER\` (Form Tutor 7B) | \`:own_groups\` (\`grp_reg_7b\`) |
| \`stf_hod_maths\` | Mr David Sterling | \`HOD\` (Head of Maths) | \`dal.pupils.read:all\`, \`dal.groups.read:all\` |
| \`stf_senco_01\` | Dr Amina Patel | \`HOD\` (SENCO) | \`dal.sen.read:all\`, \`dal.pupils.read:all\` |
| \`stf_dsl_01\` | Mr Marcus Thorne | \`DSL\` | \`dal.safeguarding.read\`, \`dal.safeguarding.create\` |
| \`stf_support_01\` | Mrs Clara Oswald | \`SUPPORT\` | \`dal.attendance.write:all\`, \`dal.medical.read:all\` |
| \`stf_admin_01\` | Dame Eleanor Vance | \`ADMIN\` | All school administrative & census scopes |

To boot \`mis-dev-server\` as a specific persona:
\`\`\`bash
node scripts/run-cli.mjs mis-dev-server --plugin . --persona stf_admin_01 --port 18080
\`\`\`

### Pre-Seeded Groups & Pupils
* **Groups:** \`grp_reg_7a\` (Year 7 Registration 7A), \`grp_reg_7b\` (Year 7 Registration 7B), \`grp_mat_7x1\` (Year 7 Maths Set 1)
* **Pupils:**
  * \`pup_yr7_alice\` — Alice Byron (Year 7, assigned to \`grp_reg_7a\` & \`grp_mat_7x1\`, SEN status \`E\` EHCP + Critical Peanut Anaphylaxis IHCP)
  * \`pup_yr7_noah\` — Noah Sterling (Year 7, assigned to \`grp_reg_7a\` & \`grp_mat_7x1\`, SEN status \`K\`)
  * \`pup_yr7_zara\` — Zara Vance (Year 7, assigned to \`grp_reg_7b\`)
  * \`pup_pri_maya\` — Maya Patel (Year 2 Primary)

---

## The 4 Automated Verification Gates (\`npm run verify\`)

Before packaging with \`npm run pack\`, \`mis-verify\` enforces four strict gates:
1. **\`MANIFEST_LUA_UI_OK\`** — Cross-checks \`manifest.json\` against exported functions in \`src/init.lua\`, verifies every \`data-mis-rpc="<method>"\` attribute in \`ui/\` is declared in \`manifest.json\`, and validates monochrome Lucide icon names (\`icon\` and \`data-mis-icon\`).
2. **\`CSP_OFFLINE_OK\`** — Ensures all HTML/JS/CSS assets in \`ui/\` run 100% offline inside \`<iframe sandbox="allow-scripts">\` (\`connect-src 'none'\`, no external \`http://\` or \`https://\` scripts/stylesheets/fetches) and contain zero non-monochrome emoji glyphs.
3. **\`MIGRATIONS_OK\`** — Verifies all SQL files in \`migrations/*.sql\` follow non-destructive **Expand-and-Contract** rules (no \`DROP COLUMN\`, \`RENAME COLUMN\`, or \`DROP TABLE\`) and execute cleanly inside an immediate transaction.
4. **\`GDPR_ZERO_ORPHANS_OK\`** — Automatically seeds synthetic pupil rows across every plugin table containing a \`pupil_id\` column, executes \`export_pupil_data\` and \`purge_pupil_data\`, and verifies **0 orphaned rows** remain.
`,

  agentsMd: () => `# Repository Rules

* Do not run git commands. Never touch git.
* NO FALLBACKS / FAIL FAST: Never introduce silent default fallbacks for missing template files, prompt configurations, required environment variables, required RPC parameters, or authenticated states. If any required file or parameter is missing, throw an explicit error immediately.
* Always write a test first (\`tests/*_test.lua\`) before implementing production Lua, SQL, or UI code.
* Test naming convention is \`test_methodname_initialstate_expectedbehaviour_test(t)\`.
* Always use \`npm\` commands (\`npm run verify\`, \`npm test\`, \`npm run dev\`, \`npm run pack\`) rather than invoking underlying scripts directly.
* Never introduce external \`http://\` or \`https://\` scripts, stylesheets, fonts, or \`fetch()\` calls in \`ui/\` (all plugin UIs execute inside \`<iframe sandbox="allow-scripts">\` with \`connect-src 'none'\`), and never use emoji glyphs (strictly use monochrome Lucide icons via \`data-mis-icon="<name>"\`).
* If you create or modify an \`npm\` script/command in \`package.json\`, you MUST document it under \`## Available NPM Commands\` in \`.agents/AGENTS.md\`.
* Always create or update a \`GEMINI.md\` in any new or heavily updated folders to describe the files and architecture inside it, in addition to updating \`<project-root>/GEMINI.md\`.

## Available NPM Commands

* \`npm run verify\`: Run the 4-gate contract, offline CSP, monochrome Lucide icon, Expand-and-Contract SQL migration, and UK GDPR zero-orphan gatekeeper (\`mis-verify .\`).
* \`npm test\`: Run the headless Lua test suite (\`tests/*_test.lua\`) against an isolated synthetic UK school SQLite database (\`mis-test . --persona stf_tch_7a\`).
* \`npm run dev\`: Boot the local Go Host container with an auto-seeded synthetic UK school \`core.db\` and hot-reload mounting for \`manifest.json\`, \`migrations/\`, \`src/\`, and \`ui/\` on \`http://127.0.0.1:18080\` (\`mis-dev-server --plugin . --persona stf_tch_7a --port 18080\`).
* \`npm run pack\`: Run \`mis-verify\`, extract the declared monochrome Lucide \`icon.svg\`, and bundle a signed \`.mispkg\` archive (\`mis-pack .\`).
`,

  agentsGeminiMd: () => `# Agent Rules & Skills (\`.agents\`)

This directory contains repository-wide rules and agent skills enforcing Test-Driven Development, Fail-Fast error handling, UK GDPR compliance, and documentation integrity.

## Files & Subdirectories

* [\`AGENTS.md\`](./AGENTS.md): Workspace guardrails and index of available \`npm\` commands.
* [\`skills/test-author/SKILL.md\`](./skills/test-author/SKILL.md): Workflow and guardrails for authoring headless Lua plugin tests (\`tests/*_test.lua\`) first before writing production code.
* [\`skills/code-implementer/SKILL.md\`](./skills/code-implementer/SKILL.md): Workflow and guardrails for implementing \`manifest.json\`, \`migrations/*.sql\`, \`src/init.lua\`, and \`ui/dist/*\` until \`npm test\` and \`npm run verify\` pass.
`,

  skillTestAuthor: () => `---
name: test-author
description: >-
  Workflow and strict guardrails for authoring headless Lua plugin tests (tests/*_test.lua)
  in an Optimis plugin repository. Use this skill whenever writing new tests or adding test
  coverage before implementing plugin features.
---

# Test Author Skill

## 1. Core Guardrails & Non-Negotiable Rules

1. **Always Write a Test First**:
   Before implementing or modifying any endpoint, worker, or GDPR hook in \`src/init.lua\`, \`migrations/*.sql\`, or \`ui/\`, write a Lua test function in \`tests/*_test.lua\` first.
2. **Test Naming Convention**:
   Every Lua test function MUST start with \`test_\` (required by \`mis-test\` discovery) and follow:
   \`test_methodname_initialstate_expectedbehaviour_test(t)\`
3. **Strict Read-Only Production Boundary**:
   When acting as \`test-author\`, only create or modify files in \`tests/\`. Do not modify \`src/init.lua\`, \`manifest.json\`, \`migrations/\`, or \`ui/\`.
4. **Fail Fast / Zero Fallbacks**:
   Assert explicit error codes for missing parameters, out-of-scope \`:own_groups\` access, and unauthorized personas.
5. **UK GDPR Coverage**:
   If \`manifest.json\` declares \`database.alias\`, always include test assertions verifying \`export_pupil_data\` (returning structured sections with \`redactable_fields\`) and \`purge_pupil_data\` (leaving 0 rows for the purged \`pupil_id\`).
6. **Never Run \`git\` Commands**.
`,

  skillCodeImplementer: () => `---
name: code-implementer
description: >-
  Test-driven implementation workflow and architectural guardrails for writing and modifying
  Optimis plugin code (manifest.json, migrations/*.sql, src/init.lua, and ui/dist/*). Use this
  skill whenever implementing features or fixing bugs against pre-written tests.
---

# Code Implementer Skill

## 1. Core Guardrails & Non-Negotiable Rules

1. **Mandatory Pre-Written Test Gate**:
   Verify that a pre-written test exists in \`tests/*_test.lua\` following \`test_methodname_initialstate_expectedbehaviour_test(t)\` before editing production code.
2. **Fail Fast / Zero Fallbacks**:
   Never introduce silent default fallbacks for missing required RPC parameters or unauthorized states. Call \`error("ERROR_CODE: message")\` immediately in Lua when mandatory arguments are missing.
3. **Default-Deny \`manifest.json\` Contract**:
   Every interactive endpoint, background worker, DAL scope (\`:own_groups\` vs \`:all\`), allowed HTTPS egress domain, view route (\`views[]\`), and preset-size widget (\`widgets[]\` with \`width: 1 | 2\` and \`height: 1 | 2\`) must be explicitly declared in \`manifest.json\`.
4. **Expand-and-Contract SQL Migrations (\`migrations/*.sql\`)**:
   Never use \`DROP COLUMN\`, \`RENAME COLUMN\`, or \`DROP TABLE\` in minor migrations.
5. **Sandboxed Offline UI & Monochrome Lucide Icons (\`ui/dist/*\`)**:
   - All UI assets run inside \`<iframe sandbox="allow-scripts">\` with \`connect-src 'none'\` (no external \`http://\` or \`https://\` URLs).
   - Use \`--mis-*\` CSS custom properties (\`/sdk/mis-theme.css\`) and \`/sdk/mis-bridge.js\`.
   - Use strictly monochrome Lucide icons via \`data-mis-icon="<name>"\` — zero emoji glyphs.
   - Every \`data-mis-rpc="<method>"\` attribute in \`ui/\` must match an endpoint declared in \`manifest.json\`.
6. **Verify via \`npm test\` and \`npm run verify\`**:
   Iterate until both \`npm test\` (\`mis-test\`) and \`npm run verify\` (\`mis-verify\` 4 gates: \`MANIFEST_LUA_UI_OK\`, \`CSP_OFFLINE_OK\`, \`MIGRATIONS_OK\`, \`GDPR_ZERO_ORPHANS_OK\`) exit \`0\`.
7. **Documentation Integrity**:
   Always update \`GEMINI.md\` and folder-level \`GEMINI.md\` files when adding or updating files, and document any new \`npm\` scripts in \`.agents/AGENTS.md\`. Never run \`git\` commands.
`,

  rootGeminiMd: () => `# Optimis Plugin Repository — Architecture & Contract Blueprint

This repository is a standalone **Optimis School MIS Plugin** built against the Optimis Plugin Contract (Sections 6, 8, 9, and 10).

---

## 1. The Plugin Contract (4 Non-Negotiable Surfaces)

### 1.1 \`manifest.json\` (Default-Deny Routing, Permissions & UI Contract)
Validated by [\`sdk/manifest.schema.json\`](./sdk/manifest.schema.json) and enforced by \`npm run verify\` (\`mis-verify\`):
* **\`id\`, \`version\`, \`core_dal_version\`, \`icon\`**: Reverse-DNS identifier (\`uk.org.optimis.starter.notes\`), SemVer version, Core DAL SemVer range (\`~1.0\`), and monochrome Lucide icon name (\`"award"\`).
* **\`database.alias\`**: Optional SQLite database alias matching \`^plugin_[a-z0-9_]+$\` (provisioned and attached dynamically by the Host).
* **\`views[]\`**: Routable full-page view patterns (\`id\`, \`path\` starting with \`/\` and never \`/ui\`, \`entry\`, \`title\`, \`query_params\`, optional \`icon\`, and \`keywords: string[]\` for Global Search discovery).
* **\`widgets[]\`**: Optional Home Dashboard mini-iframe declarations requiring strict preset dimensions (\`"width": 1 | 2\` and \`"height": 1 | 2\`), \`id\`, \`title\`, \`entry\`, and optional \`keywords: string[]\`.
* **\`endpoints.<method_name>\`**: Maps interactive Lua functions in \`src/init.lua\` to required DAL scopes (\`:own_groups\` vs \`:all\`) and \`mutates: true | false\`.
* **\`gdpr_hooks\`**: Required whenever \`database.alias\` is present; maps \`export_pupil_data\` and \`purge_pupil_data\` to exported Lua functions.

### 1.2 Injected Lua 5.1 Sandbox (\`ctx\` in \`src/init.lua\`)
Plugins execute inside a capability-sandboxed **Lua 5.1 (\`gopher-lua\`)** VM (\`os\`, \`io\`, \`debug\`, \`package\`, \`dofile\`, and \`loadfile\` are stripped). Full type definitions live in [\`.mis-types/dal.d.lua\`](./.mis-types/dal.d.lua):
* **\`ctx.caller\`**: Read-only identity (\`type\`, \`user_id\`, \`staff_id\`, \`active_roles\`, \`request_id\`).
* **\`ctx.dal.<domain>\`**: Scoped Core data access (\`pupils\`, \`groups\`, \`attendance\`, \`guardians\`, \`sen\`, \`medical\`, \`safeguarding\`, \`census\`). When a method carries \`:own_groups\`, the Go Host automatically restricts SQL results to the caller's active temporal teaching/registration groups.
* **\`ctx.plugin_db\`**: \`.query(sql, params)\` and \`.exec(sql, params)\` (only available when \`mutates: true\`).
* **\`ctx.events\`**: \`.emit(event_name, payload)\` dispatched post-commit.
* **\`ctx.http\`**: \`.get(url, headers)\` and \`.post(url, headers, body)\` to \`egress.allowed_domains\` (disallowed inside open SQLite write transactions).
* **\`ctx.xml\`**: \`.build(root_node_spec)\` and \`.validate_xsd(xml_string, xsd_schema_name)\`.

### 1.3 UK GDPR SAR & Zero-Orphan Retention Purge Contract
Every plugin with an attached \`database.alias\` must implement:
* **\`export_pupil_data(ctx, args)\`**: Returns \`{ ok = true, data = { { title = "...", records = ..., redactable_fields = { ... } } } }\` so DPOs/Admins can redact third-party names in free-text columns.
* **\`purge_pupil_data(ctx, args)\`**: Deletes 100% of rows tied to \`pupil_id\` across all plugin tables (verified automatically by \`npm run verify\` Gate 4 \`GDPR_ZERO_ORPHANS_OK\`).

### 1.4 Sandboxed UI Bridge & Monochrome Lucide Icons (\`ui/dist/*\`)
* Runs inside \`<iframe sandbox="allow-scripts">\` with \`connect-src 'none'\` (100% offline assets).
* Includes \`/sdk/mis-theme.css\` (\`--mis-*\` Light & WCAG AAA High-Contrast tokens) and \`/sdk/mis-bridge.js\` (\`data-mis-rpc\`, \`data-mis-args\`, \`data-mis-link\`, \`data-mis-icon\`, \`window.mis.rpc()\`, and \`window.mis.useMisRpc()\`).

---

## 2. Repository Map & Module Index

| Directory | Docs Index | Purpose |
| :--- | :--- | :--- |
| [\`.agents/\`](./.agents/GEMINI.md) | [\`.agents/AGENTS.md\`](./.agents/AGENTS.md) | Repository rules, available \`npm\` commands, and \`test-author\` / \`code-implementer\` skills. |
| [\`.mis-types/\`](./.mis-types/GEMINI.md) | [\`.mis-types/GEMINI.md\`](./.mis-types/GEMINI.md) | EmmyLua / LuaLS static type annotations (\`dal.d.lua\`) for VS Code autocomplete. |
| [\`bin/\`](./bin/GEMINI.md) | [\`bin/GEMINI.md\`](./bin/GEMINI.md) | Pre-compiled static SDK CLI binaries (\`mis-dev-server\`, \`mis-test\`, \`mis-verify\`, \`mis-pack\`). |
| [\`migrations/\`](./migrations/GEMINI.md) | [\`migrations/GEMINI.md\`](./migrations/GEMINI.md) | Expand-and-Contract SQLite DDL migrations for \`plugin_starter_notes.db\`. |
| [\`schema/core/\`](./schema/core/GEMINI.md) | [\`schema/core/GEMINI.md\`](./schema/core/GEMINI.md) | Editable 46-table Core SQLite DDL (\`00\`–\`06\`) and [\`SCHEMA_MAP.md\`](./schema/core/SCHEMA_MAP.md). |
| [\`scripts/\`](./scripts/GEMINI.md) | [\`scripts/GEMINI.md\`](./scripts/GEMINI.md) | Cross-platform CLI binary launcher (\`run-cli.mjs\`). |
| [\`sdk/\`](./sdk/GEMINI.md) | [\`sdk/GEMINI.md\`](./sdk/GEMINI.md) | \`manifest.schema.json\`, \`mis-bridge.js\`, \`mis-theme.css\`, and \`@optimis/*\` frontend packages. |
| [\`src/\`](./src/GEMINI.md) | [\`src/GEMINI.md\`](./src/GEMINI.md) | Sandboxed Lua 5.1 plugin endpoints and UK GDPR \`export_pupil_data\` / \`purge_pupil_data\` hooks (\`init.lua\`). |
| [\`tests/\`](./tests/GEMINI.md) | [\`tests/GEMINI.md\`](./tests/GEMINI.md) | Headless Lua test suite (\`init_test.lua\`) executed by \`npm test\` (\`mis-test\`). |
| [\`ui/\`](./ui/GEMINI.md) | [\`ui/GEMINI.md\`](./ui/GEMINI.md) | Sandboxed full-page view (\`ui/dist/index.html\`) and 1x1 dashboard widget (\`ui/dist/widget.html\`). |
`,

  misTypesGeminiMd: () => `# LuaLS Type Definitions (\`.mis-types\`)

This directory provides static EmmyLua / Lua Language Server (\`sumneko.lua\`) type definitions pre-bound via [\`.vscode/settings.json\`](../.vscode/settings.json).

## Files

* [\`dal.d.lua\`](./dal.d.lua): Defines \`OptimisPluginContext\` (\`ctx.caller\`, \`ctx.school\`, \`ctx.dal.*\`, \`ctx.plugin_db\`, \`ctx.events\`, \`ctx.http\`, \`ctx.crypto\`, \`ctx.xml\`) and global \`ctx\` for Lua 5.1 IDE autocomplete (< 500 lines).
* [\`records.d.lua\`](./records.d.lua): Defines entity record structures across all 11 Core DAL domains (< 500 lines).
`,

  binGeminiMd: () => `# Pre-Compiled Optimis SDK Toolchain Binaries (\`bin\`)

This directory contains zero-dependency static (\`CGO_ENABLED=0\`, \`-trimpath\`) builds of the four Optimis SDK CLI binaries across \`linux-amd64\`, \`darwin-arm64\`, \`darwin-amd64\`, and \`windows-amd64\`.

## Binaries

* \`mis-verify\`: Enforces the 4 automated CI gates (\`MANIFEST_LUA_UI_OK\`, \`CSP_OFFLINE_OK\`, \`MIGRATIONS_OK\`, \`GDPR_ZERO_ORPHANS_OK\`).
* \`mis-test\`: Executes headless Lua test suites (\`tests/*_test.lua\`) against an isolated synthetic UK school \`core.db\`.
* \`mis-dev-server\`: Boots the local Go Host container with an auto-seeded synthetic UK school \`core.db\` and hot-reload mounting for the plugin.
* \`mis-pack\`: Runs \`mis-verify\`, extracts the declared monochrome Lucide \`icon.svg\`, and bundles a signed \`.mispkg\` archive.
`,

  migrationsGeminiMd: () => `# Plugin SQLite Migrations (\`migrations\`)

This directory holds sequential Expand-and-Contract SQL migration scripts applied inside an immediate transaction to the plugin's attached SQLite database (\`plugin_starter_notes.db\`).

## Guardrails

* Minor migrations must strictly follow **Expand-and-Contract** rules: never use \`DROP COLUMN\`, \`RENAME COLUMN\`, or \`DROP TABLE\` (enforced by \`npm run verify\` Gate 3 \`MIGRATIONS_OK\`).
* Every table storing pupil-specific records must use \`pupil_id\` and be cleaned up by \`purge_pupil_data\` in [\`src/init.lua\`](../src/init.lua).

## Files

* [\`001_init.sql\`](./001_init.sql): Creates the \`pupil_praise_notes\` table and indexes on \`pupil_id\` and \`group_id\`.
`,

  srcGeminiMd: () => `# Sandboxed Lua 5.1 Plugin Runtime (\`src\`)

This directory contains the server-side Lua 5.1 entry point executed inside the Go Host's capability-sandboxed \`gopher-lua\` VM.

## Files

* [\`init.lua\`](./init.lua): Implements the endpoints and UK GDPR hooks declared in [\`manifest.json\`](../manifest.json):
  * \`load_group_praise_roster(ctx, args)\`: Reads scoped group pupils via \`ctx.dal.pupils.list_by_group\` and recent praise notes from \`plugin_starter_notes.pupil_praise_notes\`.
  * \`record_pupil_praise_note(ctx, args)\`: Verifies \`:own_groups\` pupil access via \`ctx.dal.pupils.get_by_id\` and inserts a praise note on \`WriteDB\`.
  * \`get_recent_praise_summary(ctx, args)\`: Aggregates recent praise counts and points for the 1x1 Home Dashboard widget.
  * \`export_pupil_data(ctx, args)\` & \`purge_pupil_data(ctx, args)\`: Mandatory UK GDPR SAR export (marking \`praise_note\` in \`redactable_fields\`) and zero-orphan retention purge hooks.
`,

  uiGeminiMd: () => `# Sandboxed Plugin Micro-Frontend UI (\`ui\`)

This directory contains the offline HTML/JS/CSS assets served inside \`<iframe sandbox="allow-scripts">\` with a strict \`connect-src 'none'\` Content Security Policy.

## Guardrails

* Zero external \`http://\` or \`https://\` scripts, stylesheets, fonts, or \`fetch()\` calls (enforced by \`npm run verify\` Gate 2 \`CSP_OFFLINE_OK\`).
* Zero emoji glyphs; all icons must use monochrome Lucide SVGs via \`data-mis-icon="<name>"\`.
* Every \`data-mis-rpc="<method>"\` attribute must match an endpoint declared in [\`manifest.json\`](../manifest.json).

## Files

* [\`dist/index.html\`](./dist/index.html): Full-page view (\`/\` and \`/student/:pupil_id\`) binding \`load_group_praise_roster\` and \`record_pupil_praise_note\` via \`/sdk/mis-bridge.js\` and \`/sdk/mis-theme.css\`.
* [\`dist/widget.html\`](./dist/widget.html): Preset-size \`1x1\` Home Dashboard mini-widget (\`praise_summary_widget\`) calling \`get_recent_praise_summary\`.
`,

  testsGeminiMd: () => `# Headless Lua Plugin Test Suite (\`tests\`)

This directory contains headless Lua test suites executed by \`npm test\` (\`mis-test\`) against an isolated synthetic UK school \`core.db\` (provisioned from [\`../schema/core/*.sql\`](../schema/core/GEMINI.md)).

## Conventions

* Test files must end with \`_test.lua\`.
* Test functions must start with \`test_\` and follow \`test_methodname_initialstate_expectedbehaviour_test(t)\`.
* Use \`t.rpc(method, args, optional_persona_id)\` or \`t.set_persona(persona_id)\` to test RBAC scopes and UK GDPR hooks.

## Files

* [\`init_test.lua\`](./init_test.lua): Verifies \`load_group_praise_roster\`, \`record_pupil_praise_note\`, \`get_recent_praise_summary\`, \`export_pupil_data\`, and \`purge_pupil_data\`.
`,

  scriptsGeminiMd: () => `# Local CLI Helper Scripts (\`scripts\`)

This directory contains cross-platform helper scripts for invoking the pre-compiled Optimis SDK binaries in [\`../bin/\`](../bin/GEMINI.md).

## Files

* [\`run-cli.mjs\`](./run-cli.mjs): Detects the host OS (\`linux\`, \`darwin\`, \`windows\`) and CPU architecture (\`amd64\`, \`arm64\`) and spawns the corresponding binary from \`bin/<os>-<arch>/\` (or \`bin/\`).
`,
};
