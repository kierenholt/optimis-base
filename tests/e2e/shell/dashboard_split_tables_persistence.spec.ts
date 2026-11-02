import { test, expect } from '@playwright/test';
import {
  setupWorkspaceEnvironment,
  cleanupWorkspaceEnvironment,
  seedTestPersonas,
  expectDashboardLayout,
  expectWidgetCoordinates,
} from './workspace_fixtures';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

test.describe('Dashboard Layouts Split Tables & Foreign Key Integrity', () => {
  let seededPupilId: string = '';
  let secondPupilId: string = '';
  let testPlugId: string = '';
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    page.on('console', (msg) => browserLogs.push(`[browser:${msg.type()}] ${msg.text()}`));
    page.on('pageerror', (err) => pageErrors.push(`[pageerror] ${err.stack || err.message}`));

    const cursorRes = await request.get('/api/test/logs/cursor');
    if (!cursorRes.ok()) throw new Error(`Failed to obtain service log cursor: ${cursorRes.status()}`);
    logCursor = (await cursorRes.json()).cursor;

    const env = await setupWorkspaceEnvironment(request);
    seededPupilId = env.seededPupilId;
    secondPupilId = env.secondPupilId;
    testPlugId = env.testPlugId;
  });

  test.afterEach(async ({ request }, testInfo) => {
    if (logCursor) {
      const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
      if (logsRes.ok()) {
        const serviceLogs: string[] = (await logsRes.json()).entries;
        if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
          console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
        }
      }
    }
    if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
      console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
      console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
    }

    await cleanupWorkspaceEnvironment(request, { testPlugId, seededPupilId, secondPupilId });
    expect(pageErrors).toEqual([]);
  });

  test('dashboardLayouts_splitRoleAndStaffTablesWithForeignKeys_persists2DCoordinatesAndEnforcesForeignKeys_test', async ({
    page,
    request,
  }) => {
    // 1. Seed test personas (Teacher, Admin, etc.)
    const personaFixture = await seedTestPersonas(request);
    const teacherStaffId = personaFixture.teacher_staff_id;
    const adminStaffId = personaFixture.admin_staff_id;

    try {
      // 2. Switch to teacher persona and persist personal 2D layout in staff_dashboard_layouts
      const switchTeacher = await request.post('/api/auth/persona', { data: { staff_id: teacherStaffId } });
      expect(switchTeacher.ok()).toBe(true);

      const saveRes = await request.put('/api/dashboard/layout', {
        data: {
          staff_id: teacherStaffId,
          column_count: 4,
          layout: [
            { plug_id: testPlugId, widget_id: 'attendance_pulse', col: 1, row: 1 },
            { plug_id: testPlugId, widget_id: 'pastoral_watch', col: 3, row: 1 },
          ],
        },
      });
      expect(saveRes.ok()).toBe(true);

      // Verify layout is returned from staff_dashboard_layouts
      await expectDashboardLayout(request, {
        layout_source: 'STAFF',
        column_count: 4,
        layout: [
          { plug_id: testPlugId, widget_id: 'attendance_pulse', col: 1, row: 1 },
          { plug_id: testPlugId, widget_id: 'pastoral_watch', col: 3, row: 1 },
        ],
      });

      // 3. Switch to Admin persona and append widget to TEACHER role_dashboard_layouts
      const switchAdmin = await request.post('/api/auth/persona', { data: { staff_id: adminStaffId } });
      expect(switchAdmin.ok()).toBe(true);

      const appendRes = await request.post('/api/admin/dashboard/role-default/append', {
        data: {
          staff_id: adminStaffId,
          role_id: 'TEACHER',
          plug_id: testPlugId,
          widget_id: 'pastoral_watch',
        },
      });
      expect(appendRes.ok()).toBe(true);

      // 4. Verify invalid role_id fails fast
      const invalidRoleRes = await request.post('/api/admin/dashboard/role-default/append', {
        data: {
          staff_id: adminStaffId,
          role_id: 'NON_EXISTENT_ROLE_XYZ',
          plug_id: testPlugId,
          widget_id: 'pastoral_watch',
        },
      });
      expect(invalidRoleRes.status()).toBeGreaterThanOrEqual(400);

      // 5. Navigate to dashboard as teacher and verify widgets render at 2D coordinates
      const switchTeacherBack = await request.post('/api/auth/persona', { data: { staff_id: teacherStaffId } });
      expect(switchTeacherBack.ok()).toBe(true);

      await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}`);
      const attendanceCard = page.getByTestId('placed-widget-card-attendance_pulse');
      const pastoralCard = page.getByTestId('placed-widget-card-pastoral_watch');
      await expect(attendanceCard).toBeVisible();
      await expect(pastoralCard).toBeVisible();
      await expectWidgetCoordinates(attendanceCard, '1', '1');
      await expectWidgetCoordinates(pastoralCard, '3', '1');
    } finally {
      await request.delete(`/api/test/seed-personas/${encodeURIComponent(personaFixture.fixture_id)}`);
    }
  });
});
