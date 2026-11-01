import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

interface SeededPersonaFixtures {
  fixture_id: string;
  pupil_id: string;
  admin_staff_id: string;
  dsl_staff_id: string;
  hod_staff_id: string;
  teacher_staff_id: string;
  terminated_staff_id: string;
}

test.describe('In-UI Topbar Persona Switcher (Section 8A)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let seeded: SeededPersonaFixtures | null = null;
  let testPlugId: string = '';

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    seeded = null;
    testPlugId = '';

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

    // 3. Ensure dev persona switcher mode is enabled and install + activate test_plugin
    const modeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!modeRes.ok()) {
      throw new Error(`Failed to enable dev persona switcher: ${modeRes.status()}`);
    }
    const installRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'test_plugin' },
    });
    if (!installRes.ok()) {
      throw new Error(`Failed to install test_plugin: ${installRes.status()}`);
    }
    testPlugId = (await installRes.json()).plug_id;
    const activateRes = await request.post('/api/admin/plugins/uk.org.optimis.test/activate', {
      data: {},
    });
    if (!activateRes.ok()) {
      throw new Error(`Failed to activate uk.org.optimis.test: ${activateRes.status()}`);
    }

    // 4. Seed isolated staff personas (Admin/DPO, DSL, HoD, Teacher, Terminated) & pupil fixture
    const seedRes = await request.post('/api/test/seed-personas', {
      data: {
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    if (!seedRes.ok()) {
      throw new Error(`Failed to seed persona fixtures: ${seedRes.status()}`);
    }
    const body = (await seedRes.json()) as SeededPersonaFixtures;
    if (
      !body.fixture_id ||
      !body.pupil_id ||
      !body.admin_staff_id ||
      !body.dsl_staff_id ||
      !body.hod_staff_id ||
      !body.teacher_staff_id ||
      !body.terminated_staff_id
    ) {
      throw new Error('Seeded persona fixture response missing mandatory identifiers (including hod_staff_id)');
    }
    seeded = body;

    // 5. Seed a safeguarding postbox note as Teacher so the DSL Vault console has an envelope to display when switched to DSL
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.teacher_staff_id },
    });
    if (!switchTeacherRes.ok()) {
      throw new Error(`Failed to switch to Teacher persona in setup: ${switchTeacherRes.status()}`);
    }

    const postboxRes = await request.post('/api/safeguarding/postbox', {
      data: {
        pupil_id: seeded.pupil_id,
        assigned_dsl_staff_id: seeded.dsl_staff_id,
        concern_category: 'WELFARE',
        incident_date: '2026-10-28',
        summary: 'Topbar persona switcher live DSL vault verification note',
        body_map: 'NONE',
        verbatim_account: 'Pupil requested to speak with pastoral lead during morning tutorial.',
        action_taken: 'Logged via write-only safeguarding postbox for DSL review',
      },
    });
    if (!postboxRes.ok()) {
      throw new Error(`Failed to seed safeguarding postbox note in setup: ${postboxRes.status()}`);
    }

    // Switch initial active persona to Admin so selecting Teacher in step 1 triggers a live onChange event
    await request.post('/api/auth/persona', {
      data: { staff_id: seeded.admin_staff_id },
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

    // 3. Restore dev persona switcher mode to true and remove runtime-installed plugin
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete('/api/admin/plugins/uk.org.optimis.test');

    // 4. Clean up seeded persona, safeguarding, and pupil fixtures
    if (seeded?.fixture_id) {
      const delRes = await request.delete(`/api/test/seed-personas/${encodeURIComponent(seeded.fixture_id)}`);
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up persona fixture ${seeded.fixture_id}: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('switchPersonaInTopbarDropdown_switchingAcrossTeacherHodDslAndAdmin_updatesSidebarPluginsMisInitScopesAndDslSarConsolesLive_test', async ({
    page,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded persona state');
    }

    // =========================================================================
    // 1. Open as Classroom Teacher (TEACHER) via ?__persona= override
    // =========================================================================
    await page.goto(
      `/?pupil_id=${encodeURIComponent(seeded.pupil_id)}&__persona=${encodeURIComponent(
        seeded.teacher_staff_id
      )}`
    );

    // Host topbar and in-UI persona switcher must be completely removed
    await expect(page.getByTestId('host-topbar')).toHaveCount(0);
    await expect(page.getByTestId('persona-switcher-container')).toHaveCount(0);
    await expect(page.getByTestId('persona-switcher-select')).toHaveCount(0);

    const dslVaultPanel = page.getByTestId('dsl-safeguarding-vault-panel');
    const sarRedactionPanel = page.getByTestId('sar-redaction-panel');
    const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);

    // Neither DSL Vault console nor Admin/DPO SAR console may be visible for Teacher
    await expect(dslVaultPanel).toHaveCount(0);
    await expect(sarRedactionPanel).toHaveCount(0);

    // Verify mis:init populated window.mis.permittedMethods and window.mis.scopes inside the sandboxed iframe for TEACHER
    await expect
      .poll(
        async () =>
          await pluginFrame.getByTestId('plugin-root').evaluate(() => {
            const w = window as unknown as {
              mis: {
                permittedMethods: string[];
                scopes: string[];
                activeRoles: string[];
                context: { staff_id: string };
              };
            };
            return {
              staffId: w.mis.context.staff_id,
              roles: w.mis.activeRoles,
              hasReadAllPupils: w.mis.scopes.includes('dal.pupils.read:all'),
              hasReadOwnGroups: w.mis.scopes.includes('dal.pupils.read:own_groups'),
              methodCount: w.mis.permittedMethods.length,
              hasGetPupilRecord: w.mis.permittedMethods.includes('get_pupil_record'),
            };
          }),
        { timeout: 5000 }
      )
      .toEqual({
        staffId: seeded.teacher_staff_id,
        roles: ['TEACHER'],
        hasReadAllPupils: false,
        hasReadOwnGroups: true,
        methodCount: 9,
        hasGetPupilRecord: false,
      });

    // Calling get_pupil_record (requires dal.pupils.read:all) from the iframe as Teacher must fail fast
    await pluginFrame.getByTestId('print-record-btn').click();
    await expect(pluginFrame.getByTestId('rpc-error-banner')).toBeVisible();
    await expect(pluginFrame.getByTestId('rpc-error-banner')).toContainText('caller lacks required DAL scopes');

    // =========================================================================
    // 2. Switch to Head of Department (HOD) via ?__persona= override
    // =========================================================================
    await page.goto(
      `/?pupil_id=${encodeURIComponent(seeded.pupil_id)}&__persona=${encodeURIComponent(
        seeded.hod_staff_id
      )}`
    );

    // Neither DSL Vault console nor Admin/DPO SAR console is visible for HoD
    await expect(dslVaultPanel).toHaveCount(0);
    await expect(sarRedactionPanel).toHaveCount(0);

    // Verify mis:init updated window.mis.permittedMethods and window.mis.scopes for HoD
    await expect
      .poll(
        async () =>
          await pluginFrame.getByTestId('plugin-root').evaluate(() => {
            const w = window as unknown as {
              mis: {
                permittedMethods: string[];
                scopes: string[];
                activeRoles: string[];
                context: { staff_id: string };
              };
            };
            return {
              staffId: w.mis.context.staff_id,
              roles: w.mis.activeRoles,
              hasReadAllPupils: w.mis.scopes.includes('dal.pupils.read:all'),
              methodCount: w.mis.permittedMethods.length,
              hasGetPupilRecord: w.mis.permittedMethods.includes('get_pupil_record'),
            };
          }),
        { timeout: 5000 }
      )
      .toEqual({
        staffId: seeded.hod_staff_id,
        roles: ['HOD'],
        hasReadAllPupils: true,
        methodCount: 12,
        hasGetPupilRecord: true,
      });

    // Clicking print-record-btn succeeds under HoD session
    await pluginFrame.getByTestId('print-record-btn').click();
    await expect(pluginFrame.getByTestId('rpc-error-banner')).toBeHidden();
    const expectedLogSubstring = `[plugin:uk.org.optimis.test] {"id":"${seeded.pupil_id}"`;
    await expect
      .poll(() => browserLogs.some((entry) => entry.includes(expectedLogSubstring)), {
        timeout: 5000,
      })
      .toBe(true);

    // =========================================================================
    // 3. Switch to Designated Safeguarding Lead (DSL) via ?__persona= override
    // =========================================================================
    await page.goto(
      `/?pupil_id=${encodeURIComponent(seeded.pupil_id)}&__persona=${encodeURIComponent(
        seeded.dsl_staff_id
      )}`
    );

    // DSL Vault Console appears; Admin/DPO SAR Console remains hidden
    await expect(dslVaultPanel).toBeVisible();
    await expect(sarRedactionPanel).toHaveCount(0);

    // Verify mis:init updated scopes inside iframe to include dal.safeguarding.read & get_pupil_guardians
    await expect
      .poll(
        async () =>
          await pluginFrame.getByTestId('plugin-root').evaluate(() => {
            const w = window as unknown as {
              mis: {
                permittedMethods: string[];
                scopes: string[];
                activeRoles: string[];
                context: { staff_id: string };
              };
            };
            return {
              staffId: w.mis.context.staff_id,
              roles: w.mis.activeRoles,
              hasSafeguardingRead: w.mis.scopes.includes('dal.safeguarding.read'),
              hasGetGuardians: w.mis.permittedMethods.includes('get_pupil_guardians'),
            };
          }),
        { timeout: 5000 }
      )
      .toEqual({
        staffId: seeded.dsl_staff_id,
        roles: ['DSL'],
        hasSafeguardingRead: true,
        hasGetGuardians: true,
      });

    // DSL can renew OAuth Cloud Key in the DSL Vault panel
    const renewCloudKeyBtn = page.getByTestId('renew-dsl-cloud-key-btn');
    await expect(renewCloudKeyBtn).toBeVisible();
    await renewCloudKeyBtn.click();
    await expect(page.getByTestId('dsl-cloud-key-input')).not.toHaveValue('');

    // =========================================================================
    // 4. Switch to System Administrator / DPO (ADMIN) via ?__persona= override
    // =========================================================================
    await page.goto(
      `/?pupil_id=${encodeURIComponent(seeded.pupil_id)}&__persona=${encodeURIComponent(
        seeded.admin_staff_id
      )}`
    );

    // DSL Vault Console hides; Admin/DPO SAR Redaction Console appears!
    await expect(dslVaultPanel).toHaveCount(0);
    await expect(sarRedactionPanel).toBeVisible();

    // Admin can load the SAR export package in the SAR Redaction Console
    const loadSarBtn = page.getByTestId('sar-load-export-btn');
    await expect(loadSarBtn).toBeVisible();
    await loadSarBtn.click();
    await expect(page.getByTestId('sar-export-status')).toHaveText('SAR_LOADED');
    await expect(page.getByTestId('sar-section-core-Core Pupil Profile & Enrolments')).toBeVisible();
  });

  test('topbarPersonaSwitcher_whenDevPersonaSwitcherDisabled_hidesPersonaSwitcherInTopbar_test', async ({
    page,
    request,
  }) => {
    // Disable dev persona switcher mode on the Host
    const modeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: false },
    });
    expect(modeRes.status()).toBe(500);

    await page.goto('/');
    await expect(page.getByTestId('host-shell')).toBeVisible();
    await expect(page.getByTestId('host-topbar')).toHaveCount(0);
    await expect(page.getByTestId('persona-switcher-container')).toHaveCount(0);
    await expect(page.getByTestId('persona-switcher-select')).toHaveCount(0);
  });
});

