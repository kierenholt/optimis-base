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

interface DiagnosticEntry {
  timestamp: string;
  plugin_id: string;
  method: string;
  error_class: string;
  file: string;
  line: number;
  stack_trace: string;
  duration_ms: number;
  sqlite_lock_wait_ms: number;
  sqlite_write_hold_ms: number;
}

interface DiagnosticBundlePayload {
  schema_version: string;
  school_urn: string;
  generated_at: string;
  generated_by_staff_id: string;
  telemetry_mode: string;
  ring_buffer_capacity: number;
  entry_count: number;
  pii_scrubbed: boolean;
  entries: DiagnosticEntry[];
  log_archive: string;
}

const SPARX_SYNC_URL = 'https://sync.sparxmaths.uk/v1/homework/sync';

test.describe('Sparx Maths Third-Party Partner Plugin Vertical: Two-Phase Read/Egress -> WriteDB Sync & Diagnostic API Key Scrubbing (Sections 4, 8 Gap 1, 9 & 10)', () => {
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

    // 3. Ensure dev persona switcher and mock HTTP egress mode are active, and reset diagnostic ring buffer
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

    const clearEgressRes = await request.delete('/api/test/egress-mock');
    if (!clearEgressRes.ok()) {
      throw new Error(`Failed to clear mock egress rules: ${clearEgressRes.status()}`);
    }

    const resetDiagRes = await request.post('/api/test/diagnostics/reset', {
      data: { capacity: 100 },
    });
    if (!resetDiagRes.ok()) {
      throw new Error(`Failed to reset diagnostic ring buffer: ${resetDiagRes.status()}`);
    }

    // 4. Seed isolated persona fixtures & Milestone 1 teaching group/pupil fixtures
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
    if (!personaBody.fixture_id || !personaBody.admin_staff_id || !personaBody.teacher_staff_id) {
      throw new Error('Seeded persona fixture response missing mandatory identifiers');
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

    // 5. Install and activate sparx_maths_plugin (uk.co.sparx.maths)
    const installSparxRes = await request.post('/api/admin/plugins/install', {
      data: {
        folder: 'sparx_maths_plugin',
        staff_id: personas.admin_staff_id,
      },
    });
    if (!installSparxRes.ok()) {
      throw new Error(`Failed to install sparx_maths_plugin: ${installSparxRes.status()} ${await installSparxRes.text()}`);
    }
    const activateSparxRes = await request.post('/api/admin/plugins/uk.co.sparx.maths/activate', {
      data: { staff_id: personas.admin_staff_id },
    });
    if (!activateSparxRes.ok()) {
      throw new Error(`Failed to activate uk.co.sparx.maths: ${activateSparxRes.status()}`);
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Clear mock egress rules & reset diagnostic ring buffer
    await request.delete('/api/test/egress-mock');
    await request.post('/api/test/diagnostics/reset', {
      data: { capacity: 100 },
    });

    // 2. Purge plugin data created during the test
    if (personas && m1Seeded) {
      await request.post('/api/auth/persona', {
        data: { staff_id: personas.admin_staff_id },
      });
      await request.post('/api/plugins/uk.co.sparx.maths/rpc/purge_pupil_data', {
        data: { args: { pupil_id: m1Seeded.active_pupil_id } },
      });
      await request.post('/api/plugins/uk.co.sparx.maths/rpc/purge_pupil_data', {
        data: { args: { pupil_id: m1Seeded.other_group_pupil_id } },
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

    // 5. Remove runtime-installed plugin
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete('/api/admin/plugins/uk.co.sparx.maths');

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

  test('configureVendorApiKeyAndTriggerTwoPhaseSync_asTeacher_commitsKeyBeforeEgressSyncsOwnGroupsAndPersistsScores_test', async ({
    page,
    request,
  }) => {
    if (!personas || !m1Seeded) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    const vendorApiKey = 'spx_live_secret_9f8e7d6c5b4a3210_do_not_leak';

    // 1. Register mock egress response for https://sync.sparxmaths.uk/v1/homework/sync
    //    Returning homework completion scores for Ada Lovelace (active_pupil_id in own_group_id)
    const mockRuleRes = await request.post('/api/test/egress-mock', {
      data: {
        method: 'POST',
        url: SPARX_SYNC_URL,
        status: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          sync_batch_id: 'sparx_batch_20261028_01',
          homework_week: '2026-W44',
          scores: [
            {
              pupil_id: m1Seeded.active_pupil_id,
              homework_week: '2026-W44',
              completion_rate: 96,
              xp_earned: 420,
              bookwork_accuracy: 92,
              teacher_note: 'Ada Lovelace completed compulsory and XP Boost tasks ahead of deadline',
            },
          ],
        }),
      },
    });
    expect(mockRuleRes.status()).toBe(201);

    // 2. Authenticate as Milestone 1 Classroom Teacher (assigned to own_group_id, holding :own_groups scopes)
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: m1Seeded.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);
    const teacherSession = await switchTeacherRes.json();

    await page.context().addCookies([
      {
        name: 'mis_session',
        value: teacherSession.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    // 3. Navigate to Host UI Shell and open the Sparx Maths plugin iframe
    const query = new URLSearchParams({
      group_id: m1Seeded.own_group_id,
      pupil_id: m1Seeded.active_pupil_id,
      reference_date: m1Seeded.reference_date,
      academic_year_id: m1Seeded.academic_year_id,
      staff_id: m1Seeded.teacher_staff_id,
    });
    await page.goto(`/?${query.toString()}`);

    await expect(page.getByTestId('host-shell')).toBeVisible();
    const sparxNavBtn = page.getByTestId('nav-item-plugin-uk.co.sparx.maths');
    await expect(sparxNavBtn).toBeVisible();
    await sparxNavBtn.click();

    const iframeEl = page.getByTestId('plugin-iframe-uk.co.sparx.maths');
    await expect(iframeEl).toBeVisible();
    await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.co.sparx.maths"]');
    await expect(pluginFrame.getByTestId('sparx-maths-plugin-root')).toBeVisible();

    // 4. Phase 1 Setup: Configure vendor API key in plugin_sparx.db (committed before HTTP call)
    const apiKeyInput = pluginFrame.getByTestId('sparx-api-key-input');
    await expect(apiKeyInput).toBeVisible();
    await apiKeyInput.fill(vendorApiKey);

    const saveKeyBtn = pluginFrame.getByTestId('save-sparx-config-btn');
    await expect(saveKeyBtn).toBeVisible();
    await expect(saveKeyBtn).toHaveAttribute('data-mis-rpc', 'configure_vendor_credentials');
    await saveKeyBtn.click();

    const configStatusBadge = pluginFrame.getByTestId('sparx-config-status');
    await expect(configStatusBadge).toBeVisible();
    await expect(configStatusBadge).toContainText('CONFIGURED');
    await expect(pluginFrame.getByTestId('sparx-key-preview')).toContainText('spx_...3210');
    await expect(pluginFrame.getByTestId('sparx-key-preview')).not.toContainText(vendorApiKey);

    // 5. Phase 2 Sync: Trigger two-phase Read/Egress -> WriteDB sync via UI button
    //    Fetches :own_groups roster (Ada Lovelace in own_group_id; excludes transferred Alan Turing & other-group Grace Hopper),
    //    posts outbound to https://sync.sparxmaths.uk/v1/homework/sync with zero open SQLite write transaction,
    //    and persists returned homework completion scores into plugin_sparx.db.
    const syncBtn = pluginFrame.getByTestId('trigger-sparx-sync-btn');
    await expect(syncBtn).toBeVisible();
    await expect(syncBtn).toHaveAttribute('data-mis-rpc', 'sync_group_homework');
    await syncBtn.click();

    const syncSummary = pluginFrame.getByTestId('sparx-sync-summary');
    await expect(syncSummary).toBeVisible();
    await expect(pluginFrame.getByTestId('sparx-sync-status-badge')).toHaveText('SYNCED');
    await expect(pluginFrame.getByTestId('sparx-sync-batch-id')).toHaveText('sparx_batch_20261028_01');
    await expect(pluginFrame.getByTestId('sparx-synced-roster-count')).toHaveText('1');
    await expect(pluginFrame.getByTestId('sparx-persisted-scores-count')).toHaveText('1');
    await expect(pluginFrame.getByTestId('sparx-two-phase-tx-badge')).toHaveText('READ_EGRESS_THEN_WRITEDB_OK');

    // Verify persisted score row in the UI table
    const scoreRow = pluginFrame.getByTestId(`sparx-score-row-${m1Seeded.active_pupil_id}`);
    await expect(scoreRow).toBeVisible();
    await expect(scoreRow).toContainText('Ada Lovelace');
    await expect(scoreRow).toContainText('2026-W44');
    await expect(
      pluginFrame.getByTestId(`sparx-score-completion-${m1Seeded.active_pupil_id}`)
    ).toHaveText('96%');
    await expect(
      pluginFrame.getByTestId(`sparx-score-xp-${m1Seeded.active_pupil_id}`)
    ).toHaveText('420');
    await expect(
      pluginFrame.getByTestId(`sparx-score-bookwork-${m1Seeded.active_pupil_id}`)
    ).toHaveText('92%');

    // Verify transferred pupil (Alan Turing) and other-group pupil (Grace Hopper) were NOT included in the :own_groups sync
    await expect(
      pluginFrame.getByTestId(`sparx-score-row-${m1Seeded.transferred_pupil_id}`)
    ).toHaveCount(0);
    await expect(
      pluginFrame.getByTestId(`sparx-score-row-${m1Seeded.other_group_pupil_id}`)
    ).toHaveCount(0);

    // 6. Verify :own_groups RBAC isolation: Teacher attempting to sync other_group_id fails fast
    const unauthGroupSyncRes = await request.post(
      '/api/plugins/uk.co.sparx.maths/rpc/sync_group_homework',
      {
        data: {
          args: {
            group_id: m1Seeded.other_group_id,
            academic_year_id: m1Seeded.academic_year_id,
            reference_date: m1Seeded.reference_date,
            homework_week: '2026-W44',
          },
        },
      }
    );
    expect(unauthGroupSyncRes.status()).toBe(400);
    const unauthGroupSyncBody = await unauthGroupSyncRes.json();
    expect(unauthGroupSyncBody.ok).toBe(false);
    expect(unauthGroupSyncBody.error_code).toContain('own_groups');
  });

  test('sparxSyncFaultAndDiagnosticExport_onSimulatedFaultAndWriteTxEgressGuard_stripsVendorApiKeyAndPurgesGdprCleanly_test', async ({
    page,
    request,
  }) => {
    if (!personas || !m1Seeded) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    const vendorApiKey = 'spx_live_secret_key_77889900aabbccddeeff_sensitive';
    const bearerHeaderToken = `Bearer ${vendorApiKey}`;

    // 1. Authenticate as Teacher and configure the vendor API key in plugin_sparx.db
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: m1Seeded.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);

    const cfgRes = await request.post(
      '/api/plugins/uk.co.sparx.maths/rpc/configure_vendor_credentials',
      {
        data: {
          args: {
            api_key: vendorApiKey,
            sync_endpoint_url: SPARX_SYNC_URL,
          },
        },
      }
    );
    expect(cfgRes.ok()).toBe(true);

    // 2. Register mock egress rule to persist a score first (so we can also verify GDPR hooks)
    await request.post('/api/test/egress-mock', {
      data: {
        method: 'POST',
        url: SPARX_SYNC_URL,
        status: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          sync_batch_id: 'sparx_batch_gdpr_01',
          homework_week: '2026-W44',
          scores: [
            {
              pupil_id: m1Seeded.active_pupil_id,
              homework_week: '2026-W44',
              completion_rate: 100,
              xp_earned: 550,
              bookwork_accuracy: 98,
              teacher_note: 'Discussed Q4 algebra steps with Charles Babbage',
            },
          ],
        }),
      },
    });

    const syncOkRes = await request.post('/api/plugins/uk.co.sparx.maths/rpc/sync_group_homework', {
      data: {
        args: {
          group_id: m1Seeded.own_group_id,
          academic_year_id: m1Seeded.academic_year_id,
          reference_date: m1Seeded.reference_date,
          homework_week: '2026-W44',
        },
      },
    });
    expect(syncOkRes.ok()).toBe(true);

    // 3. Trigger simulated fault during sync_group_homework where the stored vendor API key
    //    (loaded from plugin_sparx.db, NOT passed in RPC args) and pupil ID are present in the fault message.
    //    Also verify the Host halts any open WriteDB tx if a single-phase write-before-egress violation is attempted.
    const simFaultRes = await request.post('/api/plugins/uk.co.sparx.maths/rpc/sync_group_homework', {
      data: {
        args: {
          group_id: m1Seeded.own_group_id,
          academic_year_id: m1Seeded.academic_year_id,
          reference_date: m1Seeded.reference_date,
          homework_week: '2026-W44',
          simulate_fault: 'VENDOR_AUTH_FAULT_WITH_STORED_KEY',
        },
      },
    });
    expect(simFaultRes.status()).toBe(400);

    const writeTxFaultRes = await request.post(
      '/api/plugins/uk.co.sparx.maths/rpc/sync_group_homework',
      {
        data: {
          args: {
            group_id: m1Seeded.own_group_id,
            academic_year_id: m1Seeded.academic_year_id,
            reference_date: m1Seeded.reference_date,
            homework_week: '2026-W44',
            simulate_fault: 'WRITE_TX_BEFORE_EGRESS',
          },
        },
      }
    );
    expect(writeTxFaultRes.status()).toBe(400);
    const writeTxFaultBody = await writeTxFaultRes.json();
    expect(writeTxFaultBody.error_code).toContain(
      'ctx.http.post disallowed inside an open SQLite write transaction'
    );

    // 4. Switch to System Admin / DPO and export the Diagnostic Bundle via UI & API
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personas.admin_staff_id },
    });
    expect(switchAdminRes.ok()).toBe(true);
    const adminSession = await switchAdminRes.json();

    await page.context().addCookies([
      {
        name: 'mis_session',
        value: adminSession.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    await page.goto(`/?pupil_id=${encodeURIComponent(m1Seeded.active_pupil_id)}`);
    await expect(page.getByTestId('admin-diagnostics-panel')).toBeVisible();
    await page.getByTestId('download-plugin-diagnostic-bundle-btn').click();
    await expect(page.getByTestId('diagnostic-bundle-status')).toContainText(
      'DIAGNOSTIC_BUNDLE_EXPORTED'
    );

    const bundleJsonText = await page.getByTestId('diagnostic-bundle-json').innerText();
    const bundleLogText = await page.getByTestId('diagnostic-bundle-log').innerText();

    const bundle = JSON.parse(bundleJsonText) as DiagnosticBundlePayload;
    expect(bundle.pii_scrubbed).toBe(true);

    const sparxEntries = bundle.entries.filter((e) => e.plugin_id === 'uk.co.sparx.maths');
    expect(sparxEntries.length).toBeGreaterThanOrEqual(2);

    const faultEntry = sparxEntries.find(
      (e) => e.method === 'sync_group_homework' && e.error_class === 'LUA_RUNTIME_ERROR'
    );
    expect(faultEntry).toBeDefined();
    expect(faultEntry!.file).toBe('src/init.lua');
    expect(faultEntry!.line).toBeGreaterThan(0);

    const writeTxHaltEntry = sparxEntries.find((e) => e.error_class === 'EGRESS_WRITE_TX_HALT');
    expect(writeTxHaltEntry).toBeDefined();

    // 5. Strict Verification: Diagnostic ring buffer MUST strip the vendor API key (both raw and Bearer header) and pupil IDs
    for (const forbiddenSecret of [
      vendorApiKey,
      bearerHeaderToken,
      '77889900aabbccddeeff',
      m1Seeded.active_pupil_id,
      m1Seeded.transferred_pupil_id,
      m1Seeded.other_group_pupil_id,
    ]) {
      expect(bundleJsonText).not.toContain(forbiddenSecret);
      expect(bundleLogText).not.toContain(forbiddenSecret);
    }

    // 6. Verify UK GDPR export_pupil_data & purge_pupil_data hooks on plugin_sparx.db
    const exportRes = await request.post('/api/plugins/uk.co.sparx.maths/rpc/export_pupil_data', {
      data: { args: { pupil_id: m1Seeded.active_pupil_id } },
    });
    expect(exportRes.ok()).toBe(true);
    const exportBody = await exportRes.json();
    expect(exportBody.ok).toBe(true);
    expect(exportBody.data.sections.length).toBeGreaterThanOrEqual(1);
    expect(exportBody.data.sections[0].title).toBe('Sparx Maths Homework Completion Scores');
    expect(exportBody.data.sections[0].records.length).toBe(1);
    expect(exportBody.data.sections[0].redactable_fields).toContain('teacher_note');

    const purgeRes = await request.post('/api/plugins/uk.co.sparx.maths/rpc/purge_pupil_data', {
      data: { args: { pupil_id: m1Seeded.active_pupil_id } },
    });
    expect(purgeRes.ok()).toBe(true);
    const purgeBody = await purgeRes.json();
    expect(purgeBody.ok).toBe(true);
    expect(purgeBody.data.purged_rows).toBeGreaterThanOrEqual(1);

    const exportAfterPurgeRes = await request.post(
      '/api/plugins/uk.co.sparx.maths/rpc/export_pupil_data',
      {
        data: { args: { pupil_id: m1Seeded.active_pupil_id } },
      }
    );
    expect(exportAfterPurgeRes.ok()).toBe(true);
    const exportAfterPurgeBody = await exportAfterPurgeRes.json();
    expect(exportAfterPurgeBody.data.sections[0].records.length).toBe(0);
  });
});
