import { test, expect } from '@playwright/test';
import { execFileSync } from 'child_process';
import fs from 'fs';
import os from 'os';
import path from 'path';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

const ROOT_DIR = process.cwd();
const MIS_VERIFY_BIN = path.resolve(ROOT_DIR, 'bin/mis-verify');
const MIS_TEST_BIN = path.resolve(ROOT_DIR, 'bin/mis-test');
const MIS_PACK_BIN = path.resolve(ROOT_DIR, 'bin/mis-pack');
const MIS_DEV_SERVER_BIN = path.resolve(ROOT_DIR, 'bin/mis-dev-server');

function runBinary(binPath: string, args: string[]): { stdout: string; stderr: string; exitCode: number } {
  try {
    const stdout = execFileSync(binPath, args, {
      cwd: ROOT_DIR,
      encoding: 'utf-8',
      env: { ...process.env },
    });
    return { stdout: stdout.trim(), stderr: '', exitCode: 0 };
  } catch (err: any) {
    return {
      stdout: typeof err.stdout === 'string' ? err.stdout.trim() : (err.stdout?.toString('utf-8') || '').trim(),
      stderr: typeof err.stderr === 'string' ? err.stderr.trim() : (err.stderr?.toString('utf-8') || '').trim(),
      exitCode: typeof err.status === 'number' ? err.status : 1,
    };
  }
}

test.describe('Plugin Authoring SDK & Developer Toolchain (Sections 10A–10D)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let tempDirs: string[] = [];
  let seededPersonaFixtureId: string = '';

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    tempDirs = [];
    seededPersonaFixtureId = '';

    // 1. Tap Browser Console & Uncaught Page Errors
    page.on('console', (msg) => {
      browserLogs.push(`[browser:${msg.type()}] ${msg.text()}`);
    });
    page.on('pageerror', (err) => {
      pageErrors.push(`[pageerror] ${err.stack || err.message}`);
    });

    // 2. Tap Backend Service Logs
    const cursorRes = await request.get('/api/test/logs/cursor');
    if (!cursorRes.ok()) {
      throw new Error(`Failed to obtain service log cursor: ${cursorRes.status()}`);
    }
    logCursor = (await cursorRes.json()).cursor;

    // 3. Ensure dev persona switcher mode is enabled
    const modeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!modeRes.ok()) {
      throw new Error(`Failed to reset auth-mode in beforeEach: ${modeRes.status()}`);
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Clean up any seeded persona fixtures
    if (seededPersonaFixtureId) {
      await request.delete(`/api/test/seed-personas/${encodeURIComponent(seededPersonaFixtureId)}`);
      seededPersonaFixtureId = '';
    }

    // 2. Clean up temporary directories created during the test
    for (const dir of tempDirs) {
      fs.rmSync(dir, { recursive: true, force: true });
    }
    tempDirs = [];

    // 3. Fetch Backend Service Logs emitted during this test
    if (logCursor) {
      const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
      if (logsRes.ok()) {
        const serviceLogs: string[] = (await logsRes.json()).entries;
        if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
          console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
        }
      }
    }

    // 4. Dump tapped Browser Console logs on failure or error
    if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
      console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
      console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
    }

    expect(pageErrors).toEqual([]);
  });

  test('vscodeAndSchemaIntegration_10bAnd10cAssets_validatesManifestSchemaLuaTypesBridgeUseMisRpcAndTailwindPreset_test', async ({
    page,
    request,
  }) => {
    // 1. Verify 10B VS Code settings (.vscode/settings.json)
    const vscodeSettingsPath = path.resolve(ROOT_DIR, '.vscode/settings.json');
    expect(fs.existsSync(vscodeSettingsPath)).toBe(true);
    const vscodeSettings = JSON.parse(fs.readFileSync(vscodeSettingsPath, 'utf-8'));
    expect(vscodeSettings['Lua.runtime.version']).toBe('Lua 5.1');
    expect(Array.isArray(vscodeSettings['Lua.workspace.library'])).toBe(true);
    expect(vscodeSettings['Lua.workspace.library']).toContain('.mis-types/dal.d.lua');
    expect(vscodeSettings['Lua.diagnostics.disable'] || vscodeSettings['Lua.runtime.builtin']).toBeDefined();
    expect(Array.isArray(vscodeSettings['json.schemas'])).toBe(true);

    // 2. Verify 10B .mis-types/dal.d.lua and sdk/manifest.schema.json
    const dalLuaTypesPath = path.resolve(ROOT_DIR, '.mis-types/dal.d.lua');
    expect(fs.existsSync(dalLuaTypesPath)).toBe(true);
    const dalLuaContent = fs.readFileSync(dalLuaTypesPath, 'utf-8');
    expect(dalLuaContent).toContain('OptimisPluginContext');
    expect(dalLuaContent).toContain('OptimisDAL');

    const manifestSchemaPath = path.resolve(ROOT_DIR, 'sdk/manifest.schema.json');
    expect(fs.existsSync(manifestSchemaPath)).toBe(true);
    const manifestSchema = JSON.parse(fs.readFileSync(manifestSchemaPath, 'utf-8'));
    expect(manifestSchema.properties).toHaveProperty('id');
    expect(manifestSchema.properties).toHaveProperty('version');
    expect(manifestSchema.properties).toHaveProperty('core_dal_version');
    expect(manifestSchema.properties).toHaveProperty('endpoints');
    expect(manifestSchema.properties.widgets.items.required).toEqual([
      'id',
      'title',
      'entry',
      'width',
      'height',
    ]);
    expect(manifestSchema.properties.widgets.items.properties.width).toEqual({
      type: 'integer',
      enum: [1, 2],
    });
    expect(manifestSchema.properties.widgets.items.properties.height).toEqual({
      type: 'integer',
      enum: [1, 2],
    });
    expect(manifestSchema.properties.widgets.items.properties).not.toHaveProperty('default_col_span');
    expect(manifestSchema.properties.widgets.items.properties).not.toHaveProperty('default_row_span');

    // 3. Verify 10C @optimis/bridge and @optimis/tailwind-preset packages
    const bridgePkgPath = path.resolve(ROOT_DIR, 'sdk/packages/bridge/package.json');
    const tailwindPkgPath = path.resolve(ROOT_DIR, 'sdk/packages/tailwind-preset/package.json');
    expect(fs.existsSync(bridgePkgPath)).toBe(true);
    expect(fs.existsSync(tailwindPkgPath)).toBe(true);

    const bridgePkg = JSON.parse(fs.readFileSync(bridgePkgPath, 'utf-8'));
    const tailwindPkg = JSON.parse(fs.readFileSync(tailwindPkgPath, 'utf-8'));
    expect(bridgePkg.name).toBe('@optimis/bridge');
    expect(tailwindPkg.name).toBe('@optimis/tailwind-preset');

    // 4. Mount test_plugin in the UI and verify window.mis.useMisRpc is exposed by the bridge
    const seedRes = await request.post('/api/test/seed-personas', {
      data: {
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    expect(seedRes.ok()).toBe(true);
    const seeded = await seedRes.json();
    seededPersonaFixtureId = seeded.fixture_id;
    const adminStaffId = seeded.admin_staff_id;

    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: adminStaffId },
    });
    expect(switchAdminRes.ok()).toBe(true);

    const installBehRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'behaviour_plugin', staff_id: adminStaffId },
    });
    expect(installBehRes.ok()).toBe(true);
    const behPlugId = (await installBehRes.json()).plug_id;
    await request.post('/api/admin/plugins/uk.org.optimis.behaviour/activate', {
      data: { staff_id: adminStaffId },
    });

    await page.goto('/');
    await page.getByTestId('nav-rail-search').click();
    const navBtn = page.getByTestId(`host-search-result-view-${behPlugId}-home`);
    await expect(navBtn).toBeVisible();
    await navBtn.click();

    const iframeEl = page.locator(`[data-testid="plugin-iframe-${behPlugId}"]`);
    await expect(iframeEl).toBeAttached();
    const frame = page.frameLocator(`[data-testid="plugin-iframe-${behPlugId}"]`);
    await expect(frame.locator('body')).toBeVisible();

    const bridgeCapabilities = await frame.locator('body').evaluate(() => {
      const mis = (window as any).mis;
      return {
        hasRpc: typeof mis?.rpc === 'function',
        hasUseMisRpc: typeof mis?.useMisRpc === 'function',
        hasOnPersonaChanged: typeof mis?.onPersonaChanged === 'function',
      };
    });
    expect(bridgeCapabilities).toEqual({
      hasRpc: true,
      hasUseMisRpc: true,
      hasOnPersonaChanged: true,
    });
  });

  test('misVerifyCli_validAndBrokenPluginBundles_enforcesManifestLuaUiCrossCheckCspExpandContractAndGdprZeroOrphans_test', async () => {
    // 1. Verify valid production plugins pass all 4 gatekeeper checks
    const validBehaviour = runBinary(MIS_VERIFY_BIN, ['plugins/behaviour_plugin']);
    expect(validBehaviour.exitCode).toBe(0);
    expect(validBehaviour.stdout).toContain('MANIFEST_LUA_UI_OK');
    expect(validBehaviour.stdout).toContain('CSP_OFFLINE_OK');
    expect(validBehaviour.stdout).toContain('MIGRATIONS_OK');
    expect(validBehaviour.stdout).toContain('GDPR_ZERO_ORPHANS_OK');

    const validTimetable = runBinary(MIS_VERIFY_BIN, ['plugins/timetable_attendance_plugin']);
    expect(validTimetable.exitCode).toBe(0);
    expect(validTimetable.stdout).toContain('MANIFEST_LUA_UI_OK');

    // 2. Create temp plugin fixtures to test each of the 4 failure modes
    const tmpBase = fs.mkdtempSync(path.join(os.tmpdir(), 'optimis-verify-e2e-'));
    tempDirs.push(tmpBase);

    const copyBehaviourTo = (subName: string): string => {
      const dest = path.join(tmpBase, subName);
      fs.cpSync(path.resolve(ROOT_DIR, 'plugins/behaviour_plugin'), dest, { recursive: true });
      return dest;
    };

    // Failure A: UI HTML has data-mis-rpc pointing to an undeclared method in manifest.json
    const brokenUiDir = copyBehaviourTo('broken_ui_rpc');
    const uiIndexPath = path.join(brokenUiDir, 'ui/dist/index.html');
    fs.appendFileSync(uiIndexPath, '\n<button data-mis-rpc="undeclared_secret_endpoint">Bad</button>\n');
    const resBrokenUi = runBinary(MIS_VERIFY_BIN, [brokenUiDir]);
    expect(resBrokenUi.exitCode).not.toBe(0);
    expect(resBrokenUi.stdout + resBrokenUi.stderr).toContain('undeclared_secret_endpoint');

    // Failure A2: Widget in manifest.json declares invalid preset width/height (e.g. width: 3 or missing height)
    const brokenWidgetSizeDir = copyBehaviourTo('broken_widget_size');
    const brokenWidgetManifestPath = path.join(brokenWidgetSizeDir, 'manifest.json');
    const brokenWidgetManifest = JSON.parse(fs.readFileSync(brokenWidgetManifestPath, 'utf-8'));
    brokenWidgetManifest.widgets[0].width = 3;
    fs.writeFileSync(brokenWidgetManifestPath, JSON.stringify(brokenWidgetManifest, null, 2));
    const resBrokenWidgetSize = runBinary(MIS_VERIFY_BIN, [brokenWidgetSizeDir]);
    expect(resBrokenWidgetSize.exitCode).not.toBe(0);
    expect(resBrokenWidgetSize.stdout + resBrokenWidgetSize.stderr).toContain('width');

    // Failure B: UI HTML references an external https:// script violating offline CSP
    const brokenCspDir = copyBehaviourTo('broken_csp');
    fs.appendFileSync(
      path.join(brokenCspDir, 'ui/dist/index.html'),
      '\n<script src="https://cdn.jsdelivr.net/npm/axios/dist/axios.min.js"></script>\n'
    );
    const resBrokenCsp = runBinary(MIS_VERIFY_BIN, [brokenCspDir]);
    expect(resBrokenCsp.exitCode).not.toBe(0);
    expect(resBrokenCsp.stdout + resBrokenCsp.stderr).toContain('CSP');

    // Failure C: Destructive SQL migration (DROP COLUMN) violating Expand-and-Contract
    const brokenMigDir = copyBehaviourTo('broken_migration');
    fs.writeFileSync(
      path.join(brokenMigDir, 'migrations/002_drop.sql'),
      'ALTER TABLE behaviour_events DROP COLUMN incident_notes;\n'
    );
    const resBrokenMig = runBinary(MIS_VERIFY_BIN, [brokenMigDir]);
    expect(resBrokenMig.exitCode).not.toBe(0);
    expect(resBrokenMig.stdout + resBrokenMig.stderr).toContain('DROP COLUMN');

    // Failure D: GDPR purge_pupil_data leaves orphaned pupil rows
    const brokenGdprDir = copyBehaviourTo('broken_gdpr');
    const luaInitPath = path.join(brokenGdprDir, 'src/init.lua');
    const origLua = fs.readFileSync(luaInitPath, 'utf-8');
    const brokenLua =
      origLua +
      '\nfunction purge_pupil_data(ctx, args)\n  return { ok = true, data = { purged = false } }\nend\n';
    fs.writeFileSync(luaInitPath, brokenLua);
    const resBrokenGdpr = runBinary(MIS_VERIFY_BIN, [brokenGdprDir]);
    expect(resBrokenGdpr.exitCode).not.toBe(0);
    expect(resBrokenGdpr.stdout + resBrokenGdpr.stderr).toContain('orphan');
  });

  test('misTestAndMisPackCli_pluginLuaTestSuiteAndMispkgBundler_executesHeadlessTestsEmitsStructuredLogsAndExtractsLucideIcon_test', async () => {
    const tmpBase = fs.mkdtempSync(path.join(os.tmpdir(), 'optimis-test-pack-e2e-'));
    tempDirs.push(tmpBase);

    const pluginDir = path.join(tmpBase, 'sample_plugin');
    fs.cpSync(path.resolve(ROOT_DIR, 'plugins/behaviour_plugin'), pluginDir, { recursive: true });

    // Add icon field to manifest.json and author a Lua test file in tests/behaviour_test.lua
    const manifestPath = path.join(pluginDir, 'manifest.json');
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf-8'));
    manifest.icon = 'award';
    fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));

    const testsDir = path.join(pluginDir, 'tests');
    fs.mkdirSync(testsDir, { recursive: true });
    fs.writeFileSync(
      path.join(testsDir, 'behaviour_test.lua'),
      `
function test_log_merit_and_export_purge_pupil_data(t)
  local res = t.rpc("log_behaviour_event", {
    pupil_id = "pup_yr7_alice",
    group_id = "grp_reg_7a",
    event_type = "MERIT",
    points = 5,
    category = "Academic Excellence",
    incident_notes = "Outstanding algebra homework"
  })
  assert(res.ok == true, "expected log_behaviour_event to succeed")

  local exp = t.rpc("export_pupil_data", { pupil_id = "pup_yr7_alice" }, "stf_admin_01")
  assert(exp.ok == true, "expected export_pupil_data to succeed")
  assert(#exp.data >= 1, "expected at least 1 exported section")

  local prg = t.rpc("purge_pupil_data", { pupil_id = "pup_yr7_alice" }, "stf_admin_01")
  assert(prg.ok == true, "expected purge_pupil_data to succeed")
end
`
    );

    // 1. Run mis-test with --persona stf_tch_7a
    const testRes = runBinary(MIS_TEST_BIN, [pluginDir, '--persona', 'stf_tch_7a']);
    expect(testRes.exitCode).toBe(0);
    expect(testRes.stdout).toContain('type=rpc');
    expect(testRes.stdout).toContain('plugin=uk.org.optimis.behaviour');
    expect(testRes.stdout).toContain('method=log_behaviour_event');
    expect(testRes.stdout).toContain('sqlite_write_hold_ms=');
    expect(testRes.stdout).toContain('PASS');

    // 2. Run mis-pack to bundle into a .mispkg archive
    const outputPkg = path.join(tmpBase, 'behaviour.mispkg');
    const packRes = runBinary(MIS_PACK_BIN, [pluginDir, '-o', outputPkg]);
    expect(packRes.exitCode).toBe(0);
    expect(fs.existsSync(outputPkg)).toBe(true);
    expect(packRes.stdout).toContain('icon.svg');
    expect(packRes.stdout).toContain('SIGNATURE.json');

    // 3. Verify failing Lua test emits type=lua_err with file/line stack trace and exits non-zero
    fs.writeFileSync(
      path.join(testsDir, 'failing_test.lua'),
      `
function test_deliberate_failure(t)
  error("DELIBERATE_TEST_ASSERTION_FAILURE")
end
`
    );
    const failRes = runBinary(MIS_TEST_BIN, [pluginDir, '--persona', 'stf_tch_7a']);
    expect(failRes.exitCode).not.toBe(0);
    expect(failRes.stdout + failRes.stderr).toContain('type=lua_err');
    expect(failRes.stdout + failRes.stderr).toContain('DELIBERATE_TEST_ASSERTION_FAILURE');
  });

  test('observabilityAndPersonaResolution_cliFlagAndSession_resolvesPersonaAndEmitsStructuredStdoutLogs_test', async ({
    page,
    request,
  }) => {
    // Seed persona fixtures so stf_dsl_01, stf_admin_01, and stf_tch_01 are guaranteed present
    const seedRes = await request.post('/api/test/seed-personas', {
      data: {
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    expect(seedRes.ok()).toBe(true);
    const seeded = await seedRes.json();
    seededPersonaFixtureId = seeded.fixture_id;
    const dslStaffId: string = seeded.dsl_staff_id;
    const adminStaffId: string = seeded.admin_staff_id;

    // 1. Set active dev persona to Teacher via POST /api/auth/persona
    const teacherStaffId: string = seeded.teacher_staff_id;
    const setTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: teacherStaffId },
    });
    expect(setTeacherRes.ok()).toBe(true);

    // 2. Verify active session resolves to Teacher and HTTP headers (X-MIS-Persona / X-MIS-Staff-ID) do not override caller identity
    const sessionRes = await request.get('/api/auth/session', {
      headers: {
        'X-MIS-Persona': adminStaffId,
        'X-MIS-Staff-ID': adminStaffId,
      },
    });
    expect(sessionRes.ok()).toBe(true);
    const sessionBody = await sessionRes.json();
    expect(sessionBody.caller.staff_id).toBe(teacherStaffId);
    expect(sessionBody.caller.scopes).not.toContain('dal.safeguarding.read');

    // 3. Switch active dev persona to DSL via POST /api/auth/persona and verify UI initializes as DSL persona without topbar switcher
    const setDslRes = await request.post('/api/auth/persona', {
      data: { staff_id: dslStaffId },
    });
    expect(setDslRes.ok()).toBe(true);
    await page.goto('/');
    await expect(page.getByTestId('host-topbar')).toHaveCount(0);
    await expect(page.getByTestId('persona-switcher-select')).toHaveCount(0);
    await expect(page.getByTestId('dsl-safeguarding-vault-panel')).toBeVisible();

    // 4. Verify ./bin/mis-dev-server --help / --dry-run exits cleanly and supports --plugin & --persona CLI flags
    const devServerCheck = runBinary(MIS_DEV_SERVER_BIN, [
      '--plugin',
      'plugins/timetable_attendance_plugin',
      '--persona',
      'stf_tch_7a',
      '--dry-run',
    ]);
    expect(devServerCheck.exitCode).toBe(0);
    expect(devServerCheck.stdout).toContain('uk.org.optimis.timetable.attendance');
    expect(devServerCheck.stdout).toContain('persona=stf_tch_7a');
  });

  test('standalonePluginTemplate_trimpathPortableBinaries_verifiesTestsPacksAndDryRunsOutsideBaseRepo_test', async () => {
    const templateDir = path.resolve(ROOT_DIR, '../optimis-plugin-template');
    expect(fs.existsSync(templateDir)).toBe(true);

    // 1. Verify required template files, .agents guardrails/skills, and GEMINI.md files exist
    const requiredFiles = [
      'README.md',
      'GEMINI.md',
      '.agents/AGENTS.md',
      '.agents/GEMINI.md',
      '.agents/skills/test-author/SKILL.md',
      '.agents/skills/code-implementer/SKILL.md',
      'package.json',
      'manifest.json',
      '.vscode/settings.json',
      '.vscode/extensions.json',
      '.mis-types/dal.d.lua',
      '.mis-types/GEMINI.md',
      'bin/GEMINI.md',
      'sdk/GEMINI.md',
      'sdk/manifest.schema.json',
      'sdk/ui/mis-bridge.js',
      'sdk/ui/mis-theme.css',
      'schema/core/GEMINI.md',
      'schema/core/00_system_and_academic_cycle.sql',
      'schema/core/SCHEMA_MAP.md',
      'migrations/GEMINI.md',
      'migrations/001_init.sql',
      'src/GEMINI.md',
      'src/init.lua',
      'ui/GEMINI.md',
      'ui/dist/index.html',
      'ui/dist/widget.html',
      'tests/GEMINI.md',
      'tests/init_test.lua',
      'scripts/GEMINI.md',
      'scripts/run-cli.mjs',
    ];
    for (const rel of requiredFiles) {
      expect(fs.existsSync(path.join(templateDir, rel)), `Expected ${rel} in optimis-plugin-template`).toBe(true);
    }

    const agentsContent = fs.readFileSync(path.join(templateDir, '.agents/AGENTS.md'), 'utf-8');
    expect(agentsContent).toContain('NO FALLBACKS / FAIL FAST');
    expect(agentsContent).toContain('Available NPM Commands');
    expect(agentsContent).toContain('GEMINI.md');

    const rootGeminiContent = fs.readFileSync(path.join(templateDir, 'GEMINI.md'), 'utf-8');
    expect(rootGeminiContent).toContain('Plugin Contract');
    expect(rootGeminiContent).toContain('GDPR');

    const readmeContent = fs.readFileSync(path.join(templateDir, 'README.md'), 'utf-8');
    expect(readmeContent).toContain('npm run verify');
    expect(readmeContent).toContain('npm test');
    expect(readmeContent).toContain('npm run dev');
    expect(readmeContent).toContain('npm run pack');

    // 2. Execute npm run verify, npm test, npm run pack, and mis-dev-server --dry-run directly inside templateDir
    // (where schema/core does not exist on disk and binaries are compiled with -trimpath)
    const runInTemplate = (cmd: string, args: string[]) => {
      try {
        const stdout = execFileSync(cmd, args, {
          cwd: templateDir,
          encoding: 'utf-8',
          env: { ...process.env },
        });
        return { stdout: stdout.trim(), stderr: '', exitCode: 0 };
      } catch (err: any) {
        return {
          stdout: typeof err.stdout === 'string' ? err.stdout.trim() : (err.stdout?.toString('utf-8') || '').trim(),
          stderr: typeof err.stderr === 'string' ? err.stderr.trim() : (err.stderr?.toString('utf-8') || '').trim(),
          exitCode: typeof err.status === 'number' ? err.status : 1,
        };
      }
    };

    const verifyRes = runInTemplate('npm', ['run', 'verify']);
    expect(verifyRes.exitCode, `verify failed: ${verifyRes.stderr}\n${verifyRes.stdout}`).toBe(0);
    expect(verifyRes.stdout).toContain('MANIFEST_LUA_UI_OK');
    expect(verifyRes.stdout).toContain('CSP_OFFLINE_OK');
    expect(verifyRes.stdout).toContain('MIGRATIONS_OK');
    expect(verifyRes.stdout).toContain('GDPR_ZERO_ORPHANS_OK');

    const testRes = runInTemplate('npm', ['test']);
    expect(testRes.exitCode, `test failed: ${testRes.stderr}\n${testRes.stdout}`).toBe(0);
    expect(testRes.stdout).toContain('PASS');

    const dryRunRes = runInTemplate('node', ['scripts/run-cli.mjs', 'mis-dev-server', '--plugin', '.', '--persona', 'stf_tch_7a', '--dry-run']);
    expect(dryRunRes.exitCode, `dev-server dry-run failed: ${dryRunRes.stderr}\n${dryRunRes.stdout}`).toBe(0);
    expect(dryRunRes.stdout).toContain('status=READY');

    const tmpOutDir = fs.mkdtempSync(path.join(os.tmpdir(), 'optimis-template-pack-'));
    tempDirs.push(tmpOutDir);
    const outMispkg = path.join(tmpOutDir, 'starter.mispkg');
    const packRes = runInTemplate('node', ['scripts/run-cli.mjs', 'mis-pack', '.', '--out', outMispkg]);
    expect(packRes.exitCode, `pack failed: ${packRes.stderr}\n${packRes.stdout}`).toBe(0);
    expect(fs.existsSync(outMispkg)).toBe(true);
  });
});

