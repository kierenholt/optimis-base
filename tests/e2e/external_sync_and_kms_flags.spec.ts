import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

interface SeededPersonaFixtures {
  fixture_id: string;
  pupil_id: string;
  admin_staff_id: string;
  dsl_staff_id: string;
  teacher_staff_id: string;
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

test.describe('External Dependencies Feature Flags: Cloud KMS, Parent DMZ Container & MAT Central Sync (Sections 1, 5B, 6A & 7C)', () => {
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

    // 3. Ensure dev persona switcher and mock external dependency modes are active before seeding
    const authModeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!authModeRes.ok()) {
      throw new Error(`Failed to configure auth mode: ${authModeRes.status()}`);
    }

    const extModeRes = await request.post('/api/test/external-deps-mode', {
      data: {
        enable_real_cloud_kms: false,
        enable_real_parent_dmz_sync: false,
        enable_real_mat_sync: false,
      },
    });
    if (!extModeRes.ok()) {
      throw new Error(`Failed to configure external deps mode: ${extModeRes.status()}`);
    }

    // 4. Seed isolated persona & Milestone 1 safeguarding/guardian fixtures
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
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Restore mock external dependency modes & clear mock DMZ/MAT stores
    await request.post('/api/test/external-deps-mode', {
      data: {
        enable_real_cloud_kms: false,
        enable_real_parent_dmz_sync: false,
        enable_real_mat_sync: false,
      },
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

    // 4. Clean up seeded fixtures in core.db
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

  test('cloudKmsFeatureFlag_realModeWithoutProviderConfig_failsFastAndRestoresMockSafeguardingVault_test', async ({
    page,
    request,
  }) => {
    if (!personas) {
      throw new Error('Missing mandatory seeded persona fixtures');
    }

    // 1. Switch to DSL persona and create a safeguarding postbox note while mock Cloud KMS is enabled (ENABLE_REAL_CLOUD_KMS=false)
    const switchDslRes = await request.post('/api/auth/persona', {
      data: { staff_id: personas.dsl_staff_id },
    });
    expect(switchDslRes.ok()).toBe(true);
    const dslSessionBody = await switchDslRes.json();

    const postboxRes = await request.post('/api/safeguarding/postbox', {
      data: {
        pupil_id: personas.pupil_id,
        assigned_dsl_staff_id: personas.dsl_staff_id,
        concern_category: 'NEGLECT',
        incident_date: '2026-10-28',
        summary: 'Cloud KMS gate verification concern',
        body_map: 'NONE',
        verbatim_account: 'Child arrived without coat in freezing weather.',
        action_taken: 'Logged in DSL vault',
      },
    });
    expect(postboxRes.status()).toBe(201);
    const postboxBody = await postboxRes.json();
    const noteId: string = postboxBody.receipt.note_id;
    expect(noteId).toMatch(/^sfg_/);

    // 2. Flip ENABLE_REAL_CLOUD_KMS=true without real Cloud KMS provider config in __system_config -> must fail fast with 503 REAL_CLOUD_KMS_NOT_CONFIGURED
    const enableRealKmsRes = await request.post('/api/test/external-deps-mode', {
      data: { enable_real_cloud_kms: true },
    });
    expect(enableRealKmsRes.status()).toBe(503);
    const enableRealKmsBody = await enableRealKmsRes.json();
    expect(enableRealKmsBody.ok).toBe(false);
    expect(enableRealKmsBody.error_code).toContain('REAL_CLOUD_KMS_NOT_CONFIGURED');

    // 3. Subsequent calls to Cloud Key renewal, postbox creation, and note decryption must also fail fast with REAL_CLOUD_KMS_NOT_CONFIGURED
    const renewRealRes = await request.post('/api/safeguarding/cloud-key/renew', {
      data: { ttl_seconds: 3600 },
    });
    expect(renewRealRes.status()).toBe(503);
    const renewRealBody = await renewRealRes.json();
    expect(renewRealBody.error_code).toContain('REAL_CLOUD_KMS_NOT_CONFIGURED');

    const postboxRealRes = await request.post('/api/safeguarding/postbox', {
      data: {
        pupil_id: personas.pupil_id,
        assigned_dsl_staff_id: personas.dsl_staff_id,
        concern_category: 'NEGLECT',
        incident_date: '2026-10-28',
        summary: 'Should fail fast when real KMS unconfigured',
      },
    });
    expect(postboxRealRes.status()).toBe(503);
    expect((await postboxRealRes.json()).error_code).toContain('REAL_CLOUD_KMS_NOT_CONFIGURED');

    const decryptRealRes = await request.post(`/api/safeguarding/notes/${encodeURIComponent(noteId)}/decrypt`, {
      data: { cloud_key_token: 'any.token.here' },
    });
    expect(decryptRealRes.status()).toBe(503);
    expect((await decryptRealRes.json()).error_code).toContain('REAL_CLOUD_KMS_NOT_CONFIGURED');

    // 4. Verify UI DSL Safeguarding Vault surfaces the fail-fast REAL_CLOUD_KMS_NOT_CONFIGURED error banner when clicking Renew OAuth Cloud Key
    await page.context().addCookies([
      {
        name: 'mis_session',
        value: dslSessionBody.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);
    await page.goto(`/?pupil_id=${encodeURIComponent(personas.pupil_id)}`);

    const vaultPanel = page.getByTestId('dsl-safeguarding-vault-panel');
    await expect(vaultPanel).toBeVisible();

    const renewBtn = page.getByTestId('renew-dsl-cloud-key-btn');
    await renewBtn.click();

    const vaultErrorBanner = page.getByTestId('dsl-vault-error-banner');
    await expect(vaultErrorBanner).toBeVisible();
    await expect(vaultErrorBanner).toContainText('REAL_CLOUD_KMS_NOT_CONFIGURED');

    // 5. Restore ENABLE_REAL_CLOUD_KMS=false and verify Cloud Key renewal & decryption succeed in the UI
    const restoreMockKmsRes = await request.post('/api/test/external-deps-mode', {
      data: { enable_real_cloud_kms: false },
    });
    expect(restoreMockKmsRes.ok()).toBe(true);

    await renewBtn.click();
    await page.getByTestId(`decrypt-safeguarding-note-btn-${noteId}`).click();

    const decryptedView = page.getByTestId(`decrypted-safeguarding-note-${noteId}`);
    await expect(decryptedView).toBeVisible();
    await expect(decryptedView).toContainText('Cloud KMS gate verification concern');
  });

  test('parentDmzSync_pushSanitizedProjectionAndPullInbox_enforcesCourtOrderBlocksAndFailsFastWhenRealModeUnconfigured_test', async ({
    request,
  }) => {
    if (!personas || !m1Seeded) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // 1. Switch to Admin persona
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personas.admin_staff_id },
    });
    expect(switchAdminRes.ok()).toBe(true);

    // Record a statutory AM attendance mark for active_pupil_id so the read-projection includes attendance summary counts
    const rpcAttRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/record_group_am_attendance', {
      data: {
        args: {
          group_id: m1Seeded.own_group_id,
          pupil_id: m1Seeded.active_pupil_id,
          reference_date: m1Seeded.reference_date,
          academic_year_id: m1Seeded.academic_year_id,
          staff_id: m1Seeded.teacher_staff_id,
        },
      },
    });
    expect(rpcAttRes.ok()).toBe(true);

    // 2. Push sanitized parent read-projection to Mock Public Cloud DMZ (ENABLE_REAL_PARENT_DMZ_SYNC=false)
    const pushRes = await request.post('/api/sync/parent-dmz/push-projection', {
      data: {
        pupil_id: m1Seeded.active_pupil_id,
        reference_date: m1Seeded.reference_date,
        academic_year_id: m1Seeded.academic_year_id,
      },
    });
    expect(pushRes.status()).toBe(200);
    const pushBody = await pushRes.json();
    expect(pushBody.ok).toBe(true);
    expect(pushBody.projection.pupil_id).toBe(m1Seeded.active_pupil_id);
    expect(pushBody.projection.first_name).toBe('Ada');
    expect(pushBody.projection.last_name).toBe('Lovelace');
    expect(pushBody.projection.academic_year_id).toBe(m1Seeded.academic_year_id);
    expect(pushBody.projection.present_sessions).toBeGreaterThanOrEqual(1);
    expect(pushBody.projection.total_sessions).toBeGreaterThanOrEqual(1);

    // Strictly enforce guardian PR & court-order blocks: only unrestricted mother with PR is authorized; restricted father is excluded
    expect(pushBody.projection.authorized_guardian_ids).toContain(m1Seeded.unrestricted_mother_person_id);
    expect(pushBody.projection.authorized_guardian_ids).not.toContain(m1Seeded.restricted_father_person_id);
    expect(pushBody.projection.redacted_confidential_address_count).toBeGreaterThanOrEqual(1);

    const rawProjectionJson = JSON.stringify(pushBody.projection);
    expect(rawProjectionJson).not.toContain('Prohibited Steps Order');
    expect(rawProjectionJson).not.toContain('10 Safe Haven Way');
    expect(rawProjectionJson).not.toContain('upn');

    // 3. Enqueue two untrusted submissions in the Mock DMZ Write-Inbox:
    //    (a) Valid submission from unrestricted PR Mother (Anne Byron) -> MUST BE ACCEPTED
    //    (b) Blocked submission from court-order-restricted Father (George Byron) -> MUST BE REJECTED_SAFEGUARDING_BLOCK
    const enqueueMomRes = await request.post('/api/test/parent-dmz/enqueue-inbox', {
      data: {
        pupil_id: m1Seeded.active_pupil_id,
        submitted_by_guardian_person_id: m1Seeded.unrestricted_mother_person_id,
        submission_type: 'ABSENCE_NOTICE',
        date: m1Seeded.reference_date,
        session: 'PM',
        reason: 'Orthodontist appointment at 2pm',
      },
    });
    expect(enqueueMomRes.ok()).toBe(true);

    const enqueueDadRes = await request.post('/api/test/parent-dmz/enqueue-inbox', {
      data: {
        pupil_id: m1Seeded.active_pupil_id,
        submitted_by_guardian_person_id: m1Seeded.restricted_father_person_id,
        submission_type: 'CONTACT_UPDATE',
        date: m1Seeded.reference_date,
        reason: 'Attempted contact update by court-order-restricted individual',
      },
    });
    expect(enqueueDadRes.ok()).toBe(true);

    // 4. School Container pulls queued submissions outbound from the DMZ write-inbox
    const pullRes = await request.post('/api/sync/parent-dmz/pull-inbox', {
      data: {},
    });
    expect(pullRes.status()).toBe(200);
    const pullBody = await pullRes.json();
    expect(pullBody.ok).toBe(true);
    expect(pullBody.accepted_count).toBe(1);
    expect(pullBody.rejected_count).toBe(1);
    expect(pullBody.submissions.length).toBe(2);

    const momSub = pullBody.submissions.find(
      (s: { submitted_by_guardian_person_id: string }) =>
        s.submitted_by_guardian_person_id === m1Seeded!.unrestricted_mother_person_id
    );
    const dadSub = pullBody.submissions.find(
      (s: { submitted_by_guardian_person_id: string }) =>
        s.submitted_by_guardian_person_id === m1Seeded!.restricted_father_person_id
    );
    expect(momSub?.status).toBe('ACCEPTED');
    expect(dadSub?.status).toBe('REJECTED_SAFEGUARDING_BLOCK');

    // 5. Flip ENABLE_REAL_PARENT_DMZ_SYNC=true without real DMZ endpoint config in __system_config -> must fail fast with 503 REAL_PARENT_DMZ_NOT_CONFIGURED
    const enableRealDmzRes = await request.post('/api/test/external-deps-mode', {
      data: { enable_real_parent_dmz_sync: true },
    });
    expect(enableRealDmzRes.status()).toBe(503);
    expect((await enableRealDmzRes.json()).error_code).toContain('REAL_PARENT_DMZ_NOT_CONFIGURED');

    const realPushRes = await request.post('/api/sync/parent-dmz/push-projection', {
      data: { pupil_id: m1Seeded.active_pupil_id },
    });
    expect(realPushRes.status()).toBe(503);
    expect((await realPushRes.json()).error_code).toContain('REAL_PARENT_DMZ_NOT_CONFIGURED');

    const realPullRes = await request.post('/api/sync/parent-dmz/pull-inbox', {
      data: {},
    });
    expect(realPullRes.status()).toBe(503);
    expect((await realPullRes.json()).error_code).toContain('REAL_PARENT_DMZ_NOT_CONFIGURED');
  });

  test('matCentralSync_pullPoliciesAndPushAggregatedAnalytics_persistsPoliciesExcludesPiiAndFailsFastWhenRealModeUnconfigured_test', async ({
    request,
  }) => {
    if (!personas || !m1Seeded) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // 1. Switch to Admin persona
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personas.admin_staff_id },
    });
    expect(switchAdminRes.ok()).toBe(true);

    // 2. Seed Mock MAT Central Sync Feed (ENABLE_REAL_MAT_SYNC=false)
    const seedMatRes = await request.post('/api/test/mat/seed-feed', {
      data: {
        mat_id: 'MAT_OPTIMIS_TRUST_01',
        grading_schema_version: 'KS4_9_TO_1_V2026',
        behaviour_taxonomy: ['MERIT_EXCELLENCE', 'RESTORATIVE_CONVERSATION', 'DETENTION_L1'],
        approved_plugins: [
          { id: 'uk.gov.dfe.census', version: '1.0.0' },
          { id: 'uk.org.optimis.test', version: '0.1.0' },
        ],
      },
    });
    expect(seedMatRes.ok()).toBe(true);

    // 3. School Container pulls MAT policies outbound and persists them in core.db
    const pullPoliciesRes = await request.post('/api/sync/mat/pull-policies', {
      data: {},
    });
    expect(pullPoliciesRes.status()).toBe(200);
    const pullPoliciesBody = await pullPoliciesRes.json();
    expect(pullPoliciesBody.ok).toBe(true);
    expect(pullPoliciesBody.policy_bundle.mat_id).toBe('MAT_OPTIMIS_TRUST_01');
    expect(pullPoliciesBody.policy_bundle.grading_schema_version).toBe('KS4_9_TO_1_V2026');
    expect(pullPoliciesBody.policy_bundle.behaviour_taxonomy).toEqual([
      'MERIT_EXCELLENCE',
      'RESTORATIVE_CONVERSATION',
      'DETENTION_L1',
    ]);
    expect(pullPoliciesBody.policy_bundle.approved_plugins.length).toBe(2);

    // 4. School Container pushes aggregated school-level MAT analytics outbound (strictly zero pupil-level PII)
    const pushAnalyticsRes = await request.post('/api/sync/mat/push-analytics', {
      data: {
        academic_year_id: m1Seeded.academic_year_id,
      },
    });
    expect(pushAnalyticsRes.status()).toBe(200);
    const pushAnalyticsBody = await pushAnalyticsRes.json();
    expect(pushAnalyticsBody.ok).toBe(true);
    expect(pushAnalyticsBody.analytics.academic_year_id).toBe(m1Seeded.academic_year_id);
    expect(typeof pushAnalyticsBody.analytics.school_urn).toBe('string');
    expect(pushAnalyticsBody.analytics.enrolled_pupil_count).toBeGreaterThanOrEqual(3);
    expect(pushAnalyticsBody.analytics.sen_pupil_count).toBeGreaterThanOrEqual(1);

    const rawAnalyticsJson = JSON.stringify(pushAnalyticsBody.analytics);
    expect(rawAnalyticsJson).not.toContain(m1Seeded.active_pupil_id);
    expect(rawAnalyticsJson).not.toContain('Lovelace');
    expect(rawAnalyticsJson).not.toContain('upn');

    // 5. Flip ENABLE_REAL_MAT_SYNC=true without real MAT sync feed config in __system_config -> must fail fast with 503 REAL_MAT_SYNC_NOT_CONFIGURED
    const enableRealMatRes = await request.post('/api/test/external-deps-mode', {
      data: { enable_real_mat_sync: true },
    });
    expect(enableRealMatRes.status()).toBe(503);
    expect((await enableRealMatRes.json()).error_code).toContain('REAL_MAT_SYNC_NOT_CONFIGURED');

    const realPullPoliciesRes = await request.post('/api/sync/mat/pull-policies', {
      data: {},
    });
    expect(realPullPoliciesRes.status()).toBe(503);
    expect((await realPullPoliciesRes.json()).error_code).toContain('REAL_MAT_SYNC_NOT_CONFIGURED');

    const realPushAnalyticsRes = await request.post('/api/sync/mat/push-analytics', {
      data: { academic_year_id: m1Seeded.academic_year_id },
    });
    expect(realPushAnalyticsRes.status()).toBe(503);
    expect((await realPushAnalyticsRes.json()).error_code).toContain('REAL_MAT_SYNC_NOT_CONFIGURED');
  });
});
