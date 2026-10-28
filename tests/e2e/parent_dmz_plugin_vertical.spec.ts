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

test.describe('Parent Portal DMZ Sync Plugin Vertical: Inbox Pull Safeguarding Blocks & Sanitized Read-Projection Push (Sections 1, 3, 5 & 9)', () => {
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

    // 3. Ensure dev persona switcher, mock HTTP egress, and mock Parent DMZ sync modes are active
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

    // 4. Seed isolated persona fixtures (including Attendance Officer) & Milestone 1 pupil/guardian/household fixtures
    const personaRes = await request.post('/api/test/seed-personas', {
      data: {
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    if (!personaRes.ok()) {
      throw new Error(`Failed to seed persona fixtures: ${personaRes.status()}`);
    }
    const personaBody = (await personaRes.json()) as SeededPersonaFixtures;
    if (
      !personaBody.fixture_id ||
      !personaBody.admin_staff_id ||
      !personaBody.teacher_staff_id ||
      !personaBody.attendance_officer_staff_id
    ) {
      throw new Error('Seeded persona fixture response missing mandatory identifiers (including attendance_officer_staff_id)');
    }
    personas = personaBody;

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

    // 5. Install and activate test_plugin (for seeding AM attendance) and parent_dmz_plugin (uk.org.optimis.parent.dmz)
    const installTestRes = await request.post('/api/admin/plugins/install', {
      data: {
        folder: 'test_plugin',
        staff_id: personas.admin_staff_id,
      },
    });
    if (!installTestRes.ok()) {
      throw new Error(`Failed to install test_plugin: ${installTestRes.status()}`);
    }
    const activateTestRes = await request.post('/api/admin/plugins/uk.org.optimis.test/activate', {
      data: { staff_id: personas.admin_staff_id },
    });
    if (!activateTestRes.ok()) {
      throw new Error(`Failed to activate uk.org.optimis.test: ${activateTestRes.status()}`);
    }

    const installDmzRes = await request.post('/api/admin/plugins/install', {
      data: {
        folder: 'parent_dmz_plugin',
        staff_id: personas.admin_staff_id,
      },
    });
    if (!installDmzRes.ok()) {
      throw new Error(`Failed to install parent_dmz_plugin: ${installDmzRes.status()} ${await installDmzRes.text()}`);
    }
    const activateDmzRes = await request.post('/api/admin/plugins/uk.org.optimis.parent.dmz/activate', {
      data: { staff_id: personas.admin_staff_id },
    });
    if (!activateDmzRes.ok()) {
      throw new Error(`Failed to activate uk.org.optimis.parent.dmz: ${activateDmzRes.status()}`);
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Restore mock external dependency modes & clear mock DMZ store
    await request.post('/api/test/external-deps-mode', {
      data: {
        enable_real_cloud_kms: false,
        enable_real_parent_dmz_sync: false,
        enable_real_mat_sync: false,
      },
    });

    // 2. Purge plugin data created during the test
    if (personas && m1Seeded) {
      await request.post('/api/auth/persona', {
        data: { staff_id: personas.admin_staff_id },
      });
      await request.post('/api/plugins/uk.org.optimis.parent.dmz/rpc/purge_pupil_data', {
        data: { args: { pupil_id: m1Seeded.active_pupil_id } },
      });
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

    // 5. Remove runtime-installed plugins
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete('/api/admin/plugins/uk.org.optimis.parent.dmz');
    await request.delete('/api/admin/plugins/uk.org.optimis.test');

    // 6. Clean up seeded fixtures in core.db
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

  test('pullDmzInboxAndPushSanitizedProjection_asAttendanceOfficer_acceptsUnrestrictedPrMotherRejectsCourtOrderFatherAndStripsConfidentialAddressAndUpn_test', async ({
    page,
    request,
  }) => {
    if (!personas || !m1Seeded) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // 1. Record a statutory AM attendance session mark for Ada Lovelace so attendance sessions are populated
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

    // 2. Enqueue two untrusted parent absence notices in the Mock DMZ Write-Inbox:
    //    (a) Unrestricted Mother with Parental Responsibility (Anne Byron) -> MUST BE ACCEPTED
    //    (b) Court-order-restricted Father (George Byron, Prohibited Steps Order) -> MUST BE REJECTED_SAFEGUARDING_BLOCK
    const enqueueMomRes = await request.post('/api/test/parent-dmz/enqueue-inbox', {
      data: {
        pupil_id: m1Seeded.active_pupil_id,
        submitted_by_guardian_person_id: m1Seeded.unrestricted_mother_person_id,
        submission_type: 'ABSENCE_NOTICE',
        date: m1Seeded.reference_date,
        session: 'PM',
        reason: 'Hospital dental appointment at 14:00',
      },
    });
    expect(enqueueMomRes.status()).toBe(201);

    const enqueueDadRes = await request.post('/api/test/parent-dmz/enqueue-inbox', {
      data: {
        pupil_id: m1Seeded.active_pupil_id,
        submitted_by_guardian_person_id: m1Seeded.restricted_father_person_id,
        submission_type: 'ABSENCE_NOTICE',
        date: m1Seeded.reference_date,
        session: 'AM',
        reason: 'Unauthorized absence submission by court-order-blocked father',
      },
    });
    expect(enqueueDadRes.status()).toBe(201);

    // 3. Switch to Attendance Officer persona (SUPPORT role holding dal.pupils.read:all, dal.guardians.read:all, dal.attendance.read:all)
    const switchAttOfficerRes = await request.post('/api/auth/persona', {
      data: { staff_id: personas.attendance_officer_staff_id },
    });
    expect(switchAttOfficerRes.ok()).toBe(true);
    const attOfficerSession = await switchAttOfficerRes.json();
    expect(attOfficerSession.caller.active_roles).toContain('SUPPORT');
    expect(attOfficerSession.caller.scopes).toContain('dal.pupils.read:all');
    expect(attOfficerSession.caller.scopes).toContain('dal.guardians.read:all');
    expect(attOfficerSession.caller.scopes).toContain('dal.attendance.read:all');

    await page.context().addCookies([
      {
        name: 'mis_session',
        value: attOfficerSession.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    // 4. Navigate to Host UI Shell and open the Parent Portal DMZ Sync plugin iframe
    const query = new URLSearchParams({
      pupil_id: m1Seeded.active_pupil_id,
      reference_date: m1Seeded.reference_date,
      academic_year_id: m1Seeded.academic_year_id,
      staff_id: personas.attendance_officer_staff_id,
    });
    await page.goto(`/?${query.toString()}`);

    await expect(page.getByTestId('host-shell')).toBeVisible();
    const dmzNavBtn = page.getByTestId('nav-item-plugin-uk.org.optimis.parent.dmz');
    await expect(dmzNavBtn).toBeVisible();
    await dmzNavBtn.click();

    const iframeEl = page.getByTestId('plugin-iframe-uk.org.optimis.parent.dmz');
    await expect(iframeEl).toBeVisible();
    await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.parent.dmz"]');
    await expect(pluginFrame.getByTestId('parent-dmz-plugin-root')).toBeVisible();

    // 5. Attendance Officer clicks "Pull Queued DMZ Write-Inbox" inside the sandboxed plugin UI
    const pullBtn = pluginFrame.getByTestId('pull-dmz-inbox-btn');
    await expect(pullBtn).toBeVisible();
    await expect(pullBtn).toHaveAttribute('data-mis-rpc', 'pull_dmz_inbox');
    await pullBtn.click();

    // Verify UI displays accepted=1 (unrestricted PR mother) and rejected=1 (court-order-blocked father with REJECTED_SAFEGUARDING_BLOCK)
    await expect(pluginFrame.getByTestId('dmz-inbox-summary')).toBeVisible();
    await expect(pluginFrame.getByTestId('dmz-accepted-count')).toHaveText('1');
    await expect(pluginFrame.getByTestId('dmz-rejected-count')).toHaveText('1');

    const momStatus = pluginFrame.getByTestId(
      `dmz-submission-status-${m1Seeded.unrestricted_mother_person_id}`
    );
    await expect(momStatus).toBeVisible();
    await expect(momStatus).toHaveText('ACCEPTED');

    const dadStatus = pluginFrame.getByTestId(
      `dmz-submission-status-${m1Seeded.restricted_father_person_id}`
    );
    await expect(dadStatus).toBeVisible();
    await expect(dadStatus).toHaveText('REJECTED_SAFEGUARDING_BLOCK');

    // 6. Attendance Officer clicks "Push Sanitized Read-Projection" inside the sandboxed plugin UI
    const pushBtn = pluginFrame.getByTestId('push-dmz-projection-btn');
    await expect(pushBtn).toBeVisible();
    await expect(pushBtn).toHaveAttribute('data-mis-rpc', 'push_sanitized_projection');
    await pushBtn.click();

    const projCard = pluginFrame.getByTestId('dmz-projection-card');
    await expect(projCard).toBeVisible();
    await expect(pluginFrame.getByTestId('dmz-projection-pupil-name')).toContainText('Ada Lovelace');
    await expect(pluginFrame.getByTestId('dmz-projection-attendance')).toContainText('1 / 1');
    await expect(pluginFrame.getByTestId('dmz-projection-authorized-guardians')).toContainText(
      m1Seeded.unrestricted_mother_person_id
    );
    await expect(pluginFrame.getByTestId('dmz-projection-authorized-guardians')).not.toContainText(
      m1Seeded.restricted_father_person_id
    );
    await expect(pluginFrame.getByTestId('dmz-projection-redacted-addresses')).toHaveText('1');
    await expect(pluginFrame.getByTestId('dmz-projection-upn-badge')).toHaveText('UPN_STRIPPED');

    const jsonPreviewText = (await pluginFrame.getByTestId('dmz-projection-json-preview').textContent()) || '';
    expect(jsonPreviewText).toContain(m1Seeded.active_pupil_id);
    expect(jsonPreviewText).toContain(m1Seeded.unrestricted_mother_person_id);
    expect(jsonPreviewText).not.toContain(m1Seeded.restricted_father_person_id);
    expect(jsonPreviewText).not.toContain('10 Safe Haven Way');
    expect(jsonPreviewText).not.toContain('Confidential Refuge');
    expect(jsonPreviewText).not.toContain('SW1A 1AA');
    expect(jsonPreviewText).not.toContain('Prohibited Steps Order');
    expect(jsonPreviewText.toLowerCase()).not.toContain('"upn"');
    expect(jsonPreviewText).not.toContain('U1' + m1Seeded.fixture_id.toUpperCase() + '001');

    // 7. Verify the Mock Public Cloud DMZ received the sanitized read-projection with UPN and confidential refuge address stripped
    const dmzProjRes = await request.get('/api/test/parent-dmz/projections');
    expect(dmzProjRes.ok()).toBe(true);
    const dmzProjBody = await dmzProjRes.json();
    expect(dmzProjBody.ok).toBe(true);
    const pushedProj = dmzProjBody.projections[m1Seeded.active_pupil_id];
    expect(pushedProj).toBeDefined();
    expect(pushedProj.pupil_id).toBe(m1Seeded.active_pupil_id);
    expect(pushedProj.first_name).toBe('Ada');
    expect(pushedProj.last_name).toBe('Lovelace');
    expect(pushedProj.present_sessions).toBe(1);
    expect(pushedProj.total_sessions).toBe(1);
    expect(pushedProj.authorized_guardian_ids).toEqual([m1Seeded.unrestricted_mother_person_id]);
    expect(pushedProj.redacted_confidential_address_count).toBe(1);

    const rawDmzProjectionJson = JSON.stringify(pushedProj);
    expect(rawDmzProjectionJson).not.toContain('10 Safe Haven Way');
    expect(rawDmzProjectionJson).not.toContain('Prohibited Steps Order');
    expect(rawDmzProjectionJson.toLowerCase()).not.toContain('upn');
  });

  test('parentDmzPlugin_whenRealDmzUnconfiguredOrUnauthorizedTeacherOrGdprSarPurge_failsFastAndPurgesCleanly_test', async ({
    page,
    request,
  }) => {
    if (!personas || !m1Seeded) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // 1. RBAC Verification: Classroom Teacher (holding only :own_groups scopes) is rejected with 403 when calling parent_dmz_plugin endpoints
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: personas.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);

    const unauthPullRes = await request.post('/api/plugins/uk.org.optimis.parent.dmz/rpc/pull_dmz_inbox', {
      data: {
        args: {
          academic_year_id: m1Seeded.academic_year_id,
          reference_date: m1Seeded.reference_date,
        },
      },
    });
    expect(unauthPullRes.status()).toBe(403);
    const unauthPullBody = await unauthPullRes.json();
    expect(unauthPullBody.ok).toBe(false);
    expect(unauthPullBody.error_code).toContain('caller lacks required DAL scopes');

    const unauthPushRes = await request.post(
      '/api/plugins/uk.org.optimis.parent.dmz/rpc/push_sanitized_projection',
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
    expect(unauthPushRes.status()).toBe(403);
    const unauthPushBody = await unauthPushRes.json();
    expect(unauthPushBody.ok).toBe(false);
    expect(unauthPushBody.error_code).toContain('caller lacks required DAL scopes');

    // 2. Switch to Attendance Officer and verify fail-fast error banner when ENABLE_REAL_PARENT_DMZ_SYNC=true without real DMZ configuration
    const switchAttOfficerRes = await request.post('/api/auth/persona', {
      data: { staff_id: personas.attendance_officer_staff_id },
    });
    expect(switchAttOfficerRes.ok()).toBe(true);
    const attSession = await switchAttOfficerRes.json();

    const realDmzModeRes = await request.post('/api/test/external-deps-mode', {
      data: { enable_real_parent_dmz_sync: true },
    });
    expect(realDmzModeRes.status()).toBe(503);

    await page.context().addCookies([
      {
        name: 'mis_session',
        value: attSession.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);
    await page.goto(
      `/?pupil_id=${encodeURIComponent(m1Seeded.active_pupil_id)}&reference_date=${encodeURIComponent(
        m1Seeded.reference_date
      )}&academic_year_id=${encodeURIComponent(m1Seeded.academic_year_id)}`
    );
    await page.getByTestId('nav-item-plugin-uk.org.optimis.parent.dmz').click();

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.parent.dmz"]');
    await pluginFrame.getByTestId('push-dmz-projection-btn').click();

    const errorBanner = pluginFrame.getByTestId('dmz-error-banner');
    await expect(errorBanner).toBeVisible();
    await expect(errorBanner).toContainText('REAL_PARENT_DMZ_NOT_CONFIGURED');

    // Restore mock Parent DMZ mode
    await request.post('/api/test/external-deps-mode', {
      data: { enable_real_parent_dmz_sync: false },
    });

    // 3. Enqueue a submission, pull inbox, and push projection so plugin_parent_dmz.db holds both inbox and projection records for active_pupil_id
    await request.post('/api/test/parent-dmz/enqueue-inbox', {
      data: {
        pupil_id: m1Seeded.active_pupil_id,
        submitted_by_guardian_person_id: m1Seeded.unrestricted_mother_person_id,
        submission_type: 'ABSENCE_NOTICE',
        date: m1Seeded.reference_date,
        session: 'AM',
        reason: 'GP appointment for Ada with Dr Smith',
      },
    });

    const pullOkRes = await request.post('/api/plugins/uk.org.optimis.parent.dmz/rpc/pull_dmz_inbox', {
      data: {
        args: {
          academic_year_id: m1Seeded.academic_year_id,
          reference_date: m1Seeded.reference_date,
        },
      },
    });
    expect(pullOkRes.ok()).toBe(true);

    const pushOkRes = await request.post(
      '/api/plugins/uk.org.optimis.parent.dmz/rpc/push_sanitized_projection',
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
    expect(pushOkRes.ok()).toBe(true);

    // 4. Verify UK GDPR export_pupil_data and purge_pupil_data hooks on plugin_parent_dmz.db
    const exportRes = await request.post('/api/plugins/uk.org.optimis.parent.dmz/rpc/export_pupil_data', {
      data: { args: { pupil_id: m1Seeded.active_pupil_id } },
    });
    expect(exportRes.ok()).toBe(true);
    const exportBody = await exportRes.json();
    expect(exportBody.ok).toBe(true);
    expect(exportBody.data.sections.length).toBe(2);
    expect(exportBody.data.sections[0].title).toBe('Parent DMZ Write-Inbox Submissions');
    expect(exportBody.data.sections[0].records.length).toBeGreaterThanOrEqual(1);
    expect(exportBody.data.sections[0].redactable_fields).toContain('reason');
    expect(exportBody.data.sections[1].title).toBe('Parent DMZ Read-Projection Snapshots');
    expect(exportBody.data.sections[1].records.length).toBeGreaterThanOrEqual(1);
    expect(exportBody.data.sections[1].redactable_fields).toContain('notes');

    const purgeRes = await request.post('/api/plugins/uk.org.optimis.parent.dmz/rpc/purge_pupil_data', {
      data: { args: { pupil_id: m1Seeded.active_pupil_id } },
    });
    expect(purgeRes.ok()).toBe(true);
    const purgeBody = await purgeRes.json();
    expect(purgeBody.ok).toBe(true);
    expect(purgeBody.data.purged_rows).toBeGreaterThanOrEqual(2);

    const exportAfterPurgeRes = await request.post(
      '/api/plugins/uk.org.optimis.parent.dmz/rpc/export_pupil_data',
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
