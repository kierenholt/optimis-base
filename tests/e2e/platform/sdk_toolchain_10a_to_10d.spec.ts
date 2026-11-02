import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import {
  REQUIRED_TEMPLATE_FILES,
  EXPECTED_DAL_TYPE_SIGNATURES,
  BEHAVIOUR_LUA_TEST,
  FAILING_LUA_TEST,
  runBinary,
  copyPlugin,
  createProdDalProbePlugin,
  seedPersonaFixtures,
  installAndActivatePlugin,
  verifyVsCodeAndManifestSchema,
  verifySdkPackages,
  verifyDalAndRecordTypeStubs,
} from './sdk_toolchain_fixtures';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

const ROOT_DIR = process.cwd();
const MIS_VERIFY_BIN = path.resolve(ROOT_DIR, 'bin/mis-verify');
const MIS_TEST_BIN = path.resolve(ROOT_DIR, 'bin/mis-test');
const MIS_PACK_BIN = path.resolve(ROOT_DIR, 'bin/mis-pack');
const MIS_DEV_SERVER_BIN = path.resolve(ROOT_DIR, 'bin/mis-dev-server');

// =============================================================================
// Test Suite: Sections 10A–10D Toolchain & SDK Verification
// =============================================================================

test.describe('Plugin Authoring SDK & Developer Toolchain (Sections 10A–10D)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let tempDirs: string[] = [];
  let seededPersonaFixtureId: string = '';
  let installedPluginIds: string[] = [];

  const createTempDir = (prefix: string): string => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
    tempDirs.push(dir);
    return dir;
  };

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    tempDirs = [];
    seededPersonaFixtureId = '';
    installedPluginIds = [];

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
    // 1. Remove runtime-installed plugins (resilient teardown)
    for (const pluginId of installedPluginIds) {
      try {
        await request.delete(`/api/admin/plugins/${encodeURIComponent(pluginId)}`);
      } catch (err) {
        console.error(`Failed to uninstall plugin ${pluginId}:`, err);
      }
    }
    installedPluginIds = [];

    // 2. Clean up any seeded persona fixtures
    if (seededPersonaFixtureId) {
      try {
        await request.delete(`/api/test/seed-personas/${encodeURIComponent(seededPersonaFixtureId)}`);
      } catch (err) {
        console.error(`Failed to clean up persona fixture ${seededPersonaFixtureId}:`, err);
      }
      seededPersonaFixtureId = '';
    }

    // 3. Clean up temporary directories created during the test
    for (const dir of tempDirs) {
      try {
        if (fs.existsSync(dir)) {
          fs.rmSync(dir, { recursive: true, force: true });
        }
      } catch (err) {
        console.error(`Failed to clean up temp dir ${dir}:`, err);
      }
    }
    tempDirs = [];

    // 4. Fetch Backend Service Logs emitted during this test
    if (logCursor) {
      try {
        const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
        if (logsRes.ok()) {
          const serviceLogs: string[] = (await logsRes.json()).entries;
          if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
            console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
          }
        }
      } catch (err) {
        console.error('Failed to fetch service logs in afterEach:', err);
      }
    }

    // 5. Dump tapped Browser Console logs on failure or error
    if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
      console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
      console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
    }

    expect(pageErrors).toEqual([]);
  });

  // ===========================================================================
  // 10B & 10C: VS Code, Schema Integration & Frontend Bridge Packages
  // ===========================================================================

  test('vscodeAndSchemaIntegration_10bAnd10cAssets_validatesManifestSchemaLuaTypesBridgeUseMisRpcAndTailwindPreset_test', async ({
    page,
    request,
  }) => {
    // 1. Verify 10B VS Code settings & manifest schema
    verifyVsCodeAndManifestSchema(ROOT_DIR, expect);

    // 2. Verify 10C @optimis/bridge and @optimis/tailwind-preset packages
    verifySdkPackages(ROOT_DIR, expect);

    // 3. Mount behaviour_plugin in the UI and verify window.mis.useMisRpc is exposed by the bridge
    const seeded = await seedPersonaFixtures(request);
    seededPersonaFixtureId = seeded.fixture_id;
    const adminStaffId = seeded.admin_staff_id;

    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: adminStaffId },
    });
    expect(switchAdminRes.ok()).toBe(true);

    const behPlugId = await installAndActivatePlugin(
      request,
      'behaviour_plugin',
      'uk.org.optimis.behaviour',
      adminStaffId
    );
    installedPluginIds.push('uk.org.optimis.behaviour');

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
      const mis = (window as unknown as { mis?: Record<string, unknown> }).mis;
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

  // ===========================================================================
  // 10A: mis-verify Gatekeeper CLI
  // ===========================================================================

  test('misVerifyCli_validAndBrokenPluginBundles_enforcesManifestLuaUiCrossCheckCspExpandContractAndGdprZeroOrphans_test', async ({
    page,
    request,
  }) => {
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
    const tmpBase = createTempDir('optimis-verify-e2e-');
    const copyBehaviourTo = (subName: string): string =>
      copyPlugin('plugins/behaviour_plugin', path.join(tmpBase, subName), ROOT_DIR);

    // Failure A: UI HTML has data-mis-rpc pointing to an undeclared method in manifest.json
    const brokenUiDir = copyBehaviourTo('broken_ui_rpc');
    const uiIndexPath = path.join(brokenUiDir, 'ui/dist/index.html');
    fs.appendFileSync(uiIndexPath, '\n<button data-mis-rpc="undeclared_secret_endpoint">Bad</button>\n');
    const resBrokenUi = runBinary(MIS_VERIFY_BIN, [brokenUiDir]);
    expect(resBrokenUi.exitCode, 'Expected rejection of undeclared UI data-mis-rpc').not.toBe(0);
    expect(resBrokenUi.stdout + resBrokenUi.stderr).toContain('undeclared_secret_endpoint');

    // Failure A2: Widget in manifest.json declares invalid preset width/height (e.g. width: 3)
    const brokenWidgetSizeDir = copyBehaviourTo('broken_widget_size');
    const brokenWidgetManifestPath = path.join(brokenWidgetSizeDir, 'manifest.json');
    const brokenWidgetManifest = JSON.parse(fs.readFileSync(brokenWidgetManifestPath, 'utf-8'));
    brokenWidgetManifest.widgets[0].width = 3;
    fs.writeFileSync(brokenWidgetManifestPath, JSON.stringify(brokenWidgetManifest, null, 2));
    const resBrokenWidgetSize = runBinary(MIS_VERIFY_BIN, [brokenWidgetSizeDir]);
    expect(resBrokenWidgetSize.exitCode, 'Expected rejection of invalid widget preset width').not.toBe(0);
    expect(resBrokenWidgetSize.stdout + resBrokenWidgetSize.stderr).toContain('width');

    // Failure B: UI HTML references an external https:// script violating offline CSP
    const brokenCspDir = copyBehaviourTo('broken_csp');
    fs.appendFileSync(
      path.join(brokenCspDir, 'ui/dist/index.html'),
      '\n<script src="https://cdn.jsdelivr.net/npm/axios/dist/axios.min.js"></script>\n'
    );
    const resBrokenCsp = runBinary(MIS_VERIFY_BIN, [brokenCspDir]);
    expect(resBrokenCsp.exitCode, 'Expected rejection of external https:// script in offline CSP').not.toBe(0);
    expect(resBrokenCsp.stdout + resBrokenCsp.stderr).toContain('CSP');

    // Failure C: Destructive SQL migration (DROP COLUMN) violating Expand-and-Contract
    const brokenMigDir = copyBehaviourTo('broken_migration');
    fs.writeFileSync(
      path.join(brokenMigDir, 'migrations/002_drop.sql'),
      'ALTER TABLE behaviour_events DROP COLUMN incident_notes;\n'
    );
    const resBrokenMig = runBinary(MIS_VERIFY_BIN, [brokenMigDir]);
    expect(resBrokenMig.exitCode, 'Expected rejection of destructive DROP COLUMN migration').not.toBe(0);
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
    expect(resBrokenGdpr.exitCode, 'Expected rejection of GDPR purge leaving orphan pupil records').not.toBe(0);
    expect(resBrokenGdpr.stdout + resBrokenGdpr.stderr).toContain('orphan');
  });

  // ===========================================================================
  // 10A: mis-test & mis-pack Headless Execution & Packaging
  // ===========================================================================

  test('misTestAndMisPackCli_pluginLuaTestSuiteAndMispkgBundler_executesHeadlessTestsEmitsStructuredLogsAndExtractsLucideIcon_test', async ({
    page,
    request,
  }) => {
    const tmpBase = createTempDir('optimis-test-pack-e2e-');
    const pluginDir = copyPlugin('plugins/behaviour_plugin', path.join(tmpBase, 'sample_plugin'), ROOT_DIR);

    // Add icon field to manifest.json and author a Lua test file in tests/behaviour_test.lua
    const manifestPath = path.join(pluginDir, 'manifest.json');
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf-8'));
    manifest.icon = 'award';
    fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));

    const testsDir = path.join(pluginDir, 'tests');
    fs.mkdirSync(testsDir, { recursive: true });
    fs.writeFileSync(path.join(testsDir, 'behaviour_test.lua'), BEHAVIOUR_LUA_TEST);

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
    fs.writeFileSync(path.join(testsDir, 'failing_test.lua'), FAILING_LUA_TEST);
    const failRes = runBinary(MIS_TEST_BIN, [pluginDir, '--persona', 'stf_tch_7a']);
    expect(failRes.exitCode, 'Expected non-zero exit code on failing Lua assertion').not.toBe(0);
    expect(failRes.stdout + failRes.stderr).toContain('type=lua_err');
    expect(failRes.stdout + failRes.stderr).toContain('DELIBERATE_TEST_ASSERTION_FAILURE');
  });

  // ===========================================================================
  // 10D: Agent-First Observability & Persona Resolution
  // ===========================================================================

  test('observabilityAndPersonaResolution_cliFlagAndSession_resolvesPersonaAndEmitsStructuredStdoutLogs_test', async ({
    page,
    request,
  }) => {
    // Seed persona fixtures so stf_dsl_01, stf_admin_01, and stf_tch_01 are guaranteed present
    const seeded = await seedPersonaFixtures(request);
    seededPersonaFixtureId = seeded.fixture_id;
    const dslStaffId: string = seeded.dsl_staff_id;
    const adminStaffId: string = seeded.admin_staff_id;
    const teacherStaffId: string = seeded.teacher_staff_id;

    // 1. Set active dev persona to Teacher via POST /api/auth/persona
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

  // ===========================================================================
  // 10D: Standalone Plugin Template Verification (-trimpath Binaries)
  // ===========================================================================

  test('standalonePluginTemplate_trimpathPortableBinaries_verifiesTestsPacksAndDryRunsOutsideBaseRepo_test', async ({
    page,
    request,
  }) => {
    const templateDir = path.resolve(ROOT_DIR, '../optimis-plugin-template');
    expect(fs.existsSync(templateDir)).toBe(true);

    // Ensure template repository is populated before verification if missing generated files
    const manifestPath = path.join(templateDir, 'manifest.json');
    const recordsPath = path.join(templateDir, '.mis-types/records.d.lua');
    if (!fs.existsSync(manifestPath) || !fs.existsSync(recordsPath)) {
      const popRes = runBinary('node', ['scripts/populate_plugin_template.mjs', templateDir, '--skip-build'], ROOT_DIR);
      if (popRes.exitCode !== 0) {
        throw new Error(`FAIL FAST: populate_plugin_template failed: ${popRes.stderr}\n${popRes.stdout}`);
      }
    }

    // 1. Verify required template files, .agents guardrails/skills, and GEMINI.md files exist
    for (const rel of REQUIRED_TEMPLATE_FILES) {
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
    const verifyRes = runBinary('npm', ['run', 'verify'], templateDir);
    expect(verifyRes.exitCode, `verify failed: ${verifyRes.stderr}\n${verifyRes.stdout}`).toBe(0);
    expect(verifyRes.stdout).toContain('MANIFEST_LUA_UI_OK');
    expect(verifyRes.stdout).toContain('CSP_OFFLINE_OK');
    expect(verifyRes.stdout).toContain('MIGRATIONS_OK');
    expect(verifyRes.stdout).toContain('GDPR_ZERO_ORPHANS_OK');

    const testRes = runBinary('npm', ['test'], templateDir);
    expect(testRes.exitCode, `test failed: ${testRes.stderr}\n${testRes.stdout}`).toBe(0);
    expect(testRes.stdout).toContain('PASS');

    const dryRunRes = runBinary(
      'node',
      ['scripts/run-cli.mjs', 'mis-dev-server', '--plugin', '.', '--persona', 'stf_tch_7a', '--dry-run'],
      templateDir
    );
    expect(dryRunRes.exitCode, `dev-server dry-run failed: ${dryRunRes.stderr}\n${dryRunRes.stdout}`).toBe(0);
    expect(dryRunRes.stdout).toContain('status=READY');

    const tmpOutDir = createTempDir('optimis-template-pack-');
    const outMispkg = path.join(tmpOutDir, 'starter.mispkg');
    const packRes = runBinary(
      'node',
      ['scripts/run-cli.mjs', 'mis-pack', '.', '--out', outMispkg],
      templateDir
    );
    expect(packRes.exitCode, `pack failed: ${packRes.stderr}\n${packRes.stdout}`).toBe(0);
    expect(fs.existsSync(outMispkg)).toBe(true);
  });

  // ===========================================================================
  // 10D: 27-Plugin Production DAL Readiness Verification
  // ===========================================================================

  test('productionPluginDalReadiness_allDomainsAndLuaTypeStubs_executesDashboardFsmAdmissionsGuardiansScrAndTimetableBuilderBindings_test', async ({
    page,
    request,
  }) => {
    // 1. Verify .mis-types/dal.d.lua & records.d.lua and sdk/lua/*.lua are identical, under 500 lines, and declare all actual Go Host bindings
    verifyDalAndRecordTypeStubs(ROOT_DIR, expect);

    // 2. Verify scripts/populate_plugin_template.mjs includes mandatory widget description
    const populateScriptContent = fs.readFileSync(path.resolve(ROOT_DIR, 'scripts/populate_plugin_template.mjs'), 'utf-8');
    expect(populateScriptContent).toContain("id: 'praise_summary_widget'");
    expect(populateScriptContent).toContain('description:');

    // 3. Create a temporary plugin exercising all production DAL bindings via mis-test & mis-verify
    const tmpBase = createTempDir('optimis-prod-dal-readiness-');
    const pluginDir = path.join(tmpBase, 'prod_dal_probe_plugin');
    createProdDalProbePlugin(pluginDir);

    const verifyProbe = runBinary(MIS_VERIFY_BIN, [pluginDir]);
    expect(verifyProbe.exitCode, `verify failed: ${verifyProbe.stderr}\n${verifyProbe.stdout}`).toBe(0);

    const testProbe = runBinary(MIS_TEST_BIN, [pluginDir, '--persona', 'stf_admin_01']);
    expect(testProbe.exitCode, `test failed: ${testProbe.stderr}\n${testProbe.stdout}`).toBe(0);
    expect(testProbe.stdout).toContain('PASS');
  });
});
