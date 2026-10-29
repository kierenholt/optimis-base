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
  attendance_officer_staff_id: string;
  school_nurse_staff_id?: string;
  terminated_staff_id: string;
}

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
  transferred_pupil_id: string;
  other_group_pupil_id: string;
  unrestricted_mother_person_id: string;
  restricted_father_person_id: string;
  safe_household_id: string;
  confidential_household_id: string;
}

test.describe('Vertical 6: Medical Room & First Aid Plugin (uk.org.optimis.medical.firstaid)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let personas: SeededPersonaFixtures | null = null;
  let m1Seeded: Milestone1SeedResponse | null = null;

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    personas = null;
    m1Seeded = null;

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
      throw new Error(`Failed to configure auth mode: ${authModeRes.status()}`);
    }

    // 4. Seed isolated persona fixtures & Milestone 1 pupil/guardian/medical fixtures
    const personaRes = await request.post('/api/test/seed-personas', {
      data: {
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    if (!personaRes.ok()) {
      throw new Error(`Failed to seed persona fixtures: ${personaRes.status()}`);
    }
    personas = (await personaRes.json()) as SeededPersonaFixtures;

    const m1Res = await request.post('/api/test/seed-milestone1-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-28',
      },
    });
    if (!m1Res.ok()) {
      throw new Error(`Failed to seed Milestone 1 fixtures: ${m1Res.status()}`);
    }
    m1Seeded = (await m1Res.json()) as Milestone1SeedResponse;

    // 5. Install and activate first_aid_medical_plugin (`uk.org.optimis.medical.firstaid`)
    const installFaRes = await request.post('/api/admin/plugins/install', {
      data: {
        folder: 'first_aid_medical_plugin',
        staff_id: personas.admin_staff_id,
      },
    });
    if (!installFaRes.ok()) {
      throw new Error(
        `Failed to install first_aid_medical_plugin: ${installFaRes.status()} ${await installFaRes.text()}`
      );
    }

    const activateFaRes = await request.post(
      '/api/admin/plugins/uk.org.optimis.medical.firstaid/activate',
      {
        data: { staff_id: personas.admin_staff_id },
      }
    );
    if (!activateFaRes.ok()) {
      throw new Error(`Failed to activate uk.org.optimis.medical.firstaid: ${activateFaRes.status()}`);
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Purge plugin data created during the test
    if (personas && m1Seeded) {
      await request.post('/api/auth/persona', {
        data: { staff_id: personas.admin_staff_id },
      });
      await request.post('/api/plugins/uk.org.optimis.medical.firstaid/rpc/purge_pupil_data', {
        data: { args: { pupil_id: m1Seeded.active_pupil_id } },
      });
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

    // 4. Remove runtime-installed plugin
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete('/api/admin/plugins/uk.org.optimis.medical.firstaid');

    // 5. Clean up seeded fixtures in core.db
    if (m1Seeded?.fixture_id) {
      const delM1 = await request.delete(
        `/api/test/seed-milestone1-fixtures/${encodeURIComponent(m1Seeded.fixture_id)}`
      );
      if (!delM1.ok()) {
        throw new Error(`Failed to clean up Milestone 1 fixture ${m1Seeded.fixture_id}: ${delM1.status()}`);
      }
    }
    if (personas?.fixture_id) {
      const delP = await request.delete(
        `/api/test/seed-personas/${encodeURIComponent(personas.fixture_id)}`
      );
      if (!delP.ok()) {
        throw new Error(`Failed to clean up persona fixture ${personas.fixture_id}: ${delP.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('selectInjuredPupilViewCriticalMedicalAlertsAndSafeContactsAndLogTreatment_asFirstAider_rendersCriticalIhcpExcludesCourtOrderFatherAndRefugeAddressPersistsTreatmentAndEmitsCustomEvent_test', async ({
    page,
    request,
  }) => {
    if (!personas || !m1Seeded) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    const firstAiderStaffId = personas.school_nurse_staff_id || personas.attendance_officer_staff_id;
    expect(firstAiderStaffId).toBeTruthy();

    // 1. Switch to School Nurse / First Aider persona (SUPPORT role holding dal.pupils.read:all, dal.medical.read:all, dal.guardians.read:all)
    const switchFirstAiderRes = await request.post('/api/auth/persona', {
      data: { staff_id: firstAiderStaffId },
    });
    expect(switchFirstAiderRes.ok()).toBe(true);
    const firstAiderSession = await switchFirstAiderRes.json();
    expect(firstAiderSession.caller.active_roles).toContain('SUPPORT');
    expect(firstAiderSession.caller.scopes).toContain('dal.pupils.read:all');
    expect(firstAiderSession.caller.scopes).toContain('dal.medical.read:all');
    expect(firstAiderSession.caller.scopes).toContain('dal.guardians.read:all');

    await page.context().addCookies([
      {
        name: 'mis_session',
        value: firstAiderSession.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    // 2. Open Host UI Shell and navigate to Medical Room & First Aid plugin iframe
    const query = new URLSearchParams({
      pupil_id: m1Seeded.active_pupil_id,
      reference_date: m1Seeded.reference_date,
      academic_year_id: m1Seeded.academic_year_id,
      staff_id: firstAiderStaffId,
    });
    await page.goto(`/?${query.toString()}`);

    await expect(page.getByTestId('host-shell')).toBeVisible();
    const navBtn = page.getByTestId('nav-item-plugin-uk.org.optimis.medical.firstaid');
    await expect(navBtn).toBeVisible();
    await navBtn.click();

    const iframeEl = page.getByTestId('plugin-iframe-uk.org.optimis.medical.firstaid');
    await expect(iframeEl).toBeVisible();
    await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const frame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.medical.firstaid"]');
    await expect(frame.getByTestId('first-aid-plugin-root')).toBeVisible();

    // 3. Step 1 of Journey: First Aider selects injured pupil in UI
    const pupilSelect = frame.getByTestId('first-aid-pupil-select');
    await expect(pupilSelect).toBeVisible();
    await pupilSelect.selectOption(m1Seeded.active_pupil_id);

    const loadProfileBtn = frame.getByTestId('load-pupil-medical-profile-btn');
    await expect(loadProfileBtn).toHaveAttribute('data-mis-rpc', 'get_pupil_first_aid_profile');
    await loadProfileBtn.click();

    // 4. Step 2 of Journey: Plugin renders CRITICAL anaphylaxis/IHCP badge + emergency contacts (asserting court-order-blocked father & refuge address are excluded)
    const criticalAlertBadge = frame.getByTestId('critical-medical-alert-badge');
    await expect(criticalAlertBadge).toBeVisible();
    await expect(criticalAlertBadge).toContainText('CRITICAL');
    await expect(criticalAlertBadge).toContainText('Anaphylaxis (Peanut Allergy)');
    await expect(criticalAlertBadge).toContainText('IHCP');
    await expect(criticalAlertBadge).toContainText('Administer EpiPen immediately and call 999');

    const emergencyContactsCard = frame.getByTestId('emergency-contacts-card');
    await expect(emergencyContactsCard).toBeVisible();
    await expect(emergencyContactsCard).toContainText('Anne Byron');
    await expect(emergencyContactsCard).toContainText('MOTHER');
    await expect(emergencyContactsCard).toContainText('42 High Street');
    await expect(emergencyContactsCard).toContainText('AUTHORISED_TO_COLLECT');

    // Refuge guardian (Grandparent Judith Milbanke) is shown as emergency contact #2 with her refuge address strictly redacted
    await expect(emergencyContactsCard).toContainText('Judith Milbanke');
    await expect(emergencyContactsCard).toContainText('GRANDPARENT');
    await expect(emergencyContactsCard).toContainText('CONFIDENTIAL_ADDRESS_REDACTED');

    // Strictly assert court-order-blocked father & confidential refuge address are excluded from the entire iframe DOM
    const iframeHtmlContent = await frame.locator('body').innerHTML();
    expect(iframeHtmlContent).not.toContain('George Byron');
    expect(iframeHtmlContent).not.toContain(m1Seeded.restricted_father_person_id);
    expect(iframeHtmlContent).not.toContain('Prohibited Steps Order');
    expect(iframeHtmlContent).not.toContain('10 Safe Haven Way');
    expect(iframeHtmlContent).not.toContain('Confidential Refuge');
    expect(iframeHtmlContent).not.toContain('SW1A 1AA');

    // 5. Step 3 of Journey: Log treatment in `plugin_first_aid.db` and verify post-commit custom event + GDPR SAR hooks
    await frame.getByTestId('incident-location-input').fill('Main Playground - Netball Court');
    await frame
      .getByTestId('injury-summary-input')
      .fill('Grazed right knee and suspected wasp sting near wrist');
    await frame
      .getByTestId('treatment-given-input')
      .fill('Cleaned wound with sterile saline, applied cold compress, monitored 20 mins for allergic reaction');
    await frame.getByTestId('outcome-select').selectOption('RETURNED_TO_CLASS');

    const logTreatmentBtn = frame.getByTestId('log-first-aid-treatment-btn');
    await expect(logTreatmentBtn).toHaveAttribute('data-mis-rpc', 'log_first_aid_treatment');
    await logTreatmentBtn.click();

    const receiptBanner = frame.getByTestId('treatment-receipt-banner');
    await expect(receiptBanner).toBeVisible();
    await expect(receiptBanner).toContainText('LOGGED');
    await expect(receiptBanner).toContainText('plugin.uk.org.optimis.medical.firstaid.logged');

    const treatmentHistoryList = frame.getByTestId('treatment-history-list');
    await expect(treatmentHistoryList).toBeVisible();
    await expect(treatmentHistoryList).toContainText('Main Playground - Netball Court');
    await expect(treatmentHistoryList).toContainText('Grazed right knee and suspected wasp sting near wrist');
    await expect(treatmentHistoryList).toContainText('Cleaned wound with sterile saline');
    await expect(treatmentHistoryList).toContainText('RETURNED_TO_CLASS');

    // Reload profile to verify post-commit custom event worker persisted audit confirmation in `plugin_first_aid.db`
    await loadProfileBtn.click();
    const eventDispatchBadge = frame.getByTestId('first-aid-event-dispatch-badge');
    await expect(eventDispatchBadge).toBeVisible();
    await expect(eventDispatchBadge).toContainText('plugin.uk.org.optimis.medical.firstaid.logged');
    await expect(eventDispatchBadge).toContainText('DISPATCHED_POST_COMMIT');
  });

  test('firstAidMedicalPlugin_unauthorizedTeacherWithoutAllMedicalAndGuardianScopesOrMissingParamsAndGdprSarHooks_failsFastAndPurgesCleanly_test', async ({
    request,
  }) => {
    if (!personas || !m1Seeded) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    const firstAiderStaffId = personas.school_nurse_staff_id || personas.attendance_officer_staff_id;

    // 1. RBAC Verification: Classroom Teacher (lacking dal.medical.read:all & dal.guardians.read:all) is rejected with 403
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: personas.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);

    const forbiddenProfileRes = await request.post(
      '/api/plugins/uk.org.optimis.medical.firstaid/rpc/get_pupil_first_aid_profile',
      {
        data: {
          args: {
            pupil_id: m1Seeded.active_pupil_id,
            academic_year_id: m1Seeded.academic_year_id,
            reference_date: m1Seeded.reference_date,
          },
        },
      }
    );
    expect(forbiddenProfileRes.status()).toBe(403);
    const forbiddenProfileBody = await forbiddenProfileRes.json();
    expect(forbiddenProfileBody.ok).toBe(false);
    expect(forbiddenProfileBody.error_code).toContain('caller lacks required DAL scopes');

    const forbiddenLogRes = await request.post(
      '/api/plugins/uk.org.optimis.medical.firstaid/rpc/log_first_aid_treatment',
      {
        data: {
          args: {
            pupil_id: m1Seeded.active_pupil_id,
            academic_year_id: m1Seeded.academic_year_id,
            reference_date: m1Seeded.reference_date,
            incident_location: 'Hallway',
            injury_summary: 'Bump on elbow',
            treatment_given: 'Ice pack',
            outcome: 'RETURNED_TO_CLASS',
          },
        },
      }
    );
    expect(forbiddenLogRes.status()).toBe(403);
    const forbiddenLogBody = await forbiddenLogRes.json();
    expect(forbiddenLogBody.ok).toBe(false);
    expect(forbiddenLogBody.error_code).toContain('caller lacks required DAL scopes');

    // 2. Switch to School Nurse / First Aider and verify fail-fast 400 validation on missing mandatory fields
    const switchFirstAiderRes = await request.post('/api/auth/persona', {
      data: { staff_id: firstAiderStaffId },
    });
    expect(switchFirstAiderRes.ok()).toBe(true);

    const missingTreatmentRes = await request.post(
      '/api/plugins/uk.org.optimis.medical.firstaid/rpc/log_first_aid_treatment',
      {
        data: {
          args: {
            pupil_id: m1Seeded.active_pupil_id,
            academic_year_id: m1Seeded.academic_year_id,
            reference_date: m1Seeded.reference_date,
            incident_location: 'Hallway',
            injury_summary: 'Bump on elbow',
            treatment_given: '',
            outcome: 'RETURNED_TO_CLASS',
          },
        },
      }
    );
    expect(missingTreatmentRes.status()).toBe(400);
    const missingTreatmentBody = await missingTreatmentRes.json();
    expect(missingTreatmentBody.ok).toBe(false);
    expect(missingTreatmentBody.error_code).toContain('treatment_given is required');

    // 3. Log a valid treatment and verify UK GDPR `export_pupil_data` & `purge_pupil_data` hooks on `plugin_first_aid.db`
    const validLogRes = await request.post(
      '/api/plugins/uk.org.optimis.medical.firstaid/rpc/log_first_aid_treatment',
      {
        data: {
          args: {
            pupil_id: m1Seeded.active_pupil_id,
            academic_year_id: m1Seeded.academic_year_id,
            reference_date: m1Seeded.reference_date,
            incident_location: 'Science Lab 2',
            injury_summary: 'Minor scalding on left thumb during experiment with Babbage',
            treatment_given: 'Cooled under running water for 20 minutes, applied sterile burn dressing',
            outcome: 'SENT_HOME_WITH_GUARDIAN',
          },
        },
      }
    );
    expect(validLogRes.ok()).toBe(true);
    const validLogBody = await validLogRes.json();
    expect(validLogBody.ok).toBe(true);
    expect(validLogBody.data.emitted_event).toBe('plugin.uk.org.optimis.medical.firstaid.logged');

    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personas.admin_staff_id },
    });
    expect(switchAdminRes.ok()).toBe(true);

    const exportRes = await request.post(
      '/api/plugins/uk.org.optimis.medical.firstaid/rpc/export_pupil_data',
      {
        data: { args: { pupil_id: m1Seeded.active_pupil_id } },
      }
    );
    expect(exportRes.ok()).toBe(true);
    const exportBody = await exportRes.json();
    expect(exportBody.ok).toBe(true);
    expect(exportBody.data.sections.length).toBe(2);
    expect(exportBody.data.sections[0].title).toBe('Medical Room & First Aid Treatments');
    expect(exportBody.data.sections[0].records.length).toBeGreaterThanOrEqual(1);
    expect(exportBody.data.sections[0].redactable_fields).toContain('injury_summary');
    expect(exportBody.data.sections[0].redactable_fields).toContain('treatment_given');
    expect(exportBody.data.sections[1].title).toBe('First Aid Post-Commit Event Audit Logs');
    expect(exportBody.data.sections[1].records.length).toBeGreaterThanOrEqual(1);

    const purgeRes = await request.post(
      '/api/plugins/uk.org.optimis.medical.firstaid/rpc/purge_pupil_data',
      {
        data: { args: { pupil_id: m1Seeded.active_pupil_id } },
      }
    );
    expect(purgeRes.ok()).toBe(true);
    const purgeBody = await purgeRes.json();
    expect(purgeBody.ok).toBe(true);
    expect(purgeBody.data.purged_rows).toBeGreaterThanOrEqual(2);

    const exportAfterPurgeRes = await request.post(
      '/api/plugins/uk.org.optimis.medical.firstaid/rpc/export_pupil_data',
      {
        data: { args: { pupil_id: m1Seeded.active_pupil_id } },
      }
    );
    expect(exportAfterPurgeRes.ok()).toBe(true);
    const exportAfterPurgeBody = await exportAfterPurgeRes.json();
    expect(exportAfterPurgeBody.data.sections[0].records.length).toBe(0);
    expect(exportAfterPurgeBody.data.sections[1].records.length).toBe(0);
  });
});
