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
  transferred_pupil_id: string;
  other_group_pupil_id: string;
  unrestricted_mother_person_id: string;
  restricted_father_person_id: string;
  safe_household_id: string;
  confidential_household_id: string;
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

const PLUGIN_ID = 'uk.org.optimis.ops.trips';
const PLUGIN_FOLDER = 'trips_consent_plugin';

test.describe('Vertical 7: Educational Visits, PR Consent Verification & Trip Emergency Pack Plugin (uk.org.optimis.ops.trips)', () => {
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

    // 4. Seed Admin/DPO persona fixtures and Milestone 1 own_groups / guardian PR / court-order / SEN & Medical fixtures
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
    if (!personaSeed.fixture_id || !personaSeed.admin_staff_id) {
      throw new Error('Persona seed response missing mandatory identifiers');
    }

    const m1Res = await request.post('/api/test/seed-milestone1-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-28',
      },
    });
    if (!m1Res.ok()) {
      throw new Error(`Failed to seed Milestone 1 fixtures: ${m1Res.status()}`);
    }
    m1Seed = (await m1Res.json()) as Milestone1SeedResponse;
    if (
      !m1Seed.fixture_id ||
      !m1Seed.teacher_staff_id ||
      !m1Seed.other_teacher_staff_id ||
      !m1Seed.own_group_id ||
      !m1Seed.other_group_id ||
      !m1Seed.active_pupil_id ||
      !m1Seed.transferred_pupil_id ||
      !m1Seed.other_group_pupil_id ||
      !m1Seed.unrestricted_mother_person_id ||
      !m1Seed.restricted_father_person_id
    ) {
      throw new Error('Milestone 1 seed response missing mandatory identifiers');
    }

    // 5. Switch to Admin persona to install and activate trips_consent_plugin (uk.org.optimis.ops.trips)
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personaSeed.admin_staff_id },
    });
    if (!switchAdminRes.ok()) {
      throw new Error(`Failed to switch to Admin persona in setup: ${switchAdminRes.status()}`);
    }

    await request.delete(`/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}`);

    const installTripsRes = await request.post('/api/admin/plugins/install', {
      data: { folder: PLUGIN_FOLDER, staff_id: personaSeed.admin_staff_id },
    });
    if (!installTripsRes.ok()) {
      throw new Error(
        `Failed to install ${PLUGIN_FOLDER}: ${installTripsRes.status()} ${await installTripsRes.text()}`
      );
    }
    const installBody = await installTripsRes.json();
    if (!installBody.ok || installBody.plugin_id !== PLUGIN_ID || installBody.database_alias !== 'plugin_trips') {
      throw new Error(`Unexpected install response for ${PLUGIN_ID}: ${JSON.stringify(installBody)}`);
    }

    const activateTripsRes = await request.post(
      `/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}/activate`,
      {
        data: { staff_id: personaSeed.admin_staff_id },
      }
    );
    if (!activateTripsRes.ok()) {
      throw new Error(`Failed to activate ${PLUGIN_ID}: ${activateTripsRes.status()}`);
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Clean up trip consent records in plugin_trips.db
    if (personaSeed && m1Seed) {
      await request.post('/api/auth/persona', {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      await request.post(`/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}/activate`, {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      for (const pid of [
        m1Seed.active_pupil_id,
        m1Seed.transferred_pupil_id,
        m1Seed.other_group_pupil_id,
      ]) {
        await request.post(`/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/purge_pupil_data`, {
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

    // 4. Remove runtime-installed plugin
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete(`/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}`);

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

  test('createTripVerifyPrConsentAndGenerateEmergencyPack_teacherCreatesGroupTripRejectsNonPrAndCourtOrderGuardiansAcceptsUnrestrictedPrMotherAndGeneratesEmergencyManifest_test', async ({
    page,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded state');
    }

    const nonPrGrandparentPersonId = `per_gp_${m1Seed.fixture_id}`;

    // Open Host UI Shell as the Trip Leader / Classroom Teacher (m1Seed.teacher_staff_id)
    const params = new URLSearchParams({
      pupil_id: m1Seed.active_pupil_id,
      group_id: m1Seed.own_group_id,
      other_group_id: m1Seed.other_group_id,
      reference_date: m1Seed.reference_date,
      academic_year_id: m1Seed.academic_year_id,
      staff_id: m1Seed.teacher_staff_id,
    });
    await page.goto(`/?${params.toString()}`);

    await expect(page.getByTestId('host-shell')).toBeVisible();

    const tripsNavBtn = page.getByTestId(`nav-item-plugin-${PLUGIN_ID}`);
    await expect(tripsNavBtn).toBeVisible();
    await tripsNavBtn.click();

    const tripsIframeEl = page.getByTestId(`plugin-iframe-${PLUGIN_ID}`);
    await expect(tripsIframeEl).toBeVisible();
    await expect(tripsIframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${PLUGIN_ID}"]`);
    await expect(pluginFrame.getByTestId('trips-consent-plugin-root')).toBeVisible();

    // =========================================================================
    // Step 1: Teacher creates an educational visit for their group in plugin_trips.db
    // =========================================================================
    const tripNameInput = pluginFrame.getByTestId('trip-name-input');
    const tripDestinationInput = pluginFrame.getByTestId('trip-destination-input');
    const tripDateInput = pluginFrame.getByTestId('trip-date-input');
    const createTripBtn = pluginFrame.getByTestId('create-trip-btn');

    await expect(tripNameInput).toBeVisible();
    await expect(createTripBtn).toBeVisible();
    await expect(createTripBtn).toHaveAttribute('data-mis-rpc', 'create_trip');

    await tripNameInput.fill('Science Museum Astrophysics Visit');
    await tripDestinationInput.fill('Science Museum, South Kensington');
    await tripDateInput.fill('2026-11-15');
    await createTripBtn.click();

    const activeTripCard = pluginFrame.getByTestId('active-trip-card');
    await expect(activeTripCard).toBeVisible();
    await expect(pluginFrame.getByTestId('active-trip-name')).toHaveText('Science Museum Astrophysics Visit');
    await expect(pluginFrame.getByTestId('active-trip-destination')).toHaveText(
      'Science Museum, South Kensington'
    );
    await expect(pluginFrame.getByTestId('active-trip-eligible-count')).toHaveText('1');

    // =========================================================================
    // Step 2a: Attempt to record consent from a non-PR adult (Grandparent has_parental_responsibility=0)
    //          -> rejected (GUARDIAN_NOT_AUTHORIZED_FOR_CONSENT)
    // =========================================================================
    const guardianIdInput = pluginFrame.getByTestId('consent-guardian-id-input');
    const dietaryInput = pluginFrame.getByTestId('consent-dietary-notes-input');
    const medicalNotesInput = pluginFrame.getByTestId('consent-medical-notes-input');
    const recordConsentBtn = pluginFrame.getByTestId('record-consent-btn');

    await expect(guardianIdInput).toBeVisible();
    await expect(recordConsentBtn).toBeVisible();
    await expect(recordConsentBtn).toHaveAttribute('data-mis-rpc', 'record_trip_consent');

    await guardianIdInput.fill(nonPrGrandparentPersonId);
    await dietaryInput.fill('No nuts');
    await medicalNotesInput.fill('EpiPen in bag');
    await recordConsentBtn.click();

    const errorBanner = pluginFrame.getByTestId('trips-error-banner');
    await expect(errorBanner).toBeVisible();
    await expect(errorBanner).toContainText('GUARDIAN_NOT_AUTHORIZED_FOR_CONSENT');

    // =========================================================================
    // Step 2b: Attempt to record consent from a court-order-restricted adult (Father court_order_restricted=1)
    //          -> rejected (GUARDIAN_NOT_AUTHORIZED_FOR_CONSENT)
    // =========================================================================
    await guardianIdInput.fill(m1Seed.restricted_father_person_id);
    await recordConsentBtn.click();

    await expect(errorBanner).toBeVisible();
    await expect(errorBanner).toContainText('GUARDIAN_NOT_AUTHORIZED_FOR_CONSENT');
    const errorText = (await errorBanner.textContent()) || '';
    expect(errorText).not.toContain('Prohibited Steps Order');

    // =========================================================================
    // Step 3a: Record consent from unrestricted PR mother (Anne Byron, MOTHER, PR=1)
    // =========================================================================
    await guardianIdInput.fill(m1Seed.unrestricted_mother_person_id);
    await dietaryInput.fill('Strict peanut-free and tree-nut-free packed lunch');
    await medicalNotesInput.fill('Carries 2x EpiPen auto-injectors in red medical pouch');
    await recordConsentBtn.click();

    await expect(errorBanner).toBeHidden();
    const consentStatusBadge = pluginFrame.getByTestId('consent-status-badge');
    await expect(consentStatusBadge).toBeVisible();
    await expect(consentStatusBadge).toHaveText('CONSENTED');

    const consentedGuardianSummary = pluginFrame.getByTestId('consented-guardian-summary');
    await expect(consentedGuardianSummary).toBeVisible();
    await expect(consentedGuardianSummary).toContainText('Anne Byron');
    await expect(consentedGuardianSummary).toContainText('MOTHER');

    // =========================================================================
    // Step 3b: Generate Trip Emergency Manifest with medical/dietary/SEN flags
    // =========================================================================
    const generatePackBtn = pluginFrame.getByTestId('generate-emergency-pack-btn');
    await expect(generatePackBtn).toBeVisible();
    await expect(generatePackBtn).toHaveAttribute('data-mis-rpc', 'generate_trip_emergency_pack');
    await generatePackBtn.click();

    const emergencyPackSection = pluginFrame.getByTestId('trip-emergency-pack');
    await expect(emergencyPackSection).toBeVisible();
    await expect(pluginFrame.getByTestId('emergency-pack-consented-count')).toHaveText('1');

    const manifestRow = pluginFrame.getByTestId(`emergency-manifest-row-${m1Seed.active_pupil_id}`);
    await expect(manifestRow).toBeVisible();
    await expect(manifestRow).toContainText('Ada Lovelace');
    await expect(manifestRow).toContainText('Anne Byron');
    await expect(manifestRow).toContainText('07700900111');
    await expect(manifestRow).toContainText('Anaphylaxis (Peanut Allergy)');
    await expect(manifestRow).toContainText('IHCP');
    await expect(manifestRow).toContainText('Administer EpiPen immediately and call 999');
    await expect(manifestRow).toContainText('E (ASD)');
    await expect(manifestRow).toContainText('Strict peanut-free and tree-nut-free packed lunch');
    await expect(manifestRow).toContainText('Carries 2x EpiPen auto-injectors in red medical pouch');

    // Transferred pupil and other teacher's group pupil must not appear in emergency manifest
    await expect(
      pluginFrame.getByTestId(`emergency-manifest-row-${m1Seed.transferred_pupil_id}`)
    ).toHaveCount(0);
    await expect(
      pluginFrame.getByTestId(`emergency-manifest-row-${m1Seed.other_group_pupil_id}`)
    ).toHaveCount(0);
  });

  test('tripsConsentPluginOwnGroupsAndGdprSarPurge_teacherBlockedFromOtherTeachersGroupAndAdminExportsRedactsAndPurgesTripConsents_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded state');
    }

    // 1. Verify :own_groups isolation: Teacher 1 attempting to create a trip for Teacher 2's group (other_group_id) MUST fail fast
    const crossGroupTripRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/create_trip?staff_id=${encodeURIComponent(
        m1Seed.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            group_id: m1Seed.other_group_id,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
            trip_name: 'Unauthorized Cross-Group Trip',
            destination: 'Greenwich Observatory',
            trip_date: '2026-11-20',
          },
        },
      }
    );
    expect(crossGroupTripRes.status()).toBe(400);
    const crossGroupTripBody = await crossGroupTripRes.json();
    expect(crossGroupTripBody.ok).toBe(false);
    expect(crossGroupTripBody.error_code).toContain('access denied under own_groups scope');

    // 2. Teacher 1 creates a valid trip for own_group_id and records valid PR consent from Anne Byron
    const createOkRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/create_trip?staff_id=${encodeURIComponent(
        m1Seed.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            group_id: m1Seed.own_group_id,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
            trip_name: 'Natural History Museum Geology Fieldwork',
            destination: 'Natural History Museum, London',
            trip_date: '2026-11-22',
          },
        },
      }
    );
    expect(createOkRes.ok()).toBe(true);
    const createOkBody = await createOkRes.json();
    expect(createOkBody.ok).toBe(true);
    const tripId = createOkBody.data.trip_id as string;
    expect(tripId).toMatch(/^trp_/);

    // Verify Teacher 2 (other_teacher_staff_id) is blocked under :own_groups from generating the emergency pack for Teacher 1's trip
    const otherTeacherPackRes = await request.post(
      `/api/plugins/${encodeURIComponent(
        PLUGIN_ID
      )}/rpc/generate_trip_emergency_pack?staff_id=${encodeURIComponent(m1Seed.other_teacher_staff_id)}`,
      {
        data: {
          args: {
            trip_id: tripId,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
          },
        },
      }
    );
    expect(otherTeacherPackRes.status()).toBe(400);
    const otherTeacherPackBody = await otherTeacherPackRes.json();
    expect(otherTeacherPackBody.ok).toBe(false);
    expect(otherTeacherPackBody.error_code).toContain('access denied under own_groups scope');

    // Record valid PR consent for active_pupil_id containing a third-party name in medical_notes for SAR redaction verification
    const consentOkRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/record_trip_consent?staff_id=${encodeURIComponent(
        m1Seed.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            trip_id: tripId,
            pupil_id: m1Seed.active_pupil_id,
            guardian_person_id: m1Seed.unrestricted_mother_person_id,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
            dietary_notes: 'Vegetarian & strictly nut-free meal required',
            medical_notes: 'Paired with buddy Neville Longbottom for inhaler & EpiPen supervision',
          },
        },
      }
    );
    expect(consentOkRes.ok()).toBe(true);
    const consentOkBody = await consentOkRes.json();
    expect(consentOkBody.ok).toBe(true);
    expect(consentOkBody.data.consent_status).toBe('CONSENTED');
    expect(consentOkBody.data.has_parental_responsibility).toBe(true);

    // 3. Switch to System Admin / DPO persona, load SAR export in Host UI Shell, and redact third-party name ("Neville Longbottom")
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

    const tripsSarSection = page.getByTestId(
      `sar-section-${PLUGIN_ID}-Educational Visits & Parental Consent Records`
    );
    await expect(tripsSarSection).toBeVisible();

    const medicalNotesRedactInput = page.getByTestId(
      `sar-redact-input-${PLUGIN_ID}-Educational Visits & Parental Consent Records-0-medical_notes`
    );
    await expect(medicalNotesRedactInput).toBeVisible();
    await expect(medicalNotesRedactInput).toHaveValue(
      'Paired with buddy Neville Longbottom for inhaler & EpiPen supervision'
    );

    await medicalNotesRedactInput.fill(
      'Paired with buddy [REDACTED THIRD PARTY] for inhaler & EpiPen supervision'
    );

    await page.getByTestId('sar-finalize-export-btn').click();
    await expect(page.getByTestId('sar-export-status')).toContainText('SAR_EXPORT_FINALIZED');

    const finalizedJsonEl = page.getByTestId('sar-finalized-json');
    await expect(finalizedJsonEl).toBeVisible();
    const rawExportJson = (await finalizedJsonEl.textContent()) || '';
    expect(rawExportJson).not.toContain('Neville Longbottom');
    expect(rawExportJson).toContain('[REDACTED THIRD PARTY]');

    // 4. Verify UK GDPR purge_pupil_data removes all rows for active_pupil_id in plugin_trips.db
    const purgeRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/purge_pupil_data?staff_id=${encodeURIComponent(
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
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/export_pupil_data?staff_id=${encodeURIComponent(
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
