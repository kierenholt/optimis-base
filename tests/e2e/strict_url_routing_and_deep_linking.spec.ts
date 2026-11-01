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

test.describe('Strict Opaque URL Routing (/p/<plugid>), Manifest Views Contract & Deep-Linking State Persistence', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let m1Seed: Milestone1SeedResponse | null = null;
  let personaSeed: PersonaSeedResponse | null = null;
  let testPlugId: string = '';
  let senPlugId: string = '';

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    m1Seed = null;
    personaSeed = null;
    testPlugId = '';
    senPlugId = '';

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

    // 4. Seed Admin/HoD/Teacher persona fixtures and Milestone 1 SEN fixtures
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
    if (!personaSeed.fixture_id || !personaSeed.admin_staff_id || !personaSeed.hod_staff_id) {
      throw new Error('Persona seed response missing mandatory identifiers');
    }

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
    if (!m1Seed.fixture_id || !m1Seed.active_pupil_id || !m1Seed.second_active_pupil_id) {
      throw new Error('Milestone 1 seed response missing mandatory identifiers');
    }

    // 5. Switch to Admin persona and install + activate test_plugin and senco_provision_plugin via opaque plug_id
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
    const installTestBody = await installTestRes.json();
    testPlugId = installTestBody.plug_id || installTestBody.plugin_id;
    if (!testPlugId) {
      throw new Error('Install test_plugin response missing mandatory plug_id');
    }

    const activateTestRes = await request.post(`/api/admin/plugins/${encodeURIComponent(testPlugId)}/activate`, {
      data: { staff_id: personaSeed.admin_staff_id },
    });
    if (!activateTestRes.ok()) {
      throw new Error(`Failed to activate test_plugin (${testPlugId}): ${activateTestRes.status()}`);
    }

    const installSenRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'senco_provision_plugin', staff_id: personaSeed.admin_staff_id },
    });
    if (!installSenRes.ok()) {
      throw new Error(`Failed to install senco_provision_plugin: ${installSenRes.status()}`);
    }
    const installSenBody = await installSenRes.json();
    senPlugId = installSenBody.plug_id || installSenBody.plugin_id;
    if (!senPlugId) {
      throw new Error('Install senco_provision_plugin response missing mandatory plug_id');
    }

    const activateSenRes = await request.post(`/api/admin/plugins/${encodeURIComponent(senPlugId)}/activate`, {
      data: { staff_id: personaSeed.admin_staff_id },
    });
    if (!activateSenRes.ok()) {
      throw new Error(`Failed to activate senco_provision_plugin (${senPlugId}): ${activateSenRes.status()}`);
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    if (personaSeed && m1Seed && senPlugId) {
      await request.post('/api/auth/persona', {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      await request.post(`/api/admin/plugins/${encodeURIComponent(senPlugId)}/activate`, {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      for (const pid of [
        m1Seed.active_pupil_id,
        m1Seed.second_active_pupil_id,
        m1Seed.transferred_pupil_id,
        m1Seed.other_group_pupil_id,
      ]) {
        await request.post(`/api/p/${encodeURIComponent(senPlugId)}/rpc/purge_pupil_data`, {
          data: { args: { pupil_id: pid } },
        });
      }
    }

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

    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (senPlugId) {
      await request.delete(`/api/admin/plugins/${encodeURIComponent(senPlugId)}`);
    }
    if (testPlugId) {
      await request.delete(`/api/admin/plugins/${encodeURIComponent(testPlugId)}`);
    }

    if (m1Seed?.fixture_id) {
      const delM1 = await request.delete(
        `/api/test/seed-milestone1-fixtures/${encodeURIComponent(m1Seed.fixture_id)}`
      );
      if (!delM1.ok()) {
        throw new Error(`Failed to clean up Milestone 1 fixture ${m1Seed.fixture_id}: ${delM1.status()}`);
      }
    }
    if (personaSeed?.fixture_id) {
      const delP = await request.delete(
        `/api/test/seed-personas/${encodeURIComponent(personaSeed.fixture_id)}`
      );
      if (!delP.ok()) {
        throw new Error(`Failed to clean up persona fixture ${personaSeed.fixture_id}: ${delP.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('opaquePlugIdAndUrlRouting_navigatingViewsUpdatesBrowserUrlAndOpeningSharedUrlInNewTabRestoresExactContext_test', async ({
    page,
    context,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded state');
    }

    // 1. Verify opaque <plugid> format (8-char lowercase alphanumeric, never containing reverse-DNS dots or manifest.id)
    expect(testPlugId).toMatch(/^[a-z0-9]{8}$/);
    expect(senPlugId).toMatch(/^[a-z0-9]{8}$/);
    expect(testPlugId).not.toBe(senPlugId);

    // Verify reinstalling senco_provision_plugin on the same school instance persists and returns the exact same plug_id from core.db
    const reinstallSenRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'senco_provision_plugin', staff_id: personaSeed.admin_staff_id },
    });
    expect(reinstallSenRes.ok()).toBe(true);
    const reinstallSenBody = await reinstallSenRes.json();
    expect(reinstallSenBody.plug_id).toBe(senPlugId);
    await request.post(`/api/admin/plugins/${encodeURIComponent(senPlugId)}/activate`, {
      data: { staff_id: personaSeed.admin_staff_id },
    });

    // 2. Verify GET /api/system/ui-config and GET /api/admin/plugins never expose reverse-DNS manifest.id strings
    const uiConfigRes = await request.get(
      `/api/system/ui-config?staff_id=${encodeURIComponent(personaSeed.hod_staff_id)}`
    );
    expect(uiConfigRes.ok()).toBe(true);
    const rawUiConfigText = await uiConfigRes.text();
    expect(rawUiConfigText).not.toContain('uk.org.optimis.sen.provision');
    expect(rawUiConfigText).not.toContain('uk.org.optimis.test');

    const uiConfigBody = JSON.parse(rawUiConfigText);
    const senNavEntry = uiConfigBody.plugins.find((p: { id: string }) => p.id === senPlugId);
    expect(senNavEntry).toBeDefined();
    expect(senNavEntry.entry_url).toBe(`/p/${senPlugId}/ui/index.html`);
    expect(Array.isArray(senNavEntry.views)).toBe(true);
    expect(senNavEntry.views).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          id: 'home',
          path: '/',
          entry: 'ui/dist/index.html',
        }),
        expect.objectContaining({
          id: 'student',
          path: '/student/:pupil_id',
          entry: 'ui/dist/index.html',
          query_params: expect.arrayContaining(['tab']),
        }),
      ])
    );

    const adminListRes = await request.get(
      `/api/admin/plugins?staff_id=${encodeURIComponent(personaSeed.admin_staff_id)}`
    );
    expect(adminListRes.ok()).toBe(true);
    const rawAdminListText = await adminListRes.text();
    expect(rawAdminListText).not.toContain('uk.org.optimis.sen.provision');
    expect(rawAdminListText).not.toContain('uk.org.optimis.test');

    // 3. Switch session to Head of Department / SENCO (`personaSeed.hod_staff_id`) and open the Host Shell
    await request.post('/api/auth/persona', {
      data: { staff_id: personaSeed.hod_staff_id },
    });

    const initialParams = new URLSearchParams({
      pupil_id: m1Seed.active_pupil_id,
      second_pupil_id: m1Seed.second_active_pupil_id,
      group_id: m1Seed.own_group_id,
      reference_date: m1Seed.reference_date,
      academic_year_id: m1Seed.academic_year_id,
    });
    await page.goto(`/?${initialParams.toString()}`);

    await expect(page.getByTestId('host-shell')).toBeVisible();

    // Verify DOM does not leak reverse-DNS manifest.id in data-testid or iframe src attributes
    const domHtml = await page.content();
    expect(domHtml).not.toContain('uk.org.optimis.sen.provision');
    expect(domHtml).not.toContain('uk.org.optimis.test');

    // 4. Open Global Search from the Narrow Icon Rail and select the SEN Provision plugin view using opaque senPlugId
    await page.getByTestId('nav-rail-search').click();
    const senNavBtn = page.getByTestId(`command-palette-result-view-${senPlugId}-home`);
    await expect(senNavBtn).toBeVisible();
    await senNavBtn.click();

    const senIframeEl = page.getByTestId(`plugin-iframe-${senPlugId}`);
    await expect(senIframeEl).toBeVisible();
    await expect(senIframeEl).toHaveAttribute('src', `/p/${senPlugId}/ui/index.html`);

    const senFrame = page.frameLocator(`[data-testid="plugin-iframe-${senPlugId}"]`);
    await expect(senFrame.getByTestId('sen-provision-code-badge')).toHaveText('E');

    // Log an IEP provision for the second pupil (Grace Turing, K / SPLD) so we can verify full state restoration in a new tab
    const pupilSelect = senFrame.getByTestId('sen-pupil-select');
    await expect(pupilSelect).toBeVisible();
    await pupilSelect.selectOption(m1Seed.second_active_pupil_id);

    // Selecting the student updates the Host URL path to /p/<senPlugId>/student/<second_active_pupil_id>
    await expect
      .poll(() => new URL(page.url()).pathname)
      .toBe(`/p/${senPlugId}/student/${m1Seed.second_active_pupil_id}`);

    // Switch to the IEP tab inside the SEN plugin and verify ?tab=iep is reflected in the browser URL
    const iepTabBtn = senFrame.getByTestId('sen-tab-btn-iep');
    await expect(iepTabBtn).toBeVisible();
    await iepTabBtn.click();

    await expect(senFrame.getByTestId('sen-active-tab-indicator')).toHaveText('iep');
    await expect.poll(() => new URL(page.url()).searchParams.get('tab')).toBe('iep');

    const currentBrowserUrl = new URL(page.url());
    expect(currentBrowserUrl.pathname).toBe(
      `/p/${senPlugId}/student/${m1Seed.second_active_pupil_id}`
    );
    expect(currentBrowserUrl.searchParams.get('tab')).toBe('iep');

    // Log an IEP target on that student while on the iep tab
    await senFrame.getByTestId('sen-iep-target-input').fill('Phonics chunking and coloured overlay for K pupil.');
    await senFrame.getByTestId('sen-classroom-adjustment-input').fill('Extra 25% time and front-row seating.');
    await senFrame.getByTestId('sen-ta-hours-input').fill('8');
    await senFrame.getByTestId('sen-assigned-ta-input').fill('Mr Thomas Wright');
    await senFrame.getByTestId('log-iep-provision-btn').click();
    await expect(senFrame.getByTestId('sen-total-ta-hours')).toHaveText('8');

    // 5. Copy the canonical deep-link URL (/p/<senPlugId>/student/<second_active_pupil_id>?tab=iep&reference_date=...&academic_year_id=...) and open it in a brand-new browser tab!
    const sharedUrl = `/p/${senPlugId}/student/${encodeURIComponent(
      m1Seed.second_active_pupil_id
    )}?tab=iep&reference_date=${encodeURIComponent(
      m1Seed.reference_date
    )}&academic_year_id=${encodeURIComponent(m1Seed.academic_year_id)}`;

    const colleagueTab = await context.newPage();
    colleagueTab.on('console', (msg) => {
      browserLogs.push(`[colleagueTab:${msg.type()}] ${msg.text()}`);
    });
    colleagueTab.on('pageerror', (err) => {
      pageErrors.push(`[colleagueTab:pageerror] ${err.stack || err.message}`);
    });

    const navRes = await colleagueTab.goto(sharedUrl);
    expect(navRes?.status()).toBe(200);

    // Verify colleagueTab did NOT reset to the default test plugin or home screen — it immediately mounted and activated senPlugId with student=second_active_pupil_id and tab=iep!
    await expect(colleagueTab.getByTestId('host-shell')).toBeVisible();
    await expect(colleagueTab.getByTestId(`plugin-iframe-${senPlugId}`)).toBeVisible();
    expect(new URL(colleagueTab.url()).pathname).toBe(
      `/p/${senPlugId}/student/${m1Seed.second_active_pupil_id}`
    );
    expect(new URL(colleagueTab.url()).searchParams.get('tab')).toBe('iep');

    const colleagueSenFrame = colleagueTab.frameLocator(`[data-testid="plugin-iframe-${senPlugId}"]`);
    await expect(colleagueSenFrame.getByTestId('sen-active-tab-indicator')).toHaveText('iep');
    await expect(colleagueSenFrame.getByTestId('sen-context-status')).toContainText(
      `PUPIL:${m1Seed.second_active_pupil_id}`
    );
    await expect(colleagueSenFrame.getByTestId('sen-context-status')).toContainText('TAB:iep');
    await expect(colleagueSenFrame.getByTestId('sen-provision-code-badge')).toHaveText('K');
    await expect(colleagueSenFrame.getByTestId('sen-primary-need-badge')).toHaveText('SPLD');
    await expect(colleagueSenFrame.getByTestId('sen-total-ta-hours')).toHaveText('8');

    await colleagueTab.close();

    // 6. Cross-Plugin Entity Deep-Linking (Issue 3): Navigating from senco_provision_plugin (at ?tab=iep) to the designated student handler plugin (testPlugId) wakes testPlugId, updates URL to /p/<testPlugId>/student/<second_active_pupil_id> without leaking source query params, and passes pupil context
    await page.evaluate(
      ({ targetPupilId, targetPluginId }) => {
        window.postMessage(
          {
            type: 'mis:link:navigate',
            entity_type: 'student',
            entity_id: targetPupilId,
            pupil_id: targetPupilId,
            target_plugin: targetPluginId,
          },
          '*'
        );
      },
      { targetPupilId: m1Seed.second_active_pupil_id, targetPluginId: testPlugId }
    );

    await expect(page.getByTestId(`plugin-iframe-${testPlugId}`)).toBeVisible();
    expect(new URL(page.url()).pathname).toBe(
      `/p/${testPlugId}/student/${m1Seed.second_active_pupil_id}`
    );
    expect(new URL(page.url()).searchParams.get('tab')).toBeNull();
  });

  test('defaultDenyManifestViews_undeclaredPathOrQueryParamOrLegacyRoutes_failsFastWithExplicitError_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded state');
    }

    // 1. Legacy routes (/plugins/{id}/ui/... and /api/plugins/{id}/rpc/...) must be removed and fail with 404
    const legacyUiRes = await request.get('/plugins/uk.org.optimis.sen.provision/ui/index.html');
    expect(legacyUiRes.status()).toBe(404);

    const legacyRpcRes = await request.post(
      '/api/plugins/uk.org.optimis.sen.provision/rpc/get_sen_differentiation_summary',
      {
        data: { args: {} },
      }
    );
    expect(legacyRpcRes.status()).toBe(404);

    // 2. API/non-HTML request to an undeclared view path fails fast with 404 UNDECLARED_PLUGIN_VIEW_PATH JSON
    const undeclaredPathRes = await request.get(`/p/${senPlugId}/undeclared/route`, {
      headers: { Accept: 'application/json' },
    });
    expect(undeclaredPathRes.status()).toBe(404);
    const undeclaredPathBody = await undeclaredPathRes.json();
    expect(undeclaredPathBody.ok).toBe(false);
    expect(undeclaredPathBody.error_code).toContain('UNDECLARED_PLUGIN_VIEW_PATH');

    // 2b. Direct browser navigation (page.goto, Accept: text/html) to undeclared view path returns HTTP 404 AND boots Host SPA with host-route-error-banner (Issue 2)
    const browserUndeclaredPathRes = await page.goto(`/p/${senPlugId}/undeclared/route`);
    expect(browserUndeclaredPathRes?.status()).toBe(404);
    await expect(page.getByTestId('host-route-error-banner')).toBeVisible();
    await expect(page.getByTestId('host-route-error-banner')).toContainText('UNDECLARED_PLUGIN_VIEW_PATH');

    // 3. API/non-HTML request with an undeclared query parameter fails fast with 400 UNDECLARED_PLUGIN_VIEW_QUERY_PARAM JSON
    const undeclaredQueryRes = await request.get(
      `/p/${senPlugId}/student/${encodeURIComponent(m1Seed.active_pupil_id)}?tab=iep&undeclared_param=1`,
      {
        headers: { Accept: 'application/json' },
      }
    );
    expect(undeclaredQueryRes.status()).toBe(400);
    const undeclaredQueryBody = await undeclaredQueryRes.json();
    expect(undeclaredQueryBody.ok).toBe(false);
    expect(undeclaredQueryBody.error_code).toContain('UNDECLARED_PLUGIN_VIEW_QUERY_PARAM');

    // 3b. Direct browser navigation (page.goto, Accept: text/html) with undeclared query parameter returns HTTP 400 AND boots Host SPA with host-route-error-banner (Issue 2)
    const browserUndeclaredQueryRes = await page.goto(
      `/p/${senPlugId}/student/${encodeURIComponent(m1Seed.active_pupil_id)}?tab=iep&undeclared_param=1`
    );
    expect(browserUndeclaredQueryRes?.status()).toBe(400);
    await expect(page.getByTestId('host-route-error-banner')).toBeVisible();
    await expect(page.getByTestId('host-route-error-banner')).toContainText('UNDECLARED_PLUGIN_VIEW_QUERY_PARAM');

    // 4. Attempting to set an undeclared route path or query param via postMessage from inside the browser/plugin surfaces host-route-error-banner
    const validNavRes = await page.goto(
      `/p/${senPlugId}/student/${encodeURIComponent(m1Seed.active_pupil_id)}?tab=iep`
    );
    expect(validNavRes?.status()).toBe(200);
    await expect(page.getByTestId(`plugin-iframe-${senPlugId}`)).toBeVisible();

    await page.evaluate((plugId) => {
      window.postMessage(
        {
          type: 'mis:route:state',
          plug_id: plugId,
          path: '/student/4092',
          query: { tab: 'iep', forbidden_param: 'evil' },
        },
        '*'
      );
    }, senPlugId);

    const routeErrorBanner = page.getByTestId('host-route-error-banner');
    await expect(routeErrorBanner).toBeVisible();
    await expect(routeErrorBanner).toContainText('UNDECLARED_PLUGIN_VIEW_QUERY_PARAM');
  });
});
