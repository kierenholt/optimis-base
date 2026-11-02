#!/usr/bin/env node
/**
 * Populate the standalone `optimis-plugin-template` repository with:
 * - Pre-compiled SDK CLI binaries (`./bin/` + cross-platform `./bin/<os>-<arch>/`)
 * - Editable Core SQLite schema (`schema/core/*.sql` + `SCHEMA_MAP.md`)
 * - IDE autocomplete & validation (`.vscode/`, `.mis-types/dal.d.lua`, `sdk/`)
 * - Minimal Vanilla HTML/JS + Lua + SQLite starter plugin (`manifest.json`, `migrations/`, `src/`, `ui/dist/`, `tests/`)
 * - Local developer scripts (`package.json`, `scripts/run-cli.mjs`, `README.md`)
 */

import { execFileSync } from 'child_process';
import fs from 'fs';
import path from 'path';

const ROOT_DIR = process.cwd();
const DEFAULT_TARGET_DIR = path.resolve(ROOT_DIR, '../optimis-plugin-template');
const positionalArg = process.argv.slice(2).find((arg) => !arg.startsWith('-'));
const targetDir = positionalArg ? path.resolve(positionalArg) : DEFAULT_TARGET_DIR;

if (!fs.existsSync(targetDir)) {
  throw new Error(`Target template directory does not exist: ${targetDir}`);
}

console.log(`Populating Optimis Plugin Template at: ${targetDir}`);

const ensureDir = (relPath) => {
  fs.mkdirSync(path.join(targetDir, relPath), { recursive: true });
};

const writeFile = (relPath, content, executable = false) => {
  const fullPath = path.join(targetDir, relPath);
  fs.mkdirSync(path.dirname(fullPath), { recursive: true });
  fs.writeFileSync(fullPath, content, 'utf-8');
  if (executable) {
    fs.chmodSync(fullPath, 0o755);
  }
};

// 1. Copy .vscode/, .mis-types/, sdk/, and schema/core/
ensureDir('.vscode');
fs.copyFileSync(path.join(ROOT_DIR, '.vscode/settings.json'), path.join(targetDir, '.vscode/settings.json'));
writeFile(
  '.vscode/extensions.json',
  JSON.stringify(
    {
      recommendations: ['sumneko.lua'],
    },
    null,
    2
  ) + '\n'
);

ensureDir('.mis-types');
fs.copyFileSync(path.join(ROOT_DIR, '.mis-types/dal.d.lua'), path.join(targetDir, '.mis-types/dal.d.lua'));

fs.cpSync(path.join(ROOT_DIR, 'sdk'), path.join(targetDir, 'sdk'), { recursive: true });
fs.cpSync(path.join(ROOT_DIR, 'schema/core'), path.join(targetDir, 'schema/core'), { recursive: true });

// 2. Cross-compile portable (-trimpath, CGO_ENABLED=0) CLI binaries into ./bin/
const cliTools = ['mis-dev-server', 'mis-test', 'mis-verify', 'mis-pack'];
const targets = [
  { os: 'linux', arch: 'amd64', ext: '' },
  { os: 'darwin', arch: 'arm64', ext: '' },
  { os: 'darwin', arch: 'amd64', ext: '' },
  { os: 'windows', arch: 'amd64', ext: '.exe' },
];

const skipBuild = process.argv.includes('--skip-build');
ensureDir('bin');
if (!skipBuild) {
  for (const t of targets) {
    const platformSubdir = path.join(targetDir, 'bin', `${t.os}-${t.arch}`);
    fs.mkdirSync(platformSubdir, { recursive: true });
    for (const tool of cliTools) {
      const outPath = path.join(platformSubdir, `${tool}${t.ext}`);
      execFileSync('go', ['build', '-trimpath', '-ldflags=-s -w', '-o', outPath, `./cmd/${tool}`], {
        cwd: ROOT_DIR,
        env: {
          ...process.env,
          CGO_ENABLED: '0',
          GOOS: t.os,
          GOARCH: t.arch,
        },
        stdio: 'inherit',
      });
      if (t.os === 'linux' && t.arch === 'amd64') {
        const topBinPath = path.join(targetDir, 'bin', tool);
        fs.copyFileSync(outPath, topBinPath);
        fs.chmodSync(topBinPath, 0o755);
      }
    }
  }
}

// 3. Write OS-aware CLI runner (scripts/run-cli.mjs)
writeFile(
  'scripts/run-cli.mjs',
  `#!/usr/bin/env node
import { spawnSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const rootDir = path.resolve(__dirname, '..');

const toolName = process.argv[2];
const toolArgs = process.argv.slice(3);

if (!toolName) {
  console.error('Usage: node scripts/run-cli.mjs <mis-verify|mis-test|mis-dev-server|mis-pack> [args...]');
  process.exit(1);
}

const platformMap = {
  linux: 'linux',
  darwin: 'darwin',
  win32: 'windows',
};
const archMap = {
  x64: 'amd64',
  arm64: 'arm64',
};

const osName = platformMap[process.platform] || process.platform;
const archName = archMap[process.arch] || process.arch;
const ext = process.platform === 'win32' ? '.exe' : '';

const platformBin = path.join(rootDir, 'bin', \`\${osName}-\${archName}\`, \`\${toolName}\${ext}\`);
const fallbackBin = path.join(rootDir, 'bin', \`\${toolName}\${ext}\`);
const binPath = fs.existsSync(platformBin) ? platformBin : fallbackBin;

if (!fs.existsSync(binPath)) {
  console.error(\`Toolchain binary not found at \${platformBin} or \${fallbackBin}\`);
  process.exit(1);
}

const res = spawnSync(binPath, toolArgs, {
  cwd: rootDir,
  stdio: 'inherit',
  env: process.env,
});

process.exit(res.status ?? 1);
`,
  true
);

// 4. Write package.json with local verify, test, dev, and pack scripts
writeFile(
  'package.json',
  JSON.stringify(
    {
      name: 'optimis-plugin-starter',
      version: '1.0.0',
      private: true,
      description: 'Starter template for building sandboxed Optimis School MIS (.mispkg) plugins',
      scripts: {
        dev: 'node scripts/run-cli.mjs mis-dev-server --plugin . --persona stf_tch_7a --port 18080',
        verify: 'node scripts/run-cli.mjs mis-verify .',
        test: 'node scripts/run-cli.mjs mis-test . --persona stf_tch_7a',
        pack: 'node scripts/run-cli.mjs mis-pack .',
      },
    },
    null,
    2
  ) + '\n'
);

// 5. Write .gitignore
writeFile(
  '.gitignore',
  `node_modules/
*.db
*.db-wal
*.db-shm
*.mispkg
.DS_Store
`
);

// 6. Write Starter Plugin (manifest.json, migrations/001_init.sql, src/init.lua, ui/dist/index.html, ui/dist/widget.html, tests/init_test.lua)
writeFile(
  'manifest.json',
  JSON.stringify(
    {
      id: 'uk.org.optimis.starter.notes',
      title: 'Classroom Praise & Notes Starter',
      version: '1.0.0',
      core_dal_version: '~1.0',
      icon: 'award',
      max_concurrency: 4,
      max_memory_mb: 32,
      database: {
        alias: 'plugin_starter_notes',
        max_db_size_mb: 100,
      },
      views: [
        {
          id: 'home',
          path: '/',
          entry: 'ui/dist/index.html',
          title: 'Classroom Praise & Notes',
          icon: 'award',
          query_params: ['group_id'],
          keywords: ['praise', 'notes', 'recognition', 'starter', 'commendation'],
        },
        {
          id: 'student_notes',
          path: '/student/:pupil_id',
          entry: 'ui/dist/index.html',
          title: 'Pupil Praise Log',
          icon: 'user-check',
          query_params: ['tab'],
          keywords: ['pupil praise', 'student notes'],
        },
      ],
      widgets: [
        {
          id: 'praise_summary_widget',
          title: 'Recent Classroom Praise',
          description: 'Displays recent classroom praise and commendation notes for your assigned teaching groups.',
          icon: 'award',
          entry: 'ui/dist/widget.html',
          width: 1,
          height: 1,
          keywords: ['praise widget', 'commendations', 'recent notes'],
        },
      ],
      endpoints: {
        load_group_praise_roster: {
          scopes: ['dal.pupils.read:own_groups', 'dal.groups.read:own_groups'],
          mutates: false,
          timeout_ms: 2500,
        },
        record_pupil_praise_note: {
          scopes: ['dal.pupils.read:own_groups', 'dal.groups.read:own_groups'],
          mutates: true,
          timeout_ms: 2500,
        },
        get_recent_praise_summary: {
          scopes: ['dal.pupils.read:own_groups', 'dal.groups.read:own_groups'],
          mutates: false,
          timeout_ms: 2500,
        },
      },
      gdpr_hooks: {
        export_pupil_data: 'export_pupil_data',
        purge_pupil_data: 'purge_pupil_data',
      },
    },
    null,
    2
  ) + '\n'
);

writeFile(
  'migrations/001_init.sql',
  `-- Expand-and-Contract Migration 001: Classroom Praise & Notes
-- Note: Minor migrations must never use DROP COLUMN, RENAME COLUMN, or DROP TABLE.

CREATE TABLE IF NOT EXISTS pupil_praise_notes (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL,
    group_id TEXT NOT NULL,
    staff_id TEXT NOT NULL,
    category TEXT NOT NULL,
    points INTEGER NOT NULL DEFAULT 1,
    praise_note TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_pupil_praise_notes_pupil
    ON pupil_praise_notes(pupil_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_pupil_praise_notes_group
    ON pupil_praise_notes(group_id, created_at DESC);
`
);

writeFile(
  'src/init.lua',
  `---@param ctx OptimisPluginContext
---@param args table
function load_group_praise_roster(ctx, args)
  local group_id = args and args.group_id
  if not group_id or group_id == "" then
    local groups = ctx.dal.groups.list("REGISTRATION")
    if #groups == 0 then
      groups = ctx.dal.groups.list("")
    end
    if #groups == 0 then
      error("NO_ASSIGNED_GROUPS: caller has no active groups")
    end
    group_id = groups[1].id
  end

  local pupils = ctx.dal.pupils.list_by_group(group_id)
  local notes = ctx.plugin_db.query(
    [[SELECT id, pupil_id, group_id, staff_id, category, points, praise_note, created_at
      FROM plugin_starter_notes.pupil_praise_notes
      WHERE group_id = ?
      ORDER BY created_at DESC, id DESC]],
    { group_id }
  )

  return {
    ok = true,
    data = {
      group_id = group_id,
      caller_staff_id = ctx.caller.staff_id,
      pupils = pupils,
      notes = notes,
    },
  }
end

---@param ctx OptimisPluginContext
---@param args table
function record_pupil_praise_note(ctx, args)
  if not args or not args.pupil_id or args.pupil_id == "" then
    error("MISSING_PUPIL_ID: pupil_id is required")
  end
  if not args.group_id or args.group_id == "" then
    error("MISSING_GROUP_ID: group_id is required")
  end
  if not args.praise_note or args.praise_note == "" then
    error("MISSING_PRAISE_NOTE: praise_note is required")
  end

  -- Verify pupil belongs to caller's scoped group (:own_groups enforced automatically by Core DAL)
  local pupil = ctx.dal.pupils.get_by_id(args.pupil_id)
  if not pupil or not pupil.id then
    error("PUPIL_NOT_FOUND_OR_OUT_OF_SCOPE: " .. tostring(args.pupil_id))
  end

  local category = args.category or "Academic Effort"
  local points = tonumber(args.points) or 1
  if points < 1 then
    error("INVALID_POINTS: points must be >= 1")
  end

  local existing = ctx.plugin_db.query(
    "SELECT COUNT(*) AS cnt FROM plugin_starter_notes.pupil_praise_notes",
    {}
  )
  local next_seq = 1
  if existing and existing[1] and existing[1].cnt then
    next_seq = tonumber(existing[1].cnt) + 1
  end
  local note_id = string.format("prs_%s_%d", tostring(args.pupil_id), next_seq)

  ctx.plugin_db.exec(
    [[INSERT INTO plugin_starter_notes.pupil_praise_notes
      (id, pupil_id, group_id, staff_id, category, points, praise_note)
      VALUES (?, ?, ?, ?, ?, ?, ?)]],
    {
      note_id,
      tostring(args.pupil_id),
      tostring(args.group_id),
      tostring(ctx.caller.staff_id or "SYSTEM"),
      tostring(category),
      points,
      tostring(args.praise_note),
    }
  )

  return {
    ok = true,
    data = {
      id = note_id,
      pupil_id = tostring(args.pupil_id),
      group_id = tostring(args.group_id),
      category = tostring(category),
      points = points,
      praise_note = tostring(args.praise_note),
    },
  }
end

---@param ctx OptimisPluginContext
---@param args table
function get_recent_praise_summary(ctx, args)
  local rows = ctx.plugin_db.query(
    [[SELECT id, pupil_id, group_id, category, points, praise_note, created_at
      FROM plugin_starter_notes.pupil_praise_notes
      ORDER BY created_at DESC, id DESC
      LIMIT 5]],
    {}
  )

  local total_points = 0
  for _, row in ipairs(rows) do
    total_points = total_points + (tonumber(row.points) or 0)
  end

  return {
    ok = true,
    data = {
      recent_count = #rows,
      total_points = total_points,
      items = rows,
    },
  }
end

---@param ctx OptimisPluginContext
---@param args table|string
function export_pupil_data(ctx, args)
  local pupil_id = type(args) == "table" and args.pupil_id or args
  if not pupil_id or pupil_id == "" then
    error("MISSING_PUPIL_ID: pupil_id is required for SAR export")
  end

  local records = ctx.plugin_db.query(
    [[SELECT id, pupil_id, group_id, staff_id, category, points, praise_note, created_at
      FROM plugin_starter_notes.pupil_praise_notes
      WHERE pupil_id = ?
      ORDER BY created_at DESC, id DESC]],
    { tostring(pupil_id) }
  )

  return {
    ok = true,
    data = {
      {
        title = "Classroom Praise & Recognition Notes",
        records = records,
        redactable_fields = { "praise_note" },
      },
    },
  }
end

---@param ctx OptimisPluginContext
---@param args table|string
function purge_pupil_data(ctx, args)
  local pupil_id = type(args) == "table" and args.pupil_id or args
  if not pupil_id or pupil_id == "" then
    error("MISSING_PUPIL_ID: pupil_id is required for retention purge")
  end

  local res = ctx.plugin_db.exec(
    "DELETE FROM plugin_starter_notes.pupil_praise_notes WHERE pupil_id = ?",
    { tostring(pupil_id) }
  )

  return {
    ok = true,
    data = {
      pupil_id = tostring(pupil_id),
      rows_affected = res and res.rows_affected or 0,
    },
  }
end
`
);

writeFile(
  'ui/dist/index.html',
  `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Classroom Praise & Notes Starter</title>
  <link rel="stylesheet" href="/sdk/mis-theme.css" />
  <script src="/sdk/mis-bridge.js"></script>
</head>
<body class="mis-surface" style="padding: 1.25rem; margin: 0;">
  <main style="max-width: 960px; margin: 0 auto;">
    <header style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 1rem;">
      <div style="display: flex; align-items: center; gap: 0.625rem;">
        <span data-mis-icon="award"></span>
        <div>
          <h1 style="margin: 0; font-size: 1.25rem;">Classroom Praise &amp; Notes</h1>
          <p class="mis-text-muted" style="margin: 0.15rem 0 0 0; font-size: 0.875rem;">
            Starter Optimis Plugin — Scoped to your assigned teaching &amp; registration groups
          </p>
        </div>
      </div>
      <button
        type="button"
        id="refresh-roster-btn"
        data-testid="refresh-roster-btn"
        data-mis-rpc="load_group_praise_roster"
        data-mis-args='{"group_id":"grp_reg_7a"}'
        class="mis-btn-primary"
      >
        <span data-mis-icon="refresh-cw"></span>
        <span>Load Group 7A</span>
      </button>
    </header>

    <section class="mis-card" style="padding: 1rem; margin-bottom: 1rem;">
      <h2 style="margin-top: 0; font-size: 1rem;">Award Classroom Praise</h2>
      <div style="display: grid; grid-template-columns: 1fr 1fr 2fr auto; gap: 0.75rem; align-items: end;">
        <label style="display: flex; flex-direction: column; gap: 0.25rem; font-size: 0.875rem;">
          <span>Pupil ID</span>
          <input id="praise-pupil-id" data-testid="praise-pupil-id" value="pup_yr7_alice" style="padding: 0.45rem;" />
        </label>
        <label style="display: flex; flex-direction: column; gap: 0.25rem; font-size: 0.875rem;">
          <span>Category</span>
          <input id="praise-category" data-testid="praise-category" value="Academic Effort" style="padding: 0.45rem;" />
        </label>
        <label style="display: flex; flex-direction: column; gap: 0.25rem; font-size: 0.875rem;">
          <span>Praise Note (SAR Redactable)</span>
          <input id="praise-note" data-testid="praise-note" value="Helped peers solve linear equations" style="padding: 0.45rem;" />
        </label>
        <button
          type="button"
          id="submit-praise-btn"
          data-testid="submit-praise-btn"
          data-mis-rpc="record_pupil_praise_note"
          data-mis-args='{"pupil_id":"pup_yr7_alice","group_id":"grp_reg_7a","category":"Academic Effort","points":2,"praise_note":"Helped peers solve linear equations"}'
          class="mis-btn-primary"
        >
          <span data-mis-icon="clipboard-check"></span>
          <span>Save Praise</span>
        </button>
      </div>
    </section>

    <section class="mis-card" style="padding: 1rem;">
      <h2 style="margin-top: 0; font-size: 1rem;">Group Roster &amp; Recent Notes</h2>
      <pre id="plugin-output" data-testid="plugin-output" style="margin: 0; font-size: 0.8125rem; white-space: pre-wrap;">Ready.</pre>
    </section>
  </main>

  <script>
    (function () {
      var outputEl = document.getElementById('plugin-output');
      var submitBtn = document.getElementById('submit-praise-btn');

      function syncArgs() {
        var pupilId = document.getElementById('praise-pupil-id').value;
        var category = document.getElementById('praise-category').value;
        var note = document.getElementById('praise-note').value;
        submitBtn.setAttribute(
          'data-mis-args',
          JSON.stringify({
            pupil_id: pupilId,
            group_id: 'grp_reg_7a',
            category: category,
            points: 2,
            praise_note: note,
          })
        );
      }

      ['praise-pupil-id', 'praise-category', 'praise-note'].forEach(function (id) {
        document.getElementById(id).addEventListener('input', syncArgs);
      });

      document.addEventListener('mis:rpc:success', function (ev) {
        outputEl.textContent = JSON.stringify(ev.detail, null, 2);
      });

      document.addEventListener('mis:rpc:error', function (ev) {
        outputEl.textContent = 'RPC Error: ' + JSON.stringify(ev.detail, null, 2);
      });
    })();
  </script>
</body>
</html>
`
);

writeFile(
  'ui/dist/widget.html',
  `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Recent Classroom Praise Widget</title>
  <link rel="stylesheet" href="/sdk/mis-theme.css" />
  <script src="/sdk/mis-bridge.js"></script>
</head>
<body class="mis-surface" style="padding: 0.75rem; margin: 0;">
  <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.5rem;">
    <div style="display: flex; align-items: center; gap: 0.4rem; font-weight: 600;">
      <span data-mis-icon="award"></span>
      <span>Recent Praise</span>
    </div>
    <button
      type="button"
      data-testid="widget-refresh-btn"
      data-mis-rpc="get_recent_praise_summary"
      data-mis-args="{}"
      class="mis-btn-primary"
      style="padding: 0.25rem 0.5rem; font-size: 0.75rem;"
    >
      <span data-mis-icon="refresh-cw"></span>
    </button>
  </div>
  <div id="widget-summary" data-testid="widget-summary" class="mis-text-muted" style="font-size: 0.8125rem;">
    Click refresh or wait for handshake...
  </div>
  <script>
    window.addEventListener('message', function (ev) {
      if (ev.data && ev.data.type === 'mis:init' && window.mis && typeof window.mis.rpc === 'function') {
        window.mis.rpc('get_recent_praise_summary', {}).then(function (res) {
          var el = document.getElementById('widget-summary');
          if (res && res.data) {
            el.textContent = 'Notes logged: ' + res.data.recent_count + ' (Total points: ' + res.data.total_points + ')';
          }
        });
      }
    });
  </script>
</body>
</html>
`
);

writeFile(
  'tests/init_test.lua',
  `function test_recordAndLoadPraise_withScopedTeacher_persistsNoteAndSatisfiesGdprHooks_test(t)
  -- 1. Load group 7A praise roster as Form Tutor 7A (stf_tch_7a)
  local roster_res = t.rpc("load_group_praise_roster", { group_id = "grp_reg_7a" }, "stf_tch_7a")
  assert(roster_res.ok == true, "expected load_group_praise_roster to succeed")
  assert(roster_res.data.group_id == "grp_reg_7a", "expected group_id grp_reg_7a")
  assert(#roster_res.data.pupils >= 1, "expected synthetic pupils in grp_reg_7a")

  -- 2. Record a praise note for Alice (pup_yr7_alice in grp_reg_7a)
  local create_res = t.rpc("record_pupil_praise_note", {
    pupil_id = "pup_yr7_alice",
    group_id = "grp_reg_7a",
    category = "Mathematics Mastery",
    points = 3,
    praise_note = "Explained simultaneous equations clearly to the class",
  }, "stf_tch_7a")
  assert(create_res.ok == true, "expected record_pupil_praise_note to succeed")
  assert(create_res.data.points == 3, "expected 3 points recorded")

  -- 3. Verify widget summary endpoint reflects the newly logged praise
  local summary_res = t.rpc("get_recent_praise_summary", {}, "stf_tch_7a")
  assert(summary_res.ok == true, "expected get_recent_praise_summary to succeed")
  assert(summary_res.data.recent_count == 1, "expected recent_count == 1")
  assert(summary_res.data.total_points == 3, "expected total_points == 3")

  -- 4. Verify UK GDPR SAR export_pupil_data hook returns redactable praise_note field
  local sar_res = t.rpc("export_pupil_data", { pupil_id = "pup_yr7_alice" }, "stf_admin_01")
  assert(sar_res.ok == true, "expected export_pupil_data to succeed")
  assert(#sar_res.data == 1, "expected 1 SAR section")
  assert(#sar_res.data[1].records == 1, "expected 1 exported praise record")

  -- 5. Verify UK GDPR purge_pupil_data hook removes all rows for pup_yr7_alice
  local purge_res = t.rpc("purge_pupil_data", { pupil_id = "pup_yr7_alice" }, "stf_admin_01")
  assert(purge_res.ok == true, "expected purge_pupil_data to succeed")

  local after_sar = t.rpc("export_pupil_data", { pupil_id = "pup_yr7_alice" }, "stf_admin_01")
  assert(#after_sar.data[1].records == 0, "expected 0 records after purge_pupil_data")
end
`
);

// 7. Write comprehensive README.md
writeFile(
  'README.md',
  `# Optimis School MIS — Plugin Starter Template

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
│   └── dal.d.lua                # EmmyLua type definitions for ctx.caller, ctx.dal.*, ctx.plugin_db, etc.
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
`
);

// 8. Write .agents/ guardrails & skills and per-folder GEMINI.md files
writeFile(
  '.agents/AGENTS.md',
  `# Repository Rules

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
`
);

writeFile(
  '.agents/GEMINI.md',
  `# Agent Rules & Skills (\`.agents\`)

This directory contains repository-wide rules and agent skills enforcing Test-Driven Development, Fail-Fast error handling, UK GDPR compliance, and documentation integrity.

## Files & Subdirectories

* [\`AGENTS.md\`](./AGENTS.md): Workspace guardrails and index of available \`npm\` commands.
* [\`skills/test-author/SKILL.md\`](./skills/test-author/SKILL.md): Workflow and guardrails for authoring headless Lua plugin tests (\`tests/*_test.lua\`) first before writing production code.
* [\`skills/code-implementer/SKILL.md\`](./skills/code-implementer/SKILL.md): Workflow and guardrails for implementing \`manifest.json\`, \`migrations/*.sql\`, \`src/init.lua\`, and \`ui/dist/*\` until \`npm test\` and \`npm run verify\` pass.
`
);

writeFile(
  '.agents/skills/test-author/SKILL.md',
  `---
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
`
);

writeFile(
  '.agents/skills/code-implementer/SKILL.md',
  `---
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
`
);

writeFile(
  'GEMINI.md',
  `# Optimis Plugin Repository — Architecture & Contract Blueprint

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
`
);

writeFile(
  '.mis-types/GEMINI.md',
  `# LuaLS Type Definitions (\`.mis-types\`)

This directory provides static EmmyLua / Lua Language Server (\`sumneko.lua\`) type definitions pre-bound via [\`.vscode/settings.json\`](../.vscode/settings.json).

## Files

* [\`dal.d.lua\`](./dal.d.lua): Defines \`OptimisPluginContext\` (\`ctx.caller\`, \`ctx.school\`, \`ctx.dal.*\`, \`ctx.plugin_db\`, \`ctx.events\`, \`ctx.http\`, \`ctx.crypto\`, \`ctx.xml\`) and global \`ctx\` for Lua 5.1 IDE autocomplete.
`
);

writeFile(
  'bin/GEMINI.md',
  `# Pre-Compiled Optimis SDK Toolchain Binaries (\`bin\`)

This directory contains zero-dependency static (\`CGO_ENABLED=0\`, \`-trimpath\`) builds of the four Optimis SDK CLI binaries across \`linux-amd64\`, \`darwin-arm64\`, \`darwin-amd64\`, and \`windows-amd64\`.

## Binaries

* \`mis-verify\`: Enforces the 4 automated CI gates (\`MANIFEST_LUA_UI_OK\`, \`CSP_OFFLINE_OK\`, \`MIGRATIONS_OK\`, \`GDPR_ZERO_ORPHANS_OK\`).
* \`mis-test\`: Executes headless Lua test suites (\`tests/*_test.lua\`) against an isolated synthetic UK school \`core.db\`.
* \`mis-dev-server\`: Boots the local Go Host container with an auto-seeded synthetic UK school \`core.db\` and hot-reload mounting for the plugin.
* \`mis-pack\`: Runs \`mis-verify\`, extracts the declared monochrome Lucide \`icon.svg\`, and bundles a signed \`.mispkg\` archive.
`
);

writeFile(
  'migrations/GEMINI.md',
  `# Plugin SQLite Migrations (\`migrations\`)

This directory holds sequential Expand-and-Contract SQL migration scripts applied inside an immediate transaction to the plugin's attached SQLite database (\`plugin_starter_notes.db\`).

## Guardrails

* Minor migrations must strictly follow **Expand-and-Contract** rules: never use \`DROP COLUMN\`, \`RENAME COLUMN\`, or \`DROP TABLE\` (enforced by \`npm run verify\` Gate 3 \`MIGRATIONS_OK\`).
* Every table storing pupil-specific records must use \`pupil_id\` and be cleaned up by \`purge_pupil_data\` in [\`src/init.lua\`](../src/init.lua).

## Files

* [\`001_init.sql\`](./001_init.sql): Creates the \`pupil_praise_notes\` table and indexes on \`pupil_id\` and \`group_id\`.
`
);

writeFile(
  'src/GEMINI.md',
  `# Sandboxed Lua 5.1 Plugin Runtime (\`src\`)

This directory contains the server-side Lua 5.1 entry point executed inside the Go Host's capability-sandboxed \`gopher-lua\` VM.

## Files

* [\`init.lua\`](./init.lua): Implements the endpoints and UK GDPR hooks declared in [\`manifest.json\`](../manifest.json):
  * \`load_group_praise_roster(ctx, args)\`: Reads scoped group pupils via \`ctx.dal.pupils.list_by_group\` and recent praise notes from \`plugin_starter_notes.pupil_praise_notes\`.
  * \`record_pupil_praise_note(ctx, args)\`: Verifies \`:own_groups\` pupil access via \`ctx.dal.pupils.get_by_id\` and inserts a praise note on \`WriteDB\`.
  * \`get_recent_praise_summary(ctx, args)\`: Aggregates recent praise counts and points for the 1x1 Home Dashboard widget.
  * \`export_pupil_data(ctx, args)\` & \`purge_pupil_data(ctx, args)\`: Mandatory UK GDPR SAR export (marking \`praise_note\` in \`redactable_fields\`) and zero-orphan retention purge hooks.
`
);

writeFile(
  'ui/GEMINI.md',
  `# Sandboxed Plugin Micro-Frontend UI (\`ui\`)

This directory contains the offline HTML/JS/CSS assets served inside \`<iframe sandbox="allow-scripts">\` with a strict \`connect-src 'none'\` Content Security Policy.

## Guardrails

* Zero external \`http://\` or \`https://\` scripts, stylesheets, fonts, or \`fetch()\` calls (enforced by \`npm run verify\` Gate 2 \`CSP_OFFLINE_OK\`).
* Zero emoji glyphs; all icons must use monochrome Lucide SVGs via \`data-mis-icon="<name>"\`.
* Every \`data-mis-rpc="<method>"\` attribute must match an endpoint declared in [\`manifest.json\`](../manifest.json).

## Files

* [\`dist/index.html\`](./dist/index.html): Full-page view (\`/\` and \`/student/:pupil_id\`) binding \`load_group_praise_roster\` and \`record_pupil_praise_note\` via \`/sdk/mis-bridge.js\` and \`/sdk/mis-theme.css\`.
* [\`dist/widget.html\`](./dist/widget.html): Preset-size \`1x1\` Home Dashboard mini-widget (\`praise_summary_widget\`) calling \`get_recent_praise_summary\`.
`
);

writeFile(
  'tests/GEMINI.md',
  `# Headless Lua Plugin Test Suite (\`tests\`)

This directory contains headless Lua test suites executed by \`npm test\` (\`mis-test\`) against an isolated synthetic UK school \`core.db\` (provisioned from [\`../schema/core/*.sql\`](../schema/core/GEMINI.md)).

## Conventions

* Test files must end with \`_test.lua\`.
* Test functions must start with \`test_\` and follow \`test_methodname_initialstate_expectedbehaviour_test(t)\`.
* Use \`t.rpc(method, args, optional_persona_id)\` or \`t.set_persona(persona_id)\` to test RBAC scopes and UK GDPR hooks.

## Files

* [\`init_test.lua\`](./init_test.lua): Verifies \`load_group_praise_roster\`, \`record_pupil_praise_note\`, \`get_recent_praise_summary\`, \`export_pupil_data\`, and \`purge_pupil_data\`.
`
);

writeFile(
  'scripts/GEMINI.md',
  `# Local CLI Helper Scripts (\`scripts\`)

This directory contains cross-platform helper scripts for invoking the pre-compiled Optimis SDK binaries in [\`../bin/\`](../bin/GEMINI.md).

## Files

* [\`run-cli.mjs\`](./run-cli.mjs): Detects the host OS (\`linux\`, \`darwin\`, \`windows\`) and CPU architecture (\`amd64\`, \`arm64\`) and spawns the corresponding binary from \`bin/<os>-<arch>/\` (or \`bin/\`).
`
);

// Sanitize any absolute /home/kieren/Documents/code/optimis links in copied sdk/ and schema/core/ GEMINI.md files
const sanitizeGeminiLinks = (dir) => {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      sanitizeGeminiLinks(full);
    } else if (entry.name === 'GEMINI.md' || entry.name.endsWith('.md')) {
      const raw = fs.readFileSync(full, 'utf-8');
      const cleaned = raw.replace(/file:\/\/\/home\/kieren\/Documents\/code\/optimis\//g, './');
      if (cleaned !== raw) {
        fs.writeFileSync(full, cleaned, 'utf-8');
      }
    }
  }
};
sanitizeGeminiLinks(path.join(targetDir, 'sdk'));
sanitizeGeminiLinks(path.join(targetDir, 'schema/core'));

console.log('Successfully populated optimis-plugin-template!');

