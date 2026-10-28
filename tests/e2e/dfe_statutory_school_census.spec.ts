import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

interface SeededCensusFixtures {
  fixture_id: string;
  academic_year_id: string;
  census_date: string;
  term_code: string;
  attendance_start_date: string;
  attendance_end_date: string;
  admin_staff_id: string;
  teacher_staff_id: string;
  pupil_sen_k_fsm_id: string;
  pupil_ehcp_lac_id: string;
  pupil_no_sen_id: string;
  off_roll_pupil_id: string;
}

test.describe('DfE Statutory School Census Plugin, Host XML/XSD Validator & Mock COLLECT Service (Milestone 2B)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let seeded: SeededCensusFixtures | null = null;

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

    // 3. Ensure dev persona switcher, mock egress mode, and Mock DfE COLLECT service flag are enabled
    const authModeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!authModeRes.ok()) {
      throw new Error(`Failed to configure auth mode: ${authModeRes.status()}`);
    }

    const egressModeRes = await request.post('/api/test/egress-mode', {
      data: { enable_real_http_egress: false },
    });
    if (!egressModeRes.ok()) {
      throw new Error(`Failed to configure egress mode: ${egressModeRes.status()}`);
    }

    const censusModeRes = await request.post('/api/test/census-service-mode', {
      data: { enable_mock_dfe_census_service: true },
    });
    if (!censusModeRes.ok()) {
      throw new Error(`Failed to enable mock DfE census service: ${censusModeRes.status()}`);
    }

    const clearSubmissionsRes = await request.delete('/api/test/census-submissions');
    if (!clearSubmissionsRes.ok()) {
      throw new Error(`Failed to clear mock census submissions: ${clearSubmissionsRes.status()}`);
    }

    // 4. Seed isolated statutory school census fixtures
    const seedRes = await request.post('/api/test/seed-census-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        census_date: '2026-10-01',
        term_code: 'AUTUMN',
        attendance_start_date: '2026-09-01',
        attendance_end_date: '2026-10-01',
      },
    });
    if (!seedRes.ok()) {
      throw new Error(`Failed to seed census fixtures: ${seedRes.status()} ${await seedRes.text()}`);
    }
    const body = (await seedRes.json()) as SeededCensusFixtures;
    if (
      !body.fixture_id ||
      !body.academic_year_id ||
      !body.census_date ||
      !body.term_code ||
      !body.admin_staff_id ||
      !body.teacher_staff_id ||
      !body.pupil_sen_k_fsm_id ||
      !body.pupil_ehcp_lac_id ||
      !body.pupil_no_sen_id ||
      !body.off_roll_pupil_id
    ) {
      throw new Error('Seeded census fixture response missing mandatory identifiers');
    }
    seeded = body;

    // 5. Install and activate dfe_census_plugin (uk.gov.dfe.census) at runtime
    const installRes = await request.post('/api/admin/plugins/install', {
      data: {
        folder: 'dfe_census_plugin',
        staff_id: seeded.admin_staff_id,
      },
    });
    if (!installRes.ok()) {
      throw new Error(`Failed to install dfe_census_plugin: ${installRes.status()}`);
    }
    const activateRes = await request.post('/api/admin/plugins/uk.gov.dfe.census/activate', {
      data: {
        staff_id: seeded.admin_staff_id,
      },
    });
    if (!activateRes.ok()) {
      throw new Error(`Failed to activate uk.gov.dfe.census: ${activateRes.status()}`);
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Restore Mock DfE COLLECT service flag and clear submissions
    await request.post('/api/test/census-service-mode', {
      data: { enable_mock_dfe_census_service: true },
    });
    await request.delete('/api/test/census-submissions');

    // 2. Purge plugin census snapshots
    if (seeded) {
      await request.post('/api/auth/persona', {
        data: { staff_id: seeded.admin_staff_id },
      });
      for (const pupilId of [
        seeded.pupil_sen_k_fsm_id,
        seeded.pupil_ehcp_lac_id,
        seeded.pupil_no_sen_id,
        seeded.off_roll_pupil_id,
      ]) {
        await request.post('/api/plugins/uk.gov.dfe.census/rpc/purge_pupil_data', {
          data: { args: { pupil_id: pupilId } },
        });
      }
    }

    // 3. Fetch Backend Service Logs emitted during this test
    if (logCursor) {
      const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
      if (logsRes.ok()) {
        const serviceLogs: string[] = (await logsRes.json()).entries;
        if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
          console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
        }
      }
    }

    // 4. Dump tapped Browser Console logs on failure or error
    if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
      console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
      console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
    }

    // 5. Remove runtime-installed dfe_census_plugin
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete('/api/admin/plugins/uk.gov.dfe.census');

    // 6. Clean up seeded census fixtures in core.db
    if (seeded?.fixture_id) {
      const delRes = await request.delete(
        `/api/test/seed-census-fixtures/${encodeURIComponent(seeded.fixture_id)}`
      );
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up census fixture ${seeded.fixture_id}: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('generateValidateAndSubmitSchoolCensus_withMockCollectServiceEnabled_rendersRollupsValidatesXsdAndSubmitsReturn_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded census state');
    }

    // 1. Switch to Admin persona (who holds dal.census.read:all) and attach session cookie
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.admin_staff_id },
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

    // 2. Navigate to Host UI Shell and open the DfE Statutory School Census plugin
    const query = new URLSearchParams({
      pupil_id: seeded.pupil_sen_k_fsm_id,
      reference_date: seeded.census_date,
      academic_year_id: seeded.academic_year_id,
      staff_id: seeded.admin_staff_id,
    });
    await page.goto(`/?${query.toString()}`);

    const hostShell = page.getByTestId('host-shell');
    await expect(hostShell).toBeVisible();

    const censusNavBtn = page.getByTestId('nav-item-plugin-uk.gov.dfe.census');
    await expect(censusNavBtn).toBeVisible();
    await censusNavBtn.click();

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.gov.dfe.census"]');
    const termSelect = pluginFrame.getByTestId('census-term-select');
    const dateInput = pluginFrame.getByTestId('census-date-input');
    const generateBtn = pluginFrame.getByTestId('generate-census-btn');

    await expect(termSelect).toBeVisible();
    await expect(dateInput).toBeVisible();
    await expect(generateBtn).toBeVisible();
    await expect(generateBtn).toHaveAttribute('data-mis-rpc', 'generate_and_validate_census_return');

    // 3. Trigger Statutory School Census Generation & XSD Validation inside the sandboxed iframe
    await generateBtn.click();

    const summaryCards = pluginFrame.getByTestId('census-summary-cards');
    await expect(summaryCards).toBeVisible();

    const xsdStatus = pluginFrame.getByTestId('census-xsd-status');
    await expect(xsdStatus).toHaveText('XSD_VALID');

    const onRollCount = pluginFrame.getByTestId('census-on-roll-count');
    await expect(onRollCount).toBeVisible();
    const onRollValue = Number((await onRollCount.textContent())?.trim());
    expect(onRollValue).toBeGreaterThanOrEqual(3);

    const senBreakdown = pluginFrame.getByTestId('census-sen-breakdown');
    await expect(senBreakdown).toContainText('K:');
    await expect(senBreakdown).toContainText('E:');

    const fsmCount = pluginFrame.getByTestId('census-fsm-count');
    await expect(fsmCount).toBeVisible();
    expect(Number((await fsmCount.textContent())?.trim())).toBeGreaterThanOrEqual(2);

    const lacCount = pluginFrame.getByTestId('census-lac-count');
    await expect(lacCount).toBeVisible();
    expect(Number((await lacCount.textContent())?.trim())).toBeGreaterThanOrEqual(1);

    const attendanceSummary = pluginFrame.getByTestId('census-attendance-summary');
    await expect(attendanceSummary).toBeVisible();

    // 4. Inspect generated DfE CBDS XML payload in the UI preview
    const xmlPreview = pluginFrame.getByTestId('census-xml-preview');
    await expect(xmlPreview).toBeVisible();
    const xmlText = (await xmlPreview.textContent()) || '';

    expect(xmlText).toContain('<?xml version="1.0" encoding="UTF-8"?>');
    expect(xmlText).toContain('<Collection>SchoolCensus</Collection>');
    expect(xmlText).toContain('<Term>AUTUMN</Term>');
    expect(xmlText).toContain('<CensusDate>2026-10-01</CensusDate>');
    expect(xmlText).toContain('<SENProvision>K</SENProvision>');
    expect(xmlText).toContain('<SENProvision>E</SENProvision>');
    expect(xmlText).toContain('<FSMPeriods>');
    expect(xmlText).toContain('<LookedAfter>1</LookedAfter>');
    expect(xmlText).toContain('<PossibleSessions>');
    expect(xmlText).toContain('<SessionsPresent>');
    expect(xmlText).toContain('<AuthorisedAbsenceSessions>');
    expect(xmlText).toContain('<UnauthorisedAbsenceSessions>');

    // Verify off-roll pupil (effective_to = 2026-09-15 < census_date 2026-10-01) is strictly excluded from the generated return
    const listReturnsRes = await request.post('/api/plugins/uk.gov.dfe.census/rpc/list_census_returns', {
      data: { args: { academic_year_id: seeded.academic_year_id } },
    });
    expect(listReturnsRes.ok()).toBe(true);
    const listReturnsBody = await listReturnsRes.json();
    expect(listReturnsBody.ok).toBe(true);
    expect(listReturnsBody.data.returns.length).toBeGreaterThanOrEqual(1);

    const exportOffRollRes = await request.post('/api/plugins/uk.gov.dfe.census/rpc/export_pupil_data', {
      data: { args: { pupil_id: seeded.off_roll_pupil_id } },
    });
    expect(exportOffRollRes.ok()).toBe(true);
    const exportOffRollBody = await exportOffRollRes.json();
    expect(exportOffRollBody.ok).toBe(true);
    expect(exportOffRollBody.data.sections[0].records.length).toBe(0);

    // 5. Submit validated census return to the Mock DfE COLLECT service via UI button
    const submitBtn = pluginFrame.getByTestId('submit-census-collect-btn');
    await expect(submitBtn).toBeVisible();
    await expect(submitBtn).toHaveAttribute('data-mis-rpc', 'submit_census_return');
    await submitBtn.click();

    const collectReceipt = pluginFrame.getByTestId('census-collect-receipt');
    await expect(collectReceipt).toBeVisible();
    await expect(collectReceipt).toContainText('COLLECT_ACCEPTED');
    await expect(collectReceipt).toContainText('collect_sub_');

    // 6. Verify the Mock DfE COLLECT service recorded the outbound XML submission
    const submissionsRes = await request.get('/api/test/census-submissions');
    expect(submissionsRes.ok()).toBe(true);
    const submissionsBody = await submissionsRes.json();
    expect(submissionsBody.ok).toBe(true);
    expect(submissionsBody.submissions.length).toBe(1);
    expect(submissionsBody.submissions[0].status).toBe('COLLECT_ACCEPTED');
    expect(submissionsBody.submissions[0].term_code).toBe('AUTUMN');
    expect(submissionsBody.submissions[0].census_date).toBe('2026-10-01');
    expect(submissionsBody.submissions[0].xml_payload).toContain('<Collection>SchoolCensus</Collection>');
  });

  test('submitSchoolCensus_whenMockCollectFeatureFlagDisabledOrInvalidXsdOrUnauthorizedTeacher_failsFast_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded census state');
    }

    // 1. Switch to Admin persona and generate a valid census return first
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.admin_staff_id },
    });
    expect(switchAdminRes.ok()).toBe(true);
    const adminSessionBody = await switchAdminRes.json();

    const genRes = await request.post(
      '/api/plugins/uk.gov.dfe.census/rpc/generate_and_validate_census_return',
      {
        data: {
          args: {
            academic_year_id: seeded.academic_year_id,
            census_date: seeded.census_date,
            term_code: seeded.term_code,
            attendance_start_date: '2026-09-01',
            attendance_end_date: '2026-10-01',
          },
        },
      }
    );
    expect(genRes.ok()).toBe(true);
    const genBody = await genRes.json();
    expect(genBody.ok).toBe(true);
    expect(genBody.data.xsd_valid).toBe(true);
    const returnId = genBody.data.return_id;
    expect(returnId).toBeTruthy();

    // 2. Disable ENABLE_MOCK_DFE_CENSUS_SERVICE and verify fail-fast rejection on submission & mock endpoint
    const disableFlagRes = await request.post('/api/test/census-service-mode', {
      data: { enable_mock_dfe_census_service: false },
    });
    expect(disableFlagRes.ok()).toBe(true);

    const disabledListRes = await request.get('/api/test/census-submissions');
    expect(disabledListRes.ok()).toBe(false);
    const disabledListBody = await disabledListRes.json();
    expect(disabledListBody.error_code).toContain('MOCK_DFE_CENSUS_SERVICE_DISABLED');

    const disabledSubmitRes = await request.post(
      '/api/plugins/uk.gov.dfe.census/rpc/submit_census_return',
      {
        data: { args: { return_id: returnId } },
      }
    );
    expect(disabledSubmitRes.ok()).toBe(false);
    const disabledSubmitBody = await disabledSubmitRes.json();
    expect(disabledSubmitBody.ok).toBe(false);
    expect(disabledSubmitBody.error_code).toContain('MOCK_DFE_CENSUS_SERVICE_DISABLED');

    // Also verify the UI surfaces the fail-fast error banner when Mock COLLECT is disabled
    await page.context().addCookies([
      {
        name: 'mis_session',
        value: adminSessionBody.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);
    await page.goto(
      `/?pupil_id=${encodeURIComponent(seeded.pupil_sen_k_fsm_id)}&reference_date=2026-10-01&academic_year_id=ay_2026_2027`
    );
    await page.getByTestId('nav-item-plugin-uk.gov.dfe.census').click();

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.gov.dfe.census"]');
    await pluginFrame.getByTestId('generate-census-btn').click();
    await expect(pluginFrame.getByTestId('census-xsd-status')).toHaveText('XSD_VALID');

    await pluginFrame.getByTestId('submit-census-collect-btn').click();
    const errorBanner = pluginFrame.getByTestId('census-error-banner');
    await expect(errorBanner).toBeVisible();
    await expect(errorBanner).toContainText('MOCK_DFE_CENSUS_SERVICE_DISABLED');

    // Re-enable Mock DfE COLLECT service flag
    await request.post('/api/test/census-service-mode', {
      data: { enable_mock_dfe_census_service: true },
    });

    // 3. Verify XSD schema validation rejects malformed UPN in census XML
    const invalidXsdRes = await request.post(
      '/api/plugins/uk.gov.dfe.census/rpc/generate_and_validate_census_return',
      {
        data: {
          args: {
            academic_year_id: seeded.academic_year_id,
            census_date: seeded.census_date,
            term_code: seeded.term_code,
            attendance_start_date: '2026-09-01',
            attendance_end_date: '2026-10-01',
            tamper_invalid_upn: 'BAD_UPN_99',
          },
        },
      }
    );
    expect(invalidXsdRes.status()).toBe(400);
    const invalidXsdBody = await invalidXsdRes.json();
    expect(invalidXsdBody.ok).toBe(false);
    expect(invalidXsdBody.error_code).toContain('XSD_VALIDATION_FAILED');

    // 4. Verify RBAC: Teacher persona without dal.census.read:all is blocked from generating census return
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);

    const unauthorizedCensusRes = await request.post(
      '/api/plugins/uk.gov.dfe.census/rpc/generate_and_validate_census_return',
      {
        data: {
          args: {
            academic_year_id: seeded.academic_year_id,
            census_date: seeded.census_date,
            term_code: seeded.term_code,
          },
        },
      }
    );
    expect(unauthorizedCensusRes.status()).toBe(403);
    const unauthorizedCensusBody = await unauthorizedCensusRes.json();
    expect(unauthorizedCensusBody.ok).toBe(false);
    expect(unauthorizedCensusBody.error_code).toContain('caller lacks required DAL scopes');

    // 5. Switch back to Admin and verify UK GDPR export_pupil_data & purge_pupil_data hooks on uk.gov.dfe.census
    await request.post('/api/auth/persona', {
      data: { staff_id: seeded.admin_staff_id },
    });

    const exportRes = await request.post('/api/plugins/uk.gov.dfe.census/rpc/export_pupil_data', {
      data: { args: { pupil_id: seeded.pupil_sen_k_fsm_id } },
    });
    expect(exportRes.ok()).toBe(true);
    const exportBody = await exportRes.json();
    expect(exportBody.ok).toBe(true);
    expect(exportBody.data.sections.length).toBe(1);
    expect(exportBody.data.sections[0].title).toBe('DfE Statutory School Census Snapshots');
    expect(exportBody.data.sections[0].records.length).toBeGreaterThanOrEqual(1);
    expect(exportBody.data.sections[0].redactable_fields).toContain('notes');

    const purgeRes = await request.post('/api/plugins/uk.gov.dfe.census/rpc/purge_pupil_data', {
      data: { args: { pupil_id: seeded.pupil_sen_k_fsm_id } },
    });
    expect(purgeRes.ok()).toBe(true);
    const purgeBody = await purgeRes.json();
    expect(purgeBody.ok).toBe(true);
    expect(purgeBody.data.purged_rows).toBeGreaterThanOrEqual(1);

    const exportAfterPurgeRes = await request.post(
      '/api/plugins/uk.gov.dfe.census/rpc/export_pupil_data',
      {
        data: { args: { pupil_id: seeded.pupil_sen_k_fsm_id } },
      }
    );
    expect(exportAfterPurgeRes.ok()).toBe(true);
    const exportAfterPurgeBody = await exportAfterPurgeRes.json();
    expect(exportAfterPurgeBody.data.sections[0].records.length).toBe(0);
  });
});
