import { test, expect } from '@playwright/test';

// All tests must run sequentially
test.describe.configure({ mode: 'serial' });

test.describe('MVP Plugin Sandboxed UI & Lua DAL Console Print', () => {
  let seededPupilId: string;
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string;

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

    // 3. Seed required test data explicitly during setup
    const res = await request.post('/api/test/seed-pupil', {
      data: {
        upn: 'A123456789012',
        first_name: 'Ada',
        last_name: 'Lovelace',
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

    // 3. Always clean up seeded mock data at the end of the test
    if (seededPupilId) {
      const delRes = await request.delete(`/api/test/seed-pupil/${seededPupilId}`);
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up seeded pupil ${seededPupilId}: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('getPupilRecord_seededPupilClickedInPluginUI_printsDatabaseRecordToConsole_test', async ({ page }) => {
    await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}`);

    const hostShell = page.getByTestId('host-shell');
    await expect(hostShell).toBeVisible();

    const iframeEl = page.getByTestId('plugin-iframe-uk.org.optimis.test');
    await expect(iframeEl).toBeVisible();
    await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    const printBtn = pluginFrame.getByTestId('print-record-btn');
    await expect(printBtn).toBeVisible();
    await expect(printBtn).toHaveAttribute('data-mis-rpc', 'get_pupil_record');

    await printBtn.click();

    const expectedLogSubstring = `[plugin:uk.org.optimis.test] {"id":"${seededPupilId}","upn":"A123456789012","first_name":"Ada","last_name":"Lovelace"}`;

    await expect
      .poll(
        () => browserLogs.some((entry) => entry.includes(expectedLogSubstring)),
        {
          message: `Expected browser console logs to contain ${expectedLogSubstring}`,
          timeout: 5000,
        }
      )
      .toBe(true);
  });
});
