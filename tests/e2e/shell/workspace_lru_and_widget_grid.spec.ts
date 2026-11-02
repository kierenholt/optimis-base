import { test, expect } from '@playwright/test';
import {
  setupWorkspaceEnvironment,
  cleanupWorkspaceEnvironment,
  selectPluginViaRailSearch,
  verifyManifestWidgetsInUiConfig,
  expectWidgetGridStyles,
  expectPlacedWidgetCard,
  expectWidgetPlacementStyles,
  expectWidgetCoordinates,
  placeWidgetViaPickerAndGhost,
  seedTestPersonas,
  expectDashboardLayout,
  toggleThemeViaSettings,
} from './workspace_fixtures';

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

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    page.on('console', (msg) => browserLogs.push(`[browser:${msg.type()}] ${msg.text()}`));
    page.on('pageerror', (err) => pageErrors.push(`[pageerror] ${err.stack || err.message}`));

    const cursorRes = await request.get('/api/test/logs/cursor');
    if (!cursorRes.ok()) throw new Error(`Failed to obtain service log cursor: ${cursorRes.status()}`);
    logCursor = (await cursorRes.json()).cursor;

    const env = await setupWorkspaceEnvironment(request);
    seededPupilId = env.seededPupilId;
    secondPupilId = env.secondPupilId;
    testPlugId = env.testPlugId;
    lruPlugIds = env.lruPlugIds;
  });

  test.afterEach(async ({ request }, testInfo) => {
    if (logCursor) {
      const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
      if (logsRes.ok()) {
        const serviceLogs: string[] = (await logsRes.json()).entries;
        if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
          console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
        }
      }
    }
    if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
      console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
      console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
    }

    await cleanupWorkspaceEnvironment(request, { testPlugId, seededPupilId, secondPupilId });
    expect(pageErrors).toEqual([]);
  });

  test('keepAliveLruPool_visitingSixPluginsAndSwitchingTabs_preservesZeroReloadStateForLastFiveAndEvictsLeastRecentlyUsed_test', async ({
    page,
  }) => {
    await page.goto(`/p/${testPlugId}?pupil_id=${encodeURIComponent(seededPupilId)}&second_pupil_id=${encodeURIComponent(secondPupilId)}`);

    const lruPool = page.getByTestId('keep-alive-lru-pool');
    await expect(lruPool).toBeVisible();

    const [lru1, lru2, lru3, lru4, lru5] = lruPlugIds;

    // Default plugin testPlugId is mounted and active (pool size = 1)
    const defaultIframe = page.getByTestId(`plugin-iframe-${testPlugId}`);
    await expect(defaultIframe).toBeVisible();
    await expect(lruPool).toHaveAttribute('data-pool-size', '1');

    // Mutate in-memory DOM state inside testPlugId via Promise RPC to prove zero-reload preservation later
    const defaultFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
    await expect(defaultFrame.getByTestId('sdk-context-status')).toContainText(`PUPIL:${seededPupilId}`);
    await defaultFrame.getByTestId('sdk-promise-rpc-btn').click();
    await expect(defaultFrame.getByTestId('sdk-promise-rpc-result')).toHaveText('PROMISE_OK:Grace Hopper:W100000000001');

    // Visit lru1, lru2, lru3, lru4 -> Pool now holds 5 mounted iframes: [testPlugId, lru1, lru2, lru3, lru4]
    for (const pid of [lru1, lru2, lru3, lru4]) {
      await selectPluginViaRailSearch(page, pid);
      await expect(page.getByTestId(`plugin-iframe-${pid}`)).toBeVisible();
    }
    await expect(lruPool).toHaveAttribute('data-pool-size', '5');

    // Inactive iframes in the pool remain mounted in DOM with display: none
    await expect(defaultIframe).toBeAttached();
    await expect(defaultIframe).toBeHidden();
    expect(await defaultIframe.evaluate((el) => getComputedStyle(el).display)).toBe('none');

    // Mutate state in lru1 and switch back to testPlugId to refresh testPlugId as MRU
    await selectPluginViaRailSearch(page, lru1);
    const lru1Frame = page.frameLocator(`[data-testid="plugin-iframe-${lru1}"]`);
    await lru1Frame.getByTestId('sdk-promise-rpc-btn').click();
    await expect(lru1Frame.getByTestId('sdk-promise-rpc-result')).toHaveText('PROMISE_OK:Grace Hopper:W100000000001');

    // Switch back to testPlugId: verify instant zero-reload tab switch (previous RPC result intact)
    await selectPluginViaRailSearch(page, testPlugId);
    await expect(defaultIframe).toBeVisible();
    await expect(defaultFrame.getByTestId('sdk-promise-rpc-result')).toHaveText('PROMISE_OK:Grace Hopper:W100000000001');

    // Current LRU order from oldest to newest: [lru2, lru3, lru4, lru1, testPlugId]
    await expect(lruPool).toHaveAttribute('data-lru-order', `${lru2},${lru3},${lru4},${lru1},${testPlugId}`);

    // Visit 6th distinct plugin (lru5): pool is full (N=5), so Least-Recently-Used (lru2) is evicted
    await selectPluginViaRailSearch(page, lru5);
    await expect(page.getByTestId(`plugin-iframe-${lru5}`)).toBeVisible();
    await expect(lruPool).toHaveAttribute('data-pool-size', '5');

    // Verify lru2 evicted, while testPlugId, lru1, lru3, lru4 remain mounted
    await expect(page.getByTestId(`plugin-iframe-${lru2}`)).toHaveCount(0);
    await expect(page.getByTestId(`plugin-iframe-${testPlugId}`)).toBeAttached();
    await expect(page.getByTestId(`plugin-iframe-${lru1}`)).toBeAttached();
    await expect(page.getByTestId(`plugin-iframe-${lru3}`)).toBeAttached();
    await expect(page.getByTestId(`plugin-iframe-${lru4}`)).toBeAttached();

    // Switch back to lru1 and confirm its in-memory DOM state was preserved
    await selectPluginViaRailSearch(page, lru1);
    await expect(page.getByTestId(`plugin-iframe-${lru1}`)).toBeVisible();
    await expect(lru1Frame.getByTestId('sdk-promise-rpc-result')).toHaveText('PROMISE_OK:Grace Hopper:W100000000001');
  });

  test('dashboardWidgetGridAndEditMode_manifestWidgetsLoaded_placesPresetSizedWidgetsAutoFlowingExecutesRpcAndDeepLinksToKeepAlivePool_test', async ({
    page,
    request,
  }) => {
    const [lru1] = lruPlugIds;

    // 1. Verify GET /api/system/ui-config includes manifest widgets with preset dimensions (1|2)
    await verifyManifestWidgetsInUiConfig(request, testPlugId);

    // 2. Open Host Dashboard and inspect Mini-Iframe Widget Grid in default View Mode
    await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}&second_pupil_id=${encodeURIComponent(secondPupilId)}`);

    await expect(page.getByTestId('dashboard-widget-grid-section')).toBeVisible();
    await expect(page.getByTestId('dashboard-layout-source')).toHaveCount(0);
    await expect(page.getByTestId('dashboard-edit-mode-status')).toHaveCount(0);
    await expect(page.getByTestId('admin-role-dashboard-controls')).toHaveCount(0);

    // Default placed widget (attendance_pulse) mounted in 4 cols x 300px, 180px rows
    const widgetGrid = page.getByTestId('dashboard-widget-grid');
    await expectWidgetGridStyles(widgetGrid, 4);

    const attendanceCard = page.getByTestId('placed-widget-card-attendance_pulse');
    await expectPlacedWidgetCard(attendanceCard, { width: '2', height: '1', colSpan: '2', rowSpan: '1', gridCol: '1', gridRow: '1' });

    const attendanceIframe = page.getByTestId('widget-iframe-attendance_pulse');
    await expect(attendanceIframe).toBeVisible();
    await expect(attendanceIframe).toHaveAttribute('sandbox', 'allow-scripts');
    await expect(page.locator('[data-testid^="dashboard-empty-grid-cell-"]')).toHaveCount(0);
    await expect(page.getByTestId('widget-edit-overlay-attendance_pulse')).toHaveCount(0);

    // 3. Enter Dashboard Edit Mode -> Add/Remove Column, greyed-out overlay, 2D dotted empty cells, modal + ghost preview
    const editModeToggleBtn = page.getByTestId('dashboard-edit-mode-toggle-btn');
    await expect(editModeToggleBtn).toBeVisible();
    await editModeToggleBtn.click();
    await expect(editModeToggleBtn).toHaveText('Finish Editing Grid');
    await expect(page.getByTestId('dashboard-widget-catalog')).toHaveCount(0);
    await expect(page.getByTestId('admin-role-dashboard-controls')).toHaveCount(0);
    await expect(page.locator('[data-testid^="dashboard-gap-add-btn-"]')).toHaveCount(0);

    const colCountBadge = page.getByTestId('dashboard-column-count-badge');
    const addColBtn = page.getByTestId('dashboard-add-column-btn');
    const removeColBtn = page.getByTestId('dashboard-remove-column-btn');
    await expect(page.getByTestId('dashboard-add-widget-btn')).toBeVisible();
    await expect(colCountBadge).toHaveAttribute('data-column-count', '4');

    await addColBtn.click();
    await expect(colCountBadge).toHaveAttribute('data-column-count', '5');
    await expect(widgetGrid).toHaveAttribute('data-column-count', '5');
    expect(await widgetGrid.evaluate((el) => (el as HTMLElement).style.gridTemplateColumns)).toBe('repeat(5, 300px)');

    await removeColBtn.click();
    await expect(colCountBadge).toHaveAttribute('data-column-count', '4');
    await expect(widgetGrid).toHaveAttribute('data-column-count', '4');
    expect(await widgetGrid.evaluate((el) => (el as HTMLElement).style.gridTemplateColumns)).toBe('repeat(4, 300px)');

    // Verify placed widget is greyed out and non-functional in edit mode
    await expect(attendanceCard).toHaveAttribute('data-edit-preview', 'true');
    expect(await attendanceIframe.evaluate((el) => (el as HTMLElement).style.pointerEvents)).toBe('none');

    const moveUpAttBtn = page.getByTestId('move-widget-up-btn-attendance_pulse');
    const moveDownAttBtn = page.getByTestId('move-widget-down-btn-attendance_pulse');
    const moveLeftAttBtn = page.getByTestId('move-widget-left-btn-attendance_pulse');
    const moveRightAttBtn = page.getByTestId('move-widget-right-btn-attendance_pulse');
    await expect(page.getByTestId('widget-edit-overlay-attendance_pulse')).toBeVisible();
    await expect(moveUpAttBtn).toBeDisabled();
    await expect(moveLeftAttBtn).toBeDisabled();
    await expect(moveRightAttBtn).toBeEnabled();
    await expect(moveDownAttBtn).toBeEnabled();
    await expect(page.getByTestId('remove-widget-btn-attendance_pulse')).toBeVisible();

    const emptyCell4_2 = page.getByTestId('dashboard-empty-grid-cell-4-2');
    await expect(page.getByTestId('dashboard-empty-grid-cell-3-1')).toHaveAttribute('data-dotted-cell', 'true');
    await expect(page.getByTestId('dashboard-empty-grid-cell-4-1')).toBeVisible();
    await expect(emptyCell4_2).toBeVisible();

    // Add pastoral_watch (1x1) via modal -> live ghost preview at (3, 1) -> hover (4, 2) -> click to confirm
    await placeWidgetViaPickerAndGhost(page, 'pastoral_watch', {
      initialPreviewCol: '3',
      initialPreviewRow: '1',
      hoverCell: emptyCell4_2,
      targetPreviewCol: '4',
      targetPreviewRow: '2',
    });

    const pastoralCard = page.getByTestId('placed-widget-card-pastoral_watch');
    await expectPlacedWidgetCard(pastoralCard, { width: '1', height: '1', colSpan: '1', rowSpan: '1', gridCol: '4', gridRow: '2' });
    await expect(page.getByTestId('widget-iframe-pastoral_watch')).toBeVisible();
    await expect(removeColBtn).toBeDisabled();

    // Verify freeform resize selects absent and 2D coordinates applied
    await expect(page.getByTestId('resize-widget-col-select-attendance_pulse')).toHaveCount(0);
    await expect(page.getByTestId('resize-widget-row-select-attendance_pulse')).toHaveCount(0);
    await expectWidgetPlacementStyles(attendanceCard, '1 / span 2', '1 / span 1');
    await expectWidgetPlacementStyles(pastoralCard, '4 / span 1', '2 / span 1');

    // Exit Edit Mode -> verify layout persists and iframes become interactive
    await editModeToggleBtn.click();
    await expect(editModeToggleBtn).toHaveText('Edit Dashboard Grid');
    await expect(page.locator('[data-testid^="dashboard-empty-grid-cell-"]')).toHaveCount(0);
    await expect(page.getByTestId('widget-edit-overlay-attendance_pulse')).toHaveCount(0);
    expect(await attendanceIframe.evaluate((el) => (el as HTMLElement).style.pointerEvents)).not.toBe('none');
    await expectWidgetCoordinates(attendanceCard, '1', '1');
    await expectWidgetCoordinates(pastoralCard, '4', '2');

    // 4. Verify mini-iframe widget receives mis:init and executes window.mis.rpc()
    const attendanceFrame = page.frameLocator('[data-testid="widget-iframe-attendance_pulse"]');
    await expect(attendanceFrame.getByTestId('widget-theme-status')).toContainText('THEME:light');
    await expect(attendanceFrame.getByTestId('widget-context-status')).toContainText(`PUPIL:${seededPupilId}`);
    await attendanceFrame.getByTestId('widget-rpc-fetch-btn').click();
    await expect(attendanceFrame.getByTestId('widget-rpc-result')).toHaveText('WIDGET_RPC_OK:Grace Hopper:W100000000001');

    // 5. Deep-linking: switch workspace to lru1, return to Home, click student deep-link in widget
    await selectPluginViaRailSearch(page, lru1);
    await expect(page.getByTestId(`plugin-iframe-${lru1}`).first()).toBeVisible();
    await expect(page.getByTestId(`plugin-iframe-${testPlugId}`)).toBeHidden();

    await page.getByTestId('nav-rail-home').click();
    await attendanceFrame.getByTestId('widget-deep-link-student').click();

    await expect(page.getByTestId(`plugin-iframe-${testPlugId}`)).toBeVisible();
    await expect(page.getByTestId(`plugin-iframe-${lru1}`)).toBeHidden();
    const mainPluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
    await expect(mainPluginFrame.getByTestId('sdk-context-status')).toContainText(`PUPIL:${secondPupilId}`);
  });

  test('dashboardPersistenceCoreDb_ordered1dArrayColumnCountMoveLeftRightAndRevokedWidgetInlineError_persistsPersonalLayoutOnlyAcrossReloadsAndPersonas_test', async ({
    page,
    request,
  }) => {
    // 1. Seed persona fixtures (TEACHER, HOD, DSL, ADMIN)
    const personaFixture = await seedTestPersonas(request);
    const teacherStaffId = personaFixture.teacher_staff_id;
    const adminStaffId = personaFixture.admin_staff_id;

    try {
      // 2. Open dashboard as Teacher, enter Edit Mode, add 5th column, place pastoral_watch, move with 4-way buttons
      const switchTeacher = await request.post('/api/auth/persona', { data: { staff_id: teacherStaffId } });
      expect(switchTeacher.ok()).toBe(true);
      await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}`);

      await expect(page.getByTestId('placed-widget-card-attendance_pulse')).toBeVisible();
      await page.getByTestId('dashboard-edit-mode-toggle-btn').click();

      // Add a column (4 -> 5)
      await page.getByTestId('dashboard-add-column-btn').click();
      await expect(page.getByTestId('dashboard-widget-grid')).toHaveAttribute('data-column-count', '5');

      // Place pastoral_watch at (4, 1)
      await page.getByTestId('dashboard-add-widget-btn').click();
      await page.getByTestId('dashboard-widget-picker-modal').getByTestId('place-widget-btn-pastoral_watch').click();
      await page.getByTestId('dashboard-empty-grid-cell-4-1').hover();
      await page.getByTestId('dashboard-widget-placement-preview').click();

      const pastoralCard = page.getByTestId('placed-widget-card-pastoral_watch');
      await expectWidgetCoordinates(pastoralCard, '4', '1');

      // Move LEFT from (4, 1) to (3, 1)
      await page.getByTestId('move-widget-left-btn-pastoral_watch').click();
      await expectWidgetCoordinates(pastoralCard, '3', '1');

      // Collision guard disables Move Left on pastoral and Move Right on attendance
      await expect(page.getByTestId('move-widget-left-btn-pastoral_watch')).toBeDisabled();
      await expect(page.getByTestId('move-widget-right-btn-attendance_pulse')).toBeDisabled();

      // Move DOWN from (3, 1) to (3, 2)
      await page.getByTestId('move-widget-down-btn-pastoral_watch').click();
      await expectWidgetCoordinates(pastoralCard, '3', '2');

      // Verify core.db persisted 2D coordinates AND column_count = 5
      await expectDashboardLayout(request, {
        layout_source: 'STAFF',
        column_count: 5,
        layout: [
          { plug_id: testPlugId, widget_id: 'attendance_pulse', col: 1, row: 1 },
          { plug_id: testPlugId, widget_id: 'pastoral_watch', col: 3, row: 2 },
        ],
      });

      // Reload page and confirm 2D coordinates and column_count = 5 are restored
      await page.reload();
      await expect(page.getByTestId('dashboard-widget-grid')).toHaveAttribute('data-column-count', '5');
      const attendanceCard = page.getByTestId('placed-widget-card-attendance_pulse');
      await expectWidgetCoordinates(attendanceCard, '1', '1');
      await expectWidgetCoordinates(pastoralCard, '3', '2');

      // Teacher removes pastoral_watch in Edit Mode
      await page.getByTestId('dashboard-edit-mode-toggle-btn').click();
      await page.getByTestId('remove-widget-btn-pastoral_watch').click();
      await expect(page.getByTestId('placed-widget-card-pastoral_watch')).toHaveCount(0);

      // 3. Switch to Admin persona -> Admin "Edit Dashboard Grid" ONLY edits Admin personal dashboard
      const switchAdmin = await request.post('/api/auth/persona', { data: { staff_id: adminStaffId } });
      expect(switchAdmin.ok()).toBe(true);
      await page.goto('/');
      await expect(page.getByTestId('admin-role-dashboard-controls')).toHaveCount(0);
      await page.getByTestId('dashboard-edit-mode-toggle-btn').click();
      await expect(page.getByTestId('admin-role-dashboard-controls')).toHaveCount(0);

      // Append role default via API
      const appendRoleRes = await request.post('/api/admin/dashboard/role-default/append', {
        data: { staff_id: adminStaffId, role_id: 'TEACHER', plug_id: testPlugId, widget_id: 'pastoral_watch' },
      });
      expect(appendRoleRes.ok()).toBe(true);

      // Switch back to Teacher -> Confirm pastoral_watch appended at free slot (col=3, row=1)
      const switchTeacherAgain = await request.post('/api/auth/persona', { data: { staff_id: teacherStaffId } });
      expect(switchTeacherAgain.ok()).toBe(true);
      await page.goto('/');
      await expectWidgetCoordinates(page.getByTestId('placed-widget-card-attendance_pulse'), '1', '1');
      await expectWidgetCoordinates(page.getByTestId('placed-widget-card-pastoral_watch'), '3', '1');

      // 4. Persist revoked widget in Teacher layout and verify inline permission error card
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
      await request.delete(`/api/test/seed-personas/${encodeURIComponent(personaFixture.fixture_id)}`);
    }
  });

  test('widgetIdCollisionAcrossPlugins_twoPluginsSharingSameWidgetId_placesReordersLeftRightRemovesAndBroadcastsInitByCompositeKey_test', async ({
    page,
    request,
  }) => {
    // 1. Seed synthetic plugin lru1 WITH cloned widgets
    const seedWithWidgetsRes = await request.post('/api/test/seed-lru-plugins', {
      data: { plugin_ids: ['uk.org.optimis.lru1'], include_widgets: true },
    });
    expect(seedWithWidgetsRes.ok()).toBe(true);
    const [lru1] = (await seedWithWidgetsRes.json()).seeded_plugin_ids as string[];
    expect(lru1).toBeTruthy();

    await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}&second_pupil_id=${encodeURIComponent(secondPupilId)}`);

    const testAttendanceCard = page.locator(`[data-testid="placed-widget-card-attendance_pulse"][data-plug-id="${testPlugId}"]`);
    const lru1AttendanceCard = page.locator(`[data-testid="placed-widget-card-attendance_pulse"][data-plug-id="${lru1}"]`);
    await expect(testAttendanceCard).toBeVisible();
    await expect(lru1AttendanceCard).toHaveCount(0);

    // 2. Landing Page Search for 'Attendance Pulse'
    const landingInput = page.getByTestId('landing-search-input');
    await landingInput.fill('Attendance Pulse');
    const landingAddTestBtn = page.locator(`[data-testid="landing-search-add-widget-btn-attendance_pulse"][data-plug-id="${testPlugId}"]`);
    const landingAddLru1Btn = page.locator(`[data-testid="landing-search-add-widget-btn-attendance_pulse"][data-plug-id="${lru1}"]`);
    await expect(landingAddTestBtn).toBeDisabled();
    await expect(landingAddLru1Btn).toBeEnabled();

    await landingAddLru1Btn.click();
    await expect(lru1AttendanceCard).toBeVisible();
    await expectWidgetCoordinates(lru1AttendanceCard, '3', '1');
    await expect(landingAddLru1Btn).toBeDisabled();
    await landingInput.fill('');

    // 3. Command Palette (Ctrl+K) search 'Pastoral'
    await page.keyboard.press('Control+k');
    const paletteInput = page.getByTestId('command-palette-input');
    await expect(paletteInput).toBeVisible();
    await paletteInput.fill('Pastoral');

    const paletteAddTestPastoralBtn = page.locator(`[data-testid="command-palette-add-widget-btn-pastoral_watch"][data-plug-id="${testPlugId}"]`);
    const paletteAddLru1PastoralBtn = page.locator(`[data-testid="command-palette-add-widget-btn-pastoral_watch"][data-plug-id="${lru1}"]`);
    await expect(paletteAddTestPastoralBtn).toBeEnabled();
    await expect(paletteAddLru1PastoralBtn).toBeEnabled();

    await paletteAddTestPastoralBtn.click();
    await expect(paletteAddTestPastoralBtn).toBeDisabled();
    await expect(paletteAddLru1PastoralBtn).toBeEnabled();
    await page.keyboard.press('Escape');

    // 4. Dashboard Edit Mode -> Widget Picker Modal states and 2D movement for colliding widget IDs
    await page.getByTestId('dashboard-edit-mode-toggle-btn').click();
    await page.getByTestId('dashboard-add-widget-btn').click();

    const pickerModal = page.getByTestId('dashboard-widget-picker-modal');
    await expect(pickerModal).toBeVisible();
    const catalogPlaceTest = pickerModal.locator(`[data-testid="place-widget-btn-pastoral_watch"][data-plug-id="${testPlugId}"]`);
    const catalogPlaceLru1 = pickerModal.locator(`[data-testid="place-widget-btn-pastoral_watch"][data-plug-id="${lru1}"]`);
    await expect(catalogPlaceTest).toBeDisabled();
    await expect(catalogPlaceLru1).toBeEnabled();
    await page.getByTestId('dashboard-widget-picker-close-btn').click();
    await expect(pickerModal).toHaveCount(0);

    // Move lru1:attendance_pulse DOWN from (3, 1) to (3, 2), then UP back to (3, 1)
    await expectWidgetCoordinates(testAttendanceCard, '1', '1');
    await expectWidgetCoordinates(lru1AttendanceCard, '3', '1');

    await page.locator(`[data-testid="move-widget-down-btn-attendance_pulse"][data-plug-id="${lru1}"]`).click();
    await expectWidgetCoordinates(lru1AttendanceCard, '3', '2');
    await expectWidgetCoordinates(testAttendanceCard, '1', '1');

    await page.locator(`[data-testid="move-widget-up-btn-attendance_pulse"][data-plug-id="${lru1}"]`).click();
    await expectWidgetCoordinates(lru1AttendanceCard, '3', '1');

    // 5. Verify both colliding widget iframes receive mis:init theme broadcasts
    const testWidgetFrame = page.frameLocator(`[data-testid="widget-iframe-attendance_pulse"][data-plug-id="${testPlugId}"]`);
    const lru1WidgetFrame = page.frameLocator(`[data-testid="widget-iframe-attendance_pulse"][data-plug-id="${lru1}"]`);

    await expect(testWidgetFrame.getByTestId('widget-theme-status')).toContainText('THEME:light');
    await expect(lru1WidgetFrame.getByTestId('widget-theme-status')).toContainText('THEME:light');

    // Toggle theme to high-contrast via Settings
    await toggleThemeViaSettings(page, 'high-contrast');
    await expect(testWidgetFrame.getByTestId('widget-theme-status')).toContainText('THEME:high-contrast');
    await expect(lru1WidgetFrame.getByTestId('widget-theme-status')).toContainText('THEME:high-contrast');

    // 6. Remove ONLY lru1 attendance widget -> testPlugId widget remains mounted
    await page.locator(`[data-testid="remove-widget-btn-attendance_pulse"][data-plug-id="${lru1}"]`).click();
    await expect(lru1AttendanceCard).toHaveCount(0);
    await expect(testAttendanceCard).toBeVisible();

    // Toggle theme back to light -> remaining widget still receives mis:init
    await toggleThemeViaSettings(page, 'light');
    await expect(testWidgetFrame.getByTestId('widget-theme-status')).toContainText('THEME:light');
  });
});
