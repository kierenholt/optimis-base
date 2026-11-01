import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

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

test.describe('Host Settings Modal: Base Font Size Toggle (Small/Medium/Large/Extra Large), Iframe Propagation & staff_settings Persistence', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let m1Seed: Milestone1SeedResponse | null = null;
  let personaSeed: PersonaSeedResponse | null = null;
  let testPlugId: string = '';

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    m1Seed = null;
    personaSeed = null;
    testPlugId = '';

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
    testPlugId = (await installTestRes.json()).plug_id;
    await request.post(`/api/admin/plugins/${encodeURIComponent(testPlugId)}/activate`, {
      data: { staff_id: personaSeed.admin_staff_id },
    });
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

    // 3. Clean up user settings, dashboard layouts, and installed plugins
    await request.delete('/api/test/dashboard-layouts');
    if (personaSeed) {
      await request.post('/api/auth/persona', {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      await request.put('/api/system/user-settings', {
        data: {
          staff_id: personaSeed.admin_staff_id,
          base_font_size: 'medium',
          theme_mode: 'light',
        },
      });
      await request.post('/api/system/pinned-recents', {
        data: {
          action: 'clear',
          staff_id: personaSeed.admin_staff_id,
        },
      });
      if (m1Seed) {
        await request.put('/api/system/user-settings', {
          data: {
            staff_id: m1Seed.teacher_staff_id,
            base_font_size: 'medium',
            theme_mode: 'light',
          },
        });
        await request.post('/api/system/pinned-recents', {
          data: {
            action: 'clear',
            staff_id: m1Seed.teacher_staff_id,
          },
        });
      }
      if (testPlugId) {
        await request.delete(`/api/admin/plugins/${encodeURIComponent(testPlugId)}`);
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

  test('hostSettingsBaseFontSize_selectingSmallMediumLargeExtraLarge_updatesHostAndPluginIframeFontSizeAndPersistsInStaffSettings_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // 1. Authenticate as Teacher and place a widget on the dashboard so both plugin and widget iframes are mounted
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: m1Seed.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);
    const teacherSession = await switchTeacherRes.json();

    await page.context().addCookies([
      {
        name: 'mis_session',
        value: teacherSession.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    const placeWidgetRes = await request.put('/api/dashboard/layout', {
      data: {
        staff_id: m1Seed.teacher_staff_id,
        layout: [{ plug_id: testPlugId, widget_id: 'pastoral_watch' }],
      },
    });
    expect(placeWidgetRes.ok()).toBe(true);

    await page.goto(
      `/p/${testPlugId}?pupil_id=${encodeURIComponent(
        m1Seed.active_pupil_id
      )}&group_id=${encodeURIComponent(m1Seed.own_group_id)}`
    );

    const hostShell = page.getByTestId('host-shell');
    await expect(hostShell).toBeVisible();
    await expect(hostShell).toHaveAttribute('data-mis-font-size', 'medium');

    const pluginIframeEl = page.getByTestId(`plugin-iframe-${testPlugId}`);
    await expect(pluginIframeEl).toBeVisible();
    const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
    await expect(pluginFrame.locator('body')).toBeVisible();

    const widgetCompositeKey = `${testPlugId}:pastoral_watch`;

    // 2. Verify initial default medium (16px) state on Host root and plugin iframe root
    const initialHostMetrics = await page.evaluate(() => {
      const root = document.documentElement;
      const cs = window.getComputedStyle(root);
      const schoolNameEl = document.querySelector('[data-testid="host-school-name"]');
      const schoolNameFontSize = schoolNameEl ? parseFloat(window.getComputedStyle(schoolNameEl).fontSize) : 0;
      return {
        attr: root.getAttribute('data-mis-font-size'),
        tokenBase: root.style.getPropertyValue('--mis-font-size-base').trim(),
        tokenScale: root.style.getPropertyValue('--mis-font-scale').trim(),
        computedRootFontSize: cs.fontSize,
        schoolNameFontSize,
      };
    });
    expect(initialHostMetrics.attr).toBe('medium');
    expect(initialHostMetrics.tokenBase).toBe('16px');
    expect(initialHostMetrics.tokenScale).toBe('1');
    expect(initialHostMetrics.computedRootFontSize).toBe('16px');

    const initialPluginMetrics = await pluginFrame.locator(':root').evaluate((root) => {
      const cs = window.getComputedStyle(root);
      const bodyCs = window.getComputedStyle(document.body);
      return {
        attr: root.getAttribute('data-mis-font-size'),
        tokenBase: (root as HTMLElement).style.getPropertyValue('--mis-font-size-base').trim(),
        tokenScale: (root as HTMLElement).style.getPropertyValue('--mis-font-scale').trim(),
        computedRootFontSize: cs.fontSize,
        computedBodyFontSize: bodyCs.fontSize,
      };
    });
    expect(initialPluginMetrics.attr).toBe('medium');
    expect(initialPluginMetrics.tokenBase).toBe('16px');
    expect(initialPluginMetrics.tokenScale).toBe('1');
    expect(initialPluginMetrics.computedRootFontSize).toBe('16px');

    // Verify Home view renders the placed widget iframe
    await page.getByTestId('nav-rail-home').click();
    const widgetIframeEl = page.getByTestId('widget-iframe-pastoral_watch');
    await expect(widgetIframeEl).toBeVisible();
    await expect(widgetIframeEl).toHaveAttribute('data-widget-key', widgetCompositeKey);
    const widgetFrame = page.frameLocator('[data-testid="widget-iframe-pastoral_watch"]');
    await expect(widgetFrame.locator('body')).toBeVisible();

    // 3. Click Settings in the sidebar -> switches right-hand pane to Host Settings View (/settings), NOT a modal
    await page.getByTestId('nav-rail-settings').click();
    expect(new URL(page.url()).pathname).toBe('/settings');
    await expect(page.getByTestId('host-settings-modal')).toHaveCount(0);
    const settingsView = page.getByTestId('host-settings-view');
    await expect(settingsView).toBeVisible();

    const fontSizeControl = settingsView.getByTestId('host-settings-font-size-control');
    await expect(fontSizeControl).toBeVisible();

    const btnSmall = settingsView.getByTestId('host-settings-font-size-btn-small');
    const btnMedium = settingsView.getByTestId('host-settings-font-size-btn-medium');
    const btnLarge = settingsView.getByTestId('host-settings-font-size-btn-large');
    const btnExtraLarge = settingsView.getByTestId('host-settings-font-size-btn-extra-large');

    await expect(btnSmall).toBeVisible();
    await expect(btnMedium).toBeVisible();
    await expect(btnLarge).toBeVisible();
    await expect(btnExtraLarge).toBeVisible();

    await expect(btnMedium).toHaveAttribute('aria-pressed', 'true');
    await expect(btnSmall).toHaveAttribute('aria-pressed', 'false');
    await expect(btnLarge).toHaveAttribute('aria-pressed', 'false');
    await expect(btnExtraLarge).toHaveAttribute('aria-pressed', 'false');

    // 4. Click "Large" (18px) -> verify Host Shell and kept-alive plugin iframe update live
    await btnLarge.click();
    await expect(btnLarge).toHaveAttribute('aria-pressed', 'true');
    await expect(btnMedium).toHaveAttribute('aria-pressed', 'false');
    await expect(hostShell).toHaveAttribute('data-mis-font-size', 'large');

    const largeHostMetrics = await page.evaluate(() => {
      const root = document.documentElement;
      const cs = window.getComputedStyle(root);
      return {
        attr: root.getAttribute('data-mis-font-size'),
        tokenBase: root.style.getPropertyValue('--mis-font-size-base').trim(),
        tokenScale: root.style.getPropertyValue('--mis-font-scale').trim(),
        computedRootFontSize: cs.fontSize,
      };
    });
    expect(largeHostMetrics.attr).toBe('large');
    expect(largeHostMetrics.tokenBase).toBe('18px');
    expect(largeHostMetrics.tokenScale).toBe('1.125');
    expect(largeHostMetrics.computedRootFontSize).toBe('18px');

    await expect
      .poll(async () => {
        return await pluginFrame.locator(':root').evaluate((root) => ({
          attr: root.getAttribute('data-mis-font-size'),
          tokenBase: (root as HTMLElement).style.getPropertyValue('--mis-font-size-base').trim(),
          computedRootFontSize: window.getComputedStyle(root).fontSize,
        }));
      })
      .toEqual({
        attr: 'large',
        tokenBase: '18px',
        computedRootFontSize: '18px',
      });

    // Switch to Home view to verify widget iframe also reflects Large (18px)
    await page.getByTestId('nav-rail-home').click();
    await expect
      .poll(async () => {
        return await widgetFrame.locator(':root').evaluate((root) => ({
          attr: root.getAttribute('data-mis-font-size'),
          tokenBase: (root as HTMLElement).style.getPropertyValue('--mis-font-size-base').trim(),
          computedRootFontSize: window.getComputedStyle(root).fontSize,
        }));
      })
      .toEqual({
        attr: 'large',
        tokenBase: '18px',
        computedRootFontSize: '18px',
      });

    // Switch back to Settings view
    await page.getByTestId('nav-rail-settings').click();
    await expect(settingsView).toBeVisible();

    // 5. Click "Extra Large" (20px) -> verify 20px / 1.25 scale propagation
    await btnExtraLarge.click();
    await expect(btnExtraLarge).toHaveAttribute('aria-pressed', 'true');
    await expect(hostShell).toHaveAttribute('data-mis-font-size', 'extra-large');

    await expect
      .poll(async () => {
        return await pluginFrame.locator(':root').evaluate((root) => ({
          attr: root.getAttribute('data-mis-font-size'),
          tokenBase: (root as HTMLElement).style.getPropertyValue('--mis-font-size-base').trim(),
          tokenScale: (root as HTMLElement).style.getPropertyValue('--mis-font-scale').trim(),
          computedRootFontSize: window.getComputedStyle(root).fontSize,
        }));
      })
      .toEqual({
        attr: 'extra-large',
        tokenBase: '20px',
        tokenScale: '1.25',
        computedRootFontSize: '20px',
      });

    // 6. Click "Small" (14px) -> verify 14px / 0.875 scale propagation
    await btnSmall.click();
    await expect(btnSmall).toHaveAttribute('aria-pressed', 'true');
    await expect(hostShell).toHaveAttribute('data-mis-font-size', 'small');

    await expect
      .poll(async () => {
        return await pluginFrame.locator(':root').evaluate((root) => ({
          attr: root.getAttribute('data-mis-font-size'),
          tokenBase: (root as HTMLElement).style.getPropertyValue('--mis-font-size-base').trim(),
          tokenScale: (root as HTMLElement).style.getPropertyValue('--mis-font-scale').trim(),
          computedRootFontSize: window.getComputedStyle(root).fontSize,
        }));
      })
      .toEqual({
        attr: 'small',
        tokenBase: '14px',
        tokenScale: '0.875',
        computedRootFontSize: '14px',
      });

    // 7. Verify GET /api/system/ui-config returns persisted staff_settings for teacher_staff_id
    await expect
      .poll(async () => {
        const cfgRes = await request.get('/api/system/ui-config');
        if (!cfgRes.ok()) {
          return null;
        }
        const cfgBody = await cfgRes.json();
        return cfgBody.user_settings;
      })
      .toEqual(
        expect.objectContaining({
          base_font_size: 'small',
          theme_mode: 'light',
        })
      );

    // 8. Reload the browser on /settings and verify Teacher's "small" (14px) setting and Settings view route persist
    await page.reload();
    await expect(page.getByTestId('host-shell')).toHaveAttribute('data-mis-font-size', 'small');
    await expect(page.getByTestId('host-settings-view')).toBeVisible();
    const reloadedRootFontSize = await page.evaluate(
      () => window.getComputedStyle(document.documentElement).fontSize
    );
    expect(reloadedRootFontSize).toBe('14px');

    // 9. Switch persona to Admin via POST /api/auth/persona -> verify Admin has isolated default 'medium' (16px)
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personaSeed.admin_staff_id },
    });
    expect(switchAdminRes.ok()).toBe(true);
    const adminSession = await switchAdminRes.json();
    await page.context().addCookies([
      {
        name: 'mis_session',
        value: adminSession.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);
    await page.goto('/');
    await expect(page.getByTestId('host-shell')).toHaveAttribute('data-mis-font-size', 'medium');
    const adminRootFontSize = await page.evaluate(
      () => window.getComputedStyle(document.documentElement).fontSize
    );
    expect(adminRootFontSize).toBe('16px');
  });

  test('userSettingsEndpoint_crossStaffSpoofingAndInvalidFontSizeOrThemeMode_failsFastWith400And403_test', async ({
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // Switch active session to Teacher A
    const switchTeacherARes = await request.post('/api/auth/persona', {
      data: { staff_id: m1Seed.teacher_staff_id },
    });
    expect(switchTeacherARes.ok()).toBe(true);

    // 1. Cross-staff spoofing (Teacher A attempting to overwrite Teacher B's settings) -> 403 FORBIDDEN_STAFF_ID_MISMATCH
    const spoofRes = await request.put('/api/system/user-settings', {
      data: {
        staff_id: m1Seed.other_teacher_staff_id,
        base_font_size: 'large',
      },
    });
    expect(spoofRes.status()).toBe(403);
    const spoofBody = await spoofRes.json();
    expect(spoofBody.ok).toBe(false);
    expect(spoofBody.error_code).toContain('FORBIDDEN_STAFF_ID_MISMATCH');

    // 2. Invalid base_font_size value -> 400 INVALID_BASE_FONT_SIZE
    const invalidFontRes = await request.put('/api/system/user-settings', {
      data: {
        base_font_size: 'gigantic',
      },
    });
    expect(invalidFontRes.status()).toBe(400);
    const invalidFontBody = await invalidFontRes.json();
    expect(invalidFontBody.ok).toBe(false);
    expect(invalidFontBody.error_code).toContain('INVALID_BASE_FONT_SIZE');

    // 3. Invalid theme_mode value -> 400 INVALID_THEME_MODE
    const invalidThemeRes = await request.put('/api/system/user-settings', {
      data: {
        theme_mode: 'cyberpunk',
      },
    });
    expect(invalidThemeRes.status()).toBe(400);
    const invalidThemeBody = await invalidThemeRes.json();
    expect(invalidThemeBody.ok).toBe(false);
    expect(invalidThemeBody.error_code).toContain('INVALID_THEME_MODE');

    // 4. Empty update payload (neither base_font_size nor theme_mode provided) -> 400 MISSING_MANDATORY_PARAMETER
    const emptyUpdateRes = await request.put('/api/system/user-settings', {
      data: {},
    });
    expect(emptyUpdateRes.status()).toBe(400);
    const emptyUpdateBody = await emptyUpdateRes.json();
    expect(emptyUpdateBody.ok).toBe(false);
    expect(emptyUpdateBody.error_code).toContain('MISSING_MANDATORY_PARAMETER');

    // 5. Valid update for Teacher A persists alongside pinned_recents_json in staff_settings without clobbering pins
    const pinRes = await request.post('/api/system/pinned-recents', {
      data: {
        action: 'pin',
        route: `/p/${testPlugId}`,
        title: 'Test Workspace Pin',
      },
    });
    expect(pinRes.ok()).toBe(true);

    const validUpdateRes = await request.put('/api/system/user-settings', {
      data: {
        base_font_size: 'extra-large',
        theme_mode: 'high-contrast',
      },
    });
    expect(validUpdateRes.status()).toBe(200);
    const validUpdateBody = await validUpdateRes.json();
    expect(validUpdateBody.ok).toBe(true);
    expect(validUpdateBody.user_settings).toEqual({
      staff_id: m1Seed.teacher_staff_id,
      base_font_size: 'extra-large',
      theme_mode: 'high-contrast',
    });

    const uiConfigRes = await request.get('/api/system/ui-config');
    expect(uiConfigRes.ok()).toBe(true);
    const uiConfigBody = await uiConfigRes.json();
    expect(uiConfigBody.user_settings).toEqual({
      staff_id: m1Seed.teacher_staff_id,
      base_font_size: 'extra-large',
      theme_mode: 'high-contrast',
    });
    expect(uiConfigBody.pinned_recents).toEqual([
      expect.objectContaining({
        route: `/p/${testPlugId}`,
        plug_id: testPlugId,
      }),
    ]);
  });

  test('hostSidebar_withLargeAndExtraLargeBaseFontSize_doesNotExceedWindowHeightAndKeepsBottomActionsInViewport_test', async ({
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
    const teacherSession = await switchTeacherRes.json();

    await page.context().addCookies([
      {
        name: 'mis_session',
        value: teacherSession.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    await page.goto(
      `/p/${testPlugId}?pupil_id=${encodeURIComponent(
        m1Seed.active_pupil_id
      )}&group_id=${encodeURIComponent(m1Seed.own_group_id)}`
    );

    const sidebar = page.getByTestId('host-sidebar');
    await expect(sidebar).toBeVisible();

    await page.getByTestId('nav-rail-settings').click();
    const settingsView = page.getByTestId('host-settings-view');
    await expect(settingsView).toBeVisible();

    for (const preset of ['small', 'medium', 'large', 'extra-large'] as const) {
      await settingsView.getByTestId(`host-settings-font-size-btn-${preset}`).click();

      const geometry = await page.evaluate(() => {
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

      expect(geometry.asideHeight).toBeLessThanOrEqual(geometry.windowInnerHeight);
      expect(geometry.asideBottom).toBeLessThanOrEqual(geometry.windowInnerHeight);
      expect(geometry.settingsBottom).toBeLessThanOrEqual(geometry.windowInnerHeight);
      expect(geometry.logoutBottom).toBeLessThanOrEqual(geometry.windowInnerHeight);
      expect(geometry.asideScrollHeight).toBeLessThanOrEqual(geometry.asideOffsetHeight);
    }
  });
});

