import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

test.describe('Per-Plugin Attached SQLite (plugin_test.db) & Migration Snapshots E2E', () => {
  let seededPupilId: string = '';
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    seededPupilId = '';
    logCursor = '';

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

    // 3. Seed required test pupil explicitly during setup
    const res = await request.post('/api/test/seed-pupil', {
      data: {
        upn: 'P987654321001',
        first_name: 'Grace',
        last_name: 'Hopper',
        academic_year_id: '2026-2027',
        effective_from: '2026-09-01',
      },
    });
    if (!res.ok()) {
      throw new Error(`Failed to seed pupil fixture: ${res.status()}`);
    }
    const body = await res.json();
    if (!body.pupil_id) {
      throw new Error('Seeded pupil response missing mandatory pupil_id');
    }
    seededPupilId = body.pupil_id;
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Purge any plugin_test.db rows created for seededPupilId during the test
    if (seededPupilId) {
      await request.post('/api/plugins/uk.org.optimis.test/rpc/purge_pupil_data', {
        data: {
          args: { pupil_id: seededPupilId },
        },
      });
    }

    // 2. Fetch Backend Service Logs emitted during this test
    if (logCursor) {
      const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
      if (logsRes.ok()) {
        const serviceLogs: string[] = (await logsRes.json()).entries;
        if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
          console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
        }
      }
    }

    // 3. Dump tapped Browser Console logs on failure or error
    if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
      console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
      console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
    }

    // 4. Clean up seeded core pupil record at the end of the test
    if (seededPupilId) {
      const delRes = await request.delete(`/api/test/seed-pupil/${seededPupilId}`);
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up seeded pupil ${seededPupilId}: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('saveAndQueryPluginNote_seededPupilInSandboxedPlugin_persistsToAttachedPluginDBAndEnforcesPragmas_test', async ({
    page,
    request,
  }) => {
    await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}`);

    const hostShell = page.getByTestId('host-shell');
    await expect(hostShell).toBeVisible();

    const iframeEl = page.getByTestId('plugin-iframe-uk.org.optimis.test');
    await expect(iframeEl).toBeVisible();
    await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');

    const saveNoteBtn = pluginFrame.getByTestId('save-plugin-note-btn');
    await expect(saveNoteBtn).toBeVisible();
    await expect(saveNoteBtn).toHaveAttribute('data-mis-rpc', 'save_pupil_note');

    await saveNoteBtn.click();

    const noteStatus = pluginFrame.getByTestId('plugin-note-status');
    await expect(noteStatus).toHaveText('Persisted 1 note(s) in plugin_test.db: Excellent progress in algebra');

    const loadNotesBtn = pluginFrame.getByTestId('load-plugin-notes-btn');
    await expect(loadNotesBtn).toBeVisible();
    await expect(loadNotesBtn).toHaveAttribute('data-mis-rpc', 'list_pupil_notes');

    await loadNotesBtn.click();

    const expectedLogSubstring = `[plugin:uk.org.optimis.test:plugin_db] {"pupil_id":"${seededPupilId}","count":1,"first_note":"Excellent progress in algebra","db_alias":"plugin_test","max_page_count":16384}`;

    await expect
      .poll(
        () => browserLogs.some((entry) => entry.includes(expectedLogSubstring)),
        {
          message: `Expected browser console logs to contain ${expectedLogSubstring}`,
          timeout: 5000,
        }
      )
      .toBe(true);

    // Verify GDPR export_pupil_data and purge_pupil_data against the attached plugin_test.db
    const exportRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/export_pupil_data', {
      data: {
        args: { pupil_id: seededPupilId },
      },
    });
    expect(exportRes.ok()).toBe(true);
    const exportBody = await exportRes.json();
    expect(exportBody.ok).toBe(true);
    expect(Array.isArray(exportBody.data)).toBe(true);
    expect(exportBody.data.length).toBe(1);
    expect(exportBody.data[0].title).toBe('Plugin Pupil Notes');
    expect(exportBody.data[0].records.length).toBe(1);
    expect(exportBody.data[0].records[0].note).toBe('Excellent progress in algebra');

    const purgeRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/purge_pupil_data', {
      data: {
        args: { pupil_id: seededPupilId },
      },
    });
    expect(purgeRes.ok()).toBe(true);
    const purgeBody = await purgeRes.json();
    expect(purgeBody.ok).toBe(true);

    const listAfterPurgeRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/list_pupil_notes', {
      data: {
        args: { pupil_id: seededPupilId },
      },
    });
    expect(listAfterPurgeRes.ok()).toBe(true);
    const listAfterPurgeBody = await listAfterPurgeRes.json();
    expect(listAfterPurgeBody.ok).toBe(true);
    expect(listAfterPurgeBody.data.count).toBe(0);
  });
});
