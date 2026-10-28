import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

test.describe('Plugin UI SDK (mis-bridge.js), Design Token Theming (--mis-*) & Default Dashboard Layout (Sections 6C, 9.4 & 9.5)', () => {
  let seededPupilId: string = '';
  let secondPupilId: string = '';
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    seededPupilId = '';
    secondPupilId = '';
    logCursor = '';

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

    // 3. Ensure dev persona switcher is enabled and plugin uk.org.optimis.test is ACTIVE
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });

    // 4. Seed two isolated test pupils for RPC and Entity Deep-Linking verification
    const res1 = await request.post('/api/test/seed-pupil', {
      data: {
        upn: 'T100000000001',
        first_name: 'Ada',
        last_name: 'Lovelace',
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
    seededPupilId = body1.pupil_id;

    const res2 = await request.post('/api/test/seed-pupil', {
      data: {
        upn: 'T100000000002',
        first_name: 'Alan',
        last_name: 'Turing',
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
    secondPupilId = body2.pupil_id;
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

    // 3. Clean up seeded pupil records at the end of the test
    if (seededPupilId) {
      const del1 = await request.delete(`/api/test/seed-pupil/${encodeURIComponent(seededPupilId)}`);
      if (!del1.ok()) {
        throw new Error(`Failed to clean up seeded pupil ${seededPupilId}: ${del1.status()}`);
      }
    }
    if (secondPupilId) {
      const del2 = await request.delete(`/api/test/seed-pupil/${encodeURIComponent(secondPupilId)}`);
      if (!del2.ok()) {
        throw new Error(`Failed to clean up second pupil ${secondPupilId}: ${del2.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('misInitAndThemeTokens_defaultDashboardAndHighContrastToggle_injectsCssVariablesIntoHostAndSandboxedIframe_test', async ({
    page,
  }) => {
    // 1. Load Default Dashboard without query params first to verify unconditional mis:init on default dashboard layout
    await page.goto('/');

    const hostShell = page.getByTestId('host-shell');
    await expect(hostShell).toBeVisible();
    await expect(page.getByTestId('host-sidebar')).toBeVisible();
    await expect(page.getByTestId('host-school-name')).toBeVisible();
    await expect(page.getByTestId('host-topbar')).toBeVisible();
    await expect(page.getByTestId('host-dashboard-grid')).toBeVisible();
    await expect(page.getByTestId('dashboard-summary-strip')).toBeVisible();
    await expect(page.getByTestId('nav-item-plugin-uk.org.optimis.test')).toBeVisible();

    const themeIndicator = page.getByTestId('active-theme-mode-indicator');
    await expect(themeIndicator).toHaveText('light');

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    const sdkThemeStatus = pluginFrame.getByTestId('sdk-theme-status');
    await expect(sdkThemeStatus).toBeVisible();
    await expect(sdkThemeStatus).toContainText('THEME:light');
    await expect(sdkThemeStatus).toContainText('#1d4ed8');

    // Verify Host root CSS custom properties in light mode
    const hostLightTokens = await page.evaluate(() => {
      const style = getComputedStyle(document.documentElement);
      return {
        themeAttr: document.documentElement.getAttribute('data-mis-theme'),
        brandPrimary: style.getPropertyValue('--mis-brand-primary').trim(),
        bgCanvas: style.getPropertyValue('--mis-bg-canvas').trim(),
        textPrimary: style.getPropertyValue('--mis-text-primary').trim(),
      };
    });
    expect(hostLightTokens.themeAttr).toBe('light');
    expect(hostLightTokens.brandPrimary).toBe('#1d4ed8');
    expect(hostLightTokens.bgCanvas).toBe('#f1f5f9');
    expect(hostLightTokens.textPrimary).toBe('#0f172a');

    // 2. Toggle High-Contrast Mode in Host Topbar and verify live propagation into both Host and sandboxed iframe
    const toggleThemeBtn = page.getByTestId('theme-mode-toggle-btn');
    await expect(toggleThemeBtn).toBeVisible();
    await toggleThemeBtn.click();

    await expect(themeIndicator).toHaveText('high-contrast');
    await expect(sdkThemeStatus).toContainText('THEME:high-contrast');
    await expect(sdkThemeStatus).toContainText('#facc15');

    const hostHighContrastTokens = await page.evaluate(() => {
      const style = getComputedStyle(document.documentElement);
      return {
        themeAttr: document.documentElement.getAttribute('data-mis-theme'),
        brandPrimary: style.getPropertyValue('--mis-brand-primary').trim(),
        bgCanvas: style.getPropertyValue('--mis-bg-canvas').trim(),
        textPrimary: style.getPropertyValue('--mis-text-primary').trim(),
      };
    });
    expect(hostHighContrastTokens.themeAttr).toBe('high-contrast');
    expect(hostHighContrastTokens.brandPrimary).toBe('#facc15');
    expect(hostHighContrastTokens.bgCanvas).toBe('#000000');
    expect(hostHighContrastTokens.textPrimary).toBe('#ffffff');
  });

  test('windowMisRpcPromiseApi_validAndInvalidCalls_resolvesDataAndRejectsFailFastWithErrorCode_test', async ({
    page,
  }) => {
    await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}`);

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');

    // 1. Invoke window.mis.rpc('get_pupil_record', { pupil_id }) Promise API
    const promiseRpcBtn = pluginFrame.getByTestId('sdk-promise-rpc-btn');
    await expect(promiseRpcBtn).toBeVisible();
    await promiseRpcBtn.click();

    const promiseRpcResult = pluginFrame.getByTestId('sdk-promise-rpc-result');
    await expect(promiseRpcResult).toHaveText(`PROMISE_OK:Ada Lovelace:T100000000001`);

    // 2. Invoke window.mis.rpc with missing/invalid arguments or undeclared method to verify fail-fast Promise rejection
    const promiseRpcErrorBtn = pluginFrame.getByTestId('sdk-promise-rpc-error-btn');
    await expect(promiseRpcErrorBtn).toBeVisible();
    await promiseRpcErrorBtn.click();

    await expect(promiseRpcResult).toContainText('PROMISE_ERR:');
    await expect(promiseRpcResult).toContainText('missing mandatory parameter: pupil_id');
  });

  test('dataMisLinkDeepLink_clickingStudentAndGroupLinksInIframe_wakesDesignatedHandlerAndUpdatesContext_test', async ({
    page,
  }) => {
    await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}&second_pupil_id=${encodeURIComponent(secondPupilId)}`);

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    const contextStatus = pluginFrame.getByTestId('sdk-context-status');
    await expect(contextStatus).toContainText(`PUPIL:${seededPupilId}`);

    // 1. Click <a data-mis-link="student" data-pupil-id="<secondPupilId>"> inside the plugin iframe
    const studentDeepLink = pluginFrame.getByTestId('deep-link-student-anchor');
    await expect(studentDeepLink).toBeVisible();
    await expect(studentDeepLink).toHaveAttribute('data-mis-link', 'student');
    await studentDeepLink.click();

    // Verify Host Shell intercepts mis:link:navigate, resolves Student Handler Plugin (uk.org.optimis.test), and updates context
    const deepLinkStatus = page.getByTestId('deep-link-navigation-status');
    await expect(deepLinkStatus).toContainText(`student:${secondPupilId}`);
    await expect(deepLinkStatus).toContainText('uk.org.optimis.test');

    const activeEntityPill = page.getByTestId('active-entity-context-pill');
    await expect(activeEntityPill).toContainText(`student:${secondPupilId}`);

    // Verify child plugin iframe received updated context with secondPupilId without losing iframe mount
    await expect(contextStatus).toContainText(`PUPIL:${secondPupilId}`);

    // 2. Click <a data-mis-link="group" data-group-id="grp_year7_7A"> inside the plugin iframe
    const groupDeepLink = pluginFrame.getByTestId('deep-link-group-anchor');
    await expect(groupDeepLink).toBeVisible();
    await expect(groupDeepLink).toHaveAttribute('data-mis-link', 'group');
    await groupDeepLink.click();

    await expect(deepLinkStatus).toContainText('group:grp_year7_7A');
    await expect(contextStatus).toContainText('GROUP:grp_year7_7A');
  });

  test('connectSrcNoneCsp_sandboxedIframeAttemptsDirectExternalFetch_blocksNetworkRequest_test', async ({
    page,
  }) => {
    await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}`);

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    const cspTestBtn = pluginFrame.getByTestId('sdk-csp-fetch-test-btn');
    await expect(cspTestBtn).toBeVisible();
    await cspTestBtn.click();

    const cspStatus = pluginFrame.getByTestId('sdk-csp-fetch-status');
    await expect(cspStatus).toHaveText('CSP_BLOCKED');
  });
});
