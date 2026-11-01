import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

test.describe('Workspace Keep-Alive LRU Pool (Section 9A) & Editable NxM Mini-Iframe Widget Grid (Section 9B)', () => {
  let seededPupilId: string = '';
  let secondPupilId: string = '';
  let testPlugId: string = '';
  let lruPlugIds: string[] = [];
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
    testPlugId = '';
    lruPlugIds = [];
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

    // 3. Ensure dev persona switcher is enabled and install + activate test_plugin
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    const installRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'test_plugin' },
    });
    if (!installRes.ok()) {
      throw new Error(`Failed to install test_plugin: ${installRes.status()}`);
    }
    const installBody = await installRes.json();
    testPlugId = installBody.plug_id;
    const activateRes = await request.post(`/api/admin/plugins/${encodeURIComponent(testPlugId)}/activate`, {
      data: {},
    });
    if (!activateRes.ok()) {
      throw new Error(`Failed to activate test_plugin (${testPlugId}): ${activateRes.status()}`);
    }

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
    const seedLruBody = await seedLruRes.json();
    lruPlugIds = seedLruBody.seeded_plugin_ids;
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

    // 3. Clean up synthetic LRU plugins, dashboard layouts, and remove runtime-installed test_plugin
    await request.delete('/api/test/dashboard-layouts');
    await request.delete('/api/test/seed-lru-plugins');
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (testPlugId) {
      await request.delete(`/api/admin/plugins/${encodeURIComponent(testPlugId)}`);
    }

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
    await page.goto(`/p/${testPlugId}?pupil_id=${encodeURIComponent(seededPupilId)}&second_pupil_id=${encodeURIComponent(secondPupilId)}`);

    const lruPool = page.getByTestId('keep-alive-lru-pool');
    await expect(lruPool).toBeVisible();

    const [lru1, lru2, lru3, lru4, lru5] = lruPlugIds;

    // Initially, default plugin testPlugId is mounted and active (pool size = 1)
    const defaultIframe = page.getByTestId(`plugin-iframe-${testPlugId}`);
    await expect(defaultIframe).toBeVisible();
    await expect(lruPool).toHaveAttribute('data-pool-size', '1');

    // Mutate in-memory DOM state inside testPlugId via Promise RPC to prove zero-reload preservation later
    const defaultFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
    await expect(defaultFrame.getByTestId('sdk-context-status')).toContainText(`PUPIL:${seededPupilId}`);
    await defaultFrame.getByTestId('sdk-promise-rpc-btn').click();
    await expect(defaultFrame.getByTestId('sdk-promise-rpc-result')).toHaveText(
      'PROMISE_OK:Grace Hopper:W100000000001'
    );

    const selectPluginViaRailSearch = async (pid: string) => {
      await page.getByTestId('nav-rail-search').click();
      await page.getByTestId(`host-search-result-view-${pid}-home`).click();
    };

    // Visit lru1, lru2, lru3, lru4 -> Pool now holds 5 mounted iframes: [testPlugId, lru1, lru2, lru3, lru4]
    for (const pid of [lru1, lru2, lru3, lru4]) {
      await selectPluginViaRailSearch(pid);
      await expect(page.getByTestId(`plugin-iframe-${pid}`)).toBeVisible();
    }

    await expect(lruPool).toHaveAttribute('data-pool-size', '5');

    // Inactive iframes in the pool must remain mounted in the DOM with display: none
    await expect(defaultIframe).toBeAttached();
    await expect(defaultIframe).toBeHidden();
    const defaultDisplay = await defaultIframe.evaluate((el) => getComputedStyle(el).display);
    expect(defaultDisplay).toBe('none');

    // Mutate in-memory DOM state inside lru1 and switch back to testPlugId to refresh testPlugId as Most-Recently-Used (MRU)
    await selectPluginViaRailSearch(lru1);
    const lru1Frame = page.frameLocator(`[data-testid="plugin-iframe-${lru1}"]`);
    await lru1Frame.getByTestId('sdk-promise-rpc-btn').click();
    await expect(lru1Frame.getByTestId('sdk-promise-rpc-result')).toHaveText(
      'PROMISE_OK:Grace Hopper:W100000000001'
    );

    // Switch back to testPlugId: verify instant zero-reload tab switch (previous RPC result text is still intact without re-clicking!)
    await selectPluginViaRailSearch(testPlugId);
    await expect(defaultIframe).toBeVisible();
    await expect(defaultFrame.getByTestId('sdk-promise-rpc-result')).toHaveText(
      'PROMISE_OK:Grace Hopper:W100000000001'
    );

    // Current LRU order from oldest to newest is now: [lru2, lru3, lru4, lru1, testPlugId]
    await expect(lruPool).toHaveAttribute(
      'data-lru-order',
      `${lru2},${lru3},${lru4},${lru1},${testPlugId}`
    );

    // Visit a 6th distinct plugin (lru5): pool is full (N=5), so the Least-Recently-Used (lru2) must be evicted from the DOM!
    await selectPluginViaRailSearch(lru5);
    await expect(page.getByTestId(`plugin-iframe-${lru5}`)).toBeVisible();
    await expect(lruPool).toHaveAttribute('data-pool-size', '5');

    // Verify lru2 was evicted (detached from DOM), while testPlugId and lru1 remain mounted with their state intact
    await expect(page.getByTestId(`plugin-iframe-${lru2}`)).toHaveCount(0);
    await expect(page.getByTestId(`plugin-iframe-${testPlugId}`)).toBeAttached();
    await expect(page.getByTestId(`plugin-iframe-${lru1}`)).toBeAttached();
    await expect(page.getByTestId(`plugin-iframe-${lru3}`)).toBeAttached();
    await expect(page.getByTestId(`plugin-iframe-${lru4}`)).toBeAttached();

    // Switch back to lru1 and confirm its in-memory DOM state was never reloaded when lru5 mounted
    await selectPluginViaRailSearch(lru1);
    await expect(page.getByTestId(`plugin-iframe-${lru1}`)).toBeVisible();
    await expect(lru1Frame.getByTestId('sdk-promise-rpc-result')).toHaveText(
      'PROMISE_OK:Grace Hopper:W100000000001'
    );
  });

  test('dashboardWidgetGridAndEditMode_manifestWidgetsLoaded_placesPresetSizedWidgetsAutoFlowingExecutesRpcAndDeepLinksToKeepAlivePool_test', async ({
    page,
    request,
  }) => {
    const [lru1] = lruPlugIds;

    // 1. Verify GET /api/system/ui-config includes manifest widgets with strict preset width (1|2) and height (1|2)
    const uiConfigRes = await request.get('/api/system/ui-config');
    expect(uiConfigRes.ok()).toBe(true);
    const uiConfigBody = await uiConfigRes.json();
    expect(uiConfigBody.ok).toBe(true);

    const testPlugin = uiConfigBody.plugins.find((p: { id: string }) => p.id === testPlugId);
    expect(testPlugin).toBeDefined();
    expect(Array.isArray(testPlugin.widgets)).toBe(true);
    expect(testPlugin.widgets).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          id: 'attendance_pulse',
          title: 'Statutory Attendance Pulse',
          entry: 'ui/dist/widget.html',
          entry_url: `/p/${testPlugId}/ui/widget.html`,
          width: 2,
          height: 1,
        }),
        expect.objectContaining({
          id: 'pastoral_watch',
          title: 'Pastoral & Behaviour Watch',
          entry: 'ui/dist/widget.html',
          entry_url: `/p/${testPlugId}/ui/widget.html`,
          width: 1,
          height: 1,
        }),
      ])
    );

    // 2. Open Host Dashboard and inspect the Mini-Iframe Widget Grid in default View Mode
    await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}&second_pupil_id=${encodeURIComponent(secondPupilId)}`);

    const widgetGridSection = page.getByTestId('dashboard-widget-grid-section');
    await expect(widgetGridSection).toBeVisible();

    await expect(page.getByTestId('dashboard-layout-source')).toHaveCount(0);
    await expect(page.getByTestId('dashboard-edit-mode-status')).toHaveCount(0);
    await expect(page.getByTestId('admin-role-dashboard-controls')).toHaveCount(0);

    // Default placed widget (attendance_pulse) should be mounted inside the auto-flowing fixed-width column grid (default 4 columns x 300px, 180px rows)
    const widgetGrid = page.getByTestId('dashboard-widget-grid');
    await expect(widgetGrid).toBeVisible();
    await expect(widgetGrid).toHaveAttribute('data-column-count', '4');
    const gridStyles = await widgetGrid.evaluate((el) => {
      const s = (el as HTMLElement).style;
      return {
        gridAutoFlow: s.gridAutoFlow,
        gridTemplateColumns: s.gridTemplateColumns,
        gridAutoRows: s.gridAutoRows,
      };
    });
    expect(gridStyles.gridAutoFlow).toBe('row');
    expect(gridStyles.gridTemplateColumns).toBe('repeat(4, 300px)');
    expect(gridStyles.gridAutoRows).toBe('180px');

    const attendanceWidgetCard = page.getByTestId('placed-widget-card-attendance_pulse');
    await expect(attendanceWidgetCard).toBeVisible();
    await expect(attendanceWidgetCard).toHaveAttribute('data-width', '2');
    await expect(attendanceWidgetCard).toHaveAttribute('data-height', '1');
    await expect(attendanceWidgetCard).toHaveAttribute('data-col-span', '2');
    await expect(attendanceWidgetCard).toHaveAttribute('data-row-span', '1');
    await expect(attendanceWidgetCard).toHaveAttribute('data-grid-col', '1');
    await expect(attendanceWidgetCard).toHaveAttribute('data-grid-row', '1');

    const attendanceWidgetIframe = page.getByTestId('widget-iframe-attendance_pulse');
    await expect(attendanceWidgetIframe).toBeVisible();
    await expect(attendanceWidgetIframe).toHaveAttribute('sandbox', 'allow-scripts');

    // Outside Edit Mode, no dotted empty cells or greyed-out widget overlays exist
    await expect(page.locator('[data-testid^="dashboard-empty-grid-cell-"]')).toHaveCount(0);
    await expect(page.getByTestId('widget-edit-overlay-attendance_pulse')).toHaveCount(0);

    // 3. Enter Dashboard Edit Mode -> Verify 2D dotted grid cells, fixed column add/remove controls, greyed-out non-functional widget 4-way overlay, and Add Widget modal -> live ghost preview -> click-to-confirm
    const editModeToggleBtn = page.getByTestId('dashboard-edit-mode-toggle-btn');
    await expect(editModeToggleBtn).toBeVisible();
    await editModeToggleBtn.click();

    await expect(editModeToggleBtn).toHaveText('Finish Editing Grid');
    // Old top Available Manifest Widgets bar, Admin Role Controls, and hover-only gap buttons must not exist
    await expect(page.getByTestId('dashboard-widget-catalog')).toHaveCount(0);
    await expect(page.getByTestId('admin-role-dashboard-controls')).toHaveCount(0);
    await expect(page.locator('[data-testid^="dashboard-gap-add-btn-"]')).toHaveCount(0);

    // Verify always-visible '+ Add Widget' button and Add/Remove Column controls in Edit Mode
    const addWidgetBtn = page.getByTestId('dashboard-add-widget-btn');
    const colCountBadge = page.getByTestId('dashboard-column-count-badge');
    const addColBtn = page.getByTestId('dashboard-add-column-btn');
    const removeColBtn = page.getByTestId('dashboard-remove-column-btn');
    await expect(addWidgetBtn).toBeVisible();
    await expect(colCountBadge).toHaveAttribute('data-column-count', '4');
    await expect(addColBtn).toBeVisible();
    await expect(removeColBtn).toBeVisible();
    await expect(removeColBtn).toBeEnabled();

    await addColBtn.click();
    await expect(colCountBadge).toHaveAttribute('data-column-count', '5');
    await expect(widgetGrid).toHaveAttribute('data-column-count', '5');
    expect(await widgetGrid.evaluate((el) => (el as HTMLElement).style.gridTemplateColumns)).toBe('repeat(5, 300px)');

    await removeColBtn.click();
    await expect(colCountBadge).toHaveAttribute('data-column-count', '4');
    await expect(widgetGrid).toHaveAttribute('data-column-count', '4');
    expect(await widgetGrid.evaluate((el) => (el as HTMLElement).style.gridTemplateColumns)).toBe('repeat(4, 300px)');

    // Verify placed widget is greyed out and non-functional (pointer-events: none on iframe) with 4 directional buttons (Up, Down, Left, Right) + Delete on top
    await expect(attendanceWidgetCard).toHaveAttribute('data-edit-preview', 'true');
    const iframePointerEvents = await attendanceWidgetIframe.evaluate((el) => (el as HTMLElement).style.pointerEvents);
    expect(iframePointerEvents).toBe('none');

    const attendanceOverlay = page.getByTestId('widget-edit-overlay-attendance_pulse');
    await expect(attendanceOverlay).toBeVisible();
    const moveUpAttBtn = page.getByTestId('move-widget-up-btn-attendance_pulse');
    const moveDownAttBtn = page.getByTestId('move-widget-down-btn-attendance_pulse');
    const moveLeftAttBtn = page.getByTestId('move-widget-left-btn-attendance_pulse');
    const moveRightAttBtn = page.getByTestId('move-widget-right-btn-attendance_pulse');
    await expect(moveUpAttBtn).toBeVisible();
    await expect(moveDownAttBtn).toBeVisible();
    await expect(moveLeftAttBtn).toBeVisible();
    await expect(moveRightAttBtn).toBeVisible();
    await expect(page.getByTestId('remove-widget-btn-attendance_pulse')).toBeVisible();

    // At (col=1, row=1), Up and Left are out-of-bounds (disabled), while Right and Down are empty cells (enabled)
    await expect(moveUpAttBtn).toBeDisabled();
    await expect(moveLeftAttBtn).toBeDisabled();
    await expect(moveRightAttBtn).toBeEnabled();
    await expect(moveDownAttBtn).toBeEnabled();

    // Verify 2D dotted empty cells exist at unoccupied (col, row) coordinates
    const emptyCell3_1 = page.getByTestId('dashboard-empty-grid-cell-3-1');
    const emptyCell4_1 = page.getByTestId('dashboard-empty-grid-cell-4-1');
    const emptyCell4_2 = page.getByTestId('dashboard-empty-grid-cell-4-2');
    await expect(emptyCell3_1).toBeVisible();
    await expect(emptyCell3_1).toHaveAttribute('data-dotted-cell', 'true');
    await expect(emptyCell3_1).toHaveAttribute('data-grid-col', '3');
    await expect(emptyCell3_1).toHaveAttribute('data-grid-row', '1');
    await expect(emptyCell4_1).toBeVisible();
    await expect(emptyCell4_2).toBeVisible();

    // Click '+ Add Widget' -> opens Widget Picker Modal -> choose pastoral_watch (1x1) -> live ghost preview appears at first free slot (3, 1) -> hover (4, 2) -> click ghost preview to confirm position!
    await addWidgetBtn.click();
    const pickerModal = page.getByTestId('dashboard-widget-picker-modal');
    await expect(pickerModal).toBeVisible();

    const placePastoralBtn = pickerModal.getByTestId('place-widget-btn-pastoral_watch');
    await expect(placePastoralBtn).toBeVisible();
    await placePastoralBtn.click();

    await expect(pickerModal).toHaveCount(0);

    // Ghost preview initializes immediately at first available empty slot (col=3, row=1)
    const placementPreview = page.getByTestId('dashboard-widget-placement-preview');
    await expect(placementPreview).toBeVisible();
    await expect(placementPreview).toHaveAttribute('data-preview-col', '3');
    await expect(placementPreview).toHaveAttribute('data-preview-row', '1');

    // Hover over (col=4, row=2) -> ghost preview moves to (4, 2)
    await emptyCell4_2.hover();
    await expect(placementPreview).toHaveAttribute('data-preview-col', '4');
    await expect(placementPreview).toHaveAttribute('data-preview-row', '2');

    // Click the ghost preview to confirm placement at (col=4, row=2)
    await placementPreview.click();
    await expect(placementPreview).toHaveCount(0);

    const pastoralWidgetCard = page.getByTestId('placed-widget-card-pastoral_watch');
    await expect(pastoralWidgetCard).toBeVisible();
    await expect(pastoralWidgetCard).toHaveAttribute('data-width', '1');
    await expect(pastoralWidgetCard).toHaveAttribute('data-height', '1');
    await expect(pastoralWidgetCard).toHaveAttribute('data-col-span', '1');
    await expect(pastoralWidgetCard).toHaveAttribute('data-row-span', '1');
    await expect(pastoralWidgetCard).toHaveAttribute('data-grid-col', '4');
    await expect(pastoralWidgetCard).toHaveAttribute('data-grid-row', '2');
    await expect(page.getByTestId('widget-iframe-pastoral_watch')).toBeVisible();

    // Because pastoral_watch is now at column 4 (the rightmost column of the 4-column grid), '- Remove Column' must be disabled!
    await expect(removeColBtn).toBeDisabled();

    // Widgets cannot be arbitrarily resized by users; verify freeform resize selects are absent and explicit 2D CSS Grid coordinates are applied
    await expect(page.getByTestId('resize-widget-col-select-attendance_pulse')).toHaveCount(0);
    await expect(page.getByTestId('resize-widget-row-select-attendance_pulse')).toHaveCount(0);

    const attendanceStyles = await attendanceWidgetCard.evaluate((el) => {
      const s = (el as HTMLElement).style;
      return { gridColumn: s.gridColumn, gridRow: s.gridRow };
    });
    expect(attendanceStyles.gridColumn).toBe('1 / span 2');
    expect(attendanceStyles.gridRow).toBe('1 / span 1');

    const pastoralStyles = await pastoralWidgetCard.evaluate((el) => {
      const s = (el as HTMLElement).style;
      return { gridColumn: s.gridColumn, gridRow: s.gridRow };
    });
    expect(pastoralStyles.gridColumn).toBe('4 / span 1');
    expect(pastoralStyles.gridRow).toBe('2 / span 1');

    // Exit Dashboard Edit Mode and verify layout persists at explicit 2D coordinates and iframes become interactive again
    await editModeToggleBtn.click();
    await expect(editModeToggleBtn).toHaveText('Edit Dashboard Grid');
    await expect(page.locator('[data-testid^="dashboard-empty-grid-cell-"]')).toHaveCount(0);
    await expect(page.getByTestId('widget-edit-overlay-attendance_pulse')).toHaveCount(0);
    expect(await attendanceWidgetIframe.evaluate((el) => (el as HTMLElement).style.pointerEvents)).not.toBe('none');
    await expect(attendanceWidgetCard).toHaveAttribute('data-grid-col', '1');
    await expect(attendanceWidgetCard).toHaveAttribute('data-grid-row', '1');
    await expect(pastoralWidgetCard).toHaveAttribute('data-grid-col', '4');
    await expect(pastoralWidgetCard).toHaveAttribute('data-grid-row', '2');
    await expect(pastoralWidgetCard).toBeVisible();

    // 4. Verify mini-iframe widget independently receives mis:init and executes window.mis.rpc()
    const attendanceWidgetFrame = page.frameLocator('[data-testid="widget-iframe-attendance_pulse"]');
    await expect(attendanceWidgetFrame.getByTestId('widget-theme-status')).toContainText('THEME:light');
    await expect(attendanceWidgetFrame.getByTestId('widget-context-status')).toContainText(`PUPIL:${seededPupilId}`);

    await attendanceWidgetFrame.getByTestId('widget-rpc-fetch-btn').click();
    await expect(attendanceWidgetFrame.getByTestId('widget-rpc-result')).toHaveText(
      'WIDGET_RPC_OK:Grace Hopper:W100000000001'
    );

    // 5. Switch active workspace tab to lru1, then return to Home and click <a data-mis-link="student"> inside the mini-iframe widget
    // Verify it wakes the designated Student Handler Plugin (testPlugId) in the Keep-Alive LRU Pool with secondPupilId!
    await page.getByTestId('nav-rail-search').click();
    await page.getByTestId(`host-search-result-view-${lru1}-home`).click();
    await expect(page.getByTestId(`plugin-iframe-${lru1}`).first()).toBeVisible();
    await expect(page.getByTestId(`plugin-iframe-${testPlugId}`)).toBeHidden();

    await page.getByTestId('nav-rail-home').click();
    await attendanceWidgetFrame.getByTestId('widget-deep-link-student').click();

    await expect(page.getByTestId(`plugin-iframe-${testPlugId}`)).toBeVisible();
    await expect(page.getByTestId(`plugin-iframe-${lru1}`)).toBeHidden();

    const mainPluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
    await expect(mainPluginFrame.getByTestId('sdk-context-status')).toContainText(`PUPIL:${secondPupilId}`);
  });

  test('dashboardPersistenceCoreDb_ordered1dArrayColumnCountMoveLeftRightAndRevokedWidgetInlineError_persistsPersonalLayoutOnlyAcrossReloadsAndPersonas_test', async ({
    page,
    request,
  }) => {
    // 1. Seed persona fixtures (TEACHER, HOD, DSL, ADMIN) for personal 2D layout & column_count persistence testing
    const personaRes = await request.post('/api/test/seed-personas', {
      data: {
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    if (!personaRes.ok()) {
      throw new Error(`Failed to seed persona fixtures: ${personaRes.status()}`);
    }
    const personaFixture = await personaRes.json();
    const teacherStaffId: string = personaFixture.teacher_staff_id;
    const adminStaffId: string = personaFixture.admin_staff_id;

    try {
      // 2. Open dashboard as Teacher, enter Edit Mode, add a 5th column, place pastoral_watch via modal + ghost preview at (4, 1), and move it using 4-way empty-cell buttons
      const switchTeacher = await request.post('/api/auth/persona', {
        data: { staff_id: teacherStaffId },
      });
      expect(switchTeacher.ok()).toBe(true);
      await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}`);

      await expect(page.getByTestId('placed-widget-card-attendance_pulse')).toBeVisible();
      await page.getByTestId('dashboard-edit-mode-toggle-btn').click();

      // Add a column (4 -> 5)
      await page.getByTestId('dashboard-add-column-btn').click();
      await expect(page.getByTestId('dashboard-widget-grid')).toHaveAttribute('data-column-count', '5');

      // Click '+ Add Widget' -> pick pastoral_watch -> hover (4, 1) -> click ghost preview to confirm
      await page.getByTestId('dashboard-add-widget-btn').click();
      await page.getByTestId('dashboard-widget-picker-modal').getByTestId('place-widget-btn-pastoral_watch').click();
      await page.getByTestId('dashboard-empty-grid-cell-4-1').hover();
      await page.getByTestId('dashboard-widget-placement-preview').click();

      const pastoralCard = page.getByTestId('placed-widget-card-pastoral_watch');
      await expect(pastoralCard).toBeVisible();
      await expect(pastoralCard).toHaveAttribute('data-grid-col', '4');
      await expect(pastoralCard).toHaveAttribute('data-grid-row', '1');

      // Move pastoral_watch LEFT from (4, 1) to (3, 1)
      await page.getByTestId('move-widget-left-btn-pastoral_watch').click();
      await expect(pastoralCard).toHaveAttribute('data-grid-col', '3');
      await expect(pastoralCard).toHaveAttribute('data-grid-row', '1');

      // Now pastoral_watch at (3, 1) is immediately adjacent to attendance_pulse at (1..2, 1):
      // Verify empty-cell-only collision guard disables Move Left on pastoral_watch and Move Right on attendance_pulse!
      await expect(page.getByTestId('move-widget-left-btn-pastoral_watch')).toBeDisabled();
      await expect(page.getByTestId('move-widget-right-btn-attendance_pulse')).toBeDisabled();

      // Move pastoral_watch DOWN from (3, 1) to (3, 2)
      await page.getByTestId('move-widget-down-btn-pastoral_watch').click();
      await expect(pastoralCard).toHaveAttribute('data-grid-col', '3');
      await expect(pastoralCard).toHaveAttribute('data-grid-row', '2');

      // Verify core.db persisted the 2D { plug_id, widget_id, col, row } coordinates AND column_count = 5
      const teacherLayoutRes = await request.get('/api/dashboard/layout');
      expect(teacherLayoutRes.ok()).toBe(true);
      const teacherLayoutBody = await teacherLayoutRes.json();
      expect(teacherLayoutBody.ok).toBe(true);
      expect(teacherLayoutBody.layout_source).toBe('STAFF');
      expect(teacherLayoutBody.column_count).toBe(5);
      expect(teacherLayoutBody.layout).toEqual([
        { plug_id: testPlugId, widget_id: 'attendance_pulse', col: 1, row: 1 },
        { plug_id: testPlugId, widget_id: 'pastoral_watch', col: 3, row: 2 },
      ]);

      // Reload page and confirm 2D coordinates and column_count = 5 are restored from core.db
      await page.reload();
      await expect(page.getByTestId('dashboard-widget-grid')).toHaveAttribute('data-column-count', '5');
      await expect(page.getByTestId('placed-widget-card-attendance_pulse')).toHaveAttribute('data-grid-col', '1');
      await expect(page.getByTestId('placed-widget-card-attendance_pulse')).toHaveAttribute('data-grid-row', '1');
      await expect(page.getByTestId('placed-widget-card-pastoral_watch')).toHaveAttribute('data-grid-col', '3');
      await expect(page.getByTestId('placed-widget-card-pastoral_watch')).toHaveAttribute('data-grid-row', '2');

      // Teacher removes pastoral_watch in Edit Mode so their personal layout has only [attendance_pulse]
      await page.getByTestId('dashboard-edit-mode-toggle-btn').click();
      await page.getByTestId('remove-widget-btn-pastoral_watch').click();
      await expect(page.getByTestId('placed-widget-card-pastoral_watch')).toHaveCount(0);

      // 3. Switch to Admin persona -> Verify Admin's "Edit Dashboard Grid" button ONLY edits Admin's personal dashboard (zero admin-role-dashboard-controls on Home view)
      const switchAdmin = await request.post('/api/auth/persona', {
        data: { staff_id: adminStaffId },
      });
      expect(switchAdmin.ok()).toBe(true);
      await page.goto('/');
      await expect(page.getByTestId('admin-role-dashboard-controls')).toHaveCount(0);
      await page.getByTestId('dashboard-edit-mode-toggle-btn').click();
      await expect(page.getByTestId('admin-role-dashboard-controls')).toHaveCount(0);

      // Backend role-default append API remains functional for the dedicated Admin Dashboard Editor Plugin
      const appendRoleRes = await request.post('/api/admin/dashboard/role-default/append', {
        data: {
          staff_id: adminStaffId,
          role_id: 'TEACHER',
          plug_id: testPlugId,
          widget_id: 'pastoral_watch',
        },
      });
      expect(appendRoleRes.ok()).toBe(true);

      // Switch back to Teacher persona -> Confirm pastoral_watch was appended and auto-assigned a free 2D coordinate (col=3, row=1)!
      const switchTeacherAgain = await request.post('/api/auth/persona', {
        data: { staff_id: teacherStaffId },
      });
      expect(switchTeacherAgain.ok()).toBe(true);
      await page.goto('/');
      await expect(page.getByTestId('placed-widget-card-attendance_pulse')).toHaveAttribute('data-grid-col', '1');
      await expect(page.getByTestId('placed-widget-card-attendance_pulse')).toHaveAttribute('data-grid-row', '1');
      await expect(page.getByTestId('placed-widget-card-pastoral_watch')).toHaveAttribute('data-grid-col', '3');
      await expect(page.getByTestId('placed-widget-card-pastoral_watch')).toHaveAttribute('data-grid-row', '1');

      // 4. Persist a widget belonging to a revoked/unpermitted plugin in Teacher's 2D layout and verify inline permission error card renders in its 2D slot
      const saveRevokedRes = await request.put('/api/dashboard/layout', {
        data: {
          staff_id: teacherStaffId,
          layout: [
            { plug_id: testPlugId, widget_id: 'attendance_pulse', col: 1, row: 1 },
            { plug_id: 'revoked99', widget_id: 'restricted_safeguarding_widget', col: 3, row: 1 },
          ],
        },
      });
      expect(saveRevokedRes.ok()).toBe(true);

      await page.reload();
      await expect(page.getByTestId('placed-widget-card-attendance_pulse')).toBeVisible();
      const errorCard = page.getByTestId('widget-permission-error-card-restricted_safeguarding_widget');
      await expect(errorCard).toBeVisible();
      await expect(errorCard).toHaveAttribute('data-plug-id', 'revoked99');
    } finally {
      await request.delete(
        `/api/test/seed-personas/${encodeURIComponent(personaFixture.fixture_id)}`
      );
    }
  });

  test('widgetIdCollisionAcrossPlugins_twoPluginsSharingSameWidgetId_placesReordersLeftRightRemovesAndBroadcastsInitByCompositeKey_test', async ({
    page,
    request,
  }) => {
    // 1. Seed synthetic plugin lru1 WITH cloned widgets so testPlugId and lru1 share generic widget IDs ('attendance_pulse' & 'pastoral_watch')
    const seedWithWidgetsRes = await request.post('/api/test/seed-lru-plugins', {
      data: {
        plugin_ids: ['uk.org.optimis.lru1'],
        include_widgets: true,
      },
    });
    expect(seedWithWidgetsRes.ok()).toBe(true);
    const seedWithWidgetsBody = await seedWithWidgetsRes.json();
    const [lru1] = seedWithWidgetsBody.seeded_plugin_ids as string[];
    expect(lru1).toBeTruthy();

    await page.goto(
      `/?pupil_id=${encodeURIComponent(seededPupilId)}&second_pupil_id=${encodeURIComponent(secondPupilId)}`
    );

    // Initially, only testPlugId's attendance_pulse is placed on the dashboard at (1, 1)
    const testAttendanceCard = page.locator(
      `[data-testid="placed-widget-card-attendance_pulse"][data-plug-id="${testPlugId}"]`
    );
    const lru1AttendanceCard = page.locator(
      `[data-testid="placed-widget-card-attendance_pulse"][data-plug-id="${lru1}"]`
    );
    await expect(testAttendanceCard).toBeVisible();
    await expect(lru1AttendanceCard).toHaveCount(0);

    // 2. Search in Landing Page Top-Half Search for 'Attendance Pulse':
    // testPlugId's widget is already placed (disabled), while lru1's widget with the same ID must be enabled and placeable
    const landingInput = page.getByTestId('landing-search-input');
    await landingInput.fill('Attendance Pulse');

    const landingAddTestBtn = page.locator(
      `[data-testid="landing-search-add-widget-btn-attendance_pulse"][data-plug-id="${testPlugId}"]`
    );
    const landingAddLru1Btn = page.locator(
      `[data-testid="landing-search-add-widget-btn-attendance_pulse"][data-plug-id="${lru1}"]`
    );
    await expect(landingAddTestBtn).toBeDisabled();
    await expect(landingAddLru1Btn).toBeEnabled();

    // Place lru1's attendance_pulse via Landing Search -> auto-places at (3, 1)
    await landingAddLru1Btn.click();
    await expect(lru1AttendanceCard).toBeVisible();
    await expect(lru1AttendanceCard).toHaveAttribute('data-grid-col', '3');
    await expect(lru1AttendanceCard).toHaveAttribute('data-grid-row', '1');
    await expect(landingAddLru1Btn).toBeDisabled();
    await landingInput.fill('');

    // 3. Open Global Command Palette (Ctrl+K) and search 'Pastoral':
    // Placing testPlugId's pastoral_watch must NOT disable lru1's pastoral_watch button
    await page.keyboard.press('Control+k');
    const paletteInput = page.getByTestId('command-palette-input');
    await expect(paletteInput).toBeVisible();
    await paletteInput.fill('Pastoral');

    const paletteAddTestPastoralBtn = page.locator(
      `[data-testid="command-palette-add-widget-btn-pastoral_watch"][data-plug-id="${testPlugId}"]`
    );
    const paletteAddLru1PastoralBtn = page.locator(
      `[data-testid="command-palette-add-widget-btn-pastoral_watch"][data-plug-id="${lru1}"]`
    );
    await expect(paletteAddTestPastoralBtn).toBeEnabled();
    await expect(paletteAddLru1PastoralBtn).toBeEnabled();

    await paletteAddTestPastoralBtn.click();
    await expect(paletteAddTestPastoralBtn).toBeDisabled();
    await expect(paletteAddLru1PastoralBtn).toBeEnabled();
    await page.keyboard.press('Escape');

    // 4. Enter Dashboard Edit Mode -> Verify Widget Picker Modal states and independent 2D movement for colliding widget IDs
    await page.getByTestId('dashboard-edit-mode-toggle-btn').click();
    await page.getByTestId('dashboard-add-widget-btn').click();

    const pickerModal = page.getByTestId('dashboard-widget-picker-modal');
    await expect(pickerModal).toBeVisible();

    const catalogPlaceTestPastoral = pickerModal.locator(
      `[data-testid="place-widget-btn-pastoral_watch"][data-plug-id="${testPlugId}"]`
    );
    const catalogPlaceLru1Pastoral = pickerModal.locator(
      `[data-testid="place-widget-btn-pastoral_watch"][data-plug-id="${lru1}"]`
    );
    await expect(catalogPlaceTestPastoral).toBeDisabled();
    await expect(catalogPlaceLru1Pastoral).toBeEnabled();
    await page.getByTestId('dashboard-widget-picker-close-btn').click();
    await expect(pickerModal).toHaveCount(0);

    // Move lru1:attendance_pulse DOWN from (3, 1) to (3, 2), then UP back to (3, 1)
    await expect(testAttendanceCard).toHaveAttribute('data-grid-col', '1');
    await expect(testAttendanceCard).toHaveAttribute('data-grid-row', '1');
    await expect(lru1AttendanceCard).toHaveAttribute('data-grid-col', '3');
    await expect(lru1AttendanceCard).toHaveAttribute('data-grid-row', '1');

    await page
      .locator(`[data-testid="move-widget-down-btn-attendance_pulse"][data-plug-id="${lru1}"]`)
      .click();
    await expect(lru1AttendanceCard).toHaveAttribute('data-grid-col', '3');
    await expect(lru1AttendanceCard).toHaveAttribute('data-grid-row', '2');
    await expect(testAttendanceCard).toHaveAttribute('data-grid-col', '1');
    await expect(testAttendanceCard).toHaveAttribute('data-grid-row', '1');

    await page
      .locator(`[data-testid="move-widget-up-btn-attendance_pulse"][data-plug-id="${lru1}"]`)
      .click();
    await expect(lru1AttendanceCard).toHaveAttribute('data-grid-col', '3');
    await expect(lru1AttendanceCard).toHaveAttribute('data-grid-row', '1');

    // 5. Verify both widget iframes with colliding widget ID 'attendance_pulse' are registered in widgetIframeRefs and receive mis:init broadcasts
    const testWidgetFrame = page.frameLocator(
      `[data-testid="widget-iframe-attendance_pulse"][data-plug-id="${testPlugId}"]`
    );
    const lru1WidgetFrame = page.frameLocator(
      `[data-testid="widget-iframe-attendance_pulse"][data-plug-id="${lru1}"]`
    );

    await expect(testWidgetFrame.getByTestId('widget-theme-status')).toContainText('THEME:light');
    await expect(lru1WidgetFrame.getByTestId('widget-theme-status')).toContainText('THEME:light');

    // Toggle theme to high-contrast via full-page Settings view -> return to Home and verify BOTH colliding widget iframes receive mis:init
    await page.getByTestId('nav-rail-settings').click();
    await expect(page.getByTestId('host-settings-modal')).toHaveCount(0);
    await page.getByTestId('host-settings-view').getByTestId('host-settings-theme-toggle-btn').click();
    await page.getByTestId('nav-rail-home').click();
    await expect(testWidgetFrame.getByTestId('widget-theme-status')).toContainText(
      'THEME:high-contrast'
    );
    await expect(lru1WidgetFrame.getByTestId('widget-theme-status')).toContainText(
      'THEME:high-contrast'
    );

    // 6. Remove ONLY lru1's attendance_pulse widget -> testPlugId's attendance_pulse must remain mounted and continue receiving mis:init broadcasts
    await page
      .locator(`[data-testid="remove-widget-btn-attendance_pulse"][data-plug-id="${lru1}"]`)
      .click();
    await expect(lru1AttendanceCard).toHaveCount(0);
    await expect(testAttendanceCard).toBeVisible();

    // Toggle theme back to light via full-page Settings view -> return to Home and verify remaining testPlugId:attendance_pulse iframe still receives mis:init
    await page.getByTestId('nav-rail-settings').click();
    await page.getByTestId('host-settings-view').getByTestId('host-settings-theme-toggle-btn').click();
    await page.getByTestId('nav-rail-home').click();
    await expect(testWidgetFrame.getByTestId('widget-theme-status')).toContainText('THEME:light');
  });
});

