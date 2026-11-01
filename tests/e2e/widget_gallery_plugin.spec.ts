import { test, expect } from '@playwright/test';
import { execFileSync } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

const ROOT_DIR = process.cwd();
const MIS_VERIFY_BIN = path.resolve(ROOT_DIR, 'bin/mis-verify');
const MIS_TEST_BIN = path.resolve(ROOT_DIR, 'bin/mis-test');

interface Milestone1SeedResponse {
  fixture_id: string;
  academic_year_id: string;
  reference_date: string;
  teacher_staff_id: string;
  other_teacher_staff_id: string;
  dsl_staff_id: string;
  own_group_id: string;
  other_group_id: string;
  active_pupil_id: string;
  second_active_pupil_id: string;
  transferred_pupil_id: string;
  other_group_pupil_id: string;
}

interface PersonaSeedResponse {
  fixture_id: string;
  pupil_id: string;
  admin_staff_id: string;
  dsl_staff_id: string;
  hod_staff_id: string;
  teacher_staff_id: string;
  attendance_officer_staff_id: string;
  terminated_staff_id: string;
}

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

import * as os from 'os';

const EXPECTED_WIDGETS: Array<{
  id: string;
  title: string;
  description: string;
  width: 1 | 2;
  height: 1 | 2;
  icon: string;
  entry: string;
}> = [
  // 3x 1x1
  {
    id: 'wg_1x1_attendance_counter',
    title: 'Session Attendance Counter (1x1)',
    description: 'Compact 1x1 live statutory AM/PM session attendance counter with present and absent pupil totals.',
    width: 1,
    height: 1,
    icon: 'clipboard-check',
    entry: 'ui/dist/widget_1x1_attendance_counter.html',
  },
  {
    id: 'wg_1x1_sen_badge',
    title: 'SEN & Medical Quick Badge (1x1)',
    description: 'At-a-glance 1x1 indicator showing active SEN Support (K), EHCP (E), and medical condition counts for your teaching group.',
    width: 1,
    height: 1,
    icon: 'heart-pulse',
    entry: 'ui/dist/widget_1x1_sen_badge.html',
  },
  {
    id: 'wg_1x1_group_size_stat',
    title: 'Active Group Size Stat (1x1)',
    description: 'Single-tile 1x1 headcount summary displaying temporally active enrolled pupils across your assigned classes.',
    width: 1,
    height: 1,
    icon: 'users',
    entry: 'ui/dist/widget_1x1_group_size_stat.html',
  },
  // 3x 2x1
  {
    id: 'wg_2x1_attendance_pulse_bar',
    title: 'AM/PM Attendance Pulse Bar (2x1)',
    description: 'Wide 2x1 horizontal progress bar comparing statutory AM and PM session attendance percentages across your active group.',
    width: 2,
    height: 1,
    icon: 'activity',
    entry: 'ui/dist/widget_2x1_attendance_pulse_bar.html',
  },
  {
    id: 'wg_2x1_pupil_quick_lookup',
    title: 'Pupil Quick Lookup & Deep-Link (2x1)',
    description: 'Wide 2x1 quick-action card for inspecting a pupil summary and launching their full profile via entity deep-linking.',
    width: 2,
    height: 1,
    icon: 'search',
    entry: 'ui/dist/widget_2x1_pupil_quick_lookup.html',
  },
  {
    id: 'wg_2x1_next_lesson_banner',
    title: 'Next Timetable Lesson Banner (2x1)',
    description: 'Wide 2x1 banner highlighting the active teaching group code, assigned room, and session readiness status.',
    width: 2,
    height: 1,
    icon: 'calendar',
    entry: 'ui/dist/widget_2x1_next_lesson_banner.html',
  },
  // 3x 1x2
  {
    id: 'wg_1x2_roster_ticker',
    title: 'Class Roster Vertical Ticker (1x2)',
    description: 'Tall 1x2 vertical scrolling ticker listing active pupils in your teaching group with live attendance status badges.',
    width: 1,
    height: 2,
    icon: 'user-check',
    entry: 'ui/dist/widget_1x2_roster_ticker.html',
  },
  {
    id: 'wg_1x2_sen_medical_watch_list',
    title: 'SEN & IHCP Watch List (1x2)',
    description: 'Tall 1x2 pastoral and SEND watch list surfacing pupils with active SEN provisions or Individual Healthcare Plans.',
    width: 1,
    height: 2,
    icon: 'stethoscope',
    entry: 'ui/dist/widget_1x2_sen_medical_watch_list.html',
  },
  {
    id: 'wg_1x2_period_timeline',
    title: 'Daily Period Timeline (1x2)',
    description: 'Tall 1x2 vertical schedule timeline tracking AM registration, teaching periods, and PM statutory roll call.',
    width: 1,
    height: 2,
    icon: 'book-open',
    entry: 'ui/dist/widget_1x2_period_timeline.html',
  },
  // 3x 2x2
  {
    id: 'wg_2x2_class_overview_matrix',
    title: 'Class Overview & RPC Matrix (2x2)',
    description: 'Large 2x2 multi-metric classroom matrix combining headcount, SEN/medical flags, and live statutory attendance rollups.',
    width: 2,
    height: 2,
    icon: 'layout-grid',
    entry: 'ui/dist/widget_2x2_class_overview_matrix.html',
  },
  {
    id: 'wg_2x2_attendance_breakdown_grid',
    title: 'Session Attendance Breakdown Grid (2x2)',
    description: 'Large 2x2 per-pupil AM and PM statutory session mark breakdown table with live Core DAL refresh.',
    width: 2,
    height: 2,
    icon: 'file-check',
    entry: 'ui/dist/widget_2x2_attendance_breakdown_grid.html',
  },
  {
    id: 'wg_2x2_interactive_sandbox_bench',
    title: 'Interactive Widget RPC & Link Bench (2x2)',
    description: 'Large 2x2 interactive verification bench for testing on-demand Lua RPC calls and student/group entity deep-links.',
    width: 2,
    height: 2,
    icon: 'award',
    entry: 'ui/dist/widget_2x2_interactive_sandbox_bench.html',
  },
];

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
    logCursor = '';
    personaSeed = null;
    m1Seed = null;
    testPlugId = '';
    galleryPlugId = '';

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
    const authModeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!authModeRes.ok()) {
      throw new Error(`Failed to enable dev persona switcher: ${authModeRes.status()}`);
    }

    // 4. Seed Persona fixtures and Milestone 1 scoped fixtures
    const pSeedRes = await request.post('/api/test/seed-personas', {
      data: {
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    if (!pSeedRes.ok()) {
      throw new Error(`Failed to seed persona fixtures: ${pSeedRes.status()}`);
    }
    personaSeed = (await pSeedRes.json()) as PersonaSeedResponse;

    const m1Res = await request.post('/api/test/seed-milestone1-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-27',
        include_second_active_pupil: true,
      },
    });
    if (!m1Res.ok()) {
      throw new Error(`Failed to seed Milestone 1 fixtures: ${m1Res.status()}`);
    }
    m1Seed = (await m1Res.json()) as Milestone1SeedResponse;

    // 5. Switch to Admin persona and install + activate test_plugin and widget_gallery_plugin
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personaSeed.admin_staff_id },
    });
    if (!switchAdminRes.ok()) {
      throw new Error(`Failed to switch to Admin persona: ${switchAdminRes.status()}`);
    }

    const installTestRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'test_plugin', staff_id: personaSeed.admin_staff_id },
    });
    if (!installTestRes.ok()) {
      throw new Error(`Failed to install test_plugin: ${installTestRes.status()}`);
    }
    testPlugId = (await installTestRes.json()).plug_id;
    await request.post(`/api/admin/plugins/${encodeURIComponent(testPlugId)}/activate`, {
      data: { staff_id: personaSeed.admin_staff_id },
    });

    const installGalleryRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'widget_gallery_plugin', staff_id: personaSeed.admin_staff_id },
    });
    if (!installGalleryRes.ok()) {
      throw new Error(`Failed to install widget_gallery_plugin: ${installGalleryRes.status()}`);
    }
    galleryPlugId = (await installGalleryRes.json()).plug_id;
    const activateGalleryRes = await request.post(
      `/api/admin/plugins/${encodeURIComponent(galleryPlugId)}/activate`,
      {
        data: { staff_id: personaSeed.admin_staff_id },
      }
    );
    if (!activateGalleryRes.ok()) {
      throw new Error(`Failed to activate widget_gallery_plugin: ${activateGalleryRes.status()}`);
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

    // 3. Clean up installed plugins and seeded fixtures
    if (personaSeed) {
      await request.post('/api/auth/persona', {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      for (const pid of [galleryPlugId, testPlugId]) {
        if (pid) {
          await request.delete(`/api/admin/plugins/${encodeURIComponent(pid)}`);
        }
      }
    }
    if (m1Seed) {
      await request.delete(`/api/test/seed-milestone1-fixtures/${encodeURIComponent(m1Seed.fixture_id)}`);
      m1Seed = null;
    }
    if (personaSeed) {
      await request.delete(`/api/test/seed-personas/${encodeURIComponent(personaSeed.fixture_id)}`);
      personaSeed = null;
    }

    expect(pageErrors).toEqual([]);
  });

  test('widgetGalleryPlugin_manifestVerificationAndUiConfig_exposesTwelveWidgetsAcrossFourFootprintsAndPassesToolchainChecks_test', async ({
    request,
  }) => {
    // 1. Verify sdk/manifest.schema.json requires `description` on `widgets.items`
    const schemaPath = path.resolve(ROOT_DIR, 'sdk/manifest.schema.json');
    const manifestSchema = JSON.parse(fs.readFileSync(schemaPath, 'utf-8'));
    expect(manifestSchema.properties.widgets.items.required).toContain('description');
    expect(manifestSchema.properties.widgets.items.properties.description).toEqual({
      type: 'string',
      minLength: 1,
    });

    // 2. Verify bin/mis-verify fails fast with MISSING_WIDGET_DESCRIPTION when a widget omits description
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

    // 3. Verify bin/mis-verify and bin/mis-test pass on plugins/widget_gallery_plugin
    const verifyRes = runBinary(MIS_VERIFY_BIN, ['plugins/widget_gallery_plugin']);
    expect(verifyRes.exitCode).toBe(0);
    expect(verifyRes.stdout).toContain('MANIFEST_LUA_UI_OK');
    expect(verifyRes.stdout).toContain('CSP_OFFLINE_OK');

    const testRes = runBinary(MIS_TEST_BIN, ['plugins/widget_gallery_plugin']);
    expect(testRes.exitCode).toBe(0);
    expect(testRes.stdout).toContain('status=PASS');

    // 4. Verify GET /api/system/ui-config exposes all 12 widgets with exact preset dimensions and descriptions
    const uiConfigRes = await request.get('/api/system/ui-config');
    expect(uiConfigRes.ok()).toBe(true);
    const uiConfig = await uiConfigRes.json();

    const galleryPlugin = uiConfig.plugins.find((p: any) => p.id === galleryPlugId);
    expect(galleryPlugin).toBeDefined();
    expect(galleryPlugin.title).toBe('Widget Gallery & Layout Bench');
    expect(galleryPlugin.icon).toBe('layout-grid');
    expect(Array.isArray(galleryPlugin.widgets)).toBe(true);
    expect(galleryPlugin.widgets.length).toBe(12);

    for (const expected of EXPECTED_WIDGETS) {
      const actual = galleryPlugin.widgets.find((w: any) => w.id === expected.id);
      expect(actual).toBeDefined();
      expect(actual.title).toBe(expected.title);
      expect(actual.description).toBe(expected.description);
      expect(actual.plugin_title).toBe('Widget Gallery & Layout Bench');
      expect(actual.width).toBe(expected.width);
      expect(actual.height).toBe(expected.height);
      expect(actual.icon).toBe(expected.icon);
      expect(actual.entry).toBe(expected.entry);
      expect(Array.isArray(actual.keywords)).toBe(true);
      expect(actual.keywords.length).toBeGreaterThan(0);
    }
  });

  test('widgetGalleryPlugin_editModeModalAndMultiSizePlacement_renders1x1_2x1_1x2_And2x2WidgetsWithLiveRpcAndDeepLink_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // Switch to the seeded Classroom Teacher so :own_groups DAL queries resolve against m1Seed.own_group_id
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: m1Seed.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);

    // Start with an empty personal dashboard layout on a 4-column grid
    const clearLayoutRes = await request.put('/api/dashboard/layout', {
      data: {
        staff_id: m1Seed.teacher_staff_id,
        column_count: 4,
        layout: [],
      },
    });
    expect(clearLayoutRes.ok()).toBe(true);

    await page.goto(
      `/?pupil_id=${encodeURIComponent(m1Seed.active_pupil_id)}&second_pupil_id=${encodeURIComponent(
        m1Seed.second_active_pupil_id
      )}&group_id=${encodeURIComponent(m1Seed.own_group_id)}&reference_date=${encodeURIComponent(
        m1Seed.reference_date
      )}&academic_year_id=${encodeURIComponent(m1Seed.academic_year_id)}`
    );

    // 1. Enter Edit Dashboard Grid mode and open "+ Add Widget" modal
    await page.getByTestId('dashboard-edit-mode-toggle-btn').click();
    await page.getByTestId('dashboard-add-widget-btn').click();

    const pickerModal = page.getByTestId('dashboard-widget-picker-modal');
    await expect(pickerModal).toBeVisible();

    // Verify all 12 gallery widgets are listed in the picker modal
    for (const expected of EXPECTED_WIDGETS) {
      await expect(page.getByTestId(`place-widget-btn-${expected.id}`)).toBeVisible();
    }

    // 2. Place one widget of each size (1x1, 2x1, 1x2, 2x2) onto the 2D grid
    // Place 1x1 at (1, 1)
    await pickerModal.getByTestId('place-widget-btn-wg_1x1_attendance_counter').click();
    await expect(pickerModal).toHaveCount(0);
    const preview = page.getByTestId('dashboard-widget-placement-preview');
    await expect(preview).toBeVisible();
    await expect(preview).toHaveAttribute('data-preview-width', '1');
    await expect(preview).toHaveAttribute('data-preview-height', '1');
    await expect(preview).toHaveAttribute('data-preview-col', '1');
    await expect(preview).toHaveAttribute('data-preview-row', '1');
    await preview.click();
    await expect(preview).toHaveCount(0);
    await expect(page.getByTestId('placed-widget-card-wg_1x1_attendance_counter')).toHaveAttribute('data-grid-col', '1');
    await expect(page.getByTestId('placed-widget-card-wg_1x1_attendance_counter')).toHaveAttribute('data-grid-row', '1');
    await expect(page.getByTestId('placed-widget-card-wg_1x1_attendance_counter')).toHaveAttribute('data-width', '1');
    await expect(page.getByTestId('placed-widget-card-wg_1x1_attendance_counter')).toHaveAttribute('data-height', '1');

    // Place 2x1 at (2, 1)
    await page.getByTestId('dashboard-add-widget-btn').click();
    await expect(pickerModal).toBeVisible();
    await pickerModal.getByTestId('place-widget-btn-wg_2x1_attendance_pulse_bar').click();
    await expect(pickerModal).toHaveCount(0);
    await expect(preview).toBeVisible();
    await expect(preview).toHaveAttribute('data-preview-width', '2');
    await expect(preview).toHaveAttribute('data-preview-height', '1');
    await expect(preview).toHaveAttribute('data-preview-col', '2');
    await expect(preview).toHaveAttribute('data-preview-row', '1');
    await preview.click();
    await expect(preview).toHaveCount(0);
    await expect(page.getByTestId('placed-widget-card-wg_2x1_attendance_pulse_bar')).toHaveAttribute('data-grid-col', '2');
    await expect(page.getByTestId('placed-widget-card-wg_2x1_attendance_pulse_bar')).toHaveAttribute('data-grid-row', '1');
    await expect(page.getByTestId('placed-widget-card-wg_2x1_attendance_pulse_bar')).toHaveAttribute('data-width', '2');
    await expect(page.getByTestId('placed-widget-card-wg_2x1_attendance_pulse_bar')).toHaveAttribute('data-height', '1');

    // Place 1x2 at (4, 1)
    await page.getByTestId('dashboard-add-widget-btn').click();
    await expect(pickerModal).toBeVisible();
    await pickerModal.getByTestId('place-widget-btn-wg_1x2_roster_ticker').click();
    await expect(pickerModal).toHaveCount(0);
    await expect(preview).toBeVisible();
    await expect(preview).toHaveAttribute('data-preview-width', '1');
    await expect(preview).toHaveAttribute('data-preview-height', '2');
    await expect(preview).toHaveAttribute('data-preview-col', '4');
    await expect(preview).toHaveAttribute('data-preview-row', '1');
    await preview.click();
    await expect(preview).toHaveCount(0);
    await expect(page.getByTestId('placed-widget-card-wg_1x2_roster_ticker')).toHaveAttribute('data-grid-col', '4');
    await expect(page.getByTestId('placed-widget-card-wg_1x2_roster_ticker')).toHaveAttribute('data-grid-row', '1');
    await expect(page.getByTestId('placed-widget-card-wg_1x2_roster_ticker')).toHaveAttribute('data-width', '1');
    await expect(page.getByTestId('placed-widget-card-wg_1x2_roster_ticker')).toHaveAttribute('data-height', '2');

    // Place 2x2 at (1, 2)
    await page.getByTestId('dashboard-add-widget-btn').click();
    await expect(pickerModal).toBeVisible();
    await pickerModal.getByTestId('place-widget-btn-wg_2x2_interactive_sandbox_bench').click();
    await expect(pickerModal).toHaveCount(0);
    await expect(preview).toBeVisible();
    await expect(preview).toHaveAttribute('data-preview-width', '2');
    await expect(preview).toHaveAttribute('data-preview-height', '2');
    await expect(preview).toHaveAttribute('data-preview-col', '1');
    await expect(preview).toHaveAttribute('data-preview-row', '2');
    await preview.click();
    await expect(preview).toHaveCount(0);
    await expect(page.getByTestId('placed-widget-card-wg_2x2_interactive_sandbox_bench')).toHaveAttribute('data-grid-col', '1');
    await expect(page.getByTestId('placed-widget-card-wg_2x2_interactive_sandbox_bench')).toHaveAttribute('data-grid-row', '2');
    await expect(page.getByTestId('placed-widget-card-wg_2x2_interactive_sandbox_bench')).toHaveAttribute('data-width', '2');
    await expect(page.getByTestId('placed-widget-card-wg_2x2_interactive_sandbox_bench')).toHaveAttribute('data-height', '2');

    // 3. Exit Edit Mode so widget iframes are interactive
    await page.getByTestId('dashboard-edit-mode-toggle-btn').click();

    // Verify 1x1 widget rendered and hydrated with live RPC summary
    const frame1x1 = page.frameLocator('[data-testid="widget-iframe-wg_1x1_attendance_counter"]');
    await expect(frame1x1.getByTestId('wg-footprint-badge')).toHaveText('1x1');
    await expect(frame1x1.getByTestId('wg-rpc-status')).toContainText('RPC_OK');

    // Verify 2x1 widget rendered and hydrated
    const frame2x1 = page.frameLocator('[data-testid="widget-iframe-wg_2x1_attendance_pulse_bar"]');
    await expect(frame2x1.getByTestId('wg-footprint-badge')).toHaveText('2x1');
    await expect(frame2x1.getByTestId('wg-rpc-status')).toContainText('RPC_OK');

    // Verify 1x2 widget rendered and lists active pupils from own_group_id (excluding transferred pupil)
    const frame1x2 = page.frameLocator('[data-testid="widget-iframe-wg_1x2_roster_ticker"]');
    await expect(frame1x2.getByTestId('wg-footprint-badge')).toHaveText('1x2');
    await expect(frame1x2.getByTestId('wg-roster-list')).toContainText('Ada');

    // Verify 2x2 interactive bench widget executes on-demand RPC and triggers entity deep-linking
    const frame2x2 = page.frameLocator('[data-testid="widget-iframe-wg_2x2_interactive_sandbox_bench"]');
    await expect(frame2x2.getByTestId('wg-footprint-badge')).toHaveText('2x2');
    await frame2x2.getByTestId('wg-refresh-rpc-btn').click();
    await expect(frame2x2.getByTestId('wg-rpc-status')).toContainText('RPC_OK:2_ACTIVE_PUPILS');

    // Click student deep-link inside the 2x2 widget and verify navigation to Student Handler Plugin
    await frame2x2.getByTestId('wg-deep-link-student').click();
    await expect.poll(() => new URL(page.url()).pathname).toBe(`/p/${testPlugId}/student/${m1Seed!.active_pupil_id}`);
  });

  test('widgetPickerModal_centeredDialogWithDescriptionsSizeSearchPluginAndPlacedFilters_filtersAndPlacesSelectedWidget_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: m1Seed.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);

    // Pre-place one widget (wg_1x1_attendance_counter) so we can verify "Placed" state and "Hide already placed" filter
    const presetLayoutRes = await request.put('/api/dashboard/layout', {
      data: {
        staff_id: m1Seed.teacher_staff_id,
        column_count: 4,
        layout: [
          {
            plug_id: galleryPlugId,
            widget_id: 'wg_1x1_attendance_counter',
            grid_col: 1,
            grid_row: 1,
          },
        ],
      },
    });
    expect(presetLayoutRes.ok()).toBe(true);

    await page.goto(
      `/?pupil_id=${encodeURIComponent(m1Seed.active_pupil_id)}&second_pupil_id=${encodeURIComponent(
        m1Seed.second_active_pupil_id
      )}&group_id=${encodeURIComponent(m1Seed.own_group_id)}&reference_date=${encodeURIComponent(
        m1Seed.reference_date
      )}&academic_year_id=${encodeURIComponent(m1Seed.academic_year_id)}`
    );

    // 1. Enter Edit Mode and click "+ Add Widget" -> opens centered backdrop modal dialog
    await page.getByTestId('dashboard-edit-mode-toggle-btn').click();
    await page.getByTestId('dashboard-add-widget-btn').click();

    const backdrop = page.getByTestId('dashboard-widget-picker-backdrop');
    const modal = page.getByTestId('dashboard-widget-picker-modal');
    await expect(backdrop).toBeVisible();
    await expect(modal).toBeVisible();
    await expect(modal).toHaveAttribute('role', 'dialog');
    await expect(modal).toHaveAttribute('aria-modal', 'true');

    // Verify Escape key closes the modal, and clicking "+ Add Widget" re-opens it
    await page.keyboard.press('Escape');
    await expect(modal).toHaveCount(0);
    await page.getByTestId('dashboard-add-widget-btn').click();
    await expect(modal).toBeVisible();

    // 2. Verify each of the 12 gallery widget cards displays its title, description, visual 2x2 footprint diagram, and plugin title
    for (const expected of EXPECTED_WIDGETS) {
      const card = modal.getByTestId(`picker-widget-card-${expected.id}`);
      await expect(card).toBeVisible();
      await expect(card).toContainText(expected.title);
      await expect(modal.getByTestId(`picker-widget-description-${expected.id}`)).toHaveText(expected.description);
      const cardDiagram = card.getByTestId(`picker-widget-footprint-diagram-${expected.id}`);
      await expect(cardDiagram).toBeVisible();
      await expect(cardDiagram).toHaveAttribute('data-footprint-width', String(expected.width));
      await expect(cardDiagram).toHaveAttribute('data-footprint-height', String(expected.height));
      await expect(card).toContainText('Widget Gallery & Layout Bench');
    }

    // 3. Verify Size Filter Buttons render 2x2 footprint diagrams (1x1, 2x1, 1x2, 2x2) instead of text labels
    for (const sizeOpt of [
      { id: '1x1', w: '1', h: '1' },
      { id: '2x1', w: '2', h: '1' },
      { id: '1x2', w: '1', h: '2' },
      { id: '2x2', w: '2', h: '2' },
    ]) {
      const filterBtn = modal.getByTestId(`widget-picker-size-filter-${sizeOpt.id}`);
      const filterDiagram = filterBtn.getByTestId(`size-filter-footprint-diagram-${sizeOpt.id}`);
      await expect(filterDiagram).toBeVisible();
      await expect(filterDiagram).toHaveAttribute('data-footprint-width', sizeOpt.w);
      await expect(filterDiagram).toHaveAttribute('data-footprint-height', sizeOpt.h);
    }

    await modal.getByTestId('widget-picker-size-filter-1x1').click();
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_attendance_counter')).toBeVisible();
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_sen_badge')).toBeVisible();
    await expect(modal.getByTestId('picker-widget-card-wg_2x1_attendance_pulse_bar')).toHaveCount(0);
    await expect(modal.getByTestId('picker-widget-card-wg_1x2_roster_ticker')).toHaveCount(0);
    await expect(modal.getByTestId('picker-widget-card-wg_2x2_class_overview_matrix')).toHaveCount(0);

    await modal.getByTestId('widget-picker-size-filter-2x1').click();
    await expect(modal.getByTestId('picker-widget-card-wg_2x1_attendance_pulse_bar')).toBeVisible();
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_sen_badge')).toHaveCount(0);
    await expect(modal.getByTestId('picker-widget-card-wg_1x2_roster_ticker')).toHaveCount(0);
    await expect(modal.getByTestId('picker-widget-card-wg_2x2_class_overview_matrix')).toHaveCount(0);

    await modal.getByTestId('widget-picker-size-filter-1x2').click();
    await expect(modal.getByTestId('picker-widget-card-wg_1x2_roster_ticker')).toBeVisible();
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_sen_badge')).toHaveCount(0);
    await expect(modal.getByTestId('picker-widget-card-wg_2x1_attendance_pulse_bar')).toHaveCount(0);
    await expect(modal.getByTestId('picker-widget-card-wg_2x2_class_overview_matrix')).toHaveCount(0);

    await modal.getByTestId('widget-picker-size-filter-2x2').click();
    await expect(modal.getByTestId('picker-widget-card-wg_2x2_class_overview_matrix')).toBeVisible();
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_sen_badge')).toHaveCount(0);
    await expect(modal.getByTestId('picker-widget-card-wg_2x1_attendance_pulse_bar')).toHaveCount(0);
    await expect(modal.getByTestId('picker-widget-card-wg_1x2_roster_ticker')).toHaveCount(0);

    await modal.getByTestId('widget-picker-size-filter-all').click();
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_sen_badge')).toBeVisible();
    await expect(modal.getByTestId('picker-widget-card-wg_2x2_class_overview_matrix')).toBeVisible();

    // 4. Verify Text Search Input matching widget description / title / keywords
    const searchInput = modal.getByTestId('widget-picker-search-input');
    await searchInput.fill('Individual Healthcare Plans');
    await expect(modal.getByTestId('picker-widget-card-wg_1x2_sen_medical_watch_list')).toBeVisible();
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_attendance_counter')).toHaveCount(0);
    await expect(modal.getByTestId('picker-widget-card-wg_2x2_class_overview_matrix')).toHaveCount(0);
    await searchInput.fill('');
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_attendance_counter')).toBeVisible();

    // 5. Verify Plugin Filter dropdown
    const pluginFilter = modal.getByTestId('widget-picker-plugin-filter');
    await expect(modal.getByTestId('picker-widget-card-attendance_pulse')).toBeVisible();
    await pluginFilter.selectOption(galleryPlugId);
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_sen_badge')).toBeVisible();
    await expect(modal.getByTestId('picker-widget-card-attendance_pulse')).toHaveCount(0);
    await pluginFilter.selectOption('all');
    await expect(modal.getByTestId('picker-widget-card-attendance_pulse')).toBeVisible();

    // 6. Verify "Hide already placed" toggle filter
    const hidePlacedToggle = modal.getByTestId('widget-picker-hide-placed-toggle');
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_attendance_counter')).toBeVisible();
    await expect(modal.getByTestId('place-widget-btn-wg_1x1_attendance_counter')).toBeDisabled();
    await hidePlacedToggle.click();
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_attendance_counter')).toHaveCount(0);
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_sen_badge')).toBeVisible();
    await hidePlacedToggle.click();
    await expect(modal.getByTestId('picker-widget-card-wg_1x1_attendance_counter')).toBeVisible();

    // 7. Select a filtered widget (2x1 pulse bar) from the modal and place it onto the grid
    await modal.getByTestId('widget-picker-size-filter-2x1').click();
    await modal.getByTestId('place-widget-btn-wg_2x1_attendance_pulse_bar').click();
    await expect(modal).toHaveCount(0);

    const preview = page.getByTestId('dashboard-widget-placement-preview');
    await expect(preview).toBeVisible();
    await expect(preview).toHaveAttribute('data-preview-width', '2');
    await expect(preview).toHaveAttribute('data-preview-height', '1');
    await expect(preview).toHaveAttribute('data-preview-col', '2');
    await expect(preview).toHaveAttribute('data-preview-row', '1');
    await preview.click();
    await expect(page.getByTestId('placed-widget-card-wg_2x1_attendance_pulse_bar')).toBeVisible();
  });

  test('widgetPickerModal_galleryAndSplitPreviewModesInLargeModal_switchesBetweenCardGalleryAndTwoColumnLiveWidgetPreview_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: m1Seed.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);

    const clearLayoutRes = await request.put('/api/dashboard/layout', {
      data: {
        staff_id: m1Seed.teacher_staff_id,
        column_count: 4,
        layout: [],
      },
    });
    expect(clearLayoutRes.ok()).toBe(true);

    await page.goto(
      `/?pupil_id=${encodeURIComponent(m1Seed.active_pupil_id)}&second_pupil_id=${encodeURIComponent(
        m1Seed.second_active_pupil_id
      )}&group_id=${encodeURIComponent(m1Seed.own_group_id)}&reference_date=${encodeURIComponent(
        m1Seed.reference_date
      )}&academic_year_id=${encodeURIComponent(m1Seed.academic_year_id)}`
    );

    // 1. Open "+ Add Widget" modal and verify enlarged dimensions & default Gallery View mode
    await page.getByTestId('dashboard-edit-mode-toggle-btn').click();
    await page.getByTestId('dashboard-add-widget-btn').click();

    const modal = page.getByTestId('dashboard-widget-picker-modal');
    await expect(modal).toBeVisible();
    await expect(modal).toHaveAttribute('data-modal-view-mode', 'gallery');

    const modalStyles = await modal.evaluate((el) => {
      const cs = window.getComputedStyle(el);
      return {
        maxWidth: cs.maxWidth,
        maxHeight: cs.maxHeight,
      };
    });
    expect(modalStyles.maxWidth).toBe('1240px');
    expect(modalStyles.maxHeight).toBe('880px');

    const galleryBtn = modal.getByTestId('widget-picker-view-mode-gallery');
    const splitBtn = modal.getByTestId('widget-picker-view-mode-split');
    await expect(galleryBtn).toBeVisible();
    await expect(splitBtn).toBeVisible();
    await expect(galleryBtn).toHaveAttribute('aria-pressed', 'true');
    await expect(splitBtn).toHaveAttribute('aria-pressed', 'false');
    await expect(modal.getByTestId('widget-picker-gallery-view')).toBeVisible();
    await expect(modal.getByTestId('widget-picker-split-view')).toHaveCount(0);

    // 2. Switch to Split Preview View (Left List + Right Live Preview)
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

    // 3. Select a 1x1 widget in the left list and verify left list footprint diagram + right preview pane + live iframe (300x180)
    const item1x1 = splitList.getByTestId('split-widget-list-item-wg_1x1_attendance_counter');
    await item1x1.click();
    await expect(item1x1).toHaveAttribute('aria-selected', 'true');
    await expect(item1x1.getByTestId('split-list-footprint-diagram-wg_1x1_attendance_counter')).toHaveAttribute(
      'data-footprint-width',
      '1'
    );
    await expect(item1x1.getByTestId('split-list-footprint-diagram-wg_1x1_attendance_counter')).toHaveAttribute(
      'data-footprint-height',
      '1'
    );
    await expect(splitPreview.getByTestId('split-preview-widget-title')).toHaveText('Session Attendance Counter (1x1)');
    await expect(splitPreview.getByTestId('split-preview-widget-plugin')).toHaveText('Widget Gallery & Layout Bench');
    await expect(splitPreview.getByTestId('split-preview-size-badge')).toHaveCount(0);
    await expect(splitPreview.getByTestId('split-preview-widget-description')).toHaveText(
      EXPECTED_WIDGETS[0].description
    );
    await expect(splitPreview.getByTestId('split-preview-footprint-diagram')).toHaveAttribute(
      'data-footprint-width',
      '1'
    );
    await expect(splitPreview.getByTestId('split-preview-footprint-diagram')).toHaveAttribute(
      'data-footprint-height',
      '1'
    );

    const liveIframe = splitPreview.getByTestId('widget-picker-live-preview-iframe');
    await expect(liveIframe).toBeVisible();
    await expect(liveIframe).toHaveAttribute('data-widget-id', 'wg_1x1_attendance_counter');
    await expect(liveIframe).toHaveAttribute('data-preview-width-px', '300');
    await expect(liveIframe).toHaveAttribute('data-preview-height-px', '180');

    const previewFrame1x1 = page.frameLocator('[data-testid="widget-picker-live-preview-iframe"]');
    await expect(previewFrame1x1.getByTestId('wg-footprint-badge')).toHaveText('1x1');
    await expect(previewFrame1x1.getByTestId('wg-rpc-status')).toContainText('RPC_OK');

    // 4. Select a 2x2 widget in the left list and verify right preview updates to 612x372 live iframe
    const item2x2 = splitList.getByTestId('split-widget-list-item-wg_2x2_class_overview_matrix');
    await item2x2.click();
    await expect(item2x2).toHaveAttribute('aria-selected', 'true');
    await expect(item1x1).toHaveAttribute('aria-selected', 'false');
    await expect(item2x2.getByTestId('split-list-footprint-diagram-wg_2x2_class_overview_matrix')).toHaveAttribute(
      'data-footprint-width',
      '2'
    );
    await expect(item2x2.getByTestId('split-list-footprint-diagram-wg_2x2_class_overview_matrix')).toHaveAttribute(
      'data-footprint-height',
      '2'
    );
    await expect(splitPreview.getByTestId('split-preview-widget-title')).toHaveText('Class Overview & RPC Matrix (2x2)');
    await expect(splitPreview.getByTestId('split-preview-size-badge')).toHaveCount(0);
    await expect(splitPreview.getByTestId('split-preview-footprint-diagram')).toHaveAttribute(
      'data-footprint-width',
      '2'
    );
    await expect(splitPreview.getByTestId('split-preview-footprint-diagram')).toHaveAttribute(
      'data-footprint-height',
      '2'
    );
    await expect(liveIframe).toHaveAttribute('data-widget-id', 'wg_2x2_class_overview_matrix');
    await expect(liveIframe).toHaveAttribute('data-preview-width-px', '612');
    await expect(liveIframe).toHaveAttribute('data-preview-height-px', '372');
    await expect(previewFrame1x1.getByTestId('wg-footprint-badge')).toHaveText('2x2');
    await expect(previewFrame1x1.getByTestId('wg-rpc-status')).toContainText('RPC_OK');

    // 5. Place the selected 2x2 widget directly from the Split Preview pane CTA button
    await splitPreview.getByTestId('place-widget-btn-wg_2x2_class_overview_matrix').click();
    await expect(modal).toHaveCount(0);

    const gridPreview = page.getByTestId('dashboard-widget-placement-preview');
    await expect(gridPreview).toBeVisible();
    await expect(gridPreview).toHaveAttribute('data-preview-width', '2');
    await expect(gridPreview).toHaveAttribute('data-preview-height', '2');
    await gridPreview.click();
    await expect(page.getByTestId('placed-widget-card-wg_2x2_class_overview_matrix')).toBeVisible();
  });
});

