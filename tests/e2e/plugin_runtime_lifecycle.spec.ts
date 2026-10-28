import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

interface SeededGDPRFixtures {
  fixture_id: string;
  academic_year_id: string;
  reference_date: string;
  admin_staff_id: string;
  teacher_staff_id: string;
  dsl_staff_id: string;
  pupil_id: string;
  protected_pupil_id: string;
  pastoral_note_id: string;
  safeguarding_note_id: string;
}

test.describe('Runtime Plugin Lifecycle API: Install, Activate & Remove (Sections 6A & 9.3)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let seeded: SeededGDPRFixtures | null = null;

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    seeded = null;

    // 1. Tap Browser Console & Uncaught Page Errors
    page.on('console', (msg) => {
      browserLogs.push(`[browser:${msg.type()}] ${msg.text()}`);
    });
    page.on('pageerror', (err) => {
      pageErrors.push(`[pageerror] ${err.stack || err.message}`);
    });

    // 2. Tap Backend Service Logs (record cursor before test execution)
    const cursorRes = await request.get('/api/test/logs/cursor');
    if (!cursorRes.ok()) {
      throw new Error(`Failed to obtain service log cursor: ${cursorRes.status()}`);
    }
    logCursor = (await cursorRes.json()).cursor;

    // 3. Ensure dev persona switcher is enabled and clean up any previously installed test_plugin
    const authModeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!authModeRes.ok()) {
      throw new Error(`Failed to enable dev persona switcher: ${authModeRes.status()}`);
    }
    await request.delete('/api/admin/plugins/uk.org.optimis.test');

    // 4. Seed isolated staff personas (Admin, Teacher, DSL) & pupil fixture
    const seedRes = await request.post('/api/test/seed-gdpr-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-20',
      },
    });
    if (!seedRes.ok()) {
      throw new Error(`Failed to seed GDPR fixtures: ${seedRes.status()} ${await seedRes.text()}`);
    }
    const body = (await seedRes.json()) as SeededGDPRFixtures;
    if (!body.fixture_id || !body.admin_staff_id || !body.teacher_staff_id || !body.pupil_id) {
      throw new Error('Seeded fixture response missing mandatory identifiers');
    }
    seeded = body;
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Fetch Backend Service Logs emitted during this test
    if (logCursor) {
      const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
      if (logsRes.ok()) {
        const serviceLogs: string[] = (await logsRes.json()).entries;
        if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
          console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
        }
      }
    }

    // 2. Dump tapped Browser Console logs on failure or error
    if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
      console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
      console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
    }

    // 3. Reset auth mode and ensure test_plugin is removed if still installed
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete('/api/admin/plugins/uk.org.optimis.test');

    // 4. Clean up seeded fixtures in core.db
    if (seeded?.fixture_id) {
      const delRes = await request.delete(`/api/test/seed-gdpr-fixtures/${encodeURIComponent(seeded.fixture_id)}`);
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up fixture ${seeded.fixture_id}: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('installActivateAndRemovePlugin_adminInstallsActivatesAndRemovesPluginAtRuntime_provisionsDbExposesUiAndDetachesOnRemove_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded state');
    }

    // 1. Verify RBAC: Non-Admin teacher is rejected with 403 (UNAUTHORIZED_ADMIN_PLUGIN_SCOPE) on install
    const unauthInstallRes = await request.post('/api/admin/plugins/install', {
      data: {
        folder: 'test_plugin',
        staff_id: seeded.teacher_staff_id,
      },
    });
    expect(unauthInstallRes.status()).toBe(403);
    const unauthInstallBody = await unauthInstallRes.json();
    expect(unauthInstallBody.ok).toBe(false);
    expect(unauthInstallBody.error_code).toContain('UNAUTHORIZED_ADMIN_PLUGIN_SCOPE');

    // 2. Verify fail-fast validation on invalid/traversal folder
    const badFolderRes = await request.post('/api/admin/plugins/install', {
      data: {
        folder: '../invalid_folder',
        staff_id: seeded.admin_staff_id,
      },
    });
    expect(badFolderRes.status()).toBe(400);

    // 3. Admin installs test_plugin -> provisions & attaches plugin_test.db, starts in UNINSTALLED_DORMANT state
    const installRes = await request.post('/api/admin/plugins/install', {
      data: {
        folder: 'test_plugin',
        staff_id: seeded.admin_staff_id,
      },
    });
    expect(installRes.status()).toBe(201);
    const installBody = await installRes.json();
    expect(installBody.ok).toBe(true);
    expect(installBody.plugin_id).toBe('uk.org.optimis.test');
    expect(installBody.folder).toBe('test_plugin');
    expect(installBody.state).toBe('UNINSTALLED_DORMANT');

    // 4. Verify GET /api/admin/plugins lists uk.org.optimis.test in UNINSTALLED_DORMANT state
    const listDormantRes = await request.get(
      `/api/admin/plugins?staff_id=${encodeURIComponent(seeded.admin_staff_id)}`
    );
    expect(listDormantRes.status()).toBe(200);
    const listDormantBody = await listDormantRes.json();
    expect(listDormantBody.ok).toBe(true);
    const dormantEntry = (listDormantBody.plugins || []).find(
      (p: { id: string }) => p.id === 'uk.org.optimis.test'
    );
    expect(dormantEntry).toBeDefined();
    expect(dormantEntry.state).toBe('UNINSTALLED_DORMANT');
    expect(dormantEntry.database_alias).toBe('plugin_test');

    // 5. While in UNINSTALLED_DORMANT (before activation), GET /api/system/ui-config must NOT expose the plugin in UI
    //    and interactive RPC calls must return PLUGIN_DORMANT_QUARANTINE
    const uiConfigBeforeRes = await request.get('/api/system/ui-config');
    expect(uiConfigBeforeRes.ok()).toBe(true);
    const uiConfigBeforeBody = await uiConfigBeforeRes.json();
    const exposedBefore = (uiConfigBeforeBody.plugins || []).find(
      (p: { id: string }) => p.id === 'uk.org.optimis.test'
    );
    expect(exposedBefore).toBeUndefined();

    const dormantRpcRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/get_pupil_record', {
      data: {
        args: {
          pupil_id: seeded.pupil_id,
          staff_id: seeded.admin_staff_id,
        },
      },
    });
    expect(dormantRpcRes.status()).toBe(400);
    const dormantRpcBody = await dormantRpcRes.json();
    expect(dormantRpcBody.ok).toBe(false);
    expect(dormantRpcBody.error_code).toContain('PLUGIN_DORMANT_QUARANTINE');

    // 6. Verify RBAC on activate: Non-Admin teacher is rejected with 403
    const unauthActivateRes = await request.post('/api/admin/plugins/uk.org.optimis.test/activate', {
      data: {
        staff_id: seeded.teacher_staff_id,
      },
    });
    expect(unauthActivateRes.status()).toBe(403);
    const unauthActivateBody = await unauthActivateRes.json();
    expect(unauthActivateBody.error_code).toContain('UNAUTHORIZED_ADMIN_PLUGIN_SCOPE');

    // 7. Admin activates uk.org.optimis.test -> transitions to ACTIVE and starts Event Bus workers
    const activateRes = await request.post('/api/admin/plugins/uk.org.optimis.test/activate', {
      data: {
        staff_id: seeded.admin_staff_id,
      },
    });
    expect(activateRes.status()).toBe(200);
    const activateBody = await activateRes.json();
    expect(activateBody.ok).toBe(true);
    expect(activateBody.plugin_id).toBe('uk.org.optimis.test');
    expect(activateBody.state).toBe('ACTIVE');

    // 8. Load Host UI Shell and verify plugin-iframe-uk.org.optimis.test mounts and executes get_pupil_record RPC cleanly
    await page.goto(
      `/?pupil_id=${encodeURIComponent(seeded.pupil_id)}&staff_id=${encodeURIComponent(seeded.admin_staff_id)}`
    );
    await expect(page.getByTestId('host-shell')).toBeVisible();

    const iframeEl = page.getByTestId('plugin-iframe-uk.org.optimis.test');
    await expect(iframeEl).toBeVisible();
    await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    const printBtn = pluginFrame.getByTestId('print-record-btn');
    await expect(printBtn).toBeVisible();
    await printBtn.click();

    const expectedLogSubstring = `[plugin:uk.org.optimis.test] {"id":"${seeded.pupil_id}"`;
    await expect
      .poll(() => browserLogs.some((entry) => entry.includes(expectedLogSubstring)), {
        message: `Expected browser console logs to contain ${expectedLogSubstring}`,
        timeout: 5000,
      })
      .toBe(true);

    // 9. Verify RBAC on remove: Non-Admin teacher is rejected with 403
    const unauthRemoveRes = await request.delete(
      `/api/admin/plugins/uk.org.optimis.test?staff_id=${encodeURIComponent(seeded.teacher_staff_id)}`
    );
    expect(unauthRemoveRes.status()).toBe(403);
    const unauthRemoveBody = await unauthRemoveRes.json();
    expect(unauthRemoveBody.error_code).toContain('UNAUTHORIZED_ADMIN_PLUGIN_SCOPE');

    // 10. Admin removes uk.org.optimis.test -> stops workers, detaches plugin_test.db, removes from loaded plugins
    const removeRes = await request.delete(
      `/api/admin/plugins/uk.org.optimis.test?staff_id=${encodeURIComponent(seeded.admin_staff_id)}`
    );
    expect(removeRes.status()).toBe(200);
    const removeBody = await removeRes.json();
    expect(removeBody.ok).toBe(true);
    expect(removeBody.plugin_id).toBe('uk.org.optimis.test');
    expect(removeBody.removed).toBe(true);

    // 11. Verify GET /api/admin/plugins no longer includes uk.org.optimis.test and RPC calls fail with plugin not loaded
    const listAfterRemoveRes = await request.get(
      `/api/admin/plugins?staff_id=${encodeURIComponent(seeded.admin_staff_id)}`
    );
    expect(listAfterRemoveRes.status()).toBe(200);
    const listAfterRemoveBody = await listAfterRemoveRes.json();
    const afterEntry = (listAfterRemoveBody.plugins || []).find(
      (p: { id: string }) => p.id === 'uk.org.optimis.test'
    );
    expect(afterEntry).toBeUndefined();

    const rpcAfterRemoveRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/get_pupil_record', {
      data: {
        args: {
          pupil_id: seeded.pupil_id,
          staff_id: seeded.admin_staff_id,
        },
      },
    });
    expect([400, 404]).toContain(rpcAfterRemoveRes.status());
    const rpcAfterRemoveBody = await rpcAfterRemoveRes.json();
    expect(rpcAfterRemoveBody.ok).toBe(false);
    expect(rpcAfterRemoveBody.error_code).toContain('plugin not loaded');
  });
});
