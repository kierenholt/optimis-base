# Optimis School MIS — Local Human Testing & Operations Guide

Welcome to **Optimis School MIS** (Oakridge All-Through Academy, `URN 100001`). This guide gives you quick copy-paste commands to seed the synthetic school database, run the platform locally (with all plugins, specific plugins, or zero plugins), hot-swap plugins at runtime, and tail live server and Lua diagnostic logs.

---

## 1. Quick Start (Run Locally in 3 Steps)

### Step 1: Ensure `.env` exists
If you do not already have a `.env` file in the repository root, copy [`example.env`](file:///home/kieren/Documents/code/optimis/example.env):
```bash
cp example.env .env
```

### Step 2: Build the Seeded School Database (`npm run seed`)

The seed command builds the complete 45-table Oakridge Academy synthetic school cohort (~1,108 pupils, 95 staff, timetable, 20 days attendance).

**`npm run seed` does NOT read from `.env`**. When run without parameters, it interactively asks you where to write the database:

```bash
npm run seed
# Terminal prompt:
# Where would you like to put the seed database? [.data/core.db]: 
```
* **Press Enter** to seed into `.data/core.db`.
* **Or type `.data/qa_core.db`** (or any custom path) to keep it separate.

#### Non-Interactive Seeding Shortcuts
If you prefer not to be prompted:
```bash
# Seed directly to a dedicated QA database:
npm run seed:qa
# (or: npm run seed -- .data/qa_core.db)

# Seed directly to the default core database:
npm run seed:dev
```

#### Pointing the Server to Your Database (`DATABASE_PATH`)
There is **only ONE variable for the database in [`.env`](file:///home/kieren/Documents/code/optimis/.env)**:
```bash
# Point to your database in .env:
DATABASE_PATH=.data/core.db

# Or point directly to your dedicated QA seed database:
# DATABASE_PATH=.data/qa_core.db
```
*(Or pass the `--db` flag on the CLI: `./bin/mis-server --db .data/qa_core.db`)*

> **Tip — Resetting Attached Plugin Databases:** Attached plugin databases live alongside your core DB in `.data/plugin_*.db`. To wipe all plugin state and re-seed fresh, run:
> ```bash
> rm -f .data/plugin_*.db* && npm run seed:qa
> ```

### Step 3: Launch the Server & Open the UI
Choose how you want to boot the platform:

* **Option A — Run with ALL 11 production plugins preloaded (Recommended for full walkthrough):**
  ```bash
  # Hot-reload dev mode (rebuilds React UI + Go Host automatically on file save):
  npm run dev:all-plugins

  # OR build once and run the compiled binary directly:
  npm run build && npm run start:all-plugins
  ```
* **Option B — Run with NO plugins (Clean Base Host UI Shell only):**
  ```bash
  # Hot-reload dev mode with 0 plugins:
  npm run dev:no-plugins

  # OR build once and run the compiled binary with 0 plugins:
  npm run build && npm run start:no-plugins
  ```

Then open **[http://127.0.0.1:18080](http://127.0.0.1:18080)** in your browser.

---

## 2. Controlling Preloaded Plugins via `.env` (`PRELOAD_PLUGINS`) or CLI Flags

You can control which plugins are automatically installed, migrated (`ATTACH DATABASE`), and activated at server startup using the **`PRELOAD_PLUGINS`** environment variable in [`.env`](file:///home/kieren/Documents/code/optimis/.env) (or inline on the command line):

| Mode | `.env` / Environment Variable Setting | Description |
| :--- | :--- | :--- |
| **No Plugins** | `PRELOAD_PLUGINS=none` *(or empty)* | Boots the clean React Host Shell with zero plugins installed. |
| **All Production Plugins** | `PRELOAD_PLUGINS=all` *(or `PRELOAD_PLUGINS=./plugins`)* | Scans the folder and preloads all production plugin subdirectories containing a `manifest.json` (excludes the synthetic `test_plugin` fixture). |
| **Single Plugin Folder** | `PRELOAD_PLUGINS=timetable_attendance_plugin` *(or `PRELOAD_PLUGINS=./plugins/timetable_attendance_plugin`)* | Preloads only that single plugin folder. |
| **List of Plugin Folders** | `PRELOAD_PLUGINS=timetable_attendance_plugin,behaviour_plugin,staff_cover_plugin,first_aid_medical_plugin` | Preloads the exact comma-separated list of plugin folders in order. |

### Equivalent CLI Flags on `./bin/mis-server` (and `npm run dev --`)
You can also pass flags directly to `./bin/mis-server` or `npm run dev --`:
```bash
# Preload all production plugins:
./bin/mis-server --all-plugins

# Preload all plugins INCLUDING the synthetic test_plugin fixture:
./bin/mis-server --all-plugins --include-test-plugin

# Preload a specific comma-separated list of plugin folders:
./bin/mis-server --plugins timetable_attendance_plugin,senco_provision_plugin,sparx_maths_plugin

# Force zero plugins even if PRELOAD_PLUGINS is set in .env:
./bin/mis-server --no-plugins
```

---

## 3. Hot-Swapping Plugins While the Server Is Running

You do **not** need to restart the server to add or remove plugins. While the server is running at `http://127.0.0.1:18080`, open a second terminal and use [`scripts/manage_plugins.mjs`](file:///home/kieren/Documents/code/optimis/scripts/manage_plugins.mjs):

```bash
# Check which plugins are currently installed and their state (ACTIVE vs UNINSTALLED_DORMANT):
npm run plugins:status

# Hot-install & activate ALL production plugins on the running server:
npm run plugins:all

# Hot-install & activate one or more specific plugin folders:
npm run plugins:load -- timetable_attendance_plugin behaviour_plugin

# Remove ALL plugins from the running server (returns to a 0-plugin Host Shell):
npm run plugins:none
```

---

## 4. Viewing & Tailing Local Logs

Optimis exposes three logging & diagnostic surfaces:

1. **Local Structured Log File (`.data/mis-server.log`)**:
   Every Host service log (`INFO`, `ERROR`) and every PII-scrubbed Lua runtime/lock `DIAGNOSTIC` entry is appended in real time to `<DATA_DIR_PATH>/mis-server.log`.
2. **Live HTTP Log & Diagnostic Stream (`npm run logs:tail`)**:
   The [`scripts/tail_logs.mjs`](file:///home/kieren/Documents/code/optimis/scripts/tail_logs.mjs) script connects to the running server's log cursor (`/api/test/logs`) and Admin diagnostic ring buffer (`/api/admin/diagnostics/bundle`) and prints color-coded `INFO`, `ERROR`, and `DIAG` lines as you click around the UI.
3. **In-UI Admin Diagnostic Bundle Export**:
   Switch the topbar Persona Switcher to **Dame Eleanor Vance (`ADMIN`)**, scroll to the **Day-2 Operations: Plugin Diagnostic Bundle** panel in the Host UI, and click **Download Plugin Diagnostic Bundle** to inspect the JSON/log ring buffer directly in the browser.

### Log Tailing Commands
Run any of these in a second terminal while testing:

```bash
# Stream all historical + live logs and PII-scrubbed Lua diagnostics (colorized):
npm run logs:tail

# Stream ONLY new log lines arriving from this moment forward:
npm run logs:tail -- --new-only

# Filter live logs by plugin ID, event name, or keyword (case-insensitive):
npm run logs:tail -- --filter sparx
npm run logs:tail -- --filter core.attendance
npm run logs:tail -- --filter error

# Tail the on-disk .data/mis-server.log file directly:
npm run logs:tail:file
# (or standard unix tail):
tail -f .data/mis-server.log
```

---

## 5. Human Testing Cheat Sheet (Personas, Pupils & Plugins)

### 5.1 Topbar Persona Switcher (RBAC & `:own_groups` Testing)
Use the **Active Persona** dropdown in the top navigation bar to switch staff identities on the fly. Switching personas updates the session cookie, recalculates permitted plugin endpoints, updates the Host panels (DSL Vault, GDPR SAR, Admin Diagnostics), and fires `mis:persona:changed` inside mounted plugin iframes:

| Staff ID | Persona Name | Role | Key Permissions & What to Test |
| :--- | :--- | :--- | :--- |
| `stf_admin_01` | **Dame Eleanor Vance** | `ADMIN` | Executive Headteacher. Full `:all` scopes across all domains, `plugin.config.write`, **GDPR SAR Redaction & Purge UI**, and **Admin Diagnostic Bundle Export**. |
| `stf_dsl_01` | **Mr Marcus Thorne** | `DSL` | Designated Safeguarding Lead (`is_dsl=1`). Unlocks the **Envelope-Encrypted DSL Safeguarding Vault** (click *Issue OAuth Cloud Key* $\rightarrow$ *Unlock Vault* to decrypt `pup_yr7_alice`'s note). |
| `stf_senco_01` | **Dr Priya Nair** | `SENCO` | Whole-School SENCO. Holds `dal.sen.read:all` & `dal.sen.write:all` to manage IEP targets and TA allocations across the entire school in the **SENCO Provision** plugin. |
| `stf_hod_maths` | **Mr David Sterling** | `HOD` | Head of Maths. Holds `dal.assessment.read:all` & `plugin.config.write` (can run cross-set MAT attainment rollups and configure Sparx Maths vendor sync). |
| `stf_tch_7a` | **Ms Hannah Abbott** | `TEACHER` | Form `7A` Tutor & `7A/Ma1` Maths Teacher. Restricted to `:own_groups` (`grp_reg_7a`, `grp_tch_7a_ma`). Can view/mark `pup_yr7_alice` & `pup_yr7_noah`, but blocked from `7B` (`pup_yr7_zara`). |
| `stf_tch_7b` | **Mr Simon Barker** | `TEACHER` | Form `7B` Tutor & `7B/En1` English Teacher. Negative control for `7A` (`:own_groups` restricts him to `7B` pupils like `pup_yr7_zara` and `pup_yr7_liam`). |
| `stf_ta_01` | **Mrs Sarah Jenkins** | `TA` | Higher Level Teaching Assistant assigned to `7A` and supporting Noah Williams (`pup_yr7_noah`). |
| `stf_supply_01` | **Mr Gareth Cole** | `TEACHER` | Agency Cover Teacher. Starts with **zero** assigned groups; when assigned cover for `grp_tch_7a_ma` in the **Staff Cover** plugin, dynamically gains `:own_groups` access to take `7A/Ma1`'s register! |

### 5.2 Pinned Pupil & Group IDs for Testing
Use these deterministic IDs when searching or testing forms inside the plugins:

* **Pinned Pupils:**
  * `pup_yr7_alice` — **Alice Evans (Form 7A, `grp_reg_7a`)**: Confidential Women's Refuge address, court-order restricted father (`court_order_restricted=1`, `has_parental_responsibility=0`), active PR mother, and an encrypted DSL Safeguarding Vault record.
  * `pup_yr7_noah` — **Noah Williams (Form 7A, `grp_reg_7a`)**: EHCP (`E`) with ASD & SEMH needs, assigned TA (`stf_ta_01`), and literacy intervention (`INT-LIT-7`).
  * `pup_pri_maya` — **Maya Patel (Form 4A, `grp_reg_4a`)**: `CRITICAL` Severe Peanut Anaphylaxis with IHCP (`has_ihcp=1`) and Jext 150mcg Auto-Injector in the Medical Room.
  * `pup_yr7_zara` — **Zara Khan (Form 7B, `grp_reg_7b`)**: `:own_groups` negative control for `stf_tch_7a`; SEN Support (`K`) + EAL.
  * `pup_yr7_liam` — **Liam O'Connor (Form 7B, `grp_reg_7b`)**: Mid-term temporal group transfer (`7A` $\rightarrow$ `7B` on `2026-10-01`).
  * `pup_yr10_ryan` — **Ryan Mitchell (Form 10A, `grp_reg_10a`)**: Looked After Child (LAC Foster Care), Dual Subsidiary Registration (`S`), Alternative Provision, and fixed-term suspension.
  * `pup_yr12_chloe` — **Chloe Sterling (Form 12A, `grp_reg_12a`)**: Sixth Form Post-16 pupil with 3 A-Level learning aims.
  * `pup_eyfs_leo` — **Leo Carter (Nursery N2A, `grp_reg_n2a`)**: EYFS 30-hour funded childcare & DAF indicator.
  * `pup_left_oliver` — **Oliver Grant (Form 10B)**: Off-rolled historical pupil (`LEFT` on `2026-10-02`, excluded from current-roll census counts).
* **Key Group IDs:**
  * Registration Forms: `grp_reg_7a`, `grp_reg_7b`, `grp_reg_4a`, `grp_reg_10a`, `grp_reg_12a`
  * Teaching Classes: `grp_tch_7a_ma` (Year 7 Maths Set 1, taught by `stf_tch_7a`), `grp_tch_7b_en` (Year 7 English, taught by `stf_tch_7b`), `grp_tch_10a_ma` (Year 10 Maths, taught by `stf_hod_maths`)

### 5.3 Available Plugin Folders (`plugins/`)

| Folder Name | Plugin ID | Description |
| :--- | :--- | :--- |
| `timetable_attendance_plugin` | `uk.org.optimis.timetable.attendance` | Teacher timetable grid, class roster, and statutory `AM`/`PM` session attendance register. |
| `behaviour_plugin` | `uk.org.optimis.behaviour` | Positive/negative behaviour point ledger + Home Dashboard summary widget (`widget.html`). |
| `assessment_markbook_plugin` | `uk.org.optimis.assessment.markbook` | Class markbook with MAT grading schema validation and HoD cross-set attainment rollup. |
| `senco_provision_plugin` | `uk.org.optimis.sen.provision` | Whole-school SEN register (`K`/`E`), IEP targets, TA allocations, and deep-linking dashboard widget. |
| `first_aid_medical_plugin` | `uk.org.optimis.medical.firstaid` | Medical Room treatment log, `CRITICAL` IHCP/anaphylaxis banner, and court-order safe guardian lookup. |
| `trips_consent_plugin` | `uk.org.optimis.ops.trips` | Educational visits, strict Parental Responsibility (`PR`) consent gate, and Trip Emergency Pack generator. |
| `staff_cover_plugin` | `uk.org.optimis.ops.cover` | Absent teacher timetable lookup, free-staff finder, and temporal `COVER_TEACHER` assignment. |
| `parent_dmz_plugin` | `uk.org.optimis.parent.dmz` | Outbound sanitized parent read-projection push & untrusted DMZ absence-note inbox review. |
| `lms_sync_plugin` | `uk.org.optimis.lms.classroom` | Event-driven Google Classroom roster sync listening to `core.group.membership_changed`. |
| `sparx_maths_plugin` | `uk.co.sparx.maths` | Third-party partner sync demonstrating two-phase Read/HTTP Egress $\rightarrow$ WriteDB & secret scrubbing. |
| `dfe_census_plugin` | `uk.gov.dfe.census` | Statutory DfE School Census XML generator (`ctx.xml.build`), XSD validator, and COLLECT submission. |

---

## 6. Local Break-Glass CLI (`./bin/mis-server config ...`)

If you ever need to inspect or repair runtime configuration in `__system_config` inside `core.db` from the terminal (Section 7C Break-Glass CLI):
```bash
# List all __system_config keys in core.db:
./bin/mis-server config list

# Read or update a specific configuration key:
./bin/mis-server config get sso_provider
./bin/mis-server config set school_name "Oakridge All-Through Academy"

# Repair SSO configuration from the command line:
./bin/mis-server config repair-sso --provider ENTRA_ID --client-id my-client-id --tenant-id my-tenant-id --issuer-url https://login.microsoftonline.com/my-tenant-id/v2.0
```
