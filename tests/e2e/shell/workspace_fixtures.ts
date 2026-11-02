import { expect, type APIRequestContext, type Locator, type Page } from '@playwright/test';

// =============================================================================
// Interfaces & Types
// =============================================================================

export interface WorkspaceEnvironment {
  seededPupilId: string;
  secondPupilId: string;
  testPlugId: string;
  lruPlugIds: string[];
}

export interface PersonaSeedResult {
  fixture_id: string;
  teacher_staff_id: string;
  admin_staff_id: string;
  dsl_staff_id: string;
  hod_staff_id: string;
  terminated_staff_id: string;
}

export interface WidgetGridPlacementAttrs {
  width?: string;
  height?: string;
  colSpan?: string;
  rowSpan?: string;
  gridCol?: string;
  gridRow?: string;
}

export const SYNTHETIC_PLUGIN_IDS = [
  'uk.org.optimis.lru1',
  'uk.org.optimis.lru2',
  'uk.org.optimis.lru3',
  'uk.org.optimis.lru4',
  'uk.org.optimis.lru5',
  'uk.org.optimis.lru6',
];

// =============================================================================
// Lifecycle Setup & Teardown Helpers
// =============================================================================

export async function setupWorkspaceEnvironment(
  request: APIRequestContext,
  syntheticPluginIds: string[] = SYNTHETIC_PLUGIN_IDS
): Promise<WorkspaceEnvironment> {
  // 1. Ensure dev persona switcher is enabled
  await request.post('/api/test/auth-mode', {
    data: { enable_dev_persona_switcher: true },
  });

  // 2. Install and activate test_plugin
  const installRes = await request.post('/api/admin/plugins/install', {
    data: { folder: 'test_plugin' },
  });
  if (!installRes.ok()) {
    throw new Error(`Failed to install test_plugin: ${installRes.status()}`);
  }
  const installBody = await installRes.json();
  const testPlugId = installBody.plug_id;

  const activateRes = await request.post(`/api/admin/plugins/${encodeURIComponent(testPlugId)}/activate`, {
    data: {},
  });
  if (!activateRes.ok()) {
    throw new Error(`Failed to activate test_plugin (${testPlugId}): ${activateRes.status()}`);
  }

  // 3. Seed two isolated test pupils
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

  // 4. Seed synthetic active plugins for LRU pool
  const seedLruRes = await request.post('/api/test/seed-lru-plugins', {
    data: {
      plugin_ids: syntheticPluginIds,
    },
  });
  if (!seedLruRes.ok()) {
    throw new Error(`Failed to seed synthetic LRU plugins: ${seedLruRes.status()}`);
  }
  const seedLruBody = await seedLruRes.json();

  return {
    seededPupilId: body1.pupil_id,
    secondPupilId: body2.pupil_id,
    testPlugId,
    lruPlugIds: seedLruBody.seeded_plugin_ids,
  };
}

export async function cleanupWorkspaceEnvironment(
  request: APIRequestContext,
  env: {
    testPlugId?: string;
    seededPupilId?: string;
    secondPupilId?: string;
  }
): Promise<void> {
  await request.delete('/api/test/dashboard-layouts');
  await request.delete('/api/test/seed-lru-plugins');
  await request.post('/api/test/auth-mode', {
    data: { enable_dev_persona_switcher: true },
  });

  if (env.testPlugId) {
    await request.delete(`/api/admin/plugins/${encodeURIComponent(env.testPlugId)}`);
  }
  if (env.seededPupilId) {
    await request.delete(`/api/test/seed-pupil/${encodeURIComponent(env.seededPupilId)}`);
  }
  if (env.secondPupilId) {
    await request.delete(`/api/test/seed-pupil/${encodeURIComponent(env.secondPupilId)}`);
  }
}

// =============================================================================
// Common Test Actions & Assertions
// =============================================================================

export async function selectPluginViaRailSearch(page: Page, pid: string): Promise<void> {
  await page.getByTestId('nav-rail-search').click();
  await page.getByTestId(`host-search-result-view-${pid}-home`).click();
}

export async function verifyManifestWidgetsInUiConfig(
  request: APIRequestContext,
  testPlugId: string
): Promise<void> {
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
}

export async function expectWidgetGridStyles(grid: Locator, columnCount: number): Promise<void> {
  await expect(grid).toBeVisible();
  await expect(grid).toHaveAttribute('data-column-count', String(columnCount));
  const gridStyles = await grid.evaluate((el) => {
    const s = (el as HTMLElement).style;
    return {
      gridAutoFlow: s.gridAutoFlow,
      gridTemplateColumns: s.gridTemplateColumns,
      gridAutoRows: s.gridAutoRows,
    };
  });
  expect(gridStyles.gridAutoFlow).toBe('row');
  expect(gridStyles.gridTemplateColumns).toBe(`repeat(${columnCount}, 300px)`);
  expect(gridStyles.gridAutoRows).toBe('180px');
}

export async function expectPlacedWidgetCard(
  card: Locator,
  attrs: WidgetGridPlacementAttrs
): Promise<void> {
  await expect(card).toBeVisible();
  if (attrs.width) await expect(card).toHaveAttribute('data-width', attrs.width);
  if (attrs.height) await expect(card).toHaveAttribute('data-height', attrs.height);
  if (attrs.colSpan) await expect(card).toHaveAttribute('data-col-span', attrs.colSpan);
  if (attrs.rowSpan) await expect(card).toHaveAttribute('data-row-span', attrs.rowSpan);
  if (attrs.gridCol) await expect(card).toHaveAttribute('data-grid-col', attrs.gridCol);
  if (attrs.gridRow) await expect(card).toHaveAttribute('data-grid-row', attrs.gridRow);
}

export async function expectWidgetPlacementStyles(
  card: Locator,
  expectedCol: string,
  expectedRow: string
): Promise<void> {
  const styles = await card.evaluate((el) => {
    const s = (el as HTMLElement).style;
    return { gridColumn: s.gridColumn, gridRow: s.gridRow };
  });
  expect(styles.gridColumn).toBe(expectedCol);
  expect(styles.gridRow).toBe(expectedRow);
}

export async function expectWidgetCoordinates(card: Locator, col: string, row: string): Promise<void> {
  await expect(card).toHaveAttribute('data-grid-col', col);
  await expect(card).toHaveAttribute('data-grid-row', row);
}

export async function placeWidgetViaPickerAndGhost(
  page: Page,
  widgetId: string,
  options: {
    initialPreviewCol: string;
    initialPreviewRow: string;
    hoverCell: Locator;
    targetPreviewCol: string;
    targetPreviewRow: string;
  }
): Promise<void> {
  await page.getByTestId('dashboard-add-widget-btn').click();
  const pickerModal = page.getByTestId('dashboard-widget-picker-modal');
  await expect(pickerModal).toBeVisible();

  await pickerModal.getByTestId(`place-widget-btn-${widgetId}`).click();
  await expect(pickerModal).toHaveCount(0);

  const placementPreview = page.getByTestId('dashboard-widget-placement-preview');
  await expect(placementPreview).toBeVisible();
  await expect(placementPreview).toHaveAttribute('data-preview-col', options.initialPreviewCol);
  await expect(placementPreview).toHaveAttribute('data-preview-row', options.initialPreviewRow);

  await options.hoverCell.hover();
  await expect(placementPreview).toHaveAttribute('data-preview-col', options.targetPreviewCol);
  await expect(placementPreview).toHaveAttribute('data-preview-row', options.targetPreviewRow);

  await placementPreview.click();
  await expect(placementPreview).toHaveCount(0);
}

export async function seedTestPersonas(request: APIRequestContext): Promise<PersonaSeedResult> {
  const personaRes = await request.post('/api/test/seed-personas', {
    data: {
      academic_year_id: 'ay_2026_2027',
      effective_from: '2026-09-01',
    },
  });
  if (!personaRes.ok()) {
    throw new Error(`Failed to seed persona fixtures: ${personaRes.status()}`);
  }
  return await personaRes.json();
}

export async function expectDashboardLayout(
  request: APIRequestContext,
  expected: {
    layout_source: string;
    column_count: number;
    layout: Array<{ plug_id: string; widget_id: string; col: number; row: number }>;
  }
): Promise<void> {
  const res = await request.get('/api/dashboard/layout');
  expect(res.ok()).toBe(true);
  const body = await res.json();
  expect(body.ok).toBe(true);
  expect(body.layout_source).toBe(expected.layout_source);
  expect(body.column_count).toBe(expected.column_count);
  expect(body.layout).toEqual(expected.layout);
}

export async function toggleThemeViaSettings(page: Page, expectedTheme: string): Promise<void> {
  await page.getByTestId('nav-rail-settings').click();
  await expect(page.getByTestId('host-settings-modal')).toHaveCount(0);
  await page.getByTestId('host-settings-view').getByTestId('host-settings-theme-toggle-btn').click();
  await page.getByTestId('nav-rail-home').click();
}
