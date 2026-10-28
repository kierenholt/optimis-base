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
}

interface PersonaSeedResponse {
  fixture_id: string;
  pupil_id: string;
  admin_staff_id: string;
  dsl_staff_id: string;
  hod_staff_id: string;
  teacher_staff_id: string;
  terminated_staff_id: string;
}

const PLUGIN_ID = 'uk.org.optimis.lms.classroom';
const PLUGIN_FOLDER = 'lms_sync_plugin';
const CLASSROOM_SYNC_URL = 'https://classroom.googleapis.com/v1/courses/roster:sync';

test.describe('Google Classroom LMS Sync Plugin Vertical Slice: Group Membership Transfer -> Post-Commit Event Bus -> Outbound Roster Sync (Sections 3, 4, 9 & 10)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let m1Seeded: Milestone1SeedResponse | null = null;
  let personaSeeded: PersonaSeedResponse | null = null;

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    m1Seeded = null;
    personaSeeded = null;

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

    // 3. Ensure dev persona switcher and mock HTTP egress mode are enabled
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

    await request.delete('/api/test/egress-mock');
    await request.delete(`/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}`);

    // 4. Register mock Google Classroom HTTPS egress endpoint rule
    const mockRuleRes = await request.post('/api/test/egress-mock', {
      data: {
        method: 'POST',
        url: CLASSROOM_SYNC_URL,
        status: 200,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          sync_receipt_id: 'gclass_sync_10ma2_ok',
          status: 'SYNCED',
          provider: 'GOOGLE_CLASSROOM',
        }),
      },
    });
    if (!mockRuleRes.ok()) {
      throw new Error(`Failed to register mock Google Classroom egress rule: ${mockRuleRes.status()}`);
    }

    // 5. Seed isolated group & pupil fixtures (Year 10 Maths Set 1 & Set 2)
    const seedM1Res = await request.post('/api/test/seed-milestone1-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-28',
      },
    });
    if (!seedM1Res.ok()) {
      throw new Error(`Failed to seed Milestone 1 fixtures: ${seedM1Res.status()} ${await seedM1Res.text()}`);
    }
    const m1Body = (await seedM1Res.json()) as Milestone1SeedResponse;
    if (!m1Body.fixture_id || !m1Body.own_group_id || !m1Body.other_group_id || !m1Body.active_pupil_id) {
      throw new Error('Milestone 1 fixture response missing mandatory identifiers');
    }
    m1Seeded = m1Body;

    // 6. Seed isolated Admin, HoD, and Teacher personas
    const seedPersonasRes = await request.post('/api/test/seed-personas', {
      data: {
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    if (!seedPersonasRes.ok()) {
      throw new Error(`Failed to seed persona fixtures: ${seedPersonasRes.status()} ${await seedPersonasRes.text()}`);
    }
    const pBody = (await seedPersonasRes.json()) as PersonaSeedResponse;
    if (!pBody.fixture_id || !pBody.admin_staff_id || !pBody.hod_staff_id || !pBody.teacher_staff_id) {
      throw new Error('Persona fixture response missing mandatory identifiers');
    }
    personaSeeded = pBody;

    // 7. Install and activate lms_sync_plugin dynamically at runtime as Admin
    const installRes = await request.post('/api/admin/plugins/install', {
      data: {
        folder: PLUGIN_FOLDER,
        staff_id: personaSeeded.admin_staff_id,
      },
    });
    if (!installRes.ok()) {
      throw new Error(
        `Failed to install ${PLUGIN_FOLDER}: ${installRes.status()} ${await installRes.text()}`
      );
    }

    const activateRes = await request.post(
      `/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}/activate`,
      {
        data: {
          staff_id: personaSeeded.admin_staff_id,
        },
      }
    );
    if (!activateRes.ok()) {
      throw new Error(
        `Failed to activate ${PLUGIN_ID}: ${activateRes.status()} ${await activateRes.text()}`
      );
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Purge any plugin sync logs created for seeded pupils before removing the plugin
    if (m1Seeded && personaSeeded) {
      for (const pupilId of [
        m1Seeded.active_pupil_id,
        m1Seeded.transferred_pupil_id,
        m1Seeded.other_group_pupil_id,
      ]) {
        await request.post(`/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/purge_pupil_data`, {
          data: {
            args: {
              pupil_id: pupilId,
              staff_id: personaSeeded.admin_staff_id,
            },
          },
        });
      }
    }

    // 2. Remove runtime-installed plugin and clear mock egress rules
    if (personaSeeded?.admin_staff_id) {
      await request.delete(
        `/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}?staff_id=${encodeURIComponent(personaSeeded.admin_staff_id)}`
      );
    } else {
      await request.delete(`/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}`);
    }
    await request.delete('/api/test/egress-mock');

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

    // 5. Clean up seeded fixtures in core.db
    if (m1Seeded?.fixture_id) {
      const delM1 = await request.delete(
        `/api/test/seed-milestone1-fixtures/${encodeURIComponent(m1Seeded.fixture_id)}`
      );
      if (!delM1.ok()) {
        throw new Error(`Failed to clean up Milestone 1 fixture ${m1Seeded.fixture_id}: ${delM1.status()}`);
      }
    }
    if (personaSeeded?.fixture_id) {
      const delPersonas = await request.delete(
        `/api/test/seed-personas/${encodeURIComponent(personaSeeded.fixture_id)}`
      );
      if (!delPersonas.ok()) {
        throw new Error(`Failed to clean up persona fixture ${personaSeeded.fixture_id}: ${delPersonas.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('transferPupilBetweenSets_adminUpsertsPupilMembership_firesCoreGroupMembershipChangedWorkerSyncsGoogleClassroomAndLogsStatus_test', async ({
    page,
    request,
  }) => {
    if (!m1Seeded || !personaSeeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    // 1. Verify Event Bus catalog registers uk.org.optimis.lms.classroom worker on_membership_changed
    const catalogRes = await request.get('/api/events/catalog');
    expect(catalogRes.ok()).toBe(true);
    const catalogBody = await catalogRes.json();
    expect(catalogBody.ok).toBe(true);
    const pluginSub = catalogBody.plugin_subscriptions?.[PLUGIN_ID];
    expect(pluginSub).toBeDefined();
    expect(pluginSub.state).toBe('ACTIVE');
    expect(pluginSub.workers?.on_membership_changed?.events).toContain('core.group.membership_changed');

    // 2. Authenticate as Admin (who holds dal.groups.write:all & dal.groups.read:all)
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personaSeeded.admin_staff_id },
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

    // 3. Open Host UI Shell and navigate to the Google Classroom LMS Sync plugin
    const params = new URLSearchParams({
      pupil_id: m1Seeded.active_pupil_id,
      group_id: m1Seeded.own_group_id,
      other_group_id: m1Seeded.other_group_id,
      reference_date: m1Seeded.reference_date,
      academic_year_id: m1Seeded.academic_year_id,
      staff_id: personaSeeded.admin_staff_id,
    });
    await page.goto(`/?${params.toString()}`);
    await expect(page.getByTestId('host-shell')).toBeVisible();

    const lmsNavBtn = page.getByTestId(`nav-item-plugin-${PLUGIN_ID}`);
    await expect(lmsNavBtn).toBeVisible();
    await lmsNavBtn.click();

    const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${PLUGIN_ID}"]`);
    await expect(pluginFrame.getByTestId('lms-sync-plugin-root')).toBeVisible();

    // 4. HoD transfers active_pupil_id from Set 1 (own_group_id) into Set 2 (other_group_id) via upsert_pupil_membership
    const transferBtn = pluginFrame.getByTestId('transfer-pupil-set-btn');
    await expect(transferBtn).toBeVisible();
    await expect(transferBtn).toHaveAttribute('data-mis-rpc', 'upsert_pupil_membership');
    await transferBtn.click();

    const transferResult = pluginFrame.getByTestId('transfer-membership-result');
    await expect(transferResult).toBeVisible();
    await expect(transferResult).toContainText('MEMBERSHIP_UPSERTED');
    await expect(transferResult).toContainText(m1Seeded.other_group_id);

    // 5. Post-commit Event Bus fires core.group.membership_changed -> worker on_membership_changed
    //    posts updated roster to https://classroom.googleapis.com/v1/courses/roster:sync and logs status in plugin_lms_sync.db
    const refreshLogsBtn = pluginFrame.getByTestId('refresh-lms-sync-logs-btn');
    await expect(refreshLogsBtn).toBeVisible();
    await expect(refreshLogsBtn).toHaveAttribute('data-mis-rpc', 'list_sync_logs');

    const syncStatusBadge = pluginFrame.getByTestId('lms-sync-status-badge');
    const syncLogsJson = pluginFrame.getByTestId('lms-sync-logs-json');

    await expect(async () => {
      await refreshLogsBtn.click();
      await expect(syncStatusBadge).toHaveText('SYNCED');
      await expect(syncLogsJson).toContainText('gclass_sync_10ma2_ok');
    }).toPass({ timeout: 5000 });

    const rawLogs = await syncLogsJson.textContent();
    expect(rawLogs).toBeTruthy();
    const parsed = JSON.parse(rawLogs!);
    expect(parsed.logs.length).toBeGreaterThanOrEqual(1);

    const syncEntry = parsed.logs.find(
      (l: { pupil_id: string; group_id: string }) =>
        l.pupil_id === m1Seeded!.active_pupil_id && l.group_id === m1Seeded!.other_group_id
    );
    expect(syncEntry).toBeTruthy();
    expect(syncEntry.event_name).toBe('core.group.membership_changed');
    expect(syncEntry.hop).toBe(1);
    expect(syncEntry.status).toBe('SYNCED');
    expect(syncEntry.http_status).toBe(200);
    expect(syncEntry.external_sync_id).toBe('gclass_sync_10ma2_ok');
    expect(syncEntry.target_url).toBe(CLASSROOM_SYNC_URL);
    expect(syncEntry.caller_type).toBe('SYSTEM');
    expect(syncEntry.caller_staff_id).toBe('');
    expect(syncEntry.roster_count).toBeGreaterThanOrEqual(2);
  });

  test('lmsSyncPlugin_unauthorizedTeacherEgressFailureAndGdprHooks_enforcesScopesLogsFailedSyncAndPurgesPupilData_test', async ({
    request,
  }) => {
    if (!m1Seeded || !personaSeeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    // 1. RBAC: Classroom Teacher (who lacks dal.groups.write:all) is rejected when calling upsert_pupil_membership
    const unauthTransferRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/upsert_pupil_membership`,
      {
        data: {
          args: {
            staff_id: personaSeeded.teacher_staff_id,
            group_id: m1Seeded.other_group_id,
            pupil_id: m1Seeded.active_pupil_id,
            academic_year_id: m1Seeded.academic_year_id,
            effective_from: m1Seeded.reference_date,
          },
        },
      }
    );
    expect(unauthTransferRes.ok()).toBe(false);
    const unauthTransferBody = await unauthTransferRes.json();
    expect(unauthTransferBody.ok).toBe(false);
    expect(unauthTransferBody.error_code).toContain('caller lacks required DAL scopes');

    // 2. Register a 503 failure rule on Google Classroom API and verify worker records FAILED sync status
    await request.post('/api/test/egress-mock', {
      data: {
        method: 'POST',
        url: CLASSROOM_SYNC_URL,
        status: 503,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          error: 'GOOGLE_CLASSROOM_SERVICE_UNAVAILABLE',
        }),
      },
    });

    const adminTransferRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/upsert_pupil_membership`,
      {
        data: {
          args: {
            staff_id: personaSeeded.admin_staff_id,
            group_id: m1Seeded.other_group_id,
            pupil_id: m1Seeded.active_pupil_id,
            academic_year_id: m1Seeded.academic_year_id,
            effective_from: m1Seeded.reference_date,
          },
        },
      }
    );
    expect(adminTransferRes.ok()).toBe(true);

    // Poll list_sync_logs until the failed sync attempt is recorded in plugin_lms_sync.db
    await expect(async () => {
      const logsRes = await request.post(
        `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/list_sync_logs`,
        {
          data: {
            args: {
              staff_id: personaSeeded!.admin_staff_id,
              pupil_id: m1Seeded!.active_pupil_id,
            },
          },
        }
      );
      expect(logsRes.ok()).toBe(true);
      const logsBody = await logsRes.json();
      expect(logsBody.ok).toBe(true);
      expect(logsBody.data.logs.length).toBeGreaterThanOrEqual(1);
      expect(logsBody.data.logs[0].status).toBe('FAILED');
      expect(logsBody.data.logs[0].http_status).toBe(503);
    }).toPass({ timeout: 5000 });

    // 3. Verify UK GDPR export_pupil_data & purge_pupil_data hooks on plugin_lms_sync.db
    const exportRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/export_pupil_data`,
      {
        data: {
          args: {
            staff_id: personaSeeded.admin_staff_id,
            pupil_id: m1Seeded.active_pupil_id,
          },
        },
      }
    );
    expect(exportRes.ok()).toBe(true);
    const exportBody = await exportRes.json();
    expect(exportBody.ok).toBe(true);
    expect(exportBody.data.sections.length).toBe(1);
    expect(exportBody.data.sections[0].title).toBe('Google Classroom LMS Roster Sync Logs');
    expect(exportBody.data.sections[0].records.length).toBeGreaterThanOrEqual(1);
    expect(exportBody.data.sections[0].redactable_fields).toContain('sync_notes');

    const purgeRes = await request.post(
      `/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/purge_pupil_data`,
      {
        data: {
          args: {
            staff_id: personaSeeded.admin_staff_id,
            pupil_id: m1Seeded.active_pupil_id,
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
            staff_id: personaSeeded.admin_staff_id,
            pupil_id: m1Seeded.active_pupil_id,
          },
        },
      }
    );
    expect(exportAfterPurgeRes.ok()).toBe(true);
    const exportAfterPurgeBody = await exportAfterPurgeRes.json();
    expect(exportAfterPurgeBody.data.sections[0].records.length).toBe(0);
  });
});
