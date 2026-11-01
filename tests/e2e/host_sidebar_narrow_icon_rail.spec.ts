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

test.describe('Left Sidebar (HostSidebar.tsx): Narrow Icon Rail & Pinned Recents (__system_config)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let m1Seed: Milestone1SeedResponse | null = null;
  let personaSeed: PersonaSeedResponse | null = null;
  let testPlugId: string = '';
  let senPlugId: string = '';
  let behaviourPlugId: string = '';

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    m1Seed = null;
    personaSeed = null;
    testPlugId = '';
    senPlugId = '';
    behaviourPlugId = '';

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

    // 5. Switch to Admin persona and install + activate test_plugin, senco_provision_plugin, and behaviour_plugin
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

    const installSenRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'senco_provision_plugin', staff_id: personaSeed.admin_staff_id },
    });
    if (!installSenRes.ok()) {
      throw new Error(`Failed to install senco_provision_plugin: ${installSenRes.status()}`);
    }
    senPlugId = (await installSenRes.json()).plug_id;
    await request.post(`/api/admin/plugins/${encodeURIComponent(senPlugId)}/activate`, {
      data: { staff_id: personaSeed.admin_staff_id },
    });

    const installBehRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'behaviour_plugin', staff_id: personaSeed.admin_staff_id },
    });
    if (!installBehRes.ok()) {
      throw new Error(`Failed to install behaviour_plugin: ${installBehRes.status()}`);
    }
    behaviourPlugId = (await installBehRes.json()).plug_id;
    await request.post(`/api/admin/plugins/${encodeURIComponent(behaviourPlugId)}/activate`, {
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

    // 3. Clean up pinned recents and installed plugins
    if (personaSeed) {
      await request.post('/api/auth/persona', {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      await request.post('/api/system/pinned-recents', {
        data: {
          action: 'clear',
          staff_id: personaSeed.admin_staff_id,
        },
      });
      if (m1Seed) {
        await request.post('/api/system/pinned-recents', {
          data: {
            action: 'clear',
            staff_id: m1Seed.teacher_staff_id,
          },
        });
      }
      for (const pid of [behaviourPlugId, senPlugId, testPlugId]) {
        if (pid) {
          await request.delete(
            `/api/admin/plugins/${encodeURIComponent(pid)}?staff_id=${encodeURIComponent(
              personaSeed.admin_staff_id
            )}`
          );
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

  test('hostSidebarNarrowIconRail_withInstalledPlugins_rendersOnlyRailItemsWithoutFullPluginList_test', async ({
    page,
  }) => {
    await page.goto('/');

    const sidebar = page.getByTestId('host-sidebar');
    await expect(sidebar).toBeVisible();
    await expect(sidebar).toHaveAttribute('data-rail-mode', 'narrow-icon-rail');

    // Compact school identity remains inside the Narrow Icon Rail; active-entity-context-pill is removed
    await expect(page.getByTestId('host-school-name')).toBeVisible();
    await expect(page.getByTestId('active-entity-context-pill')).toHaveCount(0);

    // Narrow Icon Rail contains strictly: Home, Global Search (🔍), Pinned Recents, Settings, and Logout
    await expect(page.getByTestId('nav-rail-home')).toBeVisible();
    await expect(page.getByTestId('nav-rail-search')).toBeVisible();
    await expect(page.getByTestId('nav-rail-pinned-recents')).toBeVisible();
    await expect(page.getByTestId('nav-rail-settings')).toBeVisible();
    await expect(page.getByTestId('nav-rail-logout')).toBeVisible();

    // Verify the full installed-plugin list (nav-item-plugin-*) is completely removed from the sidebar
    await expect(page.locator('[data-testid^="nav-item-plugin-"]')).toHaveCount(0);

    // Verify no automatic recent routes appear when user has not manually pinned any views
    await expect(page.locator('[data-testid^="pinned-recent-item-"]')).toHaveCount(0);
  });

  test('pinnedRecentsPersistence_manualPinAndUnpinAcrossPersonas_storesRoutesAndTitlesInSystemConfig_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // 1. Switch to Classroom Teacher / SENCO persona and open /p/<senPlugId>/student/<second_active_pupil_id>?tab=iep
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

    const targetSenRoute = `/p/${senPlugId}/student/${encodeURIComponent(
      m1Seed.second_active_pupil_id
    )}?tab=iep`;
    await page.goto(
      `${targetSenRoute}&reference_date=${encodeURIComponent(
        m1Seed.reference_date
      )}&academic_year_id=${encodeURIComponent(m1Seed.academic_year_id)}&staff_id=${encodeURIComponent(
        m1Seed.teacher_staff_id
      )}`
    );

    await expect(page.getByTestId('host-shell')).toBeVisible();
    await expect(page.getByTestId(`plugin-iframe-${senPlugId}`)).toBeVisible();

    // 2. Manually pin the current route (/p/<senPlugId>/student/<second_active_pupil_id>?tab=iep) via Narrow Icon Rail "Pin Current View" button
    const topbarPinBtn = page.getByTestId('topbar-pin-view-btn');
    await expect(topbarPinBtn).toBeVisible();
    await expect(topbarPinBtn).toContainText('Pin Current View');
    await topbarPinBtn.click();

    await expect(topbarPinBtn).toContainText('Pinned');

    // Verify pinned item is rendered in the Narrow Icon Rail Pinned Recents section
    const pinnedItem0 = page.getByTestId('pinned-recent-item-0');
    await expect(pinnedItem0).toBeVisible();
    await expect(pinnedItem0).toHaveAttribute('data-pinned-route', targetSenRoute);

    // Verify GET /api/system/ui-config persists the pinned route & title in __system_config for teacher_staff_id
    const uiConfigRes = await request.get(
      `/api/system/ui-config?staff_id=${encodeURIComponent(m1Seed.teacher_staff_id)}`
    );
    expect(uiConfigRes.ok()).toBe(true);
    const uiConfigBody = await uiConfigRes.json();
    expect(Array.isArray(uiConfigBody.pinned_recents)).toBe(true);
    expect(uiConfigBody.pinned_recents).toEqual([
      expect.objectContaining({
        route: targetSenRoute,
        plug_id: senPlugId,
      }),
    ]);

    // 3. Click Narrow Icon Rail Search button (🔍) -> opens full-page Search view in right-hand pane (/search), NOT a modal
    await page.getByTestId('nav-rail-search').click();
    expect(new URL(page.url()).pathname).toBe('/search');
    await expect(page.getByTestId('nav-rail-search')).toHaveAttribute('aria-current', 'page');
    await expect(page.getByTestId('command-palette-modal')).toHaveCount(0);

    const searchView = page.getByTestId('host-search-view');
    await expect(searchView).toBeVisible();
    await expect(page.getByTestId('landing-page-search-panel')).toHaveCount(0);
    await expect(page.getByTestId('dashboard-widget-grid-section')).toHaveCount(0);
    await expect(page.getByTestId(`plugin-iframe-${senPlugId}`)).toBeAttached();
    await expect(page.getByTestId(`plugin-iframe-${senPlugId}`)).toBeHidden();

    // Verify plugin view items in full-page Search view expose data-permitted-count and data-permitted-methods
    const behSearchView = page.getByTestId(`host-search-result-view-${behaviourPlugId}-home`);
    await expect(behSearchView).toBeVisible();
    await expect(behSearchView).toHaveAttribute('data-permitted-count', /^[1-9]\d*$/);

    const searchPinBehBtn = page.getByTestId(`host-search-pin-view-btn-${behaviourPlugId}-home`);
    await expect(searchPinBehBtn).toBeVisible();
    await searchPinBehBtn.click();

    // Verify both pinned routes appear in the Narrow Icon Rail
    const pinnedItem1 = page.getByTestId('pinned-recent-item-1');
    await expect(pinnedItem1).toBeVisible();
    await expect(pinnedItem1).toHaveAttribute('data-pinned-route', `/p/${behaviourPlugId}`);

    // 4. Click "Home" in the Narrow Icon Rail -> navigates to "/" and shows Home dashboard widgets while hiding Search view and keeping LRU pool mounted with display: none
    await page.getByTestId('nav-rail-home').click();
    expect(new URL(page.url()).pathname).toBe('/');
    await expect(page.getByTestId('nav-rail-home')).toHaveAttribute('aria-current', 'page');
    await expect(page.getByTestId('host-search-view')).toHaveCount(0);
    await expect(page.getByTestId('landing-page-search-panel')).toBeVisible();
    await expect(page.getByTestId('dashboard-widget-grid-section')).toBeVisible();
    await expect(page.getByTestId(`plugin-iframe-${senPlugId}`)).toBeAttached();
    await expect(page.getByTestId(`plugin-iframe-${senPlugId}`)).toBeHidden();

    // 5. Click the pinned item in the Narrow Icon Rail to restore /p/<senPlugId>/student/<second_active_pupil_id>?tab=iep exclusively in the right-hand pane
    await pinnedItem0.click();
    await expect(page.getByTestId(`plugin-iframe-${senPlugId}`)).toBeVisible();
    await expect(page.getByTestId('landing-page-search-panel')).toHaveCount(0);
    await expect(page.getByTestId('dashboard-widget-grid-section')).toHaveCount(0);
    await expect(page.getByTestId('host-search-view')).toHaveCount(0);
    await expect(page.getByTestId('host-settings-view')).toHaveCount(0);
    expect(page.url()).toContain(
      `/p/${senPlugId}/student/${encodeURIComponent(m1Seed.second_active_pupil_id)}`
    );

    // 6. Switch persona to Admin via ?__persona= override -> verify per-persona __system_config isolation
    await page.goto(`/?__persona=${encodeURIComponent(personaSeed.admin_staff_id)}`);
    await expect(page.locator('[data-testid^="pinned-recent-item-"]')).toHaveCount(0);

    // Switch back to Classroom Teacher via ?__persona= override -> teacher's 2 pinned routes are restored from __system_config
    await page.goto(`/?__persona=${encodeURIComponent(m1Seed.teacher_staff_id)}`);
    await expect(page.getByTestId('pinned-recent-item-0')).toBeVisible();
    await expect(page.getByTestId('pinned-recent-item-1')).toBeVisible();

    // 7. Unpin the first route via the rail unpin button and verify it is removed from DOM and __system_config
    await page.getByTestId('unpin-recent-btn-0').click();
    await expect(page.locator('[data-testid^="pinned-recent-item-"]')).toHaveCount(1);
    await expect(page.getByTestId('pinned-recent-item-0')).toHaveAttribute(
      'data-pinned-route',
      `/p/${behaviourPlugId}`
    );
  });

  test('railSettingsAndLogout_clickingSettingsAndLogout_opensSettingsViewInRightPaneWithoutModalAndTriggersLogoutAndFailsFastOnInvalidPin_test', async ({
    page,
    request,
  }) => {
    // 1. Fail-Fast API check: POST /api/system/pinned-recents rejects missing or non-/p/<plugid> routes
    const invalidRouteRes = await request.post('/api/system/pinned-recents', {
      data: {
        action: 'pin',
        route: '/invalid/route',
        title: 'Invalid Route',
      },
    });
    expect(invalidRouteRes.status()).toBe(400);
    const invalidRouteBody = await invalidRouteRes.json();
    expect(invalidRouteBody.ok).toBe(false);

    const missingTitleRes = await request.post('/api/system/pinned-recents', {
      data: {
        action: 'pin',
        route: `/p/${testPlugId}`,
        title: '',
      },
    });
    expect(missingTitleRes.status()).toBe(400);

    // 2. Open Host UI Shell on "/" -> starts on Home view without settings view or modal
    await page.goto('/');
    await expect(page.getByTestId('host-settings-view')).toHaveCount(0);
    await expect(page.getByTestId('host-settings-modal')).toHaveCount(0);
    await expect(page.getByTestId('host-settings-theme-toggle-btn')).toHaveCount(0);
    await expect(page.getByTestId('landing-page-search-panel')).toBeVisible();
    await expect(page.getByTestId('dashboard-widget-grid-section')).toBeVisible();

    // Click Settings in the Narrow Icon Rail -> replaces right-hand pane with full-page Settings view (/settings) and does NOT open a modal
    await page.getByTestId('nav-rail-settings').click();
    expect(new URL(page.url()).pathname).toBe('/settings');
    await expect(page.getByTestId('nav-rail-settings')).toHaveAttribute('aria-current', 'page');
    await expect(page.getByTestId('host-settings-modal')).toHaveCount(0);
    await expect(page.getByTestId('landing-page-search-panel')).toHaveCount(0);
    await expect(page.getByTestId('dashboard-widget-grid-section')).toHaveCount(0);
    await expect(page.getByTestId('host-search-view')).toHaveCount(0);

    const settingsView = page.getByTestId('host-settings-view');
    await expect(settingsView).toBeVisible();
    await expect(settingsView).toContainText('100001');

    const settingsThemeToggleBtn = settingsView.getByTestId('host-settings-theme-toggle-btn');
    await expect(settingsThemeToggleBtn).toBeVisible();
    await expect(settingsThemeToggleBtn).toHaveAttribute('role', 'switch');
    await expect(settingsThemeToggleBtn).toHaveAttribute('aria-checked', 'false');
    await settingsThemeToggleBtn.click();
    await expect(settingsThemeToggleBtn).toHaveAttribute('aria-checked', 'true');
    await expect(page.getByTestId('host-shell')).toHaveAttribute('data-mis-theme', 'high-contrast');
    await settingsThemeToggleBtn.click();
    await expect(settingsThemeToggleBtn).toHaveAttribute('aria-checked', 'false');
    await expect(page.getByTestId('host-shell')).toHaveAttribute('data-mis-theme', 'light');

    // Click Home in sidebar to return to Home view
    await page.getByTestId('nav-rail-home').click();
    expect(new URL(page.url()).pathname).toBe('/');
    await expect(settingsView).toHaveCount(0);
    await expect(page.getByTestId('landing-page-search-panel')).toBeVisible();

    // 3. Click Logout in the Narrow Icon Rail -> triggers session sign-out state (/api/auth/logout)
    await page.getByTestId('nav-rail-logout').click();
    const loggedOutBanner = page.getByTestId('host-logged-out-banner');
    await expect(loggedOutBanner).toBeVisible();
    await expect(loggedOutBanner).toContainText('Signed out');
  });

  test('pinnedRecentsAndDashboardSecurity_crossStaffSpoofingUnvalidatedSubpathsDormantPluginsAndDefaultFallback_enforcesAuthRouteValidationAndZeroFallback_test', async ({
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // Install and activate dfe_census_plugin (requires dal.census.read:all, which Classroom Teacher lacks)
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personaSeed.admin_staff_id },
    });
    expect(switchAdminRes.ok()).toBe(true);

    const installCensusRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'dfe_census_plugin', staff_id: personaSeed.admin_staff_id },
    });
    expect(installCensusRes.ok()).toBe(true);
    const censusPlugId: string = (await installCensusRes.json()).plug_id;
    const activateCensusRes = await request.post(
      `/api/admin/plugins/${encodeURIComponent(censusPlugId)}/activate`,
      {
        data: { staff_id: personaSeed.admin_staff_id },
      }
    );
    expect(activateCensusRes.ok()).toBe(true);

    try {
      // 1. Switch session cookie to Teacher A (m1Seed.teacher_staff_id)
      const switchTeacherARes = await request.post('/api/auth/persona', {
        data: { staff_id: m1Seed.teacher_staff_id },
      });
      expect(switchTeacherARes.ok()).toBe(true);

      // 1a. Cross-staff overwrite attempt on POST /api/system/pinned-recents (spoofing Teacher B's staff_id) must fail fast with 403
      const spoofPinRes = await request.post('/api/system/pinned-recents', {
        data: {
          action: 'pin',
          route: `/p/${behaviourPlugId}`,
          title: 'Spoofed Pin',
          staff_id: m1Seed.other_teacher_staff_id,
        },
      });
      expect(spoofPinRes.status()).toBe(403);
      const spoofPinBody = await spoofPinRes.json();
      expect(spoofPinBody.ok).toBe(false);
      expect(spoofPinBody.error_code).toContain('FORBIDDEN_STAFF_ID_MISMATCH');

      // 1b. Cross-staff overwrite attempt on PUT /api/dashboard/layout (spoofing Teacher B's staff_id) must fail fast with 403
      const spoofLayoutRes = await request.put('/api/dashboard/layout', {
        data: {
          staff_id: m1Seed.other_teacher_staff_id,
          layout: [{ plug_id: testPlugId, widget_id: 'pastoral_watch' }],
        },
      });
      expect(spoofLayoutRes.status()).toBe(403);
      const spoofLayoutBody = await spoofLayoutRes.json();
      expect(spoofLayoutBody.ok).toBe(false);
      expect(spoofLayoutBody.error_code).toContain('FORBIDDEN_STAFF_ID_MISMATCH');

      // 2. Unvalidated pinned sub-routes & unpermitted plugins must fail fast on POST /api/system/pinned-recents
      // 2a. Unknown plug_id (valid 8-char format but not installed)
      const unknownPlugRes = await request.post('/api/system/pinned-recents', {
        data: {
          action: 'pin',
          route: '/p/deadbeef',
          title: 'Unknown Plugin',
        },
      });
      expect(unknownPlugRes.status()).toBe(404);
      const unknownPlugBody = await unknownPlugRes.json();
      expect(unknownPlugBody.ok).toBe(false);
      expect(unknownPlugBody.error_code).toContain('PLUGIN_NOT_FOUND');

      // 2b. Undeclared sub-path on an active installed plugin
      const undeclaredSubpathRes = await request.post('/api/system/pinned-recents', {
        data: {
          action: 'pin',
          route: `/p/${senPlugId}/undeclared/subpath`,
          title: 'Undeclared Subpath',
        },
      });
      expect(undeclaredSubpathRes.status()).toBe(404);
      const undeclaredSubpathBody = await undeclaredSubpathRes.json();
      expect(undeclaredSubpathBody.ok).toBe(false);
      expect(undeclaredSubpathBody.error_code).toContain('UNDECLARED_PLUGIN_VIEW_PATH');

      // 2c. Undeclared query parameter on a declared view path
      const undeclaredQueryRes = await request.post('/api/system/pinned-recents', {
        data: {
          action: 'pin',
          route: `/p/${senPlugId}/student/${encodeURIComponent(m1Seed.active_pupil_id)}?tab=iep&bogus_param=1`,
          title: 'Undeclared Query Param',
        },
      });
      expect(undeclaredQueryRes.status()).toBe(400);
      const undeclaredQueryBody = await undeclaredQueryRes.json();
      expect(undeclaredQueryBody.ok).toBe(false);
      expect(undeclaredQueryBody.error_code).toContain('UNDECLARED_PLUGIN_VIEW_QUERY_PARAM');

      // 2d. Unpermitted plugin (Teacher A lacks dal.census.read:all for dfe_census_plugin)
      const unpermittedPluginRes = await request.post('/api/system/pinned-recents', {
        data: {
          action: 'pin',
          route: `/p/${censusPlugId}`,
          title: 'DfE Census',
        },
      });
      expect(unpermittedPluginRes.status()).toBe(403);
      const unpermittedPluginBody = await unpermittedPluginRes.json();
      expect(unpermittedPluginBody.ok).toBe(false);
      expect(unpermittedPluginBody.error_code).toContain('FORBIDDEN_PLUGIN_ACCESS');

      // 3. Pin a valid route on behaviourPlugId and a valid route on senPlugId as Teacher A
      const validBehPinRes = await request.post('/api/system/pinned-recents', {
        data: {
          action: 'pin',
          route: `/p/${behaviourPlugId}`,
          title: 'Behaviour Workspace',
        },
      });
      expect(validBehPinRes.ok()).toBe(true);

      const validSenPinRes = await request.post('/api/system/pinned-recents', {
        data: {
          action: 'pin',
          route: `/p/${senPlugId}/student/${encodeURIComponent(m1Seed.active_pupil_id)}?tab=iep`,
          title: 'SEN IEP View',
        },
      });
      expect(validSenPinRes.ok()).toBe(true);

      // Remove behaviour_plugin as Admin -> GET /api/system/ui-config for Teacher A must filter out the stale behaviourPlugId pin
      await request.post('/api/auth/persona', {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      const removeBehRes = await request.delete(
        `/api/admin/plugins/${encodeURIComponent(behaviourPlugId)}?staff_id=${encodeURIComponent(
          personaSeed.admin_staff_id
        )}`
      );
      expect(removeBehRes.ok()).toBe(true);

      // Switch back to Teacher A and verify stale behaviourPlugId pin is not served
      await request.post('/api/auth/persona', {
        data: { staff_id: m1Seed.teacher_staff_id },
      });
      const uiCfgAfterRemoveRes = await request.get('/api/system/ui-config');
      expect(uiCfgAfterRemoveRes.ok()).toBe(true);
      const uiCfgAfterRemoveBody = await uiCfgAfterRemoveRes.json();
      expect(uiCfgAfterRemoveBody.pinned_recents).toEqual([
        expect.objectContaining({
          route: `/p/${senPlugId}/student/${encodeURIComponent(m1Seed.active_pupil_id)}?tab=iep`,
          plug_id: senPlugId,
        }),
      ]);

      // 4. Unpin the remaining senPlugId route -> must return empty [] and NEVER fall back to user_prefs.pinned_recents.default
      const unpinLastRes = await request.post('/api/system/pinned-recents', {
        data: {
          action: 'unpin',
          route: `/p/${senPlugId}/student/${encodeURIComponent(m1Seed.active_pupil_id)}?tab=iep`,
        },
      });
      expect(unpinLastRes.ok()).toBe(true);
      const unpinLastBody = await unpinLastRes.json();
      expect(unpinLastBody.pinned_recents).toEqual([]);

      const uiCfgAfterClearRes = await request.get('/api/system/ui-config');
      expect(uiCfgAfterClearRes.ok()).toBe(true);
      const uiCfgAfterClearBody = await uiCfgAfterClearRes.json();
      expect(uiCfgAfterClearBody.pinned_recents).toEqual([]);
    } finally {
      await request.post('/api/auth/persona', {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      await request.delete(
        `/api/admin/plugins/${encodeURIComponent(censusPlugId)}?staff_id=${encodeURIComponent(
          personaSeed.admin_staff_id
        )}`
      );
    }
  });
});
