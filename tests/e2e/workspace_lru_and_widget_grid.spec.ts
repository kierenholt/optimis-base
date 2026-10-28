import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

test.describe('Workspace Keep-Alive LRU Pool (Section 9A) & Editable NxM Mini-Iframe Widget Grid (Section 9B)', () => {
  let seededPupilId: string = '';
  let secondPupilId: string = '';
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';

  const syntheticPluginIds = [
    'uk.org.optimis.lru1',
    'uk.org.optimis.lru2',
    'uk.org.optimis.lru3',
    'uk.org.optimis.lru4',
    'uk.org.optimis.lru5',
    'uk.org.optimis.lru6',
  ];

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    seededPupilId = '';
    secondPupilId = '';
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

    // 3. Ensure dev persona switcher is enabled
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });

    // 4. Seed two isolated test pupils for RPC and deep-link verification across LRU pool and widgets
    const res1 = await request.post('/api/test/seed-pupil', {
      data: {
        upn: 'W100000000001',
        first_name: 'Grace',
        last_name: 'Hopper',
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    if (!res1.ok()) {
      throw new Error(`Failed to seed primary pupil fixture: ${res1.status()}`);
    }
    const body1 = await res1.json();
    if (!body1.pupil_id) {
      throw new Error('Seeded primary pupil response missing mandatory pupil_id');
    }
    seededPupilId = body1.pupil_id;

    const res2 = await request.post('/api/test/seed-pupil', {
      data: {
        upn: 'W100000000002',
        first_name: 'Katherine',
        last_name: 'Johnson',
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    if (!res2.ok()) {
      throw new Error(`Failed to seed secondary pupil fixture: ${res2.status()}`);
    }
    const body2 = await res2.json();
    if (!body2.pupil_id) {
      throw new Error('Seeded secondary pupil response missing mandatory pupil_id');
    }
    secondPupilId = body2.pupil_id;

    // 5. Seed synthetic active plugins so the Keep-Alive LRU Pool (N=5) can be tested across >5 plugins
    const seedLruRes = await request.post('/api/test/seed-lru-plugins', {
      data: {
        plugin_ids: syntheticPluginIds,
      },
    });
    if (!seedLruRes.ok()) {
      throw new Error(`Failed to seed synthetic LRU plugins: ${seedLruRes.status()}`);
    }
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

    // 3. Clean up synthetic LRU plugins
    await request.delete('/api/test/seed-lru-plugins');

    // 4. Clean up seeded pupil records
    if (seededPupilId) {
      const del1 = await request.delete(`/api/test/seed-pupil/${encodeURIComponent(seededPupilId)}`);
      if (!del1.ok()) {
        throw new Error(`Failed to clean up seeded pupil ${seededPupilId}: ${del1.status()}`);
      }
    }
    if (secondPupilId) {
      const del2 = await request.delete(`/api/test/seed-pupil/${encodeURIComponent(secondPupilId)}`);
      if (!del2.ok()) {
        throw new Error(`Failed to clean up second pupil ${secondPupilId}: ${del2.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('keepAliveLruPool_visitingSixPluginsAndSwitchingTabs_preservesZeroReloadStateForLastFiveAndEvictsLeastRecentlyUsed_test', async ({
    page,
  }) => {
    await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}&second_pupil_id=${encodeURIComponent(secondPupilId)}`);

    const lruPool = page.getByTestId('keep-alive-lru-pool');
    await expect(lruPool).toBeVisible();

    // Initially, default plugin uk.org.optimis.test is mounted and active (pool size = 1)
    const defaultIframe = page.getByTestId('plugin-iframe-uk.org.optimis.test');
    await expect(defaultIframe).toBeVisible();
    await expect(lruPool).toHaveAttribute('data-pool-size', '1');

    // Mutate in-memory DOM state inside uk.org.optimis.test via Promise RPC to prove zero-reload preservation later
    const defaultFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    await defaultFrame.getByTestId('sdk-promise-rpc-btn').click();
    await expect(defaultFrame.getByTestId('sdk-promise-rpc-result')).toHaveText(
      'PROMISE_OK:Grace Hopper:W100000000001'
    );

    // Visit lru1, lru2, lru3, lru4 -> Pool now holds 5 mounted iframes: [uk.org.optimis.test, lru1, lru2, lru3, lru4]
    for (const pid of ['uk.org.optimis.lru1', 'uk.org.optimis.lru2', 'uk.org.optimis.lru3', 'uk.org.optimis.lru4']) {
      await page.getByTestId(`nav-item-plugin-${pid}`).click();
      await expect(page.getByTestId(`plugin-iframe-${pid}`)).toBeVisible();
    }

    await expect(lruPool).toHaveAttribute('data-pool-size', '5');

    // Inactive iframes in the pool must remain mounted in the DOM with display: none
    await expect(defaultIframe).toBeAttached();
    await expect(defaultIframe).toBeHidden();
    const defaultDisplay = await defaultIframe.evaluate((el) => getComputedStyle(el).display);
    expect(defaultDisplay).toBe('none');

    // Mutate in-memory DOM state inside lru1 and switch back to uk.org.optimis.test to refresh uk.org.optimis.test as Most-Recently-Used (MRU)
    await page.getByTestId('nav-item-plugin-uk.org.optimis.lru1').click();
    const lru1Frame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.lru1"]');
    await lru1Frame.getByTestId('sdk-promise-rpc-btn').click();
    await expect(lru1Frame.getByTestId('sdk-promise-rpc-result')).toHaveText(
      'PROMISE_OK:Grace Hopper:W100000000001'
    );

    // Switch back to uk.org.optimis.test: verify instant zero-reload tab switch (previous RPC result text is still intact without re-clicking!)
    await page.getByTestId('nav-item-plugin-uk.org.optimis.test').click();
    await expect(defaultIframe).toBeVisible();
    await expect(defaultFrame.getByTestId('sdk-promise-rpc-result')).toHaveText(
      'PROMISE_OK:Grace Hopper:W100000000001'
    );

    // Current LRU order from oldest to newest is now: [lru2, lru3, lru4, lru1, uk.org.optimis.test]
    await expect(lruPool).toHaveAttribute(
      'data-lru-order',
      'uk.org.optimis.lru2,uk.org.optimis.lru3,uk.org.optimis.lru4,uk.org.optimis.lru1,uk.org.optimis.test'
    );

    // Visit a 6th distinct plugin (lru5): pool is full (N=5), so the Least-Recently-Used (lru2) must be evicted from the DOM!
    await page.getByTestId('nav-item-plugin-uk.org.optimis.lru5').click();
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.lru5')).toBeVisible();
    await expect(lruPool).toHaveAttribute('data-pool-size', '5');

    // Verify lru2 was evicted (detached from DOM), while uk.org.optimis.test and lru1 remain mounted with their state intact
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.lru2')).toHaveCount(0);
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.test')).toBeAttached();
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.lru1')).toBeAttached();
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.lru3')).toBeAttached();
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.lru4')).toBeAttached();

    // Switch back to lru1 and confirm its in-memory DOM state was never reloaded when lru5 mounted
    await page.getByTestId('nav-item-plugin-uk.org.optimis.lru1').click();
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.lru1')).toBeVisible();
    await expect(lru1Frame.getByTestId('sdk-promise-rpc-result')).toHaveText(
      'PROMISE_OK:Grace Hopper:W100000000001'
    );
  });

  test('dashboardWidgetGridAndEditMode_manifestWidgetsLoaded_placesResizesExecutesRpcAndDeepLinksToKeepAlivePool_test', async ({
    page,
    request,
  }) => {
    // 1. Verify GET /api/system/ui-config includes manifest widgets for uk.org.optimis.test
    const uiConfigRes = await request.get('/api/system/ui-config');
    expect(uiConfigRes.ok()).toBe(true);
    const uiConfigBody = await uiConfigRes.json();
    expect(uiConfigBody.ok).toBe(true);

    const testPlugin = uiConfigBody.plugins.find((p: { id: string }) => p.id === 'uk.org.optimis.test');
    expect(testPlugin).toBeDefined();
    expect(Array.isArray(testPlugin.widgets)).toBe(true);
    expect(testPlugin.widgets).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          id: 'attendance_pulse',
          title: 'Statutory Attendance Pulse',
          entry: 'ui/dist/widget.html',
          entry_url: '/plugins/uk.org.optimis.test/ui/widget.html',
        }),
        expect.objectContaining({
          id: 'pastoral_watch',
          title: 'Pastoral & Behaviour Watch',
          entry: 'ui/dist/widget.html',
          entry_url: '/plugins/uk.org.optimis.test/ui/widget.html',
        }),
      ])
    );

    // 2. Open Host Dashboard and inspect the Mini-Iframe Widget Grid in default View Mode
    await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}&second_pupil_id=${encodeURIComponent(secondPupilId)}`);

    const widgetGridSection = page.getByTestId('dashboard-widget-grid-section');
    await expect(widgetGridSection).toBeVisible();

    const editModeStatus = page.getByTestId('dashboard-edit-mode-status');
    await expect(editModeStatus).toHaveText('VIEW_MODE');

    // Default placed widget (attendance_pulse) should be mounted inside the NxM grid
    const widgetGrid = page.getByTestId('dashboard-widget-grid');
    await expect(widgetGrid).toBeVisible();

    const attendanceWidgetCard = page.getByTestId('placed-widget-card-attendance_pulse');
    await expect(attendanceWidgetCard).toBeVisible();
    await expect(attendanceWidgetCard).toHaveAttribute('data-col-span', '2');
    await expect(attendanceWidgetCard).toHaveAttribute('data-row-span', '1');

    const attendanceWidgetIframe = page.getByTestId('widget-iframe-attendance_pulse');
    await expect(attendanceWidgetIframe).toBeVisible();
    await expect(attendanceWidgetIframe).toHaveAttribute('sandbox', 'allow-scripts');

    // 3. Enter Dashboard Edit Mode -> Place a second widget (pastoral_watch) and resize attendance_pulse on the NxM grid
    const editModeToggleBtn = page.getByTestId('dashboard-edit-mode-toggle-btn');
    await expect(editModeToggleBtn).toBeVisible();
    await editModeToggleBtn.click();

    await expect(editModeStatus).toHaveText('EDIT_MODE');
    await expect(page.getByTestId('dashboard-widget-catalog')).toBeVisible();

    // Place pastoral_watch widget onto the grid
    const placePastoralBtn = page.getByTestId('place-widget-btn-pastoral_watch');
    await expect(placePastoralBtn).toBeVisible();
    await placePastoralBtn.click();

    const pastoralWidgetCard = page.getByTestId('placed-widget-card-pastoral_watch');
    await expect(pastoralWidgetCard).toBeVisible();
    await expect(page.getByTestId('widget-iframe-pastoral_watch')).toBeVisible();

    // Resize attendance_pulse widget to 3 columns x 2 rows
    const colSelect = page.getByTestId('resize-widget-col-select-attendance_pulse');
    const rowSelect = page.getByTestId('resize-widget-row-select-attendance_pulse');
    await expect(colSelect).toBeVisible();
    await expect(rowSelect).toBeVisible();

    await colSelect.selectOption('3');
    await rowSelect.selectOption('2');

    await expect(attendanceWidgetCard).toHaveAttribute('data-col-span', '3');
    await expect(attendanceWidgetCard).toHaveAttribute('data-row-span', '2');

    const gridStyles = await attendanceWidgetCard.evaluate((el) => {
      const s = (el as HTMLElement).style;
      return { gridColumn: s.gridColumn, gridRow: s.gridRow };
    });
    expect(gridStyles.gridColumn).toContain('span 3');
    expect(gridStyles.gridRow).toContain('span 2');

    // Exit Dashboard Edit Mode and verify layout persists
    await editModeToggleBtn.click();
    await expect(editModeStatus).toHaveText('VIEW_MODE');
    await expect(attendanceWidgetCard).toHaveAttribute('data-col-span', '3');
    await expect(attendanceWidgetCard).toHaveAttribute('data-row-span', '2');
    await expect(pastoralWidgetCard).toBeVisible();

    // 4. Verify mini-iframe widget independently receives mis:init and executes window.mis.rpc()
    const attendanceWidgetFrame = page.frameLocator('[data-testid="widget-iframe-attendance_pulse"]');
    await expect(attendanceWidgetFrame.getByTestId('widget-theme-status')).toContainText('THEME:light');
    await expect(attendanceWidgetFrame.getByTestId('widget-context-status')).toContainText(`PUPIL:${seededPupilId}`);

    await attendanceWidgetFrame.getByTestId('widget-rpc-fetch-btn').click();
    await expect(attendanceWidgetFrame.getByTestId('widget-rpc-result')).toHaveText(
      'WIDGET_RPC_OK:Grace Hopper:W100000000001'
    );

    // 5. Switch active workspace tab to lru1, then click <a data-mis-link="student"> inside the mini-iframe widget
    // Verify it wakes the designated Student Handler Plugin (uk.org.optimis.test) in the Keep-Alive LRU Pool with secondPupilId!
    await page.getByTestId('nav-item-plugin-uk.org.optimis.lru1').click();
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.lru1')).toBeVisible();
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.test')).toBeHidden();

    await attendanceWidgetFrame.getByTestId('widget-deep-link-student').click();

    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.test')).toBeVisible();
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.lru1')).toBeHidden();
    await expect(page.getByTestId('active-entity-context-pill')).toContainText(`student:${secondPupilId}`);

    const mainPluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    await expect(mainPluginFrame.getByTestId('sdk-context-status')).toContainText(`PUPIL:${secondPupilId}`);
  });
});
