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

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    seeded = null;

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

    // 3. Ensure dev persona switcher mode is enabled and test plugin is ACTIVE
    const modeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!modeRes.ok()) {
      throw new Error(`Failed to enable dev persona switcher: ${modeRes.status()}`);
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
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Restore dev persona switcher mode to true after test
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });

    // 2. Fetch Backend Service Logs emitted during this test
    if (logCursor) {
      const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
      if (logsRes.ok()) {
        const serviceLogs: string[] = (await logsRes.json()).entries;
        if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
          console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
        }
      }
    }

    // 3. Dump tapped Browser Console logs on failure or error
    if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
      console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
      console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
    }

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

    await page.goto(`/?pupil_id=${encodeURIComponent(seeded.pupil_id)}`);

    const hostTopbar = page.getByTestId('host-topbar');
    await expect(hostTopbar).toBeVisible();

    const switcherContainer = page.getByTestId('persona-switcher-container');
    await expect(switcherContainer).toBeVisible();

    const personaSelect = page.getByTestId('persona-switcher-select');
    await expect(personaSelect).toBeVisible();

    // Verify active personas (Teacher, HoD, DSL, Admin) are present in dropdown options, and Terminated staff is excluded
    await expect(page.getByTestId(`persona-option-${seeded.teacher_staff_id}`)).toBeAttached();
    await expect(page.getByTestId(`persona-option-${seeded.hod_staff_id}`)).toBeAttached();
    await expect(page.getByTestId(`persona-option-${seeded.dsl_staff_id}`)).toBeAttached();
    await expect(page.getByTestId(`persona-option-${seeded.admin_staff_id}`)).toBeAttached();
    await expect(page.getByTestId(`persona-option-${seeded.terminated_staff_id}`)).toHaveCount(0);

    const activePersonaBadge = page.getByTestId('active-persona-badge');
    const sidebarPluginBtn = page.getByTestId('nav-item-plugin-uk.org.optimis.test');
    const sidebarPluginMethodsCount = page.getByTestId('nav-item-plugin-methods-count-uk.org.optimis.test');
    const dslVaultPanel = page.getByTestId('dsl-safeguarding-vault-panel');
    const sarRedactionPanel = page.getByTestId('sar-redaction-panel');
    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');

    // =========================================================================
    // 1. Switch to Classroom Teacher (TEACHER)
    // =========================================================================
    await personaSelect.selectOption(seeded.teacher_staff_id);
    await expect(activePersonaBadge).toContainText(seeded.teacher_staff_id);
    await expect(activePersonaBadge).toContainText('TEACHER');

    // Sidebar plugin badge reflects 8 permitted methods for TEACHER (only :own_groups & dal.safeguarding.create)
    await expect(sidebarPluginBtn).toBeVisible();
    await expect(sidebarPluginMethodsCount).toHaveText('8');
    await expect(sidebarPluginBtn).toHaveAttribute(
      'data-permitted-methods',
      'get_group_roster_summary,list_worker_event_logs,mutate_then_illegal_egress,push_external_sparx_post,record_group_am_attendance,submit_safeguarding_postbox,sync_external_sparx_get,trigger_loop_relay'
    );

    // Neither DSL Vault console nor Admin/DPO SAR console may be visible for Teacher
    await expect(dslVaultPanel).toHaveCount(0);
    await expect(sarRedactionPanel).toHaveCount(0);

    // Verify mis:init updated window.mis.permittedMethods and window.mis.scopes live inside the sandboxed iframe
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
        methodCount: 8,
        hasGetPupilRecord: false,
      });

    // Calling get_pupil_record (requires dal.pupils.read:all) from the iframe as Teacher must fail fast
    await pluginFrame.getByTestId('print-record-btn').click();
    await expect(pluginFrame.getByTestId('rpc-error-banner')).toBeVisible();
    await expect(pluginFrame.getByTestId('rpc-error-banner')).toContainText('caller lacks required DAL scopes');

    // =========================================================================
    // 2. Switch live to Head of Department (HOD)
    // =========================================================================
    await personaSelect.selectOption(seeded.hod_staff_id);
    await expect(activePersonaBadge).toContainText(seeded.hod_staff_id);
    await expect(activePersonaBadge).toContainText('HOD');

    // Sidebar plugin badge updates live to 12 permitted methods (HOD holds dal.pupils.read:all & dal.groups.read:all)
    await expect(sidebarPluginMethodsCount).toHaveText('12');

    // Neither DSL Vault console nor Admin/DPO SAR console is visible for HoD
    await expect(dslVaultPanel).toHaveCount(0);
    await expect(sarRedactionPanel).toHaveCount(0);

    // Verify mis:init updated window.mis.permittedMethods and window.mis.scopes live for HoD
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

    // Clicking print-record-btn now succeeds under HoD session without page reload
    await pluginFrame.getByTestId('print-record-btn').click();
    await expect(pluginFrame.getByTestId('rpc-error-banner')).toBeHidden();
    const expectedLogSubstring = `[plugin:uk.org.optimis.test] {"id":"${seeded.pupil_id}"`;
    await expect
      .poll(() => browserLogs.some((entry) => entry.includes(expectedLogSubstring)), {
        timeout: 5000,
      })
      .toBe(true);

    // =========================================================================
    // 3. Switch live to Designated Safeguarding Lead (DSL)
    // =========================================================================
    await personaSelect.selectOption(seeded.dsl_staff_id);
    await expect(activePersonaBadge).toContainText(seeded.dsl_staff_id);
    await expect(activePersonaBadge).toContainText('DSL');

    // DSL Vault Console appears live; Admin/DPO SAR Console remains hidden
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

    // DSL can renew OAuth Cloud Key in the live DSL Vault panel
    const renewCloudKeyBtn = page.getByTestId('renew-dsl-cloud-key-btn');
    await expect(renewCloudKeyBtn).toBeVisible();
    await renewCloudKeyBtn.click();
    await expect(page.getByTestId('dsl-cloud-key-input')).not.toHaveValue('');

    // =========================================================================
    // 4. Switch live to System Administrator / DPO (ADMIN)
    // =========================================================================
    await personaSelect.selectOption(seeded.admin_staff_id);
    await expect(activePersonaBadge).toContainText(seeded.admin_staff_id);
    await expect(activePersonaBadge).toContainText('ADMIN');

    // Sidebar plugin badge updates live to 13 permitted methods for Admin
    await expect(sidebarPluginMethodsCount).toHaveText('13');

    // DSL Vault Console hides immediately; Admin/DPO SAR Redaction Console appears live!
    await expect(dslVaultPanel).toHaveCount(0);
    await expect(sarRedactionPanel).toBeVisible();

    // Admin can load the SAR export package live in the newly revealed SAR Redaction Console
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
    await expect(page.getByTestId('host-topbar')).toBeVisible();

    // Persona switcher dropdown must NOT be rendered when ENABLE_DEV_PERSONA_SWITCHER=false
    await expect(page.getByTestId('persona-switcher-container')).toHaveCount(0);
    await expect(page.getByTestId('persona-switcher-select')).toHaveCount(0);
  });
});
