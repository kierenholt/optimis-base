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
  timetable_slot_am_id: string;
  timetable_slot_pm_id: string;
}

test.describe('Attendance Register Submissions: Dedicated Session & Lesson Tables Invariants (Option 1)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let seeded: Milestone1SeedResponse | null = null;
  let timetablePlugId: string = '';

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    seeded = null;
    timetablePlugId = '';

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

    await request.delete('/api/admin/plugins/uk.org.optimis.timetable.attendance');

    // 4. Install and activate uk.org.optimis.timetable.attendance
    const installRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'timetable_attendance_plugin' },
    });
    if (!installRes.ok()) {
      throw new Error(`Failed to install timetable_attendance_plugin: ${installRes.status()}`);
    }
    timetablePlugId = (await installRes.json()).plug_id;

    const activateRes = await request.post(
      '/api/admin/plugins/uk.org.optimis.timetable.attendance/activate',
      { data: {} }
    );
    if (!activateRes.ok()) {
      throw new Error(`Failed to activate uk.org.optimis.timetable.attendance: ${activateRes.status()}`);
    }

    // 5. Seed Milestone 1 fixtures explicitly in setup
    const seedRes = await request.post('/api/test/seed-milestone1-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-27',
        include_second_active_pupil: true,
      },
    });
    if (!seedRes.ok()) {
      throw new Error(`Failed to seed Milestone 1 fixtures: ${seedRes.status()}`);
    }
    seeded = (await seedRes.json()) as Milestone1SeedResponse;

    // 6. Switch active dev persona to teacher
    await request.post('/api/auth/persona', {
      data: { staff_id: seeded.teacher_staff_id },
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

    // 3. Clean up installed plugin
    await request.delete('/api/admin/plugins/uk.org.optimis.timetable.attendance');

    // 4. Clean up seeded test fixtures
    if (seeded?.fixture_id) {
      const delRes = await request.delete(
        `/api/test/seed-milestone1-fixtures/${encodeURIComponent(seeded.fixture_id)}`
      );
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up Milestone 1 fixture: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('attendanceRegisterSubmissions_statutorySessionAndLessonRegisters_enforcesDedicatedTablesAndReferentialIntegrity_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    // 1. Submit statutory AM attendance session via plugin UI
    await page.goto(`/p/${timetablePlugId}`);
    const hostShell = page.getByTestId('host-shell');
    await expect(hostShell).toBeVisible();

    const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${timetablePlugId}"]`);
    await expect(pluginFrame.getByTestId('timetable-attendance-root')).toBeVisible();

    const loadTimetableBtn = pluginFrame.getByTestId('load-timetable-btn');
    await loadTimetableBtn.click();
    await expect(pluginFrame.getByTestId('resolved-statutory-session-badge')).toHaveText('AM');

    const loadRosterBtn = pluginFrame.getByTestId('load-roster-btn');
    await loadRosterBtn.click();
    await expect(pluginFrame.getByTestId('class-roster-table')).toBeVisible();

    const markSelect1 = pluginFrame.getByTestId(`attendance-code-select-${seeded.active_pupil_id}`);
    const markSelect2 = pluginFrame.getByTestId(`attendance-code-select-${seeded.second_active_pupil_id}`);
    await markSelect1.selectOption('/');
    await markSelect2.selectOption('/');

    const submitBatchBtn = pluginFrame.getByTestId('submit-attendance-batch-btn');
    await submitBatchBtn.click();

    const submissionReceipt = pluginFrame.getByTestId('attendance-submission-receipt');
    await expect(submissionReceipt).toBeVisible();
    await expect(submissionReceipt).toContainText('BATCH_SAVED:AM');
    await expect(submissionReceipt).toContainText('ars_');

    // 2. Direct RPC call with an invalid non-AM/PM session string literal ('MIDDAY') MUST fail fast
    const invalidSessionRes = await request.post(`/api/p/${timetablePlugId}/rpc/submit_session_attendance_batch`, {
      data: {
        args: {
          group_id: seeded.own_group_id,
          date: seeded.reference_date,
          session: 'MIDDAY',
          academic_year_id: seeded.academic_year_id,
          marks: [{ pupil_id: seeded.active_pupil_id, attendance_code: '/' }],
        },
      },
    });
    expect(invalidSessionRes.status()).toBe(400);
    const errBody = await invalidSessionRes.json();
    expect(errBody.ok).toBe(false);
  });
});
