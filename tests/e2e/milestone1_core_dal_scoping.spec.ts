import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

interface Milestone1SeedResponse {
  fixture_id: string;
  academic_year_id: string;
  reference_date: string;
  teacher_staff_id: string;
  other_teacher_staff_id: string;
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

test.describe('Milestone 1: Coarse-Grained Core DAL Domain Methods & Dynamic :own_groups Scoping', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let seeded: Milestone1SeedResponse | null = null;

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

    // 3. Seed isolated Milestone 1 fixtures explicitly in setup
    const seedRes = await request.post('/api/test/seed-milestone1-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-27',
      },
    });
    if (!seedRes.ok()) {
      throw new Error(`Failed to seed Milestone 1 fixtures: ${seedRes.status()}`);
    }
    const body = (await seedRes.json()) as Milestone1SeedResponse;
    if (
      !body.fixture_id ||
      !body.own_group_id ||
      !body.other_group_id ||
      !body.active_pupil_id ||
      !body.transferred_pupil_id ||
      !body.other_group_pupil_id ||
      !body.teacher_staff_id
    ) {
      throw new Error('Seeded Milestone 1 response missing mandatory fixture identifiers');
    }
    seeded = body;
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

    // 3. Clean up seeded Milestone 1 test data at the end of the test
    if (seeded?.fixture_id) {
      const delRes = await request.delete(
        `/api/test/seed-milestone1-fixtures/${encodeURIComponent(seeded.fixture_id)}`
      );
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up Milestone 1 fixture ${seeded.fixture_id}: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  function buildWorkspaceUrl(f: Milestone1SeedResponse): string {
    const params = new URLSearchParams({
      pupil_id: f.active_pupil_id,
      group_id: f.own_group_id,
      other_group_id: f.other_group_id,
      reference_date: f.reference_date,
      academic_year_id: f.academic_year_id,
      staff_id: f.teacher_staff_id,
    });
    return `/?${params.toString()}`;
  }

  test('getGroupRosterSummary_teacherWithOwnGroupsScope_returnsOnlyActiveOwnGroupPupilsAndExcludesTransferred_test', async ({
    page,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    await page.goto(buildWorkspaceUrl(seeded));

    const hostShell = page.getByTestId('host-shell');
    await expect(hostShell).toBeVisible();

    const iframeEl = page.getByTestId('plugin-iframe-uk.org.optimis.test');
    await expect(iframeEl).toBeVisible();
    await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    const loadRosterBtn = pluginFrame.getByTestId('load-own-group-roster-btn');
    await expect(loadRosterBtn).toBeVisible();
    await expect(loadRosterBtn).toHaveAttribute('data-mis-rpc', 'get_group_roster_summary');

    await loadRosterBtn.click();

    const rosterResult = pluginFrame.getByTestId('roster-result');
    await expect(rosterResult).toBeVisible();
    await expect(rosterResult).toContainText(seeded.active_pupil_id);
    await expect(rosterResult).toContainText('Ada Lovelace');
    await expect(rosterResult).toContainText('E');
    await expect(rosterResult).toContainText('Anaphylaxis (Peanut Allergy)');

    // Transferred pupil (effective_to < reference_date) and other teacher's pupil MUST NOT appear
    const rosterText = await rosterResult.textContent();
    expect(rosterText).not.toContain(seeded.transferred_pupil_id);
    expect(rosterText).not.toContain(seeded.other_group_pupil_id);
  });

  test('getGroupRosterSummary_teacherWithOwnGroupsScopeRequestingOtherTeachersGroup_returnsAccessDeniedError_test', async ({
    page,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    await page.goto(buildWorkspaceUrl(seeded));

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    const loadOtherGroupBtn = pluginFrame.getByTestId('load-other-group-roster-btn');
    await expect(loadOtherGroupBtn).toBeVisible();
    await expect(loadOtherGroupBtn).toHaveAttribute('data-mis-rpc', 'get_group_roster_summary');

    await loadOtherGroupBtn.click();

    const errorBanner = pluginFrame.getByTestId('rpc-error-banner');
    await expect(errorBanner).toBeVisible();
    await expect(errorBanner).toContainText('own_groups');

    const errorBannerText = await errorBanner.textContent();
    expect(errorBannerText).not.toContain(seeded.other_group_pupil_id);
  });

  test('getPupilGuardians_pupilWithCourtOrderRestrictedFatherAndConfidentialHousehold_returnsOnlyUnrestrictedPRGuardianAndSafeAddress_test', async ({
    page,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    await page.goto(buildWorkspaceUrl(seeded));

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    const loadGuardiansBtn = pluginFrame.getByTestId('load-pupil-guardians-btn');
    await expect(loadGuardiansBtn).toBeVisible();
    await expect(loadGuardiansBtn).toHaveAttribute('data-mis-rpc', 'get_pupil_guardians');

    await loadGuardiansBtn.click();

    const guardiansResult = pluginFrame.getByTestId('guardians-result');
    await expect(guardiansResult).toBeVisible();
    await expect(guardiansResult).toContainText('Anne Byron');
    await expect(guardiansResult).toContainText('MOTHER');
    await expect(guardiansResult).toContainText('42 High Street');

    // Court-order-restricted father and confidential refuge address MUST NEVER leak to UI
    const guardiansText = await guardiansResult.textContent();
    expect(guardiansText).not.toContain('George Byron');
    expect(guardiansText).not.toContain(seeded.restricted_father_person_id);
    expect(guardiansText).not.toContain('Prohibited Steps Order');
    expect(guardiansText).not.toContain('10 Safe Haven Way');
    expect(guardiansText).not.toContain('SW1A 1AA');
  });

  test('recordGroupAmAttendance_validAmMarksForOwnGroup_persistsStatutoryAmMarksAndEmitsEvent_test', async ({
    page,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    await page.goto(buildWorkspaceUrl(seeded));

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    const recordAttendanceBtn = pluginFrame.getByTestId('record-am-attendance-btn');
    await expect(recordAttendanceBtn).toBeVisible();
    await expect(recordAttendanceBtn).toHaveAttribute('data-mis-rpc', 'record_group_am_attendance');

    await recordAttendanceBtn.click();

    const attendanceResult = pluginFrame.getByTestId('attendance-result');
    await expect(attendanceResult).toBeVisible();
    await expect(attendanceResult).toContainText('SESSION_SAVED:AM');
    await expect(attendanceResult).toContainText(seeded.active_pupil_id);
    await expect(attendanceResult).toContainText('attendance.session_marks_recorded');
  });

  test('submitSafeguardingPostbox_generalTeacherSubmitsConcern_createsEncryptedPostboxNoteAndBlocksRead_test', async ({
    page,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    await page.goto(buildWorkspaceUrl(seeded));

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    const submitSafeguardingBtn = pluginFrame.getByTestId('submit-safeguarding-postbox-btn');
    await expect(submitSafeguardingBtn).toBeVisible();
    await expect(submitSafeguardingBtn).toHaveAttribute('data-mis-rpc', 'submit_safeguarding_postbox');

    await submitSafeguardingBtn.click();

    const safeguardingResult = pluginFrame.getByTestId('safeguarding-postbox-result');
    await expect(safeguardingResult).toBeVisible();
    await expect(safeguardingResult).toContainText('POSTBOX_ACCEPTED');
    await expect(safeguardingResult).toContainText('sfg_');
    await expect(safeguardingResult).toContainText('"read_back_blocked":true');
  });
});
