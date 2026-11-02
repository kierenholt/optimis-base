import { expect, type Page, type FrameLocator, type APIRequestContext, type TestInfo } from '@playwright/test';

// =============================================================================
// Interfaces & Types
// =============================================================================

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

export interface HostSettingsTestState {
  browserLogs: string[];
  pageErrors: string[];
  logCursor: string;
  m1Seed: Milestone1SeedResponse | null;
  personaSeed: PersonaSeedResponse | null;
  testPlugId: string;
}

export interface RootFontMetrics {
  attr: string | null;
  tokenBase: string;
  tokenScale?: string;
  computedRootFontSize: string;
}

export interface SidebarGeometry {
  windowInnerHeight: number;
  asideTop: number;
  asideBottom: number;
  asideHeight: number;
  asideOffsetHeight: number;
  asideScrollHeight: number;
  settingsBottom: number;
  logoutBottom: number;
}

export interface HostSettingsAndSidebarFontMetrics {
  headingPx: number;
  controlPx: number;
  btnLargePx: number;
  schoolNamePx: number;
  navHomePx: number;
  navSettingsPx: number;
}

export interface PluginFontMetrics {
  btnPx: number;
  sdkBarPx: number;
}

// =============================================================================
// Environment Setup & Teardown
// =============================================================================

export async function setupHostSettingsTestEnv(
  page: Page,
  request: APIRequestContext
): Promise<HostSettingsTestState> {
  const browserLogs: string[] = [];
  const pageErrors: string[] = [];

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
  const logCursor = (await cursorRes.json()).cursor;

  // 3. Ensure dev persona switcher is enabled
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
  const personaSeed = (await pSeedRes.json()) as PersonaSeedResponse;

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
  const m1Seed = (await m1Res.json()) as Milestone1SeedResponse;

  // 5. Switch to Admin persona and install + activate test_plugin
  const switchAdminRes = await request.post('/api/auth/persona', {
    data: { staff_id: personaSeed.admin_staff_id },
  });
  if (!switchAdminRes.ok()) {
    throw new Error(`Failed to switch to Admin persona in setup: ${switchAdminRes.status()}`);
  }

  const installTestRes = await request.post('/api/admin/plugins/install', {
    data: { folder: 'test_plugin', staff_id: personaSeed.admin_staff_id },
  });
  if (!installTestRes.ok()) {
    throw new Error(`Failed to install test_plugin: ${installTestRes.status()}`);
  }
  const testPlugId = (await installTestRes.json()).plug_id;
  await request.post(`/api/admin/plugins/${encodeURIComponent(testPlugId)}/activate`, {
    data: { staff_id: personaSeed.admin_staff_id },
  });

  return {
    browserLogs,
    pageErrors,
    logCursor,
    m1Seed,
    personaSeed,
    testPlugId,
  };
}

export async function teardownHostSettingsTestEnv(
  request: APIRequestContext,
  state: HostSettingsTestState,
  testInfo: TestInfo
): Promise<void> {
  // 1. Fetch Backend Service Logs emitted during this test
  if (state.logCursor) {
    const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(state.logCursor)}`);
    if (logsRes.ok()) {
      const serviceLogs: string[] = (await logsRes.json()).entries;
      if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
        console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
      }
    }
  }

  // 2. Dump tapped Browser Console logs on failure or error
  if (testInfo.status !== testInfo.expectedStatus || state.pageErrors.length > 0) {
    console.log('--- TAPPED BROWSER CONSOLE ---\n' + state.browserLogs.join('\n'));
    console.log('--- TAPPED PAGE ERRORS ---\n' + state.pageErrors.join('\n'));
  }

  // 3. Clean up user settings, dashboard layouts, and installed plugins
  await request.delete('/api/test/dashboard-layouts');
  if (state.personaSeed) {
    await request.post('/api/auth/persona', {
      data: { staff_id: state.personaSeed.admin_staff_id },
    });
    await request.put('/api/system/user-settings', {
      data: {
        staff_id: state.personaSeed.admin_staff_id,
        base_font_size: 'medium',
        theme_mode: 'light',
      },
    });
    await request.post('/api/system/pinned-recents', {
      data: {
        action: 'clear',
        staff_id: state.personaSeed.admin_staff_id,
      },
    });
    if (state.m1Seed) {
      await request.put('/api/system/user-settings', {
        data: {
          staff_id: state.m1Seed.teacher_staff_id,
          base_font_size: 'medium',
          theme_mode: 'light',
        },
      });
      await request.post('/api/system/pinned-recents', {
        data: {
          action: 'clear',
          staff_id: state.m1Seed.teacher_staff_id,
        },
      });
    }
    if (state.testPlugId) {
      await request.delete(`/api/admin/plugins/${encodeURIComponent(state.testPlugId)}`);
    }
  }
  if (state.m1Seed) {
    await request.delete(`/api/test/seed-milestone1-fixtures/${encodeURIComponent(state.m1Seed.fixture_id)}`);
    state.m1Seed = null;
  }
  if (state.personaSeed) {
    await request.delete(`/api/test/seed-personas/${encodeURIComponent(state.personaSeed.fixture_id)}`);
    state.personaSeed = null;
  }

  expect(state.pageErrors).toEqual([]);
}

// =============================================================================
// Session & Navigation Helpers
// =============================================================================

export async function authenticateTeacherSession(
  page: Page,
  request: APIRequestContext,
  teacherStaffId: string
): Promise<string> {
  const switchTeacherRes = await request.post('/api/auth/persona', {
    data: { staff_id: teacherStaffId },
  });
  if (!switchTeacherRes.ok()) {
    throw new Error(`Failed to switch to Teacher persona: ${switchTeacherRes.status()}`);
  }
  const teacherSession = await switchTeacherRes.json();

  await page.context().addCookies([
    {
      name: 'mis_session',
      value: teacherSession.session_token,
      url: 'http://127.0.0.1:18080',
      sameSite: 'Strict',
    },
  ]);
  return teacherSession.session_token;
}

export async function mountTeacherWorkspaceWithWidget(
  page: Page,
  request: APIRequestContext,
  m1Seed: Milestone1SeedResponse,
  testPlugId: string
): Promise<{
  pluginFrame: FrameLocator;
  widgetFrame: FrameLocator;
  widgetCompositeKey: string;
}> {
  await authenticateTeacherSession(page, request, m1Seed.teacher_staff_id);

  const placeWidgetRes = await request.put('/api/dashboard/layout', {
    data: {
      staff_id: m1Seed.teacher_staff_id,
      layout: [{ plug_id: testPlugId, widget_id: 'pastoral_watch' }],
    },
  });
  if (!placeWidgetRes.ok()) {
    throw new Error(`Failed to place widget layout: ${placeWidgetRes.status()}`);
  }

  await page.goto(
    `/p/${testPlugId}?pupil_id=${encodeURIComponent(
      m1Seed.active_pupil_id
    )}&group_id=${encodeURIComponent(m1Seed.own_group_id)}`
  );

  const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
  const widgetFrame = page.frameLocator('[data-testid="widget-iframe-pastoral_watch"]');
  const widgetCompositeKey = `${testPlugId}:pastoral_watch`;

  return { pluginFrame, widgetFrame, widgetCompositeKey };
}

// =============================================================================
// Font & Geometry Measurement Helpers
// =============================================================================

export async function measureHostRootFontMetrics(page: Page): Promise<RootFontMetrics> {
  return await page.evaluate(() => {
    const root = document.documentElement;
    const cs = window.getComputedStyle(root);
    return {
      attr: root.getAttribute('data-mis-font-size'),
      tokenBase: root.style.getPropertyValue('--mis-font-size-base').trim(),
      tokenScale: root.style.getPropertyValue('--mis-font-scale').trim(),
      computedRootFontSize: cs.fontSize,
    };
  });
}

export async function measureFrameRootFontMetrics(frame: FrameLocator): Promise<RootFontMetrics> {
  return await frame.locator(':root').evaluate((root) => {
    const cs = window.getComputedStyle(root);
    return {
      attr: root.getAttribute('data-mis-font-size'),
      tokenBase: (root as HTMLElement).style.getPropertyValue('--mis-font-size-base').trim(),
      tokenScale: (root as HTMLElement).style.getPropertyValue('--mis-font-scale').trim(),
      computedRootFontSize: cs.fontSize,
    };
  });
}

export async function measureHostSidebarGeometry(page: Page): Promise<SidebarGeometry> {
  return await page.evaluate(() => {
    const aside = document.querySelector('[data-testid="host-sidebar"]') as HTMLElement | null;
    const logoutBtn = document.querySelector('[data-testid="nav-rail-logout"]') as HTMLElement | null;
    const settingsBtn = document.querySelector('[data-testid="nav-rail-settings"]') as HTMLElement | null;
    if (!aside || !logoutBtn || !settingsBtn) {
      throw new Error('Missing sidebar or bottom rail action elements');
    }
    const asideRect = aside.getBoundingClientRect();
    const logoutRect = logoutBtn.getBoundingClientRect();
    const settingsRect = settingsBtn.getBoundingClientRect();
    return {
      windowInnerHeight: window.innerHeight,
      asideTop: Math.round(asideRect.top),
      asideBottom: Math.round(asideRect.bottom),
      asideHeight: Math.round(asideRect.height),
      asideOffsetHeight: aside.offsetHeight,
      asideScrollHeight: aside.scrollHeight,
      settingsBottom: Math.round(settingsRect.bottom),
      logoutBottom: Math.round(logoutRect.bottom),
    };
  });
}

export async function measureHostSettingsAndSidebarFonts(page: Page): Promise<HostSettingsAndSidebarFontMetrics> {
  return await page.evaluate(() => {
    const headingEl = document.querySelector('[data-testid="host-settings-view"] h2');
    const controlEl = document.querySelector('[data-testid="host-settings-font-size-control"]');
    const btnLargeEl = document.querySelector('[data-testid="host-settings-font-size-btn-large"]');
    const schoolNameEl = document.querySelector('[data-testid="host-school-name"]');
    const navHomeEl = document.querySelector('[data-testid="nav-rail-home"]');
    const navSettingsEl = document.querySelector('[data-testid="nav-rail-settings"]');
    return {
      headingPx: headingEl ? parseFloat(window.getComputedStyle(headingEl).fontSize) : 0,
      controlPx: controlEl ? parseFloat(window.getComputedStyle(controlEl).fontSize) : 0,
      btnLargePx: btnLargeEl ? parseFloat(window.getComputedStyle(btnLargeEl).fontSize) : 0,
      schoolNamePx: schoolNameEl ? parseFloat(window.getComputedStyle(schoolNameEl).fontSize) : 0,
      navHomePx: navHomeEl ? parseFloat(window.getComputedStyle(navHomeEl).fontSize) : 0,
      navSettingsPx: navSettingsEl ? parseFloat(window.getComputedStyle(navSettingsEl).fontSize) : 0,
    };
  });
}

export async function measurePluginFonts(pluginFrame: FrameLocator): Promise<PluginFontMetrics> {
  return await pluginFrame.locator(':root').evaluate(() => {
    const btnEl = document.querySelector('[data-testid="print-record-btn"]');
    const sdkBarEl = document.querySelector('.sdk-bar');
    return {
      btnPx: btnEl ? parseFloat(window.getComputedStyle(btnEl).fontSize) : 0,
      sdkBarPx: sdkBarEl ? parseFloat(window.getComputedStyle(sdkBarEl).fontSize) : 0,
    };
  });
}
