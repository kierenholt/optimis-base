import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

interface SeededGDPRFixtures {
  fixture_id: string;
  academic_year_id: string;
  reference_date: string;
  admin_staff_id: string;
  teacher_staff_id: string;
  dsl_staff_id: string;
  pupil_id: string;
  protected_pupil_id: string;
  pastoral_note_id: string;
  safeguarding_note_id: string;
}

test.describe('UK GDPR SAR Export, Retention Purge & Admin Redaction UI (Sections 5C, 7B & 9.3)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let seeded: SeededGDPRFixtures | null = null;

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

    // 3. Ensure dev persona switcher is enabled and plugin uk.org.optimis.test is ACTIVE
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });

    // 4. Seed isolated GDPR test fixtures (Admin, Teacher, DSL, eligible pupil, protected KCSIE pupil, pastoral & SEN/medical/attendance data)
    const seedRes = await request.post('/api/test/seed-gdpr-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-20',
      },
    });
    if (!seedRes.ok()) {
      throw new Error(`Failed to seed GDPR fixtures: ${seedRes.status()} ${await seedRes.text()}`);
    }
    const body = (await seedRes.json()) as SeededGDPRFixtures;
    if (
      !body.fixture_id ||
      !body.admin_staff_id ||
      !body.teacher_staff_id ||
      !body.pupil_id ||
      !body.protected_pupil_id ||
      !body.pastoral_note_id ||
      !body.safeguarding_note_id
    ) {
      throw new Error('Seeded GDPR fixture response missing mandatory identifiers');
    }
    seeded = body;

    // Switch to Admin so we can ensure uk.org.optimis.test is ACTIVE and seed a plugin note
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.admin_staff_id },
    });
    if (!switchAdminRes.ok()) {
      throw new Error(`Failed to switch to Admin persona in setup: ${switchAdminRes.status()}`);
    }

    await request.post('/api/admin/plugins/uk.org.optimis.test/state', {
      data: { state: 'ACTIVE' },
    });

    // Seed a plugin note containing a third-party pupil name in plugin_test.db for both pupils
    const saveNote1 = await request.post('/api/plugins/uk.org.optimis.test/rpc/save_pupil_note', {
      data: {
        args: {
          pupil_id: seeded.pupil_id,
          note_id: `pnote_${seeded.fixture_id}_1`,
          note: 'Discussed homework pairing with Oliver Twist during algebra workshop.',
        },
      },
    });
    if (!saveNote1.ok()) {
      throw new Error(`Failed to seed plugin note for pupil_id: ${saveNote1.status()}`);
    }

    const saveNote2 = await request.post('/api/plugins/uk.org.optimis.test/rpc/save_pupil_note', {
      data: {
        args: {
          pupil_id: seeded.protected_pupil_id,
          note_id: `pnote_${seeded.fixture_id}_2`,
          note: 'Protected pupil algebra observation note.',
        },
      },
    });
    if (!saveNote2.ok()) {
      throw new Error(`Failed to seed plugin note for protected_pupil_id: ${saveNote2.status()}`);
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Restore plugin state to ACTIVE and clean up plugin notes & fixtures
    if (seeded) {
      await request.post('/api/auth/persona', {
        data: { staff_id: seeded.admin_staff_id },
      });
      await request.post('/api/admin/plugins/uk.org.optimis.test/state', {
        data: { state: 'ACTIVE' },
      });
      await request.post('/api/plugins/uk.org.optimis.test/rpc/purge_pupil_data', {
        data: { args: { pupil_id: seeded.pupil_id } },
      });
      await request.post('/api/plugins/uk.org.optimis.test/rpc/purge_pupil_data', {
        data: { args: { pupil_id: seeded.protected_pupil_id } },
      });
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

    // 4. Clean up seeded GDPR fixtures in core.db
    if (seeded?.fixture_id) {
      const delRes = await request.delete(`/api/test/seed-gdpr-fixtures/${encodeURIComponent(seeded.fixture_id)}`);
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up GDPR fixture ${seeded.fixture_id}: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('exportAndRedactPupilSar_adminReviewsCoreAndPluginSectionsInRedactionUi_exportsRedactedBundle_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded GDPR state');
    }

    // 1. Verify RBAC: Classroom Teacher (non-Admin/DPO) MUST be rejected when attempting SAR export
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);

    const unauthorizedSarRes = await request.post('/api/admin/gdpr/sar/export', {
      data: { pupil_id: seeded.pupil_id },
    });
    expect(unauthorizedSarRes.status()).toBe(403);
    const unauthorizedSarBody = await unauthorizedSarRes.json();
    expect(unauthorizedSarBody.ok).toBe(false);
    expect(unauthorizedSarBody.error_code).toContain('UNAUTHORIZED_ADMIN_GDPR_SCOPE');

    // 2. Switch to System Admin / DPO persona and attach session cookie to browser
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

    // 3. Navigate to Host UI Shell and open the Admin SAR Redaction UI
    await page.goto(`/?pupil_id=${encodeURIComponent(seeded.pupil_id)}`);
    await expect(page.getByTestId('host-shell')).toBeVisible();

    const sarPanel = page.getByTestId('sar-redaction-panel');
    await expect(sarPanel).toBeVisible();

    const loadSarBtn = page.getByTestId('sar-load-export-btn');
    await expect(loadSarBtn).toBeVisible();
    await loadSarBtn.click();

    // 4. Verify Core & Plugin SAR sections are rendered in the Redaction UI
    await expect(page.getByTestId('sar-section-core-Core Pupil Profile & Enrolments')).toBeVisible();
    await expect(page.getByTestId('sar-section-core-Core Pastoral Notes')).toBeVisible();
    await expect(page.getByTestId('sar-section-uk.org.optimis.test-Plugin Pupil Notes')).toBeVisible();

    // 5. Inspect and redact third-party pupil names in Core Pastoral Notes ('note_text') and Plugin Pupil Notes ('note')
    const corePastoralInput = page.getByTestId('sar-redact-input-core-Core Pastoral Notes-0-note_text');
    await expect(corePastoralInput).toBeVisible();
    await expect(corePastoralInput).toHaveValue(
      'Conflict in corridor involving third-party pupil Harry Potter before morning registration.'
    );

    await corePastoralInput.fill(
      'Conflict in corridor involving third-party pupil [REDACTED THIRD PARTY] before morning registration.'
    );

    const pluginNoteInput = page.getByTestId(
      'sar-redact-input-uk.org.optimis.test-Plugin Pupil Notes-0-note'
    );
    await expect(pluginNoteInput).toBeVisible();
    await expect(pluginNoteInput).toHaveValue(
      'Discussed homework pairing with Oliver Twist during algebra workshop.'
    );

    // Use the quick [REDACTED] button on the plugin note field or fill redacted text
    const pluginNoteRedactBtn = page.getByTestId(
      'sar-redact-mask-btn-uk.org.optimis.test-Plugin Pupil Notes-0-note'
    );
    await expect(pluginNoteRedactBtn).toBeVisible();
    await pluginNoteRedactBtn.click();
    await expect(pluginNoteInput).toHaveValue('[REDACTED]');

    // 6. Finalize and export the redacted SAR package
    const finalizeBtn = page.getByTestId('sar-finalize-export-btn');
    await expect(finalizeBtn).toBeVisible();
    await finalizeBtn.click();

    const finalizedStatus = page.getByTestId('sar-export-status');
    await expect(finalizedStatus).toContainText('SAR_EXPORT_FINALIZED');

    const finalizedJsonEl = page.getByTestId('sar-finalized-json');
    await expect(finalizedJsonEl).toBeVisible();
    const rawExportJson = await finalizedJsonEl.textContent();
    expect(rawExportJson).toBeTruthy();

    const parsedExport = JSON.parse(rawExportJson!);
    expect(parsedExport.pupil_id).toBe(seeded.pupil_id);
    expect(parsedExport.redacted_by_staff_id).toBe(seeded.admin_staff_id);
    expect(rawExportJson).not.toContain('Harry Potter');
    expect(rawExportJson).not.toContain('Oliver Twist');
    expect(rawExportJson).toContain('[REDACTED THIRD PARTY]');
    expect(rawExportJson).toContain('[REDACTED]');
  });

  test('purgePupilData_acrossCoreAndDormantPluginWithKcsieGuard_purgesAllRecordsAndBlocksProtectedSafeguarding_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded GDPR state');
    }

    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.admin_staff_id },
    });
    expect(switchAdminRes.ok()).toBe(true);
    const adminSessionBody = await switchAdminRes.json();

    // 1. Transition uk.org.optimis.test into UNINSTALLED_DORMANT quarantine state (Section 8 Gap 3 & Section 9.3.4)
    const dormantRes = await request.post('/api/admin/plugins/uk.org.optimis.test/state', {
      data: { state: 'UNINSTALLED_DORMANT' },
    });
    expect(dormantRes.ok()).toBe(true);

    // Normal interactive RPC calls to a dormant plugin MUST be blocked immediately
    const rpcBlockedRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/list_pupil_notes', {
      data: { args: { pupil_id: seeded.pupil_id } },
    });
    expect(rpcBlockedRes.status()).toBe(400);
    const rpcBlockedBody = await rpcBlockedRes.json();
    expect(rpcBlockedBody.ok).toBe(false);
    expect(rpcBlockedBody.error_code).toContain('PLUGIN_DORMANT_QUARANTINE');

    // However, SAR export MUST still invoke gdpr_hooks.export_pupil_data on the UNINSTALLED_DORMANT plugin
    const sarDormantRes = await request.post('/api/admin/gdpr/sar/export', {
      data: { pupil_id: seeded.pupil_id },
    });
    expect(sarDormantRes.ok()).toBe(true);
    const sarDormantBody = await sarDormantRes.json();
    expect(sarDormantBody.ok).toBe(true);
    const pluginSection = sarDormantBody.bundle.sections.find(
      (s: { source_id: string; title: string }) =>
        s.source_id === 'uk.org.optimis.test' && s.title === 'Plugin Pupil Notes'
    );
    expect(pluginSection).toBeTruthy();
    expect(pluginSection.plugin_state).toBe('UNINSTALLED_DORMANT');
    expect(pluginSection.records.length).toBe(1);

    // 2. Attempting a retention purge on protected_pupil_id (who has an active KCSIE safeguarding note with retention_purge_eligible=0) MUST fail fast
    const blockedPurgeRes = await request.post('/api/admin/gdpr/purge', {
      data: { pupil_id: seeded.protected_pupil_id },
    });
    expect(blockedPurgeRes.status()).toBe(409);
    const blockedPurgeBody = await blockedPurgeRes.json();
    expect(blockedPurgeBody.ok).toBe(false);
    expect(blockedPurgeBody.error_code).toContain('KCSIE_RETENTION_LOCK');

    // 3. Execute Retention Purge for eligible seeded.pupil_id from the Admin UI and verify 100% removal across Core and Dormant Plugin DB
    await page.context().addCookies([
      {
        name: 'mis_session',
        value: adminSessionBody.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    await page.goto(`/?pupil_id=${encodeURIComponent(seeded.pupil_id)}`);
    const purgeBtn = page.getByTestId('sar-purge-pupil-btn');
    await expect(purgeBtn).toBeVisible();
    await purgeBtn.click();

    const purgeStatus = page.getByTestId('sar-purge-status');
    await expect(purgeStatus).toContainText('RETENTION_PURGE_COMPLETED');

    // 4. Reactivate plugin to verify zero orphaned rows remain in plugin_test.db or core.db
    await request.post('/api/admin/plugins/uk.org.optimis.test/state', {
      data: { state: 'ACTIVE' },
    });

    const listNotesAfterPurge = await request.post('/api/plugins/uk.org.optimis.test/rpc/list_pupil_notes', {
      data: { args: { pupil_id: seeded.pupil_id } },
    });
    expect(listNotesAfterPurge.ok()).toBe(true);
    const listNotesBody = await listNotesAfterPurge.json();
    expect(listNotesBody.data.count).toBe(0);

    const getPupilAfterPurge = await request.post('/api/plugins/uk.org.optimis.test/rpc/get_pupil_record', {
      data: { args: { pupil_id: seeded.pupil_id } },
    });
    expect(getPupilAfterPurge.status()).toBe(400);
  });
});
