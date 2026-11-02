import { expect, type Page, type Locator, type APIRequestContext } from '@playwright/test';
import { execFileSync } from 'child_process';
import * as path from 'path';

// =============================================================================
// Constants & Configuration
// =============================================================================

export const ROOT_DIR = process.cwd();
export const MIS_VERIFY_BIN = path.resolve(ROOT_DIR, 'bin/mis-verify');
export const MIS_TEST_BIN = path.resolve(ROOT_DIR, 'bin/mis-test');

export const MODAL_MAX_WIDTH = '1240px';
export const MODAL_MAX_HEIGHT = '880px';

// =============================================================================
// Interfaces & Types
// =============================================================================

export interface CommandResult {
  stdout: string;
  stderr: string;
  exitCode: number;
}

export interface Milestone1SeedResponse {
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

export interface PersonaSeedResponse {
  fixture_id: string;
  pupil_id: string;
  admin_staff_id: string;
  dsl_staff_id: string;
  hod_staff_id: string;
  teacher_staff_id: string;
  attendance_officer_staff_id: string;
  terminated_staff_id: string;
}

export interface WidgetDefinition {
  id: string;
  title: string;
  description: string;
  width: 1 | 2;
  height: 1 | 2;
  icon: string;
  entry: string;
}

export interface UiConfigWidget extends WidgetDefinition {
  plugin_title: string;
  keywords: string[];
}

export interface UiConfigPlugin {
  id: string;
  title: string;
  icon: string;
  widgets: UiConfigWidget[];
}

export interface UiConfigResponse {
  plugins: UiConfigPlugin[];
}

export interface SizeFilterOption {
  id: '1x1' | '2x1' | '1x2' | '2x2';
  w: string;
  h: string;
}

export interface GalleryTestEnvironment {
  personaSeed: PersonaSeedResponse;
  m1Seed: Milestone1SeedResponse;
  testPlugId: string;
  galleryPlugId: string;
}

// =============================================================================
// Fixtures & Widget Catalog
// =============================================================================

export const SIZE_FILTERS: SizeFilterOption[] = [
  { id: '1x1', w: '1', h: '1' },
  { id: '2x1', w: '2', h: '1' },
  { id: '1x2', w: '1', h: '2' },
  { id: '2x2', w: '2', h: '2' },
];

export const EXPECTED_WIDGETS: WidgetDefinition[] = [
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

// =============================================================================
// Helper Functions
// =============================================================================

export function runBinary(binPath: string, args: string[]): CommandResult {
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

export function buildDashboardUrl(seed: Milestone1SeedResponse): string {
  const params = new URLSearchParams({
    pupil_id: seed.active_pupil_id,
    second_pupil_id: seed.second_active_pupil_id,
    group_id: seed.own_group_id,
    reference_date: seed.reference_date,
    academic_year_id: seed.academic_year_id,
  });
  return `/?${params.toString()}`;
}

export async function switchPersona(request: APIRequestContext, staffId: string): Promise<void> {
  const res = await request.post('/api/auth/persona', {
    data: { staff_id: staffId },
  });
  if (!res.ok()) {
    throw new Error(`Failed to switch to persona with staff_id ${staffId}: ${res.status()}`);
  }
}

export async function setDashboardLayout(
  request: APIRequestContext,
  staffId: string,
  layout: Array<{ plug_id: string; widget_id: string; grid_col: number; grid_row: number }>,
  columnCount = 4
): Promise<void> {
  const res = await request.put('/api/dashboard/layout', {
    data: {
      staff_id: staffId,
      column_count: columnCount,
      layout,
    },
  });
  if (!res.ok()) {
    throw new Error(`Failed to update dashboard layout for staff ${staffId}: ${res.status()}`);
  }
}

export async function setupWidgetGalleryEnvironment(request: APIRequestContext): Promise<GalleryTestEnvironment> {
  const authModeRes = await request.post('/api/test/auth-mode', {
    data: { enable_dev_persona_switcher: true },
  });
  if (!authModeRes.ok()) throw new Error(`Failed to enable dev persona switcher: ${authModeRes.status()}`);

  const pSeedRes = await request.post('/api/test/seed-personas', {
    data: { academic_year_id: 'ay_2026_2027', effective_from: '2026-09-01' },
  });
  if (!pSeedRes.ok()) throw new Error(`Failed to seed persona fixtures: ${pSeedRes.status()}`);
  const personaSeed = (await pSeedRes.json()) as PersonaSeedResponse;

  const m1Res = await request.post('/api/test/seed-milestone1-fixtures', {
    data: { academic_year_id: 'ay_2026_2027', reference_date: '2026-10-27', include_second_active_pupil: true },
  });
  if (!m1Res.ok()) throw new Error(`Failed to seed Milestone 1 fixtures: ${m1Res.status()}`);
  const m1Seed = (await m1Res.json()) as Milestone1SeedResponse;

  await switchPersona(request, personaSeed.admin_staff_id);

  const installTestRes = await request.post('/api/admin/plugins/install', {
    data: { folder: 'test_plugin', staff_id: personaSeed.admin_staff_id },
  });
  if (!installTestRes.ok()) throw new Error(`Failed to install test_plugin: ${installTestRes.status()}`);
  const testPlugId = (await installTestRes.json()).plug_id;
  await request.post(`/api/admin/plugins/${encodeURIComponent(testPlugId)}/activate`, {
    data: { staff_id: personaSeed.admin_staff_id },
  });

  const installGalleryRes = await request.post('/api/admin/plugins/install', {
    data: { folder: 'widget_gallery_plugin', staff_id: personaSeed.admin_staff_id },
  });
  if (!installGalleryRes.ok()) throw new Error(`Failed to install widget_gallery_plugin: ${installGalleryRes.status()}`);
  const galleryPlugId = (await installGalleryRes.json()).plug_id;
  await request.post(`/api/admin/plugins/${encodeURIComponent(galleryPlugId)}/activate`, {
    data: { staff_id: personaSeed.admin_staff_id },
  });

  return { personaSeed, m1Seed, testPlugId, galleryPlugId };
}

export async function cleanupWidgetGalleryEnvironment(
  request: APIRequestContext,
  env: {
    personaSeed: PersonaSeedResponse | null;
    m1Seed: Milestone1SeedResponse | null;
    galleryPlugId: string;
    testPlugId: string;
  }
): Promise<void> {
  if (env.personaSeed) {
    try {
      await switchPersona(request, env.personaSeed.admin_staff_id);
    } catch {
      // Continue cleanup
    }
    for (const pid of [env.galleryPlugId, env.testPlugId]) {
      if (pid) {
        try {
          await request.delete(`/api/admin/plugins/${encodeURIComponent(pid)}`);
        } catch {
          // Best effort
        }
      }
    }
  }
  if (env.m1Seed) {
    try {
      await request.delete(`/api/test/seed-milestone1-fixtures/${encodeURIComponent(env.m1Seed.fixture_id)}`);
    } catch {
      // Best effort
    }
  }
  if (env.personaSeed) {
    try {
      await request.delete(`/api/test/seed-personas/${encodeURIComponent(env.personaSeed.fixture_id)}`);
    } catch {
      // Best effort
    }
  }
}

export async function placeWidgetFromPicker(
  page: Page,
  widgetId: string,
  width: number,
  height: number,
  col: number,
  row: number,
  openPicker = false
): Promise<void> {
  if (openPicker) {
    await page.getByTestId('dashboard-add-widget-btn').click();
    await expect(page.getByTestId('dashboard-widget-picker-modal')).toBeVisible();
  }

  const pickerModal = page.getByTestId('dashboard-widget-picker-modal');
  await pickerModal.getByTestId(`place-widget-btn-${widgetId}`).click();
  await expect(pickerModal).toHaveCount(0);

  const preview = page.getByTestId('dashboard-widget-placement-preview');
  await expect(preview).toBeVisible();
  await expect(preview).toHaveAttribute('data-preview-width', String(width));
  await expect(preview).toHaveAttribute('data-preview-height', String(height));
  await expect(preview).toHaveAttribute('data-preview-col', String(col));
  await expect(preview).toHaveAttribute('data-preview-row', String(row));
  await preview.click();
  await expect(preview).toHaveCount(0);

  const placedCard = page.getByTestId(`placed-widget-card-${widgetId}`);
  await expect(placedCard).toHaveAttribute('data-grid-col', String(col));
  await expect(placedCard).toHaveAttribute('data-grid-row', String(row));
  await expect(placedCard).toHaveAttribute('data-width', String(width));
  await expect(placedCard).toHaveAttribute('data-height', String(height));
}

export async function expectFootprintDiagram(
  locator: Locator,
  width: number | string,
  height: number | string
): Promise<void> {
  await expect(locator).toBeVisible();
  await expect(locator).toHaveAttribute('data-footprint-width', String(width));
  await expect(locator).toHaveAttribute('data-footprint-height', String(height));
}

export async function expectWidgetIframeHydrated(
  page: Page,
  widgetId: string,
  expectedBadge: string,
  expectedRpcStatus = 'RPC_OK'
): Promise<void> {
  const frame = page.frameLocator(`[data-testid="widget-iframe-${widgetId}"]`);
  await expect(frame.getByTestId('wg-footprint-badge')).toHaveText(expectedBadge);
  await expect(frame.getByTestId('wg-rpc-status')).toContainText(expectedRpcStatus);
}

export async function expectSplitPreviewSelection(
  page: Page,
  splitList: Locator,
  splitPreview: Locator,
  widget: WidgetDefinition,
  pxWidth: number,
  pxHeight: number
): Promise<void> {
  const item = splitList.getByTestId(`split-widget-list-item-${widget.id}`);
  await item.click();
  await expect(item).toHaveAttribute('aria-selected', 'true');
  await expectFootprintDiagram(
    item.getByTestId(`split-list-footprint-diagram-${widget.id}`),
    widget.width,
    widget.height
  );

  await expect(splitPreview.getByTestId('split-preview-widget-title')).toHaveText(widget.title);
  await expect(splitPreview.getByTestId('split-preview-widget-plugin')).toHaveText('Widget Gallery & Layout Bench');
  await expect(splitPreview.getByTestId('split-preview-size-badge')).toHaveCount(0);
  await expect(splitPreview.getByTestId('split-preview-widget-description')).toHaveText(widget.description);
  await expectFootprintDiagram(splitPreview.getByTestId('split-preview-footprint-diagram'), widget.width, widget.height);

  const liveIframe = splitPreview.getByTestId('widget-picker-live-preview-iframe');
  await expect(liveIframe).toBeVisible();
  await expect(liveIframe).toHaveAttribute('data-widget-id', widget.id);
  await expect(liveIframe).toHaveAttribute('data-preview-width-px', String(pxWidth));
  await expect(liveIframe).toHaveAttribute('data-preview-height-px', String(pxHeight));

  const previewFrame = page.frameLocator('[data-testid="widget-picker-live-preview-iframe"]');
  await expect(previewFrame.getByTestId('wg-footprint-badge')).toHaveText(`${widget.width}x${widget.height}`);
  await expect(previewFrame.getByTestId('wg-rpc-status')).toContainText('RPC_OK');
}
