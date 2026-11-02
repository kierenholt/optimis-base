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
}

test.describe('Installed Plugins Persistence & Schema Migration (Sections 6A, 7C & 9.1)', () => {
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

    // 3. Ensure dev persona switcher is enabled and clean up test plugin if installed
    const authModeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!authModeRes.ok()) {
      throw new Error(`Failed to enable dev persona switcher: ${authModeRes.status()}`);
    }
    await request.delete('/api/admin/plugins/uk.org.optimis.test');

    // 4. Seed isolated staff personas & pupil fixture
    const seedRes = await request.post('/api/test/seed-gdpr-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-20',
      },
    });
    if (!seedRes.ok()) {
      throw new Error(`Failed to seed fixtures: ${seedRes.status()} ${await seedRes.text()}`);
    }
    seeded = (await seedRes.json()) as SeededGDPRFixtures;
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

    // 3. Reset test plugin state
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete('/api/admin/plugins/uk.org.optimis.test');

    // 4. Clean up seeded fixtures in core.db
    if (seeded?.fixture_id) {
      await request.delete(`/api/test/seed-gdpr-fixtures/${encodeURIComponent(seeded.fixture_id)}`);
    }
  });

  test('installedPluginsPersistence_installAndLifecycleTransitions_persistsRelationalStateAndIsoTimestamps_test', async ({
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded state');
    }

    // ISO-8601 UTC regex: e.g. "2026-11-02T10:15:30Z" (must NOT be SQLite CURRENT_TIMESTAMP "2026-11-02 10:15:30")
    const iso8601UtcRegex = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/;

    // 1. Authenticate as Admin persona
    await request.post('/api/auth/persona', {
      data: { staff_id: seeded.admin_staff_id },
    });

    // 2. Install test_plugin
    const installRes = await request.post('/api/admin/plugins/install', {
      data: {
        folder: 'test_plugin',
        staff_id: seeded.admin_staff_id,
      },
    });
    expect(installRes.status()).toBe(201);
    const installBody = await installRes.json();
    expect(installBody.ok).toBe(true);
    expect(installBody.plug_id).toMatch(/^[a-z0-9]{8}$/);
    const plugId: string = installBody.plug_id;
    expect(installBody.state).toBe('UNINSTALLED_DORMANT');
    expect(installBody.installed_at).toMatch(iso8601UtcRegex);
    expect(installBody.updated_at).toMatch(iso8601UtcRegex);

    // 3. Verify dedicated relational table installed_plugins via inspection endpoint
    const recordRes = await request.get('/api/test/installed-plugins/uk.org.optimis.test');
    expect(recordRes.status()).toBe(200);
    const recordBody = await recordRes.json();
    expect(recordBody.ok).toBe(true);
    expect(recordBody.installed_plugin).toBeDefined();

    const p = recordBody.installed_plugin;
    expect(p.manifest_id).toBe('uk.org.optimis.test');
    expect(p.plug_id).toBe(plugId);
    expect(p.version).toBe('0.2.0');
    expect(p.state).toBe('UNINSTALLED_DORMANT');
    expect(p.database_alias).toBe('plugin_test');
    expect(p.folder).toBe('test_plugin');
    expect(p.installed_at).toMatch(iso8601UtcRegex);
    expect(p.updated_at).toMatch(iso8601UtcRegex);
    expect(p.installed_at).not.toContain(' '); // Not SQLite CURRENT_TIMESTAMP
    expect(p.updated_at).not.toContain(' ');

    // 4. Activate the plugin -> lifecycle state transitions to ACTIVE
    const activateRes = await request.post(`/api/admin/plugins/${encodeURIComponent(plugId)}/activate`, {
      data: { staff_id: seeded.admin_staff_id },
    });
    expect(activateRes.status()).toBe(200);
    const activateBody = await activateRes.json();
    expect(activateBody.ok).toBe(true);
    expect(activateBody.state).toBe('ACTIVE');

    // 5. Verify updated state in installed_plugins relational table
    const activeRecordRes = await request.get('/api/test/installed-plugins/uk.org.optimis.test');
    expect(activeRecordRes.status()).toBe(200);
    const activeRecord = (await activeRecordRes.json()).installed_plugin;
    expect(activeRecord.state).toBe('ACTIVE');
    expect(activeRecord.updated_at).toMatch(iso8601UtcRegex);

    // 6. Verify GET /api/admin/plugins lists relational fields and ISO timestamps
    const listRes = await request.get('/api/admin/plugins');
    expect(listRes.status()).toBe(200);
    const listBody = await listRes.json();
    const pluginEntry = (listBody.plugins || []).find((entry: { plug_id: string }) => entry.plug_id === plugId);
    expect(pluginEntry).toBeDefined();
    expect(pluginEntry.state).toBe('ACTIVE');
    expect(pluginEntry.installed_at).toMatch(iso8601UtcRegex);
    expect(pluginEntry.updated_at).toMatch(iso8601UtcRegex);

    // 7. Remove the plugin -> state transitions to UNINSTALLED_DORMANT in database
    const removeRes = await request.delete(`/api/admin/plugins/${encodeURIComponent(plugId)}`);
    expect(removeRes.status()).toBe(200);

    const postRemoveRecordRes = await request.get('/api/test/installed-plugins/uk.org.optimis.test');
    expect(postRemoveRecordRes.status()).toBe(200);
    const postRemoveRecord = (await postRemoveRecordRes.json()).installed_plugin;
    expect(postRemoveRecord.state).toBe('UNINSTALLED_DORMANT');
  });
});
