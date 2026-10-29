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

test.describe('SENCO & Classroom Differentiation Plugin Vertical Slice (uk.org.optimis.sen.provision)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let m1Seed: Milestone1SeedResponse | null = null;
  let personaSeed: PersonaSeedResponse | null = null;

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    m1Seed = null;
    personaSeed = null;

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

    // 4. Seed Admin/DPO/HoD/DSL/Teacher persona fixtures and Milestone 1 fixtures (with both E and K SEN pupils)
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
    if (
      !m1Seed.fixture_id ||
      !m1Seed.teacher_staff_id ||
      !m1Seed.own_group_id ||
      !m1Seed.other_group_id ||
      !m1Seed.active_pupil_id ||
      !m1Seed.second_active_pupil_id ||
      !m1Seed.other_group_pupil_id
    ) {
      throw new Error('Milestone 1 seed response missing mandatory identifiers');
    }

    // 5. Switch to Admin persona to install and activate both test_plugin (student deep-link handler) and senco_provision_plugin (uk.org.optimis.sen.provision)
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
    const activateTestRes = await request.post('/api/admin/plugins/uk.org.optimis.test/activate', {
      data: { staff_id: personaSeed.admin_staff_id },
    });
    if (!activateTestRes.ok()) {
      throw new Error(`Failed to activate uk.org.optimis.test: ${activateTestRes.status()}`);
    }

    const installSenRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'senco_provision_plugin', staff_id: personaSeed.admin_staff_id },
    });
    if (!installSenRes.ok()) {
      throw new Error(
        `Failed to install senco_provision_plugin: ${installSenRes.status()} ${await installSenRes.text()}`
      );
    }
    const activateSenRes = await request.post(
      '/api/admin/plugins/uk.org.optimis.sen.provision/activate',
      {
        data: { staff_id: personaSeed.admin_staff_id },
      }
    );
    if (!activateSenRes.ok()) {
      throw new Error(`Failed to activate uk.org.optimis.sen.provision: ${activateSenRes.status()}`);
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Clean up sen_provisions in plugin_sen_provision.db
    if (personaSeed && m1Seed) {
      await request.post('/api/auth/persona', {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      await request.post('/api/admin/plugins/uk.org.optimis.sen.provision/activate', {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      for (const pid of [
        m1Seed.active_pupil_id,
        m1Seed.second_active_pupil_id,
        m1Seed.transferred_pupil_id,
        m1Seed.other_group_pupil_id,
      ]) {
        await request.post('/api/plugins/uk.org.optimis.sen.provision/rpc/purge_pupil_data', {
          data: { args: { pupil_id: pid } },
        });
      }
    }

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

    // 4. Remove runtime-installed plugins
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete('/api/admin/plugins/uk.org.optimis.sen.provision');
    await request.delete('/api/admin/plugins/uk.org.optimis.test');

    // 5. Clean up seeded fixtures in core.db
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

  test('sencoLogsIepAndTaAllocation_teacherViewsDifferentiationWidgetAndClicksStudentBadgeToWakeKeepAlivePool_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded state');
    }

    // 1. Verify GET /api/system/ui-config exposes uk.org.optimis.sen.provision and its sen_differentiation widget
    const uiConfigRes = await request.get(
      `/api/system/ui-config?staff_id=${encodeURIComponent(personaSeed.hod_staff_id)}`
    );
    expect(uiConfigRes.ok()).toBe(true);
    const uiConfigBody = await uiConfigRes.json();
    expect(uiConfigBody.ok).toBe(true);

    const senPlugin = uiConfigBody.plugins.find(
      (p: { id: string }) => p.id === 'uk.org.optimis.sen.provision'
    );
    expect(senPlugin).toBeDefined();
    expect(senPlugin.permitted_methods).toEqual(
      expect.arrayContaining(['get_sen_differentiation_summary', 'log_iep_provision'])
    );
    expect(senPlugin.widgets).toEqual([
      expect.objectContaining({
        id: 'sen_differentiation',
        plugin_id: 'uk.org.optimis.sen.provision',
        entry: 'ui/dist/widget.html',
        entry_url: '/plugins/uk.org.optimis.sen.provision/ui/widget.html',
      }),
    ]);

    // Also verify Classroom Teacher only sees `get_sen_differentiation_summary` in permitted_methods (since `log_iep_provision` requires `dal.sen.read:all`)
    const teacherUiConfigRes = await request.get(
      `/api/system/ui-config?staff_id=${encodeURIComponent(m1Seed.teacher_staff_id)}`
    );
    expect(teacherUiConfigRes.ok()).toBe(true);
    const teacherUiConfigBody = await teacherUiConfigRes.json();
    const teacherSenPlugin = teacherUiConfigBody.plugins.find(
      (p: { id: string }) => p.id === 'uk.org.optimis.sen.provision'
    );
    expect(teacherSenPlugin).toBeDefined();
    expect(teacherSenPlugin.permitted_methods).toEqual(['get_sen_differentiation_summary']);

    // 2. Open Host UI Shell as SENCO / HoD (holding `dal.sen.read:all` & `dal.pupils.read:all`)
    const sencoParams = new URLSearchParams({
      pupil_id: m1Seed.active_pupil_id,
      second_pupil_id: m1Seed.second_active_pupil_id,
      group_id: m1Seed.own_group_id,
      other_group_id: m1Seed.other_group_id,
      reference_date: m1Seed.reference_date,
      academic_year_id: m1Seed.academic_year_id,
      staff_id: personaSeed.hod_staff_id,
    });
    await page.goto(`/?${sencoParams.toString()}`);

    await expect(page.getByTestId('host-shell')).toBeVisible();

    // Place the `sen_differentiation` Mini-Iframe Widget onto the Home Dashboard Grid FIRST so it is mounted before SENCO logs IEP provisions
    const editModeToggleBtn = page.getByTestId('dashboard-edit-mode-toggle-btn');
    await expect(editModeToggleBtn).toBeVisible();
    await editModeToggleBtn.click();

    const placeSenWidgetBtn = page.getByTestId('place-widget-btn-sen_differentiation');
    await expect(placeSenWidgetBtn).toBeVisible();
    await placeSenWidgetBtn.click();

    await editModeToggleBtn.click();

    const widgetIframeEl = page.getByTestId('widget-iframe-sen_differentiation');
    await expect(widgetIframeEl).toBeVisible();
    await expect(widgetIframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const widgetFrame = page.frameLocator('[data-testid="widget-iframe-sen_differentiation"]');
    // Both K (Grace Turing) and E (Ada Lovelace) pupils in own_group_id should be surfaced in the widget
    await expect(widgetFrame.getByTestId('widget-sen-pupil-count')).toHaveText('2');
    await expect(
      widgetFrame.getByTestId(`widget-sen-badge-${m1Seed.active_pupil_id}`)
    ).toContainText('E (ASD)');
    await expect(
      widgetFrame.getByTestId(`widget-sen-badge-${m1Seed.second_active_pupil_id}`)
    ).toContainText('K (SPLD)');

    // 3. SENCO navigates to the SEN Provision workspace tab and logs an IEP classroom adjustment & TA allocation for the E (EHCP) pupil (`active_pupil_id`, Ada Lovelace)
    const senNavBtn = page.getByTestId('nav-item-plugin-uk.org.optimis.sen.provision');
    await expect(senNavBtn).toBeVisible();
    await senNavBtn.click();

    const senIframeEl = page.getByTestId('plugin-iframe-uk.org.optimis.sen.provision');
    await expect(senIframeEl).toBeVisible();
    await expect(senIframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const senFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.sen.provision"]');

    await expect(senFrame.getByTestId('sen-provision-code-badge')).toHaveText('E');
    await expect(senFrame.getByTestId('sen-primary-need-badge')).toHaveText('ASD');
    await expect(senFrame.getByTestId('sen-core-strategies')).toContainText(
      'Chunk instructions and provide visual timetable'
    );

    const iepTargetInput = senFrame.getByTestId('sen-iep-target-input');
    const adjustmentInput = senFrame.getByTestId('sen-classroom-adjustment-input');
    const taHoursInput = senFrame.getByTestId('sen-ta-hours-input');
    const assignedTaInput = senFrame.getByTestId('sen-assigned-ta-input');
    const reviewNotesInput = senFrame.getByTestId('sen-review-notes-input');
    const logProvisionBtn = senFrame.getByTestId('log-iep-provision-btn');

    await expect(logProvisionBtn).toBeVisible();
    await expect(logProvisionBtn).toHaveAttribute('data-mis-rpc', 'log_iep_provision');

    await iepTargetInput.fill('Use structured algebraic scaffolds for 3-step word problems.');
    await adjustmentInput.fill('Seat near front desk; provide noise-cancelling ear defenders and 1:1 TA check-in.');
    await taHoursInput.fill('15');
    await assignedTaInput.fill('Mrs Sarah Jenkins');
    await reviewNotesInput.fill(
      'Agreed with parent and peer buddy Neville Longbottom during autumn EHCP annual review.'
    );

    // Verify 1H: declarative args do not leak staff_id
    const rawArgsAttr = (await logProvisionBtn.getAttribute('data-mis-args')) || '{}';
    expect(JSON.parse(rawArgsAttr)).not.toHaveProperty('staff_id');

    await logProvisionBtn.click();

    await expect(senFrame.getByTestId('sen-total-ta-hours')).toHaveText('15');
    await expect(senFrame.getByTestId('sen-provision-count')).toHaveText('1');
    await expect(senFrame.getByTestId('sen-provisions-list')).toContainText(
      'Seat near front desk; provide noise-cancelling ear defenders and 1:1 TA check-in.'
    );
    await expect(senFrame.getByTestId('sen-provisions-list')).toContainText('Mrs Sarah Jenkins');

    // Also log an IEP classroom adjustment & TA allocation for the K (SEN Support) pupil (`second_active_pupil_id`, Grace Turing)
    const pupilSelect = senFrame.getByTestId('sen-pupil-select');
    await expect(pupilSelect).toBeVisible();
    await pupilSelect.selectOption(m1Seed.second_active_pupil_id);

    await expect(senFrame.getByTestId('sen-provision-code-badge')).toHaveText('K');
    await expect(senFrame.getByTestId('sen-primary-need-badge')).toHaveText('SPLD');

    await iepTargetInput.fill('Complete extended reading comprehension using dyslexia-friendly font.');
    await adjustmentInput.fill('Provide cream tinted overlay, extra 25% reading time, and key vocabulary bank.');
    await taHoursInput.fill('5');
    await assignedTaInput.fill('Mr Thomas Wright');
    await reviewNotesInput.fill('Reviewed with Literacy Lead; steady progress in decoding.');
    await logProvisionBtn.click();

    await expect(senFrame.getByTestId('sen-total-ta-hours')).toHaveText('5');
    await expect(senFrame.getByTestId('sen-provision-count')).toHaveText('1');

    // 4. Switch to the Classroom Teacher persona (`m1Seed.teacher_staff_id`) via Topbar Persona Switcher and verify the Home Dashboard SEN Differentiation Mini-Widget
    const personaSelect = page.getByTestId('persona-switcher-select');
    await expect(personaSelect).toBeVisible();
    await personaSelect.selectOption(m1Seed.teacher_staff_id);

    // Verify the SEN Differentiation Mini-Widget on the Home Dashboard displays both K & E pupils' adjustments and TA allocations
    await expect(
      widgetFrame.getByTestId(`widget-sen-adjustment-${m1Seed.active_pupil_id}`)
    ).toContainText('Seat near front desk; provide noise-cancelling ear defenders and 1:1 TA check-in.');
    await expect(widgetFrame.getByTestId(`widget-sen-ta-${m1Seed.active_pupil_id}`)).toContainText(
      '15 hrs/wk — Mrs Sarah Jenkins'
    );

    await expect(
      widgetFrame.getByTestId(`widget-sen-adjustment-${m1Seed.second_active_pupil_id}`)
    ).toContainText('Provide cream tinted overlay, extra 25% reading time, and key vocabulary bank.');
    await expect(
      widgetFrame.getByTestId(`widget-sen-ta-${m1Seed.second_active_pupil_id}`)
    ).toContainText('5 hrs/wk — Mr Thomas Wright');

    // 5. Teacher clicks a pupil badge (`data-mis-link="student"`) inside the SEN Differentiation Mini-Widget and verifies it wakes the plugin in the Keep-Alive LRU pool for that pupil
    const firstPupilLink = widgetFrame.getByTestId(
      `widget-sen-student-link-${m1Seed.active_pupil_id}`
    );
    await expect(firstPupilLink).toBeVisible();
    await expect(firstPupilLink).toHaveAttribute('data-mis-link', 'student');
    await expect(firstPupilLink).toHaveAttribute('data-pupil-id', m1Seed.active_pupil_id);

    await firstPupilLink.click();

    // Verify Keep-Alive LRU pool wakes the student handler (`uk.org.optimis.test`) while keeping `uk.org.optimis.sen.provision` mounted and updated to `active_pupil_id`
    const lruPool = page.getByTestId('keep-alive-lru-pool');
    await expect(lruPool).toHaveAttribute(
      'data-lru-order',
      'uk.org.optimis.sen.provision,uk.org.optimis.test'
    );
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.test')).toBeVisible();
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.sen.provision')).toBeHidden();
    await expect(page.getByTestId('active-entity-context-pill')).toContainText(
      `student:${m1Seed.active_pupil_id}`
    );
    await expect(page.getByTestId('deep-link-navigation-status')).toContainText(
      `student:${m1Seed.active_pupil_id} -> uk.org.optimis.test`
    );

    // Switch back to the cached `uk.org.optimis.sen.provision` tab in the Keep-Alive LRU Pool and verify it immediately reflects `active_pupil_id` (Ada Lovelace, E / ASD)
    await senNavBtn.click();
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.sen.provision')).toBeVisible();
    await expect(senFrame.getByTestId('sen-context-status')).toContainText(
      `PUPIL:${m1Seed.active_pupil_id}`
    );
    await expect(senFrame.getByTestId('sen-provision-code-badge')).toHaveText('E');
    await expect(senFrame.getByTestId('sen-total-ta-hours')).toHaveText('15');
  });

  test('senProvisionRbacScopingAndGdprHooks_teacherBlockedFromWriteAndOtherGroupReadAndAdminRedactsAndPurgesSar_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded state');
    }

    // 1. Verify Classroom Teacher (`m1Seed.teacher_staff_id`, who holds only `dal.sen.read:own_groups`) is blocked with 403 from calling `log_iep_provision` (`dal.sen.read:all`)
    const teacherWriteRes = await request.post(
      `/api/plugins/uk.org.optimis.sen.provision/rpc/log_iep_provision?staff_id=${encodeURIComponent(
        m1Seed.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            pupil_id: m1Seed.active_pupil_id,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
            iep_target: 'Unauthorized teacher write attempt',
            classroom_adjustment: 'Extra time',
            ta_allocation_hours: 4,
            assigned_ta_name: 'Mrs Jenkins',
          },
        },
      }
    );
    expect(teacherWriteRes.status()).toBe(403);
    const teacherWriteBody = await teacherWriteRes.json();
    expect(teacherWriteBody.ok).toBe(false);
    expect(teacherWriteBody.error_code).toContain('caller lacks required DAL scopes');

    // 2. Verify Attendance Officer (`personaSeed.attendance_officer_staff_id`, `SUPPORT` role without `dal.sen.read`) is blocked with 403 from `get_sen_differentiation_summary`
    const supportReadRes = await request.post(
      `/api/plugins/uk.org.optimis.sen.provision/rpc/get_sen_differentiation_summary?staff_id=${encodeURIComponent(
        personaSeed.attendance_officer_staff_id
      )}`,
      {
        data: {
          args: {
            pupil_id: m1Seed.active_pupil_id,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
          },
        },
      }
    );
    expect(supportReadRes.status()).toBe(403);
    const supportReadBody = await supportReadRes.json();
    expect(supportReadBody.ok).toBe(false);
    expect(supportReadBody.error_code).toContain('caller lacks required DAL scopes');

    // 3. Verify Classroom Teacher calling `get_sen_differentiation_summary` with `strict_own_groups: true` is blocked from reading another group's pupil (`other_group_pupil_id`) or a transferred pupil (`transferred_pupil_id`)
    const crossGroupRes = await request.post(
      `/api/plugins/uk.org.optimis.sen.provision/rpc/get_sen_differentiation_summary?staff_id=${encodeURIComponent(
        m1Seed.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            pupil_id: m1Seed.other_group_pupil_id,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
            strict_own_groups: true,
          },
        },
      }
    );
    expect(crossGroupRes.status()).toBe(400);
    const crossGroupBody = await crossGroupRes.json();
    expect(crossGroupBody.ok).toBe(false);
    expect(crossGroupBody.error_code).toContain('access denied under own_groups scope');

    const transferredRes = await request.post(
      `/api/plugins/uk.org.optimis.sen.provision/rpc/get_sen_differentiation_summary?staff_id=${encodeURIComponent(
        m1Seed.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            pupil_id: m1Seed.transferred_pupil_id,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
            strict_own_groups: true,
          },
        },
      }
    );
    expect(transferredRes.status()).toBe(400);
    const transferredBody = await transferredRes.json();
    expect(transferredBody.ok).toBe(false);
    expect(transferredBody.error_code).toContain('access denied under own_groups scope');

    // 4. Verify fail-fast validation when SENCO attempts to log an IEP provision for a pupil with SEN provision `'N'` (No SEN)
    const nonSenLogRes = await request.post(
      `/api/plugins/uk.org.optimis.sen.provision/rpc/log_iep_provision?staff_id=${encodeURIComponent(
        personaSeed.hod_staff_id
      )}`,
      {
        data: {
          args: {
            pupil_id: m1Seed.other_group_pupil_id,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
            iep_target: 'Target for non-SEN pupil',
            classroom_adjustment: 'Front row seating',
            ta_allocation_hours: 2,
            assigned_ta_name: 'Mrs Jenkins',
          },
        },
      }
    );
    expect(nonSenLogRes.status()).toBe(400);
    const nonSenLogBody = await nonSenLogRes.json();
    expect(nonSenLogBody.ok).toBe(false);
    expect(nonSenLogBody.error_code).toContain('pupil does not have active K or E SEN provision');

    // 5. SENCO logs a valid IEP provision for `active_pupil_id` containing a third-party pupil name in `review_notes`, then Admin runs UK GDPR SAR Redaction UI and purges the record
    const validLogRes = await request.post(
      `/api/plugins/uk.org.optimis.sen.provision/rpc/log_iep_provision?staff_id=${encodeURIComponent(
        personaSeed.hod_staff_id
      )}`,
      {
        data: {
          args: {
            pupil_id: m1Seed.active_pupil_id,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
            iep_target: 'Complete multi-step problem solving with visual prompt card.',
            classroom_adjustment: 'Quiet breakout table shared with peer mentor Hermione Granger.',
            ta_allocation_hours: 12,
            assigned_ta_name: 'Mrs Sarah Jenkins',
            review_notes: 'Discussed small-group pairing with third-party pupil Hermione Granger.',
          },
        },
      }
    );
    expect(validLogRes.ok()).toBe(true);
    const validLogBody = await validLogRes.json();
    expect(validLogBody.ok).toBe(true);
    expect(validLogBody.data.logged_provision_id).toMatch(/^iep_[0-9a-f-]{36}$/);

    // Switch to System Admin / DPO and verify UK GDPR SAR Redaction UI
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personaSeed.admin_staff_id },
    });
    expect(switchAdminRes.ok()).toBe(true);
    const adminSessionBody = await switchAdminRes.json();

    await page.context().addCookies([
      {
        name: 'mis_session',
        value: adminSessionBody.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    await page.goto(
      `/?pupil_id=${encodeURIComponent(m1Seed.active_pupil_id)}&reference_date=${encodeURIComponent(
        m1Seed.reference_date
      )}&academic_year_id=${encodeURIComponent(m1Seed.academic_year_id)}`
    );

    const sarPanel = page.getByTestId('sar-redaction-panel');
    await expect(sarPanel).toBeVisible();

    await page.getByTestId('sar-load-export-btn').click();
    await expect(page.getByTestId('sar-export-status')).toHaveText('SAR_LOADED');

    const senSarSection = page.getByTestId(
      'sar-section-uk.org.optimis.sen.provision-Plugin SEN IEP & TA Provision Records'
    );
    await expect(senSarSection).toBeVisible();

    const adjustmentRedactInput = page.getByTestId(
      'sar-redact-input-uk.org.optimis.sen.provision-Plugin SEN IEP & TA Provision Records-0-classroom_adjustment'
    );
    const notesRedactInput = page.getByTestId(
      'sar-redact-input-uk.org.optimis.sen.provision-Plugin SEN IEP & TA Provision Records-0-review_notes'
    );
    await expect(adjustmentRedactInput).toBeVisible();
    await expect(notesRedactInput).toBeVisible();

    await adjustmentRedactInput.fill('Quiet breakout table shared with peer mentor [REDACTED THIRD PARTY].');
    await notesRedactInput.fill('Discussed small-group pairing with third-party pupil [REDACTED THIRD PARTY].');

    await page.getByTestId('sar-finalize-export-btn').click();
    await expect(page.getByTestId('sar-export-status')).toContainText('SAR_EXPORT_FINALIZED');

    const finalizedJsonEl = page.getByTestId('sar-finalized-json');
    await expect(finalizedJsonEl).toBeVisible();
    const rawExportJson = (await finalizedJsonEl.textContent()) || '';
    expect(rawExportJson).not.toContain('Hermione Granger');
    expect(rawExportJson).toContain('[REDACTED THIRD PARTY]');

    // Purge pupil data and verify zero orphaned rows remain
    const purgeRes = await request.post(
      `/api/plugins/uk.org.optimis.sen.provision/rpc/purge_pupil_data?staff_id=${encodeURIComponent(
        personaSeed.admin_staff_id
      )}`,
      {
        data: { args: { pupil_id: m1Seed.active_pupil_id } },
      }
    );
    expect(purgeRes.ok()).toBe(true);
    const purgeBody = await purgeRes.json();
    expect(purgeBody.ok).toBe(true);
    expect(purgeBody.data.purged_rows).toBe(1);

    const exportAfterPurgeRes = await request.post(
      `/api/plugins/uk.org.optimis.sen.provision/rpc/export_pupil_data?staff_id=${encodeURIComponent(
        personaSeed.admin_staff_id
      )}`,
      {
        data: { args: { pupil_id: m1Seed.active_pupil_id } },
      }
    );
    expect(exportAfterPurgeRes.ok()).toBe(true);
    const exportAfterBody = await exportAfterPurgeRes.json();
    expect(exportAfterBody.data[0].records.length).toBe(0);
  });
});
