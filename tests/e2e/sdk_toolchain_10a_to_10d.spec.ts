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

    // 3. Verify 10C @ukmis/bridge and @ukmis/tailwind-preset packages
    const bridgePkgPath = path.resolve(ROOT_DIR, 'sdk/packages/bridge/package.json');
    const tailwindPkgPath = path.resolve(ROOT_DIR, 'sdk/packages/tailwind-preset/package.json');
    expect(fs.existsSync(bridgePkgPath)).toBe(true);
    expect(fs.existsSync(tailwindPkgPath)).toBe(true);

    const bridgePkg = JSON.parse(fs.readFileSync(bridgePkgPath, 'utf-8'));
    const tailwindPkg = JSON.parse(fs.readFileSync(tailwindPkgPath, 'utf-8'));
    expect(bridgePkg.name).toBe('@ukmis/bridge');
    expect(tailwindPkg.name).toBe('@ukmis/tailwind-preset');

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

    await request.post('/api/admin/plugins/install', {
      data: { folder: 'behaviour_plugin', staff_id: adminStaffId },
    });
    await request.post('/api/admin/plugins/uk.org.optimis.behaviour/activate', {
      data: { staff_id: adminStaffId },
    });

    await page.goto('/');
    const navBtn = page.getByTestId('nav-item-plugin-uk.org.optimis.behaviour');
    await expect(navBtn).toBeVisible();
    await navBtn.click();

    const iframeEl = page.locator('[data-testid="plugin-iframe-uk.org.optimis.behaviour"]');
    await expect(iframeEl).toBeAttached();
    const frame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.behaviour"]');
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

  test('observabilityAndPersonaSwitching_urlParamAndHeaderAndDevServer_switchesPersonaViaUrlParamAndEmitsStructuredStdoutLogs_test', async ({
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

    // 1. Verify ?__persona= URL param on /api/auth/session and /api/system/ui-config
    const dslSessRes = await request.get(`/api/auth/session?__persona=${encodeURIComponent(dslStaffId)}`);
    expect(dslSessRes.ok()).toBe(true);
    const dslSessBody = await dslSessRes.json();
    expect(dslSessBody.caller.staff_id).toBe(dslStaffId);
    expect(dslSessBody.caller.scopes).toContain('dal.safeguarding.read');

    // 2. Verify X-MIS-Persona HTTP header on /api/auth/session
    const adminHeaderRes = await request.get('/api/auth/session', {
      headers: { 'X-MIS-Persona': adminStaffId },
    });
    expect(adminHeaderRes.ok()).toBe(true);
    const adminHeaderBody = await adminHeaderRes.json();
    expect(adminHeaderBody.caller.staff_id).toBe(adminStaffId);

    // 3. Navigate browser with ?__persona=<dslStaffId> and verify UI initializes as DSL persona
    await page.goto(`/?__persona=${encodeURIComponent(dslStaffId)}`);
    await expect(page.getByTestId('persona-switcher-select')).toHaveValue(dslStaffId);
    await expect(page.getByTestId('dsl-safeguarding-vault-panel')).toBeVisible();

    // 4. Verify ./bin/mis-dev-server --help / --check-config exits cleanly and supports --plugin & --persona flags
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
});
