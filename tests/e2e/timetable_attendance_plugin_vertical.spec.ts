import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

interface Milestone1TimetableSeedResponse {
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
  timetable_slot_am_id: string;
  timetable_slot_pm_id: string;
}

const PLUGIN_ID = 'uk.org.optimis.timetable.attendance';
const PLUGIN_FOLDER = 'timetable_attendance_plugin';

test.describe('Vertical 1: Timetable & Statutory Session Attendance Plugin (uk.org.optimis.timetable.attendance)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let seeded: Milestone1TimetableSeedResponse | null = null;

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

    // 3. Ensure dev persona switcher is enabled and clean up any previous plugin state
    const authModeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!authModeRes.ok()) {
      throw new Error(`Failed to configure auth mode: ${authModeRes.status()}`);
    }
    await request.delete(`/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}`);

    // 4. Runtime-install and activate timetable_attendance_plugin (pure Core DAL, no attached DB)
    const installRes = await request.post('/api/admin/plugins/install', {
      data: { folder: PLUGIN_FOLDER },
    });
    if (!installRes.ok()) {
      throw new Error(
        `Failed to install ${PLUGIN_FOLDER}: ${installRes.status()} ${await installRes.text()}`
      );
    }
    const installBody = await installRes.json();
    if (!installBody.ok || installBody.plugin_id !== PLUGIN_ID) {
      throw new Error(`Unexpected install response for ${PLUGIN_ID}: ${JSON.stringify(installBody)}`);
    }
    if (installBody.database_alias) {
      throw new Error(`Expected pure Core DAL plugin with empty database_alias, got ${installBody.database_alias}`);
    }

    const activateRes = await request.post(`/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}/activate`, {
      data: {},
    });
    if (!activateRes.ok()) {
      throw new Error(`Failed to activate ${PLUGIN_ID}: ${activateRes.status()}`);
    }

    // 5. Seed isolated timetable, group, SEN/Medical, and pupil fixtures
    const seedRes = await request.post('/api/test/seed-milestone1-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-27',
        include_second_active_pupil: true,
      },
    });
    if (!seedRes.ok()) {
      throw new Error(`Failed to seed fixtures: ${seedRes.status()} ${await seedRes.text()}`);
    }
    const body = (await seedRes.json()) as Milestone1TimetableSeedResponse;
    if (
      !body.fixture_id ||
      !body.teacher_staff_id ||
      !body.other_teacher_staff_id ||
      !body.own_group_id ||
      !body.other_group_id ||
      !body.active_pupil_id ||
      !body.second_active_pupil_id ||
      !body.transferred_pupil_id ||
      !body.other_group_pupil_id ||
      !body.timetable_slot_am_id ||
      !body.timetable_slot_pm_id
    ) {
      throw new Error('Seeded fixture response missing mandatory timetable/attendance identifiers');
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

    // 3. Remove runtime-installed plugin
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete(`/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}`);

    // 4. Clean up seeded fixtures in core.db
    if (seeded?.fixture_id) {
      const delRes = await request.delete(
        `/api/test/seed-milestone1-fixtures/${encodeURIComponent(seeded.fixture_id)}`
      );
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up fixture ${seeded.fixture_id}: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('loadTimetableResolveSessionViewRosterAndSubmitAttendance_teacherWithOwnGroupsScope_resolvesAmPmLoadsSenMedicalFlagsAndUpsertsBatch_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    // 1. Switch to Teacher persona (holds :own_groups scopes for groups, pupils, and attendance)
    const switchRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.teacher_staff_id },
    });
    expect(switchRes.ok()).toBe(true);
    const sessionBody = await switchRes.json();
    expect(sessionBody.ok).toBe(true);

    await page.context().addCookies([
      {
        name: 'mis_session',
        value: sessionBody.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    const params = new URLSearchParams({
      pupil_id: seeded.active_pupil_id,
      group_id: seeded.own_group_id,
      other_group_id: seeded.other_group_id,
      reference_date: seeded.reference_date,
      academic_year_id: seeded.academic_year_id,
      staff_id: seeded.teacher_staff_id,
    });
    await page.goto(`/?${params.toString()}`);

    await expect(page.getByTestId('host-shell')).toBeVisible();

    const navItem = page.getByTestId(`nav-item-plugin-${PLUGIN_ID}`);
    await expect(navItem).toBeVisible();
    await navItem.click();

    const iframeEl = page.getByTestId(`plugin-iframe-${PLUGIN_ID}`);
    await expect(iframeEl).toBeVisible();
    await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${PLUGIN_ID}"]`);
    await expect(pluginFrame.getByTestId('timetable-attendance-root')).toBeVisible();

    // =========================================================================
    // Step 1: Teacher loads rolling timetable (get_staff_timetable_slots) -> resolves AM/PM statutory session
    // =========================================================================
    const cycleWeekSelect = pluginFrame.getByTestId('timetable-cycle-week-select');
    const daySelect = pluginFrame.getByTestId('timetable-day-select');
    const loadTimetableBtn = pluginFrame.getByTestId('load-timetable-btn');

    await expect(cycleWeekSelect).toBeVisible();
    await expect(daySelect).toBeVisible();
    await expect(loadTimetableBtn).toBeVisible();
    await expect(loadTimetableBtn).toHaveAttribute('data-mis-rpc', 'load_staff_timetable');

    // Verify Week 2 resolves statutory PM session first
    await cycleWeekSelect.selectOption('2');
    await loadTimetableBtn.click();

    const resolvedSessionBadge = pluginFrame.getByTestId('resolved-statutory-session-badge');
    await expect(resolvedSessionBadge).toBeVisible();
    await expect(resolvedSessionBadge).toHaveText('PM');
    await expect(pluginFrame.getByTestId(`timetable-slot-card-${seeded.timetable_slot_pm_id}`)).toBeVisible();

    // Switch back to Week 1 and verify it resolves statutory AM session for OwnGroupID
    await cycleWeekSelect.selectOption('1');
    await loadTimetableBtn.click();

    await expect(resolvedSessionBadge).toHaveText('AM');
    const amSlotCard = pluginFrame.getByTestId(`timetable-slot-card-${seeded.timetable_slot_am_id}`);
    await expect(amSlotCard).toBeVisible();
    await expect(amSlotCard).toContainText('P1_AM');
    await expect(amSlotCard).toContainText('AM');

    // =========================================================================
    // Step 2: Teacher views class roster with SEN/Medical flags (list_by_group + list_sen_and_medical_batch)
    // =========================================================================
    const loadRosterBtn = pluginFrame.getByTestId('load-roster-btn');
    await expect(loadRosterBtn).toBeVisible();
    await expect(loadRosterBtn).toHaveAttribute('data-mis-rpc', 'load_slot_roster_with_flags');
    await loadRosterBtn.click();

    const rosterTable = pluginFrame.getByTestId('class-roster-table');
    await expect(rosterTable).toBeVisible();

    // Active Pupil 1 (Ada Lovelace): SEN 'E' (ASD) + Medical 'Anaphylaxis (Peanut Allergy)' (IHCP)
    const rowPupil1 = pluginFrame.getByTestId(`roster-row-${seeded.active_pupil_id}`);
    await expect(rowPupil1).toBeVisible();
    await expect(rowPupil1).toContainText('Ada Lovelace');
    await expect(rowPupil1).toContainText('[Y10]');
    await expect(pluginFrame.getByTestId(`sen-flag-${seeded.active_pupil_id}`)).toContainText('E (ASD)');
    await expect(pluginFrame.getByTestId(`medical-flag-${seeded.active_pupil_id}`)).toContainText(
      'Anaphylaxis (Peanut Allergy)'
    );
    await expect(pluginFrame.getByTestId(`medical-flag-${seeded.active_pupil_id}`)).toContainText('IHCP');

    // Active Pupil 2 (Grace Turing): SEN 'K' (SPLD)
    const rowPupil2 = pluginFrame.getByTestId(`roster-row-${seeded.second_active_pupil_id}`);
    await expect(rowPupil2).toBeVisible();
    await expect(rowPupil2).toContainText('Grace Turing');
    await expect(rowPupil2).toContainText('[Y10]');
    await expect(pluginFrame.getByTestId(`sen-flag-${seeded.second_active_pupil_id}`)).toContainText('K (SPLD)');

    // Transferred pupil (effective_to < reference_date) and other teacher's pupil MUST NOT appear
    await expect(pluginFrame.getByTestId(`roster-row-${seeded.transferred_pupil_id}`)).toHaveCount(0);
    await expect(pluginFrame.getByTestId(`roster-row-${seeded.other_group_pupil_id}`)).toHaveCount(0);

    // =========================================================================
    // Step 3: Teacher submits batch attendance (upsert_session_batch)
    // =========================================================================
    const markSelect1 = pluginFrame.getByTestId(`attendance-code-select-${seeded.active_pupil_id}`);
    const markSelect2 = pluginFrame.getByTestId(`attendance-code-select-${seeded.second_active_pupil_id}`);
    const minutesLateInput2 = pluginFrame.getByTestId(`minutes-late-input-${seeded.second_active_pupil_id}`);
    const commentInput2 = pluginFrame.getByTestId(`attendance-comment-input-${seeded.second_active_pupil_id}`);

    await markSelect1.selectOption('/');
    await markSelect2.selectOption('L');
    await minutesLateInput2.fill('10');
    await commentInput2.fill('Northern Line delay');

    const submitBatchBtn = pluginFrame.getByTestId('submit-attendance-batch-btn');
    await expect(submitBatchBtn).toBeVisible();
    await expect(submitBatchBtn).toHaveAttribute('data-mis-rpc', 'submit_session_attendance_batch');
    await submitBatchBtn.click();

    const submissionReceipt = pluginFrame.getByTestId('attendance-submission-receipt');
    await expect(submissionReceipt).toBeVisible();
    await expect(submissionReceipt).toContainText('BATCH_SAVED:AM');
    await expect(submissionReceipt).toContainText('Upserted: 2');
    await expect(submissionReceipt).toContainText('ars_');
    await expect(submissionReceipt).toContainText('Timestamp:');

    // Verify persisted attendance marks are reflected on the roster rows
    await expect(pluginFrame.getByTestId(`persisted-mark-badge-${seeded.active_pupil_id}`)).toContainText('/');
    await expect(pluginFrame.getByTestId(`persisted-mark-badge-${seeded.second_active_pupil_id}`)).toContainText(
      'L (10m)'
    );
  });

  test('timetableAndAttendanceEndpoints_crossTeacherOwnGroupsViolationOrInvalidStatutorySession_failsFast_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    const switchRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.teacher_staff_id },
    });
    expect(switchRes.ok()).toBe(true);
    const sessionBody = await switchRes.json();

    // 1. Teacher 1 attempting to load Teacher 2's timetable under :own_groups MUST fail fast
    const crossTimetableRes = await request.post(`/api/plugins/${PLUGIN_ID}/rpc/load_staff_timetable`, {
      data: {
        args: {
          target_staff_id: seeded.other_teacher_staff_id,
          cycle_week: 1,
          day_of_week_id: 1,
          reference_date: seeded.reference_date,
          academic_year_id: seeded.academic_year_id,
        },
      },
    });
    expect(crossTimetableRes.status()).toBe(400);
    const crossTimetableBody = await crossTimetableRes.json();
    expect(crossTimetableBody.ok).toBe(false);
    expect(crossTimetableBody.error_code).toContain('access denied under own_groups scope');

    // 2. Teacher 1 attempting to submit attendance for an invalid non-AM/PM session ('BREAK') MUST fail fast
    const invalidSessionRes = await request.post(`/api/plugins/${PLUGIN_ID}/rpc/submit_session_attendance_batch`, {
      data: {
        args: {
          group_id: seeded.own_group_id,
          date: seeded.reference_date,
          session: 'BREAK',
          academic_year_id: seeded.academic_year_id,
          marks: [{ pupil_id: seeded.active_pupil_id, attendance_code: '/' }],
        },
      },
    });
    expect(invalidSessionRes.status()).toBe(400);
    const invalidSessionBody = await invalidSessionRes.json();
    expect(invalidSessionBody.ok).toBe(false);
    expect(invalidSessionBody.error_code).toContain('invalid statutory session');

    // 3. Verify UI surfaces fail-fast access-denied banner when Teacher 1 requests Teacher 2's group roster
    await page.context().addCookies([
      {
        name: 'mis_session',
        value: sessionBody.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    const params = new URLSearchParams({
      pupil_id: seeded.active_pupil_id,
      group_id: seeded.own_group_id,
      other_group_id: seeded.other_group_id,
      reference_date: seeded.reference_date,
      academic_year_id: seeded.academic_year_id,
      staff_id: seeded.teacher_staff_id,
    });
    await page.goto(`/?${params.toString()}`);

    await page.getByTestId(`nav-item-plugin-${PLUGIN_ID}`).click();
    const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${PLUGIN_ID}"]`);

    const unauthorizedGroupBtn = pluginFrame.getByTestId('load-other-group-roster-btn');
    await expect(unauthorizedGroupBtn).toBeVisible();
    await expect(unauthorizedGroupBtn).toHaveAttribute('data-mis-rpc', 'load_slot_roster_with_flags');
    await unauthorizedGroupBtn.click();

    const errorBanner = pluginFrame.getByTestId('timetable-error-banner');
    await expect(errorBanner).toBeVisible();
    await expect(errorBanner).toContainText('own_groups');
  });
});
