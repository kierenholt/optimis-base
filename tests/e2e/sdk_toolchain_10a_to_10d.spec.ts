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
      'description',
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

  test('productionPluginDalReadiness_allDomainsAndLuaTypeStubs_executesDashboardFsmAdmissionsGuardiansScrAndTimetableBuilderBindings_test', async () => {
    // 1. Verify .mis-types/dal.d.lua and sdk/lua/ctx_types.lua are identical and declare all actual Go Host bindings
    const dalTypesContent = fs.readFileSync(path.resolve(ROOT_DIR, '.mis-types/dal.d.lua'), 'utf-8');
    const sdkCtxTypesContent = fs.readFileSync(path.resolve(ROOT_DIR, 'sdk/lua/ctx_types.lua'), 'utf-8');
    expect(dalTypesContent).toBe(sdkCtxTypesContent);

    const expectedTypeSignatures = [
      'OptimisAcademicDAL',
      'list_years',
      'upsert_year',
      'list_terms',
      'upsert_term',
      'list_days_of_week',
      'upsert_day_of_week',
      'list_special_days',
      'upsert_special_day',
      'list_alert_levels',
      'OptimisDashboardDAL',
      'list_available_widgets',
      'list_role_defaults',
      'save_role_default',
      'get_staff_layout',
      'save_staff_layout',
      'reset_staff_layout',
      'list_fsm_periods_batch',
      'upsert_fsm_period',
      'upsert_pupil_admission',
      'upsert_year_enrolment',
      'list_exclusions',
      'upsert_exclusion',
      'list_alternative_provisions',
      'upsert_alternative_provision',
      'list_post16_learning_aims',
      'upsert_post16_learning_aim',
      'list_pastoral_notes',
      'upsert_pastoral_note',
      'list_need_types',
      'upsert_need_type',
      'upsert_sen_record',
      'upsert_sen_need',
      'upsert_sen_assistant',
      'list_conditions',
      'upsert_condition',
      'upsert_guardian_contact',
      'list_siblings_for_pupil',
      'list_households',
      'list_scr_register',
      'list_contracts',
      'list_absences',
      'upsert_scr_record',
      'upsert_contract',
      'upsert_absence',
      'upsert_staff_member',
      'list_roles',
      'list_cover_assignments',
      'upsert_cover_assignment',
      'list_houses',
      'upsert_house',
      'list_rooms',
      'upsert_room',
      'list_courses',
      'upsert_course',
      'list_departments',
      'upsert_department',
      'list_year_groups',
      'upsert_year_group',
      'list_timetable_periods',
      'list_timetable_slots',
      'get_pupil_timetable_slots',
      'list_pupil_memberships',
      'upsert_group',
      'upsert_timetable_period',
      'upsert_timetable_slot',
      'get_staff_timetable_slots_batch',
      'list_free_staff_for_period',
      'revoke_staff_assignment',
      'list_codes',
      'upsert_code',
      'get_group_lesson',
      'upsert_lesson_batch',
      'list_session_marks',
      'list_register_submissions',
      'create_postbox',
      'get_group_session',
      'get_pupil_summary',
    ];
    for (const sig of expectedTypeSignatures) {
      expect(dalTypesContent, `Expected ${sig} in .mis-types/dal.d.lua`).toContain(sig);
    }

    // 2. Verify scripts/populate_plugin_template.mjs includes mandatory widget description
    const populateScriptContent = fs.readFileSync(path.resolve(ROOT_DIR, 'scripts/populate_plugin_template.mjs'), 'utf-8');
    expect(populateScriptContent).toContain("id: 'praise_summary_widget'");
    expect(populateScriptContent).toContain('description:');

    // 3. Create a temporary plugin exercising all production DAL bindings via mis-test & mis-verify
    const tmpBase = fs.mkdtempSync(path.join(os.tmpdir(), 'optimis-prod-dal-readiness-'));
    tempDirs.push(tmpBase);
    const pluginDir = path.join(tmpBase, 'prod_dal_probe_plugin');
    fs.mkdirSync(path.join(pluginDir, 'src'), { recursive: true });
    fs.mkdirSync(path.join(pluginDir, 'ui/dist'), { recursive: true });
    fs.mkdirSync(path.join(pluginDir, 'tests'), { recursive: true });

    const manifest = {
      id: 'uk.org.optimis.test.prod_dal_probe',
      title: 'Production DAL Readiness Probe',
      version: '1.0.0',
      core_dal_version: '~1.4',
      icon: 'database',
      views: [
        {
          id: 'home',
          path: '/',
          entry: 'ui/dist/index.html',
          title: 'Production DAL Probe',
          icon: 'database',
          keywords: ['dal', 'probe'],
        },
      ],
      endpoints: {
        exercise_all_new_dal_domains: {
          scopes: [
            'dal.academic.read:all',
            'dal.academic.write:all',
            'dal.dashboard.read:all',
            'dal.dashboard.write:all',
            'dal.pupils.read:all',
            'dal.pupils.write:all',
            'dal.sen.read:all',
            'dal.sen.write:all',
            'dal.medical.read:all',
            'dal.medical.write:all',
            'dal.guardians.read:all',
            'dal.guardians.write:all',
            'dal.staff.read:admin',
            'dal.staff.write:admin',
            'dal.groups.read:all',
            'dal.groups.write:all',
            'dal.attendance.read:all',
            'dal.attendance.write:all',
            'dal.census.read:all',
            'plugin.config.write',
          ],
          mutates: true,
          timeout_ms: 5000,
        },
      },
    };
    fs.writeFileSync(path.join(pluginDir, 'manifest.json'), JSON.stringify(manifest, null, 2));
    fs.writeFileSync(
      path.join(pluginDir, 'ui/dist/index.html'),
      '<!doctype html><html><body><button data-mis-rpc="exercise_all_new_dal_domains">Run</button></body></html>'
    );
    fs.writeFileSync(
      path.join(pluginDir, 'src/init.lua'),
      `
function exercise_all_new_dal_domains(ctx, args)
  -- 0. Academic Calendar & Lookups DAL
  local ayUpserted = ctx.dal.academic.upsert_year({
    academic_year_id = "ay_2027_2028",
    name = "2027/2028",
    status = "FUTURE",
    sequence_number = 99,
    first_day = "2027-09-01",
    last_day = "2028-07-21"
  })
  local years = ctx.dal.academic.list_years()
  local termUpserted = ctx.dal.academic.upsert_term({
    academic_year_id = "ay_2026_2027",
    term_code = "AUT",
    name = "Autumn Term",
    sequence_number = 1,
    first_day = "2026-09-01",
    last_day = "2026-12-18",
    half_term_start = "2026-10-26",
    half_term_end = "2026-10-30"
  })
  local terms = ctx.dal.academic.list_terms({ academic_year_id = "ay_2026_2027" })
  local dayUpserted = ctx.dal.academic.upsert_day_of_week({
    id = 1,
    name = "Monday",
    name_short = "Mon",
    is_school_day = true,
    school_open_time = "08:00",
    am_register_time = "08:45",
    pm_register_time = "13:30",
    school_close_time = "16:00"
  })
  local daysOfWeek = ctx.dal.academic.list_days_of_week()
  local specDay = ctx.dal.academic.upsert_special_day({
    academic_year_id = "ay_2026_2027",
    date = "2026-09-02",
    day_type = "INSET_STAFF_ONLY",
    StatutoryAttendanceCode = "#",
    name = "Staff Training Day",
    applies_to_phase = "ALL"
  })
  local specialDays = ctx.dal.academic.list_special_days({ academic_year_id = "ay_2026_2027" })
  local alertLevels = ctx.dal.academic.list_alert_levels()

  -- 1. Dashboard DAL (Admin Dashboard Editor Plugin)
  local widgets = ctx.dal.dashboard.list_available_widgets()
  local savedRole = ctx.dal.dashboard.save_role_default({
    role_id = "TEACHER",
    layout = { { plug_id = "uk.org.optimis.test.prod_dal_probe", widget_id = "w1", col = 1, row = 1 } },
    append_to_existing_staff = true
  })
  local roleDef = ctx.dal.dashboard.get_role_default("TEACHER")
  local savedStaff = ctx.dal.dashboard.save_staff_layout({
    staff_id = "stf_tch_7a",
    column_count = 5,
    layout = { { plug_id = "uk.org.optimis.test.prod_dal_probe", widget_id = "w1", col = 2, row = 1 } }
  })
  local staffLayout = ctx.dal.dashboard.get_staff_layout("stf_tch_7a")
  local resetStaff = ctx.dal.dashboard.reset_staff_layout("stf_tch_7a")

  -- 2. Pupils DAL (Full CBDS, Admissions, Year Promotion, FSM, Exclusions, AP, Post-16 & Pastoral)
  local admitted = ctx.dal.pupils.upsert_pupil_admission({
    upn = "U100001202699",
    uln = "9876543210",
    first_name = " Clara ",
    last_name = "Barton",
    dob = "2014-05-12",
    sex = "F",
    ethnicity_code = "WBRI",
    first_language_code = "ENG",
    eal_flag = false,
    pupil_premium_flag = true,
    service_child_flag = true,
    looked_after_flag = false,
    nc_year_group = "7",
    academic_year_id = "ay_2026_2027",
    effective_from = "2026-09-01",
    fsm_eligible = true,
    fsm_effective_from = "2026-09-01"
  })
  local promoted = ctx.dal.pupils.upsert_year_enrolment({
    pupil_id = admitted.pupil_id,
    academic_year_id = "ay_2026_2027",
    nc_year_group = "8",
    enrolment_status = "C",
    ey_funded_hours = 15.0,
    effective_from = "2026-09-01"
  })
  local fetchedPupil = ctx.dal.pupils.get_by_id(admitted.pupil_id)
  local fsmUpserted = ctx.dal.pupils.upsert_fsm_period({
    pupil_id = admitted.pupil_id,
    fsm_type = "STATUTORY_FSM",
    ever6_fsm_flag = true,
    effective_from = "2026-09-01"
  })
  local fsmBatch = ctx.dal.pupils.list_fsm_periods_batch({ admitted.pupil_id })
  local excl = ctx.dal.pupils.upsert_exclusion({
    pupil_id = admitted.pupil_id,
    academic_year_id = "ay_2026_2027",
    category = "SUSPENSION",
    dfe_reason_code = "DB",
    start_date = "2026-10-05",
    end_date = "2026-10-06",
    sessions_lost = 4,
    authorising_staff_id = "stf_admin_01"
  })
  local exclusions = ctx.dal.pupils.list_exclusions({ pupil_id = admitted.pupil_id })
  local ap = ctx.dal.pupils.upsert_alternative_provision({
    pupil_id = admitted.pupil_id,
    academic_year_id = "ay_2026_2027",
    provider_urn = "139999",
    reason_code = "B",
    sessions_per_week = 4,
    effective_from = "2026-09-15"
  })
  local aps = ctx.dal.pupils.list_alternative_provisions({ pupil_id = admitted.pupil_id })
  local aim = ctx.dal.pupils.upsert_post16_learning_aim({
    pupil_id = admitted.pupil_id,
    academic_year_id = "ay_2026_2027",
    qan_code = "6031164X",
    discount_code = "FK2B",
    subject_title = "A Level Mathematics",
    planned_glh = 360,
    start_date = "2026-09-01",
    planned_end_date = "2028-06-30",
    completion_status = "1",
    outcome = "8"
  })
  local aims = ctx.dal.pupils.list_post16_learning_aims({ pupil_id = admitted.pupil_id })
  local pNote = ctx.dal.pupils.upsert_pastoral_note({
    pupil_id = admitted.pupil_id,
    recorded_by_staff_id = "stf_tch_7a",
    category = "ACHIEVEMENT",
    alert_level_id = 4,
    summary = "Excellent settling-in week"
  })
  local pNotes = ctx.dal.pupils.list_pastoral_notes(admitted.pupil_id)

  -- 3. SEN & Medical Write DAL
  local needType = ctx.dal.sen.upsert_need_type({
    code = "SPLD",
    name = "Specific Learning Difficulty",
    sequence_number = 1
  })
  local needTypes = ctx.dal.sen.list_need_types()
  local senRec = ctx.dal.sen.upsert_sen_record({
    pupil_id = admitted.pupil_id,
    sen_status_code = "K",
    key_worker_staff_id = "stf_senco_01",
    exam_access_extra_time_pct = 25,
    effective_from = "2026-09-01"
  })
  local senNeed = ctx.dal.sen.upsert_sen_need({
    pupil_id = admitted.pupil_id,
    sen_need_type_code = "SPLD",
    ranking = 1,
    effective_from = "2026-09-01"
  })
  local senAsst = ctx.dal.sen.upsert_sen_assistant({
    pupil_id = admitted.pupil_id,
    staff_id = "stf_support_01",
    support_hours_per_week = 10.5,
    effective_from = "2026-09-01"
  })
  local medCond = ctx.dal.medical.upsert_condition({
    pupil_id = admitted.pupil_id,
    condition_type = "ASTHMA",
    condition_name = "Mild Asthma",
    alert_level_id = 3,
    emergency_protocol = "Blue inhaler 2 puffs",
    medication_held_in_school = true,
    medication_location = "Medical Room Cabinet A",
    medication_expiry_date = "2027-09-01",
    ihp_review_date = "2027-03-01"
  })
  local medConds = ctx.dal.medical.list_conditions({ pupil_id = admitted.pupil_id })

  -- 4. Guardians, Households & Siblings DAL
  local sibling = ctx.dal.pupils.upsert_pupil_admission({
    upn = "U100001202698",
    first_name = "Leo",
    last_name = "Barton",
    dob = "2016-03-20",
    sex = "M",
    nc_year_group = "5",
    academic_year_id = "ay_2026_2027",
    effective_from = "2026-09-01"
  })
  local upsertedGuardian = ctx.dal.guardians.upsert_guardian_contact({
    pupil_id = admitted.pupil_id,
    guardian_person_id = "per_gdn_stephen_barton",
    household_id = "hh_barton_family",
    first_name = "Stephen",
    last_name = "Barton",
    email = "s.barton@household.uk",
    phone_primary = "07700900888",
    relationship_code = "FATHER",
    has_parental_responsibility = true,
    contact_priority = 1,
    address_line_1 = "42 High Street",
    town_city = "London",
    postcode = "SW1A 1AA",
    effective_from = "2026-09-01"
  })
  ctx.dal.guardians.upsert_guardian_contact({
    pupil_id = sibling.pupil_id,
    guardian_person_id = "per_gdn_stephen_barton",
    household_id = "hh_barton_family",
    first_name = "Stephen",
    last_name = "Barton",
    email = "s.barton@household.uk",
    phone_primary = "07700900888",
    relationship_code = "FATHER",
    has_parental_responsibility = true,
    contact_priority = 1,
    address_line_1 = "42 High Street",
    town_city = "London",
    postcode = "SW1A 1AA",
    effective_from = "2026-09-01"
  })
  local guardians = ctx.dal.guardians.list_for_pupil(admitted.pupil_id, { require_parental_responsibility = true })
  local siblings = ctx.dal.guardians.list_siblings_for_pupil(admitted.pupil_id)
  local households = ctx.dal.guardians.list_households({ pupil_id = admitted.pupil_id })

  -- 5. Staff HR / SCR / SWC / Cover Assignments DAL
  local supplyStaff = ctx.dal.staff.upsert_staff_member({
    staff_id = "stf_supply_probe",
    staff_code = "SUP99",
    first_name = "Arthur",
    last_name = "Pendelton",
    email = "a.pendelton@supply.uk",
    staff_type = "SUPPLY_AGENCY",
    job_title = "Supply Teacher",
    role_id = "TEACHER",
    qts_flag = true,
    hire_date = "2026-09-01"
  })
  local allStaff = ctx.dal.staff.list_all({ include_leavers = true })
  local roles = ctx.dal.staff.list_roles()
  local scrUpdated = ctx.dal.staff.upsert_scr_record({
    staff_id = "stf_tch_7a",
    trn = "7912345",
    dbs_check_date = "2026-08-01",
    dbs_certificate_number = "001999888777",
    prohibition_check_date = "2026-08-01",
    right_to_work_check_date = "2026-08-01",
    first_aid_qualified = true,
    first_aid_expiry_date = "2029-08-01"
  })
  local scrList = ctx.dal.staff.list_scr_register()
  local contract = ctx.dal.staff.upsert_contract({
    staff_id = "stf_tch_7a",
    contract_type = "PERMANENT",
    swc_post_code = "TCH",
    job_title = "Teacher of Maths",
    pay_scale = "UPS",
    pay_spine_point = "U1",
    fte = 1.0,
    hours_per_week = 32.5,
    weeks_per_year = 52.0,
    effective_from = "2026-09-01"
  })
  local contracts = ctx.dal.staff.list_contracts({ staff_id = "stf_tch_7a" })
  local absence = ctx.dal.staff.upsert_absence({
    staff_id = "stf_tch_7a",
    academic_year_id = "ay_2026_2027",
    absence_category_code = "TRN",
    reason_detail = "NPQ Leadership Day",
    status = "APPROVED",
    cover_required = true,
    start_date = "2026-10-15",
    end_date = "2026-10-15",
    working_days_lost = 1.0
  })
  local absences = ctx.dal.staff.list_absences({ staff_id = "stf_tch_7a" })

  -- 6. Groups, Houses, Rooms, Courses & Timetable Builder DAL
  local house = ctx.dal.groups.upsert_house({
    code = "DAR",
    name = "Darwin House",
    colour_hex = "#16A34A",
    head_of_house_staff_id = "stf_tch_7a"
  })
  local houses = ctx.dal.groups.list_houses()
  local dept = ctx.dal.groups.upsert_department({
    name = "Computer Science",
    name_short = "CSC",
    type = "LEARNING_AREA",
    head_of_department_staff_id = "stf_hod_maths"
  })
  local room = ctx.dal.groups.upsert_room({
    code = "IT01",
    name = "IT Suite 1",
    room_type = "IT_SUITE",
    capacity = 30,
    is_active = true,
    phone_extension = "2010"
  })
  local yg = ctx.dal.groups.upsert_year_group({
    nc_code = "12",
    name = "Year 12",
    phase = "SIXTH_FORM",
    key_stage = "KS5",
    sequence_number = 12,
    head_of_year_staff_id = "stf_hod_maths"
  })
  local course = ctx.dal.groups.upsert_course({
    academic_year_id = "ay_2026_2027",
    department_id = dept.id,
    code = "KS3-CSC",
    name = "KS3 Computer Science",
    dfe_subject_code = "I100"
  })
  local createdGroup = ctx.dal.groups.upsert_group({
    academic_year_id = "ay_2026_2027",
    group_type = "TEACHING_CLASS",
    code = "8Sc/Z1",
    name = "Year 8 Science Set 1",
    course_id = course.id,
    default_room_id = room.id,
    takes_attendance = true,
    effective_from = "2026-09-01"
  })
  ctx.dal.groups.upsert_pupil_membership({
    group_id = createdGroup.group_id,
    pupil_id = admitted.pupil_id,
    academic_year_id = "ay_2026_2027",
    effective_from = "2026-09-01"
  })
  local memberships = ctx.dal.groups.list_pupil_memberships({ group_id = createdGroup.group_id })
  local createdPeriod = ctx.dal.groups.upsert_timetable_period({
    academic_year_id = "ay_2026_2027",
    cycle_week = 1,
    day_of_week_id = 1,
    period_code = "P1_PROBE",
    name = "Period 1 Probe",
    start_time = "09:00",
    end_time = "10:00",
    period_type = "LESSON",
    maps_to_statutory_session = "AM",
    sequence_number = 10
  })
  local createdSlot = ctx.dal.groups.upsert_timetable_slot({
    timetable_period_id = createdPeriod.period_id,
    group_id = createdGroup.group_id,
    staff_id = "stf_tch_7a",
    room_id = room.id,
    effective_from = "2026-09-01"
  })
  local cover = ctx.dal.staff.upsert_cover_assignment({
    timetable_slot_id = createdSlot.slot_id,
    date = "2026-10-15",
    absent_staff_id = "stf_tch_7a",
    staff_absence_id = absence.absence_id,
    covering_staff_id = supplyStaff.staff_id,
    room_id = room.id,
    cover_notes = "Complete worksheet 4B",
    assigned_by_staff_id = "stf_admin_01"
  })
  local covers = ctx.dal.staff.list_cover_assignments({ date = "2026-10-15" })
  local pupilSlots = ctx.dal.groups.get_pupil_timetable_slots(admitted.pupil_id, { cycle_week = 1, day_of_week_id = 1 })
  local periods = ctx.dal.groups.list_timetable_periods({ cycle_week = 1, day_of_week_id = 1 })
  local slots = ctx.dal.groups.list_timetable_slots({ group_id = createdGroup.group_id })
  local rooms = ctx.dal.groups.list_rooms()
  local courses = ctx.dal.groups.list_courses()
  local depts = ctx.dal.groups.list_departments()
  local yearGroups = ctx.dal.groups.list_year_groups()

  -- 7. Attendance Codes, Lesson Marks, Session Marks & Register Submissions DAL
  local attCode = ctx.dal.attendance.upsert_code({
    code = "/",
    dfe_code = "/",
    description = "Present (AM)",
    statistical_meaning = "PRESENT",
    physical_presence = "ON_SITE",
    is_active = true
  })
  local attCodes = ctx.dal.attendance.list_codes()
  local lessonUpsert = ctx.dal.attendance.upsert_lesson_batch({
    group_id = createdGroup.group_id,
    timetable_slot_id = createdSlot.slot_id,
    date = "2026-10-12",
    marks = {
      {
        pupil_id = admitted.pupil_id,
        attendance_code = "/",
        minutes_late = 0,
        comment = "On time"
      }
    }
  })
  local lessonMarks = ctx.dal.attendance.get_group_lesson(createdGroup.group_id, createdSlot.slot_id, "2026-10-12")
  local sessionMarks = ctx.dal.attendance.list_session_marks({ pupil_id = admitted.pupil_id })
  local regSubs = ctx.dal.attendance.list_register_submissions({ group_id = createdGroup.group_id })

  -- 8. Expanded Statutory Census Rollup
  local rollup = ctx.dal.census.get_statutory_rollup({
    academic_year_id = "ay_2026_2027",
    reference_date = "2026-10-15"
  })

  return {
    ok = true,
    data = {
      years_count = #years,
      terms_count = #terms,
      days_count = #daysOfWeek,
      special_days_count = #specialDays,
      alert_levels_count = #alertLevels,
      widgets_type = type(widgets),
      role_def_count = #roleDef,
      staff_override_before_reset = staffLayout.has_override,
      reset_ok = resetStaff.reset,
      admitted_pupil_id = admitted.pupil_id,
      pupil_uln = fetchedPupil.uln,
      pupil_pp = fetchedPupil.pupil_premium_flag,
      promoted_year = promoted.nc_year_group,
      fsm_count = #(fsmBatch[admitted.pupil_id] or {}),
      exclusions_count = #exclusions,
      aps_count = #aps,
      aims_count = #aims,
      pastoral_count = #pNotes,
      need_types_count = #needTypes,
      med_conds_count = #medConds,
      guardian_count = #guardians,
      siblings_count = #siblings,
      households_count = #households,
      all_staff_count = #allStaff,
      roles_count = #roles,
      scr_count = #scrList,
      contract_id = contract.contract_id,
      contracts_count = #contracts,
      absence_id = absence.absence_id,
      absences_count = #absences,
      covers_count = #covers,
      houses_count = #houses,
      memberships_count = #memberships,
      pupil_slots_count = #pupilSlots,
      group_id = createdGroup.group_id,
      period_id = createdPeriod.period_id,
      slot_id = createdSlot.slot_id,
      periods_count = #periods,
      slots_count = #slots,
      att_codes_count = #attCodes,
      lesson_marks_count = #lessonMarks,
      reg_subs_count = #regSubs,
      rollup_exclusions_count = rollup.summary.exclusions_count
    }
  }
end
`
    );
    fs.writeFileSync(
      path.join(pluginDir, 'tests/probe_test.lua'),
      `
function test_production_dal_domains_readiness(t)
  local res = t.rpc("exercise_all_new_dal_domains", {}, "stf_admin_01")
  assert(res.ok == true, "expected exercise_all_new_dal_domains to succeed")
  assert(res.data.years_count >= 2, "expected academic years")
  assert(res.data.terms_count >= 1, "expected academic terms")
  assert(res.data.days_count == 7, "expected 7 days of week")
  assert(res.data.special_days_count >= 1, "expected special days")
  assert(res.data.alert_levels_count >= 1, "expected alert levels")
  assert(res.data.role_def_count == 1, "expected 1 role default widget")
  assert(res.data.staff_override_before_reset == true, "expected staff override before reset")
  assert(res.data.reset_ok == true, "expected staff reset to succeed")
  assert(res.data.pupil_uln == "9876543210", "expected pupil ULN to round-trip")
  assert(res.data.pupil_pp == true, "expected pupil_premium_flag to round-trip")
  assert(res.data.promoted_year == "8", "expected promoted year group to be 8")
  assert(res.data.fsm_count >= 1, "expected at least 1 FSM period for admitted pupil")
  assert(res.data.exclusions_count == 1, "expected 1 exclusion for admitted pupil")
  assert(res.data.aps_count == 1, "expected 1 alternative provision for admitted pupil")
  assert(res.data.aims_count == 1, "expected 1 post-16 learning aim for admitted pupil")
  assert(res.data.pastoral_count == 1, "expected 1 pastoral note for admitted pupil")
  assert(res.data.need_types_count >= 1, "expected SEN need types")
  assert(res.data.med_conds_count == 1, "expected 1 medical condition")
  assert(res.data.guardian_count == 1, "expected 1 PR guardian for admitted pupil")
  assert(res.data.siblings_count >= 1, "expected sibling link")
  assert(res.data.households_count >= 1, "expected household")
  assert(res.data.all_staff_count >= 2, "expected all staff")
  assert(res.data.roles_count >= 1, "expected roles")
  assert(res.data.scr_count >= 1, "expected SCR register entries")
  assert(res.data.contracts_count >= 1, "expected staff contract entries")
  assert(res.data.absences_count >= 1, "expected staff absence entries")
  assert(res.data.covers_count == 1, "expected 1 cover assignment")
  assert(res.data.houses_count >= 1, "expected houses")
  assert(res.data.memberships_count == 1, "expected 1 group pupil membership")
  assert(res.data.pupil_slots_count >= 1, "expected pupil timetable slots")
  assert(res.data.periods_count >= 1, "expected timetable periods")
  assert(res.data.slots_count == 1, "expected 1 timetable slot for new group")
  assert(res.data.att_codes_count >= 1, "expected attendance codes")
  assert(res.data.lesson_marks_count == 1, "expected 1 lesson attendance mark")
  assert(res.data.reg_subs_count >= 1, "expected register submission audit row")
  assert(res.data.rollup_exclusions_count >= 1, "expected census rollup exclusions count")
end
`
    );

    const verifyProbe = runBinary(MIS_VERIFY_BIN, [pluginDir]);
    expect(verifyProbe.exitCode, `verify failed: ${verifyProbe.stderr}\n${verifyProbe.stdout}`).toBe(0);

    const testProbe = runBinary(MIS_TEST_BIN, [pluginDir, '--persona', 'stf_admin_01']);
    expect(testProbe.exitCode, `test failed: ${testProbe.stderr}\n${testProbe.stdout}`).toBe(0);
    expect(testProbe.stdout).toContain('PASS');
  });
});


