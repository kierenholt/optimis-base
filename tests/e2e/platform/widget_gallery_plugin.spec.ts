import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import {
  ROOT_DIR,
  MIS_VERIFY_BIN,
  MIS_TEST_BIN,
  MODAL_MAX_WIDTH,
  MODAL_MAX_HEIGHT,
  SIZE_FILTERS,
  EXPECTED_WIDGETS,
  type Milestone1SeedResponse,
  type PersonaSeedResponse,
  type UiConfigResponse,
  runBinary,
  buildDashboardUrl,
  switchPersona,
  setDashboardLayout,
  setupWidgetGalleryEnvironment,
  cleanupWidgetGalleryEnvironment,
  placeWidgetFromPicker,
  expectFootprintDiagram,
  expectWidgetIframeHydrated,
  expectSplitPreviewSelection,
} from './widget_gallery_fixtures';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

test.describe('Widget Gallery Test Plugin (plugins/widget_gallery_plugin, 12 Multi-Size Widgets)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let personaSeed: PersonaSeedResponse | null = null;
  let m1Seed: Milestone1SeedResponse | null = null;
  let testPlugId: string = '';
  let galleryPlugId: string = '';

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    page.on('console', (msg) => browserLogs.push(`[browser:${msg.type()}] ${msg.text()}`));
    page.on('pageerror', (err) => pageErrors.push(`[pageerror] ${err.stack || err.message}`));

    const cursorRes = await request.get('/api/test/logs/cursor');
    if (!cursorRes.ok()) throw new Error(`Failed to obtain service log cursor: ${cursorRes.status()}`);
    logCursor = (await cursorRes.json()).cursor;

    const env = await setupWidgetGalleryEnvironment(request);
    personaSeed = env.personaSeed;
    m1Seed = env.m1Seed;
    testPlugId = env.testPlugId;
    galleryPlugId = env.galleryPlugId;
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

    await cleanupWidgetGalleryEnvironment(request, { personaSeed, m1Seed, galleryPlugId, testPlugId });
    expect(pageErrors).toEqual([]);
  });

  test('widgetGalleryPlugin_manifestVerificationAndUiConfig_exposesTwelveWidgetsAcrossFourFootprintsAndPassesToolchainChecks_test', async ({
    request,
  }) => {
    const schemaPath = path.resolve(ROOT_DIR, 'sdk/manifest.schema.json');
    const manifestSchema = JSON.parse(fs.readFileSync(schemaPath, 'utf-8'));
    expect(manifestSchema.properties.widgets.items.required).toContain('description');
    expect(manifestSchema.properties.widgets.items.properties.description).toEqual({
      type: 'string',
      minLength: 1,
    });

    const tmpPluginDir = fs.mkdtempSync(path.join(os.tmpdir(), 'mis-verify-missing-widget-desc-'));
    try {
      fs.cpSync(path.resolve(ROOT_DIR, 'plugins/widget_gallery_plugin'), tmpPluginDir, { recursive: true });
      const tmpManifestPath = path.join(tmpPluginDir, 'manifest.json');
      const tmpManifest = JSON.parse(fs.readFileSync(tmpManifestPath, 'utf-8'));
      tmpManifest.widgets[0].description = '   ';
      fs.writeFileSync(tmpManifestPath, JSON.stringify(tmpManifest, null, 2));

      const badVerifyRes = runBinary(MIS_VERIFY_BIN, [tmpPluginDir]);
      expect(badVerifyRes.exitCode).not.toBe(0);
      expect(badVerifyRes.stdout + '\n' + badVerifyRes.stderr).toContain('MISSING_WIDGET_DESCRIPTION');
    } finally {
      fs.rmSync(tmpPluginDir, { recursive: true, force: true });
    }

    const verifyRes = runBinary(MIS_VERIFY_BIN, ['plugins/widget_gallery_plugin']);
    expect(verifyRes.exitCode).toBe(0);
    expect(verifyRes.stdout).toContain('MANIFEST_LUA_UI_OK');
    expect(verifyRes.stdout).toContain('CSP_OFFLINE_OK');

    const testRes = runBinary(MIS_TEST_BIN, ['plugins/widget_gallery_plugin']);
    expect(testRes.exitCode).toBe(0);
    expect(testRes.stdout).toContain('status=PASS');

    const uiConfigRes = await request.get('/api/system/ui-config');
    expect(uiConfigRes.ok()).toBe(true);
    const uiConfig = (await uiConfigRes.json()) as UiConfigResponse;

    const galleryPlugin = uiConfig.plugins.find((p) => p.id === galleryPlugId);
    expect(galleryPlugin).toBeDefined();
    expect(galleryPlugin!.title).toBe('Widget Gallery & Layout Bench');
    expect(galleryPlugin!.icon).toBe('layout-grid');
    expect(Array.isArray(galleryPlugin!.widgets)).toBe(true);
    expect(galleryPlugin!.widgets.length).toBe(12);

    for (const expected of EXPECTED_WIDGETS) {
      const actual = galleryPlugin!.widgets.find((w) => w.id === expected.id);
      expect(actual).toBeDefined();
      expect(actual!.title).toBe(expected.title);
      expect(actual!.description).toBe(expected.description);
      expect(actual!.plugin_title).toBe('Widget Gallery & Layout Bench');
      expect(actual!.width).toBe(expected.width);
      expect(actual!.height).toBe(expected.height);
      expect(actual!.icon).toBe(expected.icon);
      expect(actual!.entry).toBe(expected.entry);
      expect(Array.isArray(actual!.keywords)).toBe(true);
      expect(actual!.keywords.length).toBeGreaterThan(0);
    }
  });

  test('widgetGalleryPlugin_editModeModalAndMultiSizePlacement_renders1x1_2x1_1x2_And2x2WidgetsWithLiveRpcAndDeepLink_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) throw new Error('Missing mandatory seeded fixtures');

    await switchPersona(request, m1Seed.teacher_staff_id);
    await setDashboardLayout(request, m1Seed.teacher_staff_id, []);
    await page.goto(buildDashboardUrl(m1Seed));

    await page.getByTestId('dashboard-edit-mode-toggle-btn').click();
    await page.getByTestId('dashboard-add-widget-btn').click();

    const pickerModal = page.getByTestId('dashboard-widget-picker-modal');
    await expect(pickerModal).toBeVisible();

    for (const expected of EXPECTED_WIDGETS) {
      await expect(page.getByTestId(`place-widget-btn-${expected.id}`)).toBeVisible();
    }

    await placeWidgetFromPicker(page, 'wg_1x1_attendance_counter', 1, 1, 1, 1, false);
    await placeWidgetFromPicker(page, 'wg_2x1_attendance_pulse_bar', 2, 1, 2, 1, true);
    await placeWidgetFromPicker(page, 'wg_1x2_roster_ticker', 1, 2, 4, 1, true);
    await placeWidgetFromPicker(page, 'wg_2x2_interactive_sandbox_bench', 2, 2, 1, 2, true);

    await page.getByTestId('dashboard-edit-mode-toggle-btn').click();

    await expectWidgetIframeHydrated(page, 'wg_1x1_attendance_counter', '1x1');
    await expectWidgetIframeHydrated(page, 'wg_2x1_attendance_pulse_bar', '2x1');

    const frame1x2 = page.frameLocator('[data-testid="widget-iframe-wg_1x2_roster_ticker"]');
    await expect(frame1x2.getByTestId('wg-footprint-badge')).toHaveText('1x2');
    await expect(frame1x2.getByTestId('wg-roster-list')).toContainText('Ada');

    const frame2x2 = page.frameLocator('[data-testid="widget-iframe-wg_2x2_interactive_sandbox_bench"]');
    await expect(frame2x2.getByTestId('wg-footprint-badge')).toHaveText('2x2');
    await frame2x2.getByTestId('wg-refresh-rpc-btn').click();
    await expect(frame2x2.getByTestId('wg-rpc-status')).toContainText('RPC_OK:2_ACTIVE_PUPILS');

    await frame2x2.getByTestId('wg-deep-link-student').click();
    await expect.poll(() => new URL(page.url()).pathname).toBe(`/p/${testPlugId}/student/${m1Seed!.active_pupil_id}`);
  });

  test('widgetPickerModal_centeredDialogWithDescriptionsSizeSearchPluginAndPlacedFilters_filtersAndPlacesSelectedWidget_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) throw new Error('Missing mandatory seeded fixtures');

    await switchPersona(request, m1Seed.teacher_staff_id);
    await setDashboardLayout(request, m1Seed.teacher_staff_id, [
      { plug_id: galleryPlugId, widget_id: 'wg_1x1_attendance_counter', grid_col: 1, grid_row: 1 },
    ]);

    await page.goto(buildDashboardUrl(m1Seed));

    await page.getByTestId('dashboard-edit-mode-toggle-btn').click();
    await page.getByTestId('dashboard-add-widget-btn').click();

    const backdrop = page.getByTestId('dashboard-widget-picker-backdrop');
    const modal = page.getByTestId('dashboard-widget-picker-modal');
    await expect(backdrop).toBeVisible();
    await expect(modal).toBeVisible();
    await expect(modal).toHaveAttribute('role', 'dialog');
    await expect(modal).toHaveAttribute('aria-modal', 'true');

    await page.keyboard.press('Escape');
    await expect(modal).toHaveCount(0);
    await page.getByTestId('dashboard-add-widget-btn').click();
    await expect(modal).toBeVisible();

    for (const expected of EXPECTED_WIDGETS) {
      const card = modal.getByTestId(`picker-widget-card-${expected.id}`);
      await expect(card).toBeVisible();
      await expect(card).toContainText(expected.title);
      await expect(modal.getByTestId(`picker-widget-description-${expected.id}`)).toHaveText(expected.description);
      const cardDiagram = card.getByTestId(`picker-widget-footprint-diagram-${expected.id}`);
      await expectFootprintDiagram(cardDiagram, expected.width, expected.height);
      await expect(card).toContainText('Widget Gallery & Layout Bench');
    }

    for (const sizeOpt of SIZE_FILTERS) {
      const filterBtn = modal.getByTestId(`widget-picker-size-filter-${sizeOpt.id}`);
      const filterDiagram = filterBtn.getByTestId(`size-filter-footprint-diagram-${sizeOpt.id}`);
      await expectFootprintDiagram(filterDiagram, sizeOpt.w, sizeOpt.h);
    }

    const sizeFiltersTest = [
      {
        size: '1x1',
        visible: ['wg_1x1_attendance_counter', 'wg_1x1_sen_badge'],
        hidden: ['wg_2x1_attendance_pulse_bar', 'wg_1x2_roster_ticker', 'wg_2x2_class_overview_matrix'],
      },
      {
        size: '2x1',
        visible: ['wg_2x1_attendance_pulse_bar'],
        hidden: ['wg_1x1_sen_badge', 'wg_1x2_roster_ticker', 'wg_2x2_class_overview_matrix'],
      },
      {
        size: '1x2',
        visible: ['wg_1x2_roster_ticker'],
        hidden: ['wg_1x1_sen_badge', 'wg_2x1_attendance_pulse_bar', 'wg_2x2_class_overview_matrix'],
      },
      {
        size: '2x2',
        visible: ['wg_2x2_class_overview_matrix'],
        hidden: ['wg_1x1_sen_badge', 'wg_2x1_attendance_pulse_bar', 'wg_1x2_roster_ticker'],
      },
    ];

    for (const st of sizeFiltersTest) {
      await modal.getByTestId(`widget-picker-size-filter-${st.size}`).click();
      for (const id of st.visible) await expect(modal.getByTestId(`picker-widget-card-${id}`)).toBeVisible();
      for (const id of st.hidden) await expect(modal.getByTestId(`picker-widget-card-${id}`)).toHaveCount(0);
    }

    await modal.getByTestId('widget-picker-size-filter-all').click();
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_sen_badge')).toBeVisible();
    await expect(modal.getByTestId('picker-widget-card-wg_2x2_class_overview_matrix')).toBeVisible();

    const searchInput = modal.getByTestId('widget-picker-search-input');
    await searchInput.fill('Individual Healthcare Plans');
    await expect(modal.getByTestId('picker-widget-card-wg_1x2_sen_medical_watch_list')).toBeVisible();
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_attendance_counter')).toHaveCount(0);
    await expect(modal.getByTestId('picker-widget-card-wg_2x2_class_overview_matrix')).toHaveCount(0);
    await searchInput.fill('');
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_attendance_counter')).toBeVisible();

    const pluginFilter = modal.getByTestId('widget-picker-plugin-filter');
    await expect(modal.getByTestId('picker-widget-card-attendance_pulse')).toBeVisible();
    await pluginFilter.selectOption(galleryPlugId);
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_sen_badge')).toBeVisible();
    await expect(modal.getByTestId('picker-widget-card-attendance_pulse')).toHaveCount(0);
    await pluginFilter.selectOption('all');
    await expect(modal.getByTestId('picker-widget-card-attendance_pulse')).toBeVisible();

    const hidePlacedToggle = modal.getByTestId('widget-picker-hide-placed-toggle');
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_attendance_counter')).toBeVisible();
    await expect(modal.getByTestId('place-widget-btn-wg_1x1_attendance_counter')).toBeDisabled();
    await hidePlacedToggle.click();
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_attendance_counter')).toHaveCount(0);
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_sen_badge')).toBeVisible();
    await hidePlacedToggle.click();
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_attendance_counter')).toBeVisible();

    await modal.getByTestId('widget-picker-size-filter-2x1').click();
    await placeWidgetFromPicker(page, 'wg_2x1_attendance_pulse_bar', 2, 1, 2, 1, false);
  });

  test('widgetPickerModal_galleryAndSplitPreviewModesInLargeModal_switchesBetweenCardGalleryAndTwoColumnLiveWidgetPreview_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) throw new Error('Missing mandatory seeded fixtures');

    await switchPersona(request, m1Seed.teacher_staff_id);
    await setDashboardLayout(request, m1Seed.teacher_staff_id, []);
    await page.goto(buildDashboardUrl(m1Seed));

    await page.getByTestId('dashboard-edit-mode-toggle-btn').click();
    await page.getByTestId('dashboard-add-widget-btn').click();

    const modal = page.getByTestId('dashboard-widget-picker-modal');
    await expect(modal).toBeVisible();
    await expect(modal).toHaveAttribute('data-modal-view-mode', 'gallery');

    const modalStyles = await modal.evaluate((el) => {
      const cs = window.getComputedStyle(el);
      return { maxWidth: cs.maxWidth, maxHeight: cs.maxHeight };
    });
    expect(modalStyles.maxWidth).toBe(MODAL_MAX_WIDTH);
    expect(modalStyles.maxHeight).toBe(MODAL_MAX_HEIGHT);

    const galleryBtn = modal.getByTestId('widget-picker-view-mode-gallery');
    const splitBtn = modal.getByTestId('widget-picker-view-mode-split');
    await expect(galleryBtn).toBeVisible();
    await expect(splitBtn).toBeVisible();
    await expect(galleryBtn).toHaveAttribute('aria-pressed', 'true');
    await expect(splitBtn).toHaveAttribute('aria-pressed', 'false');
    await expect(modal.getByTestId('widget-picker-gallery-view')).toBeVisible();
    await expect(modal.getByTestId('widget-picker-split-view')).toHaveCount(0);

    await splitBtn.click();
    await expect(modal).toHaveAttribute('data-modal-view-mode', 'split');
    await expect(galleryBtn).toHaveAttribute('aria-pressed', 'false');
    await expect(splitBtn).toHaveAttribute('aria-pressed', 'true');
    await expect(modal.getByTestId('widget-picker-gallery-view')).toHaveCount(0);

    const splitView = modal.getByTestId('widget-picker-split-view');
    const splitList = modal.getByTestId('widget-picker-split-list');
    const splitPreview = modal.getByTestId('widget-picker-split-preview');
    await expect(splitView).toBeVisible();
    await expect(splitList).toBeVisible();
    await expect(splitPreview).toBeVisible();

    const widget1x1 = EXPECTED_WIDGETS[0];
    await expectSplitPreviewSelection(page, splitList, splitPreview, widget1x1, 300, 180);

    const widget2x2 = EXPECTED_WIDGETS[9];
    await expectSplitPreviewSelection(page, splitList, splitPreview, widget2x2, 612, 372);
    await expect(splitList.getByTestId(`split-widget-list-item-${widget1x1.id}`)).toHaveAttribute(
      'aria-selected',
      'false'
    );

    await splitPreview.getByTestId(`place-widget-btn-${widget2x2.id}`).click();
    await expect(modal).toHaveCount(0);

    const gridPreview = page.getByTestId('dashboard-widget-placement-preview');
    await expect(gridPreview).toBeVisible();
    await expect(gridPreview).toHaveAttribute('data-preview-width', '2');
    await expect(gridPreview).toHaveAttribute('data-preview-height', '2');
    await gridPreview.click();
    await expect(page.getByTestId(`placed-widget-card-${widget2x2.id}`)).toBeVisible();
  });
});
