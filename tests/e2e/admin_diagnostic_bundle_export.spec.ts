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

test.describe('Admin-Exported PII-Scrubbed Diagnostic Bundle (Section 8 Gap 1, Section 9.3 & Section 10.3)', () => {
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

    // 2. Tap Backend Service Logs
    const cursorRes = await request.get('/api/test/logs/cursor');
    if (!cursorRes.ok()) {
      throw new Error(`Failed to obtain service log cursor: ${cursorRes.status()}`);
    }
    logCursor = (await cursorRes.json()).cursor;

    // 3. Ensure dev persona switcher is enabled
    const authModeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!authModeRes.ok()) {
      throw new Error(`Failed to enable dev persona switcher: ${authModeRes.status()}`);
    }

    // 4. Reset diagnostic ring buffer to default capacity before each test
    const resetDiagRes = await request.post('/api/test/diagnostics/reset', {
      data: { capacity: 100 },
    });
    if (!resetDiagRes.ok()) {
      throw new Error(`Failed to reset diagnostic ring buffer: ${resetDiagRes.status()} ${await resetDiagRes.text()}`);
    }

    // 5. Seed isolated staff personas & pupil fixtures
    const seedRes = await request.post('/api/test/seed-gdpr-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-20',
      },
    });
    if (!seedRes.ok()) {
      throw new Error(`Failed to seed fixtures: ${seedRes.status()} ${await seedRes.text()}`);
    }
    const body = (await seedRes.json()) as SeededGDPRFixtures;
    if (!body.fixture_id || !body.admin_staff_id || !body.teacher_staff_id || !body.dsl_staff_id || !body.pupil_id) {
      throw new Error('Seeded fixture response missing mandatory identifiers');
    }
    seeded = body;

    // Ensure uk.org.optimis.test is ACTIVE
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.admin_staff_id },
    });
    if (!switchAdminRes.ok()) {
      throw new Error(`Failed to switch to Admin persona in setup: ${switchAdminRes.status()}`);
    }
    await request.post('/api/admin/plugins/uk.org.optimis.test/state', {
      data: { state: 'ACTIVE' },
    });
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Restore default diagnostic ring buffer capacity & plugin state
    await request.post('/api/test/diagnostics/reset', {
      data: { capacity: 100 },
    });

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

    // 4. Clean up seeded fixtures in core.db
    if (seeded?.fixture_id) {
      const delRes = await request.delete(`/api/test/seed-gdpr-fixtures/${encodeURIComponent(seeded.fixture_id)}`);
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up fixture ${seeded.fixture_id}: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('exportDiagnosticBundle_nonAdminPersonasRejectedAndAdminExportsSanitizedRingBufferViaUi_stripsAllPupilIdsAndLocalVariables_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded state');
    }

    // 1. Trigger plugin Lua runtime error containing pupil IDs, UPN, ULN, and sensitive local/payload strings
    const leakedPupilId = 'pup_secret_leaked_999';
    const leakedUpn = 'A123456789012';
    const leakedUln = '9876543210';
    const leakedLocalSecret = 'Confidential local variable for Oliver Twist';

    const rpcErr1 = await request.post('/api/plugins/uk.org.optimis.test/rpc/get_pupil_record', {
      data: {
        args: {
          pupil_id: leakedPupilId,
          upn: leakedUpn,
          uln: leakedUln,
          local_debug_secret: leakedLocalSecret,
        },
      },
    });
    expect(rpcErr1.status()).toBe(400);

    // 2. Trigger plugin mutation + illegal HTTP egress inside an open WriteDB transaction
    //    This acquires the SQLite write lock (recording lock wait & hold durations) and halts at src/init.lua:365
    const secretNoteId = `pnote_${seeded.fixture_id}_diag_secret`;
    const secretNoteText = 'Confidential safeguarding disclosure involving Harry Potter UPN B987654321012';
    const rpcErr2 = await request.post('/api/plugins/uk.org.optimis.test/rpc/mutate_then_illegal_egress', {
      data: {
        args: {
          pupil_id: seeded.pupil_id,
          note_id: secretNoteId,
          note: secretNoteText,
          target_url: 'https://sync.sparxmaths.uk/v1/sync',
        },
      },
    });
    expect(rpcErr2.status()).toBe(400);

    // 3. Verify RBAC: Classroom Teacher and DSL (non-Admin/DPO) MUST be rejected with 403 on GET /api/admin/diagnostics/bundle
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);
    const teacherSessionBody = await switchTeacherRes.json();

    const teacherDiagRes = await request.get('/api/admin/diagnostics/bundle');
    expect(teacherDiagRes.status()).toBe(403);
    const teacherDiagBody = await teacherDiagRes.json();
    expect(teacherDiagBody.ok).toBe(false);
    expect(teacherDiagBody.error_code).toContain('UNAUTHORIZED_ADMIN_DIAGNOSTICS_SCOPE');

    const switchDslRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.dsl_staff_id },
    });
    expect(switchDslRes.ok()).toBe(true);

    const dslDiagRes = await request.get('/api/admin/diagnostics/bundle');
    expect(dslDiagRes.status()).toBe(403);
    const dslDiagBody = await dslDiagRes.json();
    expect(dslDiagBody.ok).toBe(false);
    expect(dslDiagBody.error_code).toContain('UNAUTHORIZED_ADMIN_DIAGNOSTICS_SCOPE');

    // 4. Open Host UI Shell as Classroom Teacher first and verify Admin Diagnostics Panel is hidden
    await page.context().addCookies([
      {
        name: 'mis_session',
        value: teacherSessionBody.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    await page.goto(`/?pupil_id=${encodeURIComponent(seeded.pupil_id)}`);
    await expect(page.getByTestId('host-shell')).toBeVisible();
    await expect(page.getByTestId('admin-diagnostics-panel')).toHaveCount(0);
    await expect(page.getByTestId('download-plugin-diagnostic-bundle-btn')).toHaveCount(0);

    // 5. Switch to System Admin / DPO via Topbar Persona Switcher and verify "Download Plugin Diagnostic Bundle" appears
    const personaSelect = page.getByTestId('persona-switcher-select');
    await expect(personaSelect).toBeVisible();
    await personaSelect.selectOption(seeded.admin_staff_id);

    const diagPanel = page.getByTestId('admin-diagnostics-panel');
    await expect(diagPanel).toBeVisible();

    const downloadBtn = page.getByTestId('download-plugin-diagnostic-bundle-btn');
    await expect(downloadBtn).toBeVisible();
    await expect(downloadBtn).toContainText('Download Plugin Diagnostic Bundle');

    // 6. Click "Download Plugin Diagnostic Bundle" and verify exported JSON & .log archive
    await downloadBtn.click();

    await expect(page.getByTestId('diagnostic-bundle-status')).toContainText('DIAGNOSTIC_BUNDLE_EXPORTED');

    const bundleJsonText = await page.getByTestId('diagnostic-bundle-json').innerText();
    const bundleLogText = await page.getByTestId('diagnostic-bundle-log').innerText();

    const bundle = JSON.parse(bundleJsonText) as DiagnosticBundlePayload;
    expect(bundle.school_urn).toBe('100001');
    expect(bundle.telemetry_mode).toBe('ADMIN_EXPORT_ONLY_ZERO_OUTBOUND');
    expect(bundle.pii_scrubbed).toBe(true);
    expect(bundle.generated_by_staff_id).toBe(seeded.admin_staff_id);
    expect(bundle.entry_count).toBeGreaterThanOrEqual(2);
    expect(bundle.entries.length).toBe(bundle.entry_count);

    // Verify get_pupil_record Lua stack trace entry (src/init.lua line 13)
    const getPupilEntry = bundle.entries.find(
      (e) => e.plugin_id === 'uk.org.optimis.test' && e.method === 'get_pupil_record' && e.error_class === 'LUA_RUNTIME_ERROR'
    );
    expect(getPupilEntry).toBeDefined();
    expect(getPupilEntry!.file).toBe('src/init.lua');
    expect(getPupilEntry!.line).toBe(13);
    expect(getPupilEntry!.stack_trace).toContain('src/init.lua:13');
    expect(getPupilEntry!.stack_trace).toContain('get_pupil_record');
    expect(getPupilEntry!.duration_ms).toBeGreaterThanOrEqual(0);

    // Verify mutate_then_illegal_egress stack trace and SQLite write lock wait/hold durations
    const illegalEgressRows = bundle.entries.filter(
      (e) =>
        e.plugin_id === 'uk.org.optimis.test' &&
        (e.method === 'mutate_then_illegal_egress' || e.method === 'ctx.http.post')
    );
    expect(illegalEgressRows.length).toBeGreaterThanOrEqual(1);

    const mutateMethodEntry = bundle.entries.find(
      (e) => e.plugin_id === 'uk.org.optimis.test' && e.method === 'mutate_then_illegal_egress'
    );
    expect(mutateMethodEntry).toBeDefined();
    expect(mutateMethodEntry!.file).toBe('src/init.lua');
    expect(mutateMethodEntry!.line).toBe(365);
    expect(mutateMethodEntry!.stack_trace).toContain('src/init.lua:365');
    expect(mutateMethodEntry!.stack_trace).toContain('mutate_then_illegal_egress');
    expect(typeof mutateMethodEntry!.sqlite_lock_wait_ms).toBe('number');
    expect(mutateMethodEntry!.sqlite_lock_wait_ms).toBeGreaterThanOrEqual(0);
    expect(typeof mutateMethodEntry!.sqlite_write_hold_ms).toBe('number');
    expect(mutateMethodEntry!.sqlite_write_hold_ms).toBeGreaterThanOrEqual(0);

    // 7. Strict Zero-PII & Zero-Local-Variable Verification across both JSON and .log representations
    for (const forbiddenToken of [
      leakedPupilId,
      seeded.pupil_id,
      seeded.protected_pupil_id,
      leakedUpn,
      'B987654321012',
      leakedUln,
      'Oliver Twist',
      'Harry Potter',
      secretNoteId,
      secretNoteText,
      leakedLocalSecret,
    ]) {
      expect(bundleJsonText).not.toContain(forbiddenToken);
      expect(bundleLogText).not.toContain(forbiddenToken);
    }
  });

  test('exportDiagnosticBundle_ringBufferExceedsCapacity_evictsOldestEntriesAndFiltersByPlugin_test', async ({
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded state');
    }

    // 1. Configure ring buffer capacity to 3
    const resetRes = await request.post('/api/test/diagnostics/reset', {
      data: { capacity: 3 },
    });
    expect(resetRes.ok()).toBe(true);

    // 2. Switch to Admin persona
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.admin_staff_id },
    });
    expect(switchAdminRes.ok()).toBe(true);

    // 3. Trigger 5 sequential undeclared endpoint faults
    for (let i = 1; i <= 5; i++) {
      const res = await request.post(`/api/plugins/uk.org.optimis.test/rpc/undeclared_fault_${i}`, {
        data: { args: { pupil_id: seeded.pupil_id } },
      });
      expect(res.status()).toBe(400);
    }

    // 4. Export diagnostic bundle via GET /api/admin/diagnostics/bundle?plugin_id=uk.org.optimis.test
    const bundleRes = await request.get('/api/admin/diagnostics/bundle?plugin_id=uk.org.optimis.test');
    expect(bundleRes.status()).toBe(200);
    expect(bundleRes.headers()['content-disposition']).toContain('attachment; filename="optimis-plugin-diagnostics-100001.json"');

    const body = await bundleRes.json();
    expect(body.ok).toBe(true);

    const bundle = body.bundle as DiagnosticBundlePayload;
    expect(bundle.ring_buffer_capacity).toBe(3);
    expect(bundle.entry_count).toBe(3);
    expect(bundle.entries.map((e) => e.method)).toEqual([
      'undeclared_fault_3',
      'undeclared_fault_4',
      'undeclared_fault_5',
    ]);
    expect(bundle.log_archive).not.toContain('undeclared_fault_1');
    expect(bundle.log_archive).not.toContain('undeclared_fault_2');
    expect(bundle.log_archive).toContain('undeclared_fault_5');
    expect(bundle.log_archive).not.toContain(seeded.pupil_id);
  });
});
