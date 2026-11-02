import { test, expect } from '@playwright/test';
import {
  type HostSettingsTestState,
  setupHostSettingsTestEnv,
  teardownHostSettingsTestEnv,
  authenticateTeacherSession,
  mountTeacherWorkspaceWithWidget,
  measureHostRootFontMetrics,
  measureFrameRootFontMetrics,
  measureHostSidebarGeometry,
  measureHostSettingsAndSidebarFonts,
  measurePluginFonts,
} from './host_settings_fixtures';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

test.describe('Host Settings Modal: Base Font Size Toggle (Small/Medium/Large/Extra Large), Iframe Propagation & staff_settings Persistence', () => {
  let state: HostSettingsTestState;

  test.beforeEach(async ({ page, request }) => {
    state = await setupHostSettingsTestEnv(page, request);
  });

  test.afterEach(async ({ request }, testInfo) => {
    await teardownHostSettingsTestEnv(request, state, testInfo);
  });

  test('hostSettingsBaseFontSize_selectingSmallMediumLargeExtraLarge_updatesHostAndPluginIframeFontSizeAndPersistsInStaffSettings_test', async ({
    page,
    request,
  }) => {
    if (!state.m1Seed || !state.personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // 1. Authenticate as Teacher and place a widget on the dashboard so both plugin and widget iframes are mounted
    const { pluginFrame, widgetFrame, widgetCompositeKey } = await mountTeacherWorkspaceWithWidget(
      page,
      request,
      state.m1Seed,
      state.testPlugId
    );

    const hostShell = page.getByTestId('host-shell');
    await expect(hostShell).toBeVisible();
    await expect(hostShell).toHaveAttribute('data-mis-font-size', 'medium');

    const pluginIframeEl = page.getByTestId(`plugin-iframe-${state.testPlugId}`);
    await expect(pluginIframeEl).toBeVisible();
    await expect(pluginFrame.locator('body')).toBeVisible();

    // 2. Verify initial default medium (16px) state on Host root and plugin iframe root
    const initialHostMetrics = await measureHostRootFontMetrics(page);
    expect(initialHostMetrics.attr).toBe('medium');
    expect(initialHostMetrics.tokenBase).toBe('16px');
    expect(initialHostMetrics.tokenScale).toBe('1');
    expect(initialHostMetrics.computedRootFontSize).toBe('16px');

    const initialPluginMetrics = await measureFrameRootFontMetrics(pluginFrame);
    expect(initialPluginMetrics.attr).toBe('medium');
    expect(initialPluginMetrics.tokenBase).toBe('16px');
    expect(initialPluginMetrics.tokenScale).toBe('1');
    expect(initialPluginMetrics.computedRootFontSize).toBe('16px');

    // Verify Home view renders the placed widget iframe
    await page.getByTestId('nav-rail-home').click();
    const widgetIframeEl = page.getByTestId('widget-iframe-pastoral_watch');
    await expect(widgetIframeEl).toBeVisible();
    await expect(widgetIframeEl).toHaveAttribute('data-widget-key', widgetCompositeKey);
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

    const largeHostMetrics = await measureHostRootFontMetrics(page);
    expect(largeHostMetrics.attr).toBe('large');
    expect(largeHostMetrics.tokenBase).toBe('18px');
    expect(largeHostMetrics.tokenScale).toBe('1.125');
    expect(largeHostMetrics.computedRootFontSize).toBe('18px');

    await expect
      .poll(async () => {
        const m = await measureFrameRootFontMetrics(pluginFrame);
        return { attr: m.attr, tokenBase: m.tokenBase, computedRootFontSize: m.computedRootFontSize };
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
        const m = await measureFrameRootFontMetrics(widgetFrame);
        return { attr: m.attr, tokenBase: m.tokenBase, computedRootFontSize: m.computedRootFontSize };
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
        const m = await measureFrameRootFontMetrics(pluginFrame);
        return {
          attr: m.attr,
          tokenBase: m.tokenBase,
          tokenScale: m.tokenScale,
          computedRootFontSize: m.computedRootFontSize,
        };
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
        const m = await measureFrameRootFontMetrics(pluginFrame);
        return {
          attr: m.attr,
          tokenBase: m.tokenBase,
          tokenScale: m.tokenScale,
          computedRootFontSize: m.computedRootFontSize,
        };
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
      data: { staff_id: state.personaSeed.admin_staff_id },
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
    if (!state.m1Seed || !state.personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // Switch active session to Teacher A
    const switchTeacherARes = await request.post('/api/auth/persona', {
      data: { staff_id: state.m1Seed.teacher_staff_id },
    });
    expect(switchTeacherARes.ok()).toBe(true);

    // 1. Cross-staff spoofing (Teacher A attempting to overwrite Teacher B's settings) -> 403 FORBIDDEN_STAFF_ID_MISMATCH
    const spoofRes = await request.put('/api/system/user-settings', {
      data: {
        staff_id: state.m1Seed.other_teacher_staff_id,
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
        route: `/p/${state.testPlugId}`,
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
      staff_id: state.m1Seed.teacher_staff_id,
      base_font_size: 'extra-large',
      theme_mode: 'high-contrast',
    });

    const uiConfigRes = await request.get('/api/system/ui-config');
    expect(uiConfigRes.ok()).toBe(true);
    const uiConfigBody = await uiConfigRes.json();
    expect(uiConfigBody.user_settings).toEqual({
      staff_id: state.m1Seed.teacher_staff_id,
      base_font_size: 'extra-large',
      theme_mode: 'high-contrast',
    });
    expect(uiConfigBody.pinned_recents).toEqual([
      expect.objectContaining({
        route: `/p/${state.testPlugId}`,
        plug_id: state.testPlugId,
      }),
    ]);
  });

  test('hostSidebar_withLargeAndExtraLargeBaseFontSize_doesNotExceedWindowHeightAndKeepsBottomActionsInViewport_test', async ({
    page,
    request,
  }) => {
    if (!state.m1Seed || !state.personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    await authenticateTeacherSession(page, request, state.m1Seed.teacher_staff_id);

    await page.goto(
      `/p/${state.testPlugId}?pupil_id=${encodeURIComponent(
        state.m1Seed.active_pupil_id
      )}&group_id=${encodeURIComponent(state.m1Seed.own_group_id)}`
    );

    const sidebar = page.getByTestId('host-sidebar');
    await expect(sidebar).toBeVisible();

    await page.getByTestId('nav-rail-settings').click();
    const settingsView = page.getByTestId('host-settings-view');
    await expect(settingsView).toBeVisible();

    for (const preset of ['small', 'medium', 'large', 'extra-large'] as const) {
      await settingsView.getByTestId(`host-settings-font-size-btn-${preset}`).click();

      const geometry = await measureHostSidebarGeometry(page);

      expect(geometry.asideHeight).toBeLessThanOrEqual(geometry.windowInnerHeight);
      expect(geometry.asideBottom).toBeLessThanOrEqual(geometry.windowInnerHeight);
      expect(geometry.settingsBottom).toBeLessThanOrEqual(geometry.windowInnerHeight);
      expect(geometry.logoutBottom).toBeLessThanOrEqual(geometry.windowInnerHeight);
      expect(geometry.asideScrollHeight).toBeLessThanOrEqual(geometry.asideOffsetHeight);
    }
  });

  test('hostSettingsBaseFontSize_selectingPresets_scalesComputedTextSizeOfSettingsButtonsHeadingsSidebarDashboardAndPluginElements_test', async ({
    page,
    request,
  }) => {
    if (!state.m1Seed || !state.personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    const { pluginFrame } = await mountTeacherWorkspaceWithWidget(
      page,
      request,
      state.m1Seed,
      state.testPlugId
    );

    await expect(pluginFrame.getByTestId('print-record-btn')).toBeVisible();

    await page.getByTestId('nav-rail-settings').click();
    const settingsView = page.getByTestId('host-settings-view');
    await expect(settingsView).toBeVisible();

    // 1. Verify Medium (1.0x) baseline computed font sizes on actual text elements
    const mediumHost = await measureHostSettingsAndSidebarFonts(page);
    expect(mediumHost.headingPx).toBeCloseTo(16, 2);
    expect(mediumHost.controlPx).toBeCloseTo(13, 2);
    expect(mediumHost.btnLargePx).toBeCloseTo(12, 2);
    expect(mediumHost.schoolNamePx).toBeCloseTo(10, 2);
    expect(mediumHost.navHomePx).toBeCloseTo(11, 2);
    expect(mediumHost.navSettingsPx).toBeCloseTo(10, 2);

    const mediumPlugin = await measurePluginFonts(pluginFrame);
    expect(mediumPlugin.btnPx).toBeCloseTo(13, 2);
    expect(mediumPlugin.sdkBarPx).toBeCloseTo(12, 2);

    // 2. Click "Large" (1.125x) -> verify all text elements scale by 1.125x
    await settingsView.getByTestId('host-settings-font-size-btn-large').click();
    const largeHost = await measureHostSettingsAndSidebarFonts(page);
    expect(largeHost.headingPx).toBeCloseTo(16 * 1.125, 2);
    expect(largeHost.controlPx).toBeCloseTo(13 * 1.125, 2);
    expect(largeHost.btnLargePx).toBeCloseTo(12 * 1.125, 2);
    expect(largeHost.schoolNamePx).toBeCloseTo(10 * 1.125, 2);
    expect(largeHost.navHomePx).toBeCloseTo(11 * 1.125, 2);
    expect(largeHost.navSettingsPx).toBeCloseTo(10 * 1.125, 2);

    await expect
      .poll(async () => (await measurePluginFonts(pluginFrame)).btnPx)
      .toBeCloseTo(13 * 1.125, 2);
    const largePlugin = await measurePluginFonts(pluginFrame);
    expect(largePlugin.sdkBarPx).toBeCloseTo(12 * 1.125, 2);

    // 3. Click "Extra Large" (1.25x) -> verify all text elements scale by 1.25x
    await settingsView.getByTestId('host-settings-font-size-btn-extra-large').click();
    const xlHost = await measureHostSettingsAndSidebarFonts(page);
    expect(xlHost.headingPx).toBeCloseTo(16 * 1.25, 2);
    expect(xlHost.controlPx).toBeCloseTo(13 * 1.25, 2);
    expect(xlHost.btnLargePx).toBeCloseTo(12 * 1.25, 2);
    expect(xlHost.schoolNamePx).toBeCloseTo(10 * 1.25, 2);
    expect(xlHost.navHomePx).toBeCloseTo(11 * 1.25, 2);
    expect(xlHost.navSettingsPx).toBeCloseTo(10 * 1.25, 2);

    await expect
      .poll(async () => (await measurePluginFonts(pluginFrame)).btnPx)
      .toBeCloseTo(13 * 1.25, 2);
    const xlPlugin = await measurePluginFonts(pluginFrame);
    expect(xlPlugin.sdkBarPx).toBeCloseTo(12 * 1.25, 2);

    // Verify Home Dashboard search & widget iframe text also reflect Extra Large (1.25x)
    await page.getByTestId('nav-rail-home').click();
    const xlDashboardMetrics = await page.evaluate(() => {
      const searchInput = document.querySelector('[data-testid="landing-search-input"]');
      const editBtn = document.querySelector('[data-testid="dashboard-edit-mode-toggle-btn"]');
      return {
        searchInputPx: searchInput ? parseFloat(window.getComputedStyle(searchInput).fontSize) : 0,
        editBtnPx: editBtn ? parseFloat(window.getComputedStyle(editBtn).fontSize) : 0,
      };
    });
    expect(xlDashboardMetrics.searchInputPx).toBeCloseTo(13 * 1.25, 2);
    expect(xlDashboardMetrics.editBtnPx).toBeCloseTo(12 * 1.25, 2);

    const widgetFrame = page.frameLocator('[data-testid="widget-iframe-pastoral_watch"]');
    await expect
      .poll(async () => {
        return await widgetFrame.locator(':root').evaluate(() => {
          const btn = document.querySelector('[data-testid="widget-rpc-fetch-btn"]');
          return btn ? parseFloat(window.getComputedStyle(btn).fontSize) : 0;
        });
      })
      .toBeCloseTo(12 * 1.25, 2);

    // 4. Switch back to Settings and click "Small" (0.875x) -> verify all text elements scale by 0.875x
    await page.getByTestId('nav-rail-settings').click();
    await settingsView.getByTestId('host-settings-font-size-btn-small').click();
    const smallHost = await measureHostSettingsAndSidebarFonts(page);
    expect(smallHost.headingPx).toBeCloseTo(16 * 0.875, 2);
    expect(smallHost.controlPx).toBeCloseTo(13 * 0.875, 2);
    expect(smallHost.btnLargePx).toBeCloseTo(12 * 0.875, 2);
    expect(smallHost.schoolNamePx).toBeCloseTo(10 * 0.875, 2);
    expect(smallHost.navHomePx).toBeCloseTo(11 * 0.875, 2);
    expect(smallHost.navSettingsPx).toBeCloseTo(10 * 0.875, 2);

    await expect
      .poll(async () => (await measurePluginFonts(pluginFrame)).btnPx)
      .toBeCloseTo(13 * 0.875, 2);
  });
});
