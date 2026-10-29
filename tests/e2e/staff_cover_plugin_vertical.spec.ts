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

interface Milestone1CoverSeedResponse {
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

const PLUGIN_ID = 'uk.org.optimis.ops.cover';
const PLUGIN_FOLDER = 'staff_cover_plugin';
const PLUGIN_DB_ALIAS = 'plugin_cover';

test.describe('Vertical 8: Staff Absence & Dynamic Cover Assignment Plugin (uk.org.optimis.ops.cover)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let personas: SeededPersonaFixtures | null = null;
  let seeded: Milestone1CoverSeedResponse | null = null;

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    personas = null;
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

    // 3. Enable dev persona switcher and clean up any prior plugin installation
    const authModeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!authModeRes.ok()) {
      throw new Error(`Failed to configure auth mode: ${authModeRes.status()}`);
    }
    await request.delete(`/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}`);

    // 4. Seed personas to get admin_staff_id (Cover Manager / Admin)
    const personaRes = await request.post('/api/test/seed-personas', {
      data: {
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    if (!personaRes.ok()) {
      throw new Error(`Failed to seed persona fixtures: ${personaRes.status()} ${await personaRes.text()}`);
    }
    const personaBody = (await personaRes.json()) as SeededPersonaFixtures;
    if (!personaBody.fixture_id || !personaBody.admin_staff_id) {
      throw new Error('Seeded persona fixture response missing mandatory admin_staff_id');
    }
    personas = personaBody;

    // 5. Seed Milestone 1 fixtures (Absent Teacher, Cover Teacher, classes, pupils, and AM/PM timetable slots)
    const seedRes = await request.post('/api/test/seed-milestone1-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-28',
        include_second_active_pupil: true,
      },
    });
    if (!seedRes.ok()) {
      throw new Error(`Failed to seed Milestone 1 fixtures: ${seedRes.status()} ${await seedRes.text()}`);
    }
    const seedBody = (await seedRes.json()) as Milestone1CoverSeedResponse;
    if (
      !seedBody.fixture_id ||
      !seedBody.teacher_staff_id ||
      !seedBody.other_teacher_staff_id ||
      !seedBody.own_group_id ||
      !seedBody.other_group_id ||
      !seedBody.active_pupil_id ||
      !seedBody.second_active_pupil_id ||
      !seedBody.transferred_pupil_id ||
      !seedBody.timetable_slot_am_id
    ) {
      throw new Error('Seeded Milestone 1 fixture response missing mandatory staff cover identifiers');
    }
    seeded = seedBody;

    // 6. Runtime-install and activate staff_cover_plugin (uk.org.optimis.ops.cover)
    const installRes = await request.post('/api/admin/plugins/install', {
      data: {
        folder: PLUGIN_FOLDER,
        staff_id: personas.admin_staff_id,
      },
    });
    if (!installRes.ok()) {
      throw new Error(
        `Failed to install ${PLUGIN_FOLDER}: ${installRes.status()} ${await installRes.text()}`
      );
    }
    const installBody = await installRes.json();
    expect(installBody.ok).toBe(true);
    expect(installBody.plugin_id).toBe(PLUGIN_ID);
    expect(installBody.database_alias).toBe(PLUGIN_DB_ALIAS);

    const activateRes = await request.post(
      `/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}/activate`,
      {
        data: { staff_id: personas.admin_staff_id },
      }
    );
    if (!activateRes.ok()) {
      throw new Error(
        `Failed to activate ${PLUGIN_ID}: ${activateRes.status()} ${await activateRes.text()}`
      );
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Purge plugin pupil data created during the test
    if (personas && seeded) {
      await request.post('/api/auth/persona', {
        data: { staff_id: personas.admin_staff_id },
      });
      for (const pupilId of [
        seeded.active_pupil_id,
        seeded.second_active_pupil_id,
        seeded.transferred_pupil_id,
      ]) {
        await request.post(`/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/purge_pupil_data`, {
          data: { args: { pupil_id: pupilId } },
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
    if (seeded?.fixture_id) {
      const delM1 = await request.delete(
        `/api/test/seed-milestone1-fixtures/${encodeURIComponent(seeded.fixture_id)}`
      );
      if (!delM1.ok()) {
        throw new Error(`Failed to clean up Milestone 1 fixture ${seeded.fixture_id}: ${delM1.status()}`);
      }
    }
    if (personas?.fixture_id) {
      const delPersonas = await request.delete(
        `/api/test/seed-personas/${encodeURIComponent(personas.fixture_id)}`
      );
      if (!delPersonas.ok()) {
        throw new Error(`Failed to clean up persona fixture ${personas.fixture_id}: ${delPersonas.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('assignCoverTeacherAndTakeRegister_coverTeacherInitiallyLacksOwnGroupsAccessThenAssignedCover_grantsDynamicOwnGroupsAccessLoadsRosterAndSubmitsAttendance_test', async ({
    page,
    request,
  }) => {
    if (!personas || !seeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    // =========================================================================
    // Step 1: Cover Teacher initially lacks :own_groups access to Absent Teacher's class (own_group_id)
    // =========================================================================
    const switchCoverRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.other_teacher_staff_id },
    });
    expect(switchCoverRes.ok()).toBe(true);
    const coverSessionBody = await switchCoverRes.json();
    expect(coverSessionBody.ok).toBe(true);

    await page.context().addCookies([
      {
        name: 'mis_session',
        value: coverSessionBody.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    const params = new URLSearchParams({
      pupil_id: seeded.active_pupil_id,
      group_id: seeded.own_group_id,
      absent_staff_id: seeded.teacher_staff_id,
      cover_staff_id: seeded.other_teacher_staff_id,
      slot_id: seeded.timetable_slot_am_id,
      reference_date: seeded.reference_date,
      academic_year_id: seeded.academic_year_id,
      staff_id: seeded.other_teacher_staff_id,
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
    await expect(pluginFrame.getByTestId('staff-cover-root')).toBeVisible();

    const loadCoverRosterBtn = pluginFrame.getByTestId('load-cover-roster-btn');
    await expect(loadCoverRosterBtn).toBeVisible();
    await expect(loadCoverRosterBtn).toHaveAttribute('data-mis-rpc', 'load_cover_class_roster');
    await loadCoverRosterBtn.click();

    const coverErrorBanner = pluginFrame.getByTestId('cover-error-banner');
    await expect(coverErrorBanner).toBeVisible();
    await expect(coverErrorBanner).toContainText('access denied under own_groups scope');

    // Also verify via direct RPC that submitting attendance for own_group_id as Cover Teacher fails before cover assignment
    const preCoverSubmitRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/submit_cover_attendance_batch`,
      {
        data: {
          args: {
            group_id: seeded.own_group_id,
            date: seeded.reference_date,
            session: 'AM',
            academic_year_id: seeded.academic_year_id,
            marks: [{ pupil_id: seeded.active_pupil_id, attendance_code: '/' }],
          },
        },
      }
    );
    expect(preCoverSubmitRes.status()).toBe(400);
    const preCoverSubmitBody = await preCoverSubmitRes.json();
    expect(preCoverSubmitBody.ok).toBe(false);
    expect(preCoverSubmitBody.error_code).toContain('access denied under own_groups scope');

    // =========================================================================
    // Step 2: Cover Manager (personas.admin_staff_id) records teacher absence and assigns Cover Teacher to Absent Teacher's timetable slot
    // =========================================================================
    const personaSelect = page.getByTestId('persona-switcher-select');
    await expect(personaSelect).toBeVisible();
    await personaSelect.selectOption(personas.admin_staff_id);
    await expect(page.getByTestId('topbar-active-persona')).toContainText(personas.admin_staff_id);
    await expect(pluginFrame.getByTestId('cover-active-persona-badge')).toContainText(personas.admin_staff_id);

    const loadAbsentTimetableBtn = pluginFrame.getByTestId('load-absent-timetable-btn');
    await expect(loadAbsentTimetableBtn).toBeVisible();
    await expect(loadAbsentTimetableBtn).toHaveAttribute('data-mis-rpc', 'load_absent_teacher_timetable');
    await loadAbsentTimetableBtn.click();

    const absentSlotCard = pluginFrame.getByTestId(`absent-slot-card-${seeded.timetable_slot_am_id}`);
    await expect(absentSlotCard).toBeVisible();
    await expect(absentSlotCard).toContainText('P1_AM');
    await expect(absentSlotCard).toContainText('AM');
    await expect(absentSlotCard).toContainText('Free Cover Staff:');

    const absenceReasonInput = pluginFrame.getByTestId('absence-reason-input');
    const coverNotesInput = pluginFrame.getByTestId('cover-notes-input');
    await expect(absenceReasonInput).toBeVisible();
    await expect(coverNotesInput).toBeVisible();
    await absenceReasonInput.fill('Flu / Medical Absence');
    await coverNotesInput.fill('Complete quadratic equations worksheet on desk');

    const assignCoverBtn = pluginFrame.getByTestId('assign-cover-btn');
    await expect(assignCoverBtn).toBeVisible();
    await expect(assignCoverBtn).toHaveAttribute('data-mis-rpc', 'record_absence_and_assign_cover');
    await assignCoverBtn.click();

    const assignmentReceipt = pluginFrame.getByTestId('cover-assignment-receipt');
    await expect(assignmentReceipt).toBeVisible();
    await expect(assignmentReceipt).toContainText('COVER_ASSIGNED');
    await expect(assignmentReceipt).toContainText(seeded.teacher_staff_id);
    await expect(assignmentReceipt).toContainText(seeded.other_teacher_staff_id);
    await expect(assignmentReceipt).toContainText(seeded.own_group_id);
    await expect(assignmentReceipt).toContainText('COVER_TEACHER');

    // =========================================================================
    // Step 3: Switch back to Cover Teacher persona (seeded.other_teacher_staff_id) -> :own_groups now permits loading the class roster and submitting the statutory attendance register
    // =========================================================================
    await personaSelect.selectOption(seeded.other_teacher_staff_id);
    await expect(page.getByTestId('topbar-active-persona')).toContainText(seeded.other_teacher_staff_id);
    await expect(pluginFrame.getByTestId('cover-active-persona-badge')).toContainText(seeded.other_teacher_staff_id);

    await loadCoverRosterBtn.click();

    await expect(coverErrorBanner).toBeHidden();
    const coverRosterTable = pluginFrame.getByTestId('cover-roster-table');
    await expect(coverRosterTable).toBeVisible();

    const rowPupil1 = pluginFrame.getByTestId(`cover-roster-row-${seeded.active_pupil_id}`);
    await expect(rowPupil1).toBeVisible();
    await expect(rowPupil1).toContainText('Ada Lovelace');
    await expect(rowPupil1).toContainText('E (ASD)');
    await expect(rowPupil1).toContainText('Anaphylaxis (Peanut Allergy)');

    const rowPupil2 = pluginFrame.getByTestId(`cover-roster-row-${seeded.second_active_pupil_id}`);
    await expect(rowPupil2).toBeVisible();
    await expect(rowPupil2).toContainText('Grace Turing');
    await expect(rowPupil2).toContainText('K (SPLD)');

    await expect(pluginFrame.getByTestId(`cover-roster-row-${seeded.transferred_pupil_id}`)).toHaveCount(0);

    // Select attendance marks and submit statutory register
    await pluginFrame.getByTestId(`cover-attendance-code-select-${seeded.active_pupil_id}`).selectOption('/');
    await pluginFrame
      .getByTestId(`cover-attendance-code-select-${seeded.second_active_pupil_id}`)
      .selectOption('L');
    await pluginFrame.getByTestId(`cover-minutes-late-input-${seeded.second_active_pupil_id}`).fill('5');
    await pluginFrame
      .getByTestId(`cover-attendance-comment-input-${seeded.second_active_pupil_id}`)
      .fill('Covered by supply teacher');

    const submitCoverAttBtn = pluginFrame.getByTestId('submit-cover-attendance-btn');
    await expect(submitCoverAttBtn).toBeVisible();
    await expect(submitCoverAttBtn).toHaveAttribute('data-mis-rpc', 'submit_cover_attendance_batch');
    await submitCoverAttBtn.click();

    const attendanceReceipt = pluginFrame.getByTestId('cover-attendance-receipt');
    await expect(attendanceReceipt).toBeVisible();
    await expect(attendanceReceipt).toContainText('COVER_REGISTER_SAVED:AM');
    await expect(attendanceReceipt).toContainText('Upserted: 2');
    await expect(attendanceReceipt).toContainText('ars_');

    await expect(pluginFrame.getByTestId(`cover-persisted-mark-${seeded.active_pupil_id}`)).toContainText('/');
    await expect(
      pluginFrame.getByTestId(`cover-persisted-mark-${seeded.second_active_pupil_id}`)
    ).toContainText('L (5m)');
  });

  test('recordAbsenceAndGdprHooks_unauthorizedTeacherAndSarExportPurge_rejectsUnauthorizedCoverAssignmentAndExportsPurgesPupilCoverLogs_test', async ({
    request,
  }) => {
    if (!personas || !seeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    // 1. Verify regular Classroom Teacher (holding only :own_groups scopes and lacking dal.groups.write:all) is rejected when attempting to assign cover
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.other_teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);

    const unauthAssignRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/record_absence_and_assign_cover`,
      {
        data: {
          args: {
            absent_staff_id: seeded.teacher_staff_id,
            cover_staff_id: seeded.other_teacher_staff_id,
            group_id: seeded.own_group_id,
            slot_id: seeded.timetable_slot_am_id,
            date: seeded.reference_date,
            academic_year_id: seeded.academic_year_id,
            absence_reason: 'Unauthorized self-assignment attempt',
            cover_notes: 'Should be blocked by RBAC',
          },
        },
      }
    );
    expect(unauthAssignRes.status()).toBe(403);
    const unauthAssignBody = await unauthAssignRes.json();
    expect(unauthAssignBody.ok).toBe(false);
    expect(unauthAssignBody.error_code).toContain('caller lacks required DAL scopes');

    // 2. Switch to Cover Manager / Admin (personas.admin_staff_id) and assign Cover Teacher to Absent Teacher's slot
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personas.admin_staff_id },
    });
    expect(switchAdminRes.ok()).toBe(true);

    const assignRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/record_absence_and_assign_cover`,
      {
        data: {
          args: {
            absent_staff_id: seeded.teacher_staff_id,
            cover_staff_id: seeded.other_teacher_staff_id,
            group_id: seeded.own_group_id,
            slot_id: seeded.timetable_slot_am_id,
            date: seeded.reference_date,
            academic_year_id: seeded.academic_year_id,
            absence_reason: 'External Exam Board Moderation',
            cover_notes: 'Complete Past Paper 2 under timed conditions',
          },
        },
      }
    );
    expect(assignRes.ok()).toBe(true);
    const assignBody = await assignRes.json();
    expect(assignBody.ok).toBe(true);

    // 3. Switch to Cover Teacher (seeded.other_teacher_staff_id) and submit cover attendance batch with register comment
    const switchCoverRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.other_teacher_staff_id },
    });
    expect(switchCoverRes.ok()).toBe(true);

    const submitAttRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/submit_cover_attendance_batch`,
      {
        data: {
          args: {
            group_id: seeded.own_group_id,
            date: seeded.reference_date,
            session: 'AM',
            academic_year_id: seeded.academic_year_id,
            marks: [
              {
                pupil_id: seeded.active_pupil_id,
                attendance_code: '/',
                minutes_late: 0,
                comment: 'Helper note with Charles Babbage mentioned',
              },
            ],
          },
        },
      }
    );
    expect(submitAttRes.ok()).toBe(true);
    const submitAttBody = await submitAttRes.json();
    expect(submitAttBody.ok).toBe(true);

    // 4. Switch to Admin (personas.admin_staff_id) and verify UK GDPR export_pupil_data returns structured sections with redactable_fields containing 'register_comment'
    await request.post('/api/auth/persona', {
      data: { staff_id: personas.admin_staff_id },
    });

    const exportRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/export_pupil_data`,
      {
        data: {
          args: {
            pupil_id: seeded.active_pupil_id,
          },
        },
      }
    );
    expect(exportRes.ok()).toBe(true);
    const exportBody = await exportRes.json();
    expect(exportBody.ok).toBe(true);
    expect(Array.isArray(exportBody.data.sections)).toBe(true);
    expect(exportBody.data.sections.length).toBeGreaterThanOrEqual(1);
    const section = exportBody.data.sections[0];
    expect(typeof section.title).toBe('string');
    expect(section.redactable_fields).toContain('register_comment');
    expect(section.records.length).toBeGreaterThanOrEqual(1);
    expect(JSON.stringify(section.records)).toContain('Helper note with Charles Babbage mentioned');

    // 5. Verify UK GDPR purge_pupil_data deletes the pupil's rows and subsequent export_pupil_data returns 0 records
    const purgeRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/purge_pupil_data`,
      {
        data: {
          args: {
            pupil_id: seeded.active_pupil_id,
          },
        },
      }
    );
    expect(purgeRes.ok()).toBe(true);
    const purgeBody = await purgeRes.json();
    expect(purgeBody.ok).toBe(true);
    expect(purgeBody.data.purged_rows).toBeGreaterThanOrEqual(1);

    const exportAfterPurgeRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/export_pupil_data`,
      {
        data: {
          args: {
            pupil_id: seeded.active_pupil_id,
          },
        },
      }
    );
    expect(exportAfterPurgeRes.ok()).toBe(true);
    const exportAfterPurgeBody = await exportAfterPurgeRes.json();
    expect(exportAfterPurgeBody.ok).toBe(true);
    expect(exportAfterPurgeBody.data.sections[0].records.length).toBe(0);

    // 6. Verify revoke_cover_assignment (ctx.dal.groups.revoke_staff_assignment) strips dynamic :own_groups access
    const revokeRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/revoke_cover_assignment`,
      {
        data: {
          args: {
            cover_staff_id: seeded.other_teacher_staff_id,
            group_id: seeded.own_group_id,
            slot_id: seeded.timetable_slot_am_id,
            date: seeded.reference_date,
            academic_year_id: seeded.academic_year_id,
          },
        },
      }
    );
    expect(revokeRes.ok()).toBe(true);
    const revokeBody = await revokeRes.json();
    expect(revokeBody.ok).toBe(true);
    expect(revokeBody.data.revoked).toBe(true);

    await request.post('/api/auth/persona', {
      data: { staff_id: seeded.other_teacher_staff_id },
    });
    const postRevokeRosterRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/load_cover_class_roster`,
      {
        data: {
          args: {
            group_id: seeded.own_group_id,
            date: seeded.reference_date,
            session: 'AM',
            academic_year_id: seeded.academic_year_id,
          },
        },
      }
    );
    expect(postRevokeRosterRes.status()).toBe(400);
    const postRevokeRosterBody = await postRevokeRosterRes.json();
    expect(postRevokeRosterBody.ok).toBe(false);
    expect(postRevokeRosterBody.error_code).toContain('access denied under own_groups scope');
  });
});
