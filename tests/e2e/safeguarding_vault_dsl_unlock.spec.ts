import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

interface SeededPersonaFixtures {
  fixture_id: string;
  pupil_id: string;
  admin_staff_id: string;
  dsl_staff_id: string;
  teacher_staff_id: string;
  terminated_staff_id: string;
}

test.describe('Safeguarding Vault Envelope Encryption & DSL Cloud Key Unlock (Section 5B)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let seeded: SeededPersonaFixtures | null = null;

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

    // 3. Install and activate test_plugin at runtime for this test
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    const installRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'test_plugin' },
    });
    if (!installRes.ok()) {
      throw new Error(`Failed to install test_plugin: ${installRes.status()}`);
    }
    const activateRes = await request.post('/api/admin/plugins/uk.org.optimis.test/activate', {
      data: {},
    });
    if (!activateRes.ok()) {
      throw new Error(`Failed to activate uk.org.optimis.test: ${activateRes.status()}`);
    }

    // 4. Seed isolated staff personas (Teacher, DSL, Admin) & pupil fixture in setup
    const seedRes = await request.post('/api/test/seed-personas', {
      data: {
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    if (!seedRes.ok()) {
      throw new Error(`Failed to seed persona fixtures: ${seedRes.status()}`);
    }
    const body = (await seedRes.json()) as SeededPersonaFixtures;
    if (
      !body.fixture_id ||
      !body.pupil_id ||
      !body.admin_staff_id ||
      !body.dsl_staff_id ||
      !body.teacher_staff_id ||
      !body.terminated_staff_id
    ) {
      throw new Error('Seeded persona fixture response missing mandatory identifiers');
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

    // 3. Reset auth mode and remove runtime-installed plugin
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete('/api/admin/plugins/uk.org.optimis.test');

    // 4. Clean up seeded persona, safeguarding notes, audit logs, & pupil fixtures at the end of the test
    if (seeded?.fixture_id) {
      const delRes = await request.delete(`/api/test/seed-personas/${encodeURIComponent(seeded.fixture_id)}`);
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up persona fixture ${seeded.fixture_id}: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('safeguardingPostboxAndDslCloudKeyUnlock_teacherSubmitsPostboxAndDslUnlocksWithSignedCloudKey_encryptsAtRestAndDecryptsForDslWithRenewal_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded persona state');
    }

    // 1. Switch session to Classroom Teacher (holds dal.safeguarding.create, NOT dal.safeguarding.read)
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);
    const teacherBody = await switchTeacherRes.json();
    expect(teacherBody.caller.staff_id).toBe(seeded.teacher_staff_id);
    expect(teacherBody.caller.scopes).toContain('dal.safeguarding.create');
    expect(teacherBody.caller.scopes).not.toContain('dal.safeguarding.read');

    // 2. Teacher submits a concern into the Write-Only Safeguarding Postbox
    const postboxRes = await request.post('/api/safeguarding/postbox', {
      data: {
        pupil_id: seeded.pupil_id,
        assigned_dsl_staff_id: seeded.dsl_staff_id,
        concern_category: 'EMOTIONAL',
        incident_date: '2026-10-28',
        summary: 'Distressed during morning registration',
        body_map: 'NONE',
        verbatim_account: 'Pupil stated they had not eaten since yesterday evening.',
        action_taken: 'Escalated immediately to DSL via Safeguarding Postbox',
      },
    });
    expect(postboxRes.status()).toBe(201);
    const postboxBody = await postboxRes.json();
    expect(postboxBody.ok).toBe(true);
    expect(postboxBody.receipt.note_id).toMatch(/^sfg_/);
    expect(postboxBody.receipt.status).toBe('OPEN_TRIAGE');
    const noteId: string = postboxBody.receipt.note_id;

    // 3. Teacher attempting to list DSL vault notes, issue a DSL cloud key, or decrypt the note MUST fail fast (403)
    const teacherListRes = await request.get(`/api/safeguarding/notes?pupil_id=${encodeURIComponent(seeded.pupil_id)}`);
    expect(teacherListRes.status()).toBe(403);

    const teacherKeyRes = await request.post('/api/safeguarding/cloud-key/renew', {
      data: { ttl_seconds: 3600 },
    });
    expect(teacherKeyRes.status()).toBe(403);

    const teacherDecryptRes = await request.post(`/api/safeguarding/notes/${encodeURIComponent(noteId)}/decrypt`, {
      data: { cloud_key_token: 'forged.token' },
    });
    expect(teacherDecryptRes.status()).toBe(403);

    // 4. Switch session to Designated Safeguarding Lead (DSL)
    const switchDslRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.dsl_staff_id },
    });
    expect(switchDslRes.ok()).toBe(true);
    const dslSessionBody = await switchDslRes.json();
    expect(dslSessionBody.caller.staff_id).toBe(seeded.dsl_staff_id);
    expect(dslSessionBody.caller.scopes).toContain('dal.safeguarding.read');

    // 5. Verify DSL Vault UI in Browser: encrypted metadata is listed without plaintext until unlocked with Cloud Key
    await page.context().addCookies([
      {
        name: 'mis_session',
        value: dslSessionBody.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    await page.goto(`/?pupil_id=${encodeURIComponent(seeded.pupil_id)}`);
    const vaultPanel = page.getByTestId('dsl-safeguarding-vault-panel');
    await expect(vaultPanel).toBeVisible();

    const noteCard = page.getByTestId(`safeguarding-note-card-${noteId}`);
    await expect(noteCard).toBeVisible();
    await expect(noteCard).toContainText('EMOTIONAL');
    await expect(noteCard).toContainText('OPEN_TRIAGE');
    // Plaintext MUST NOT be visible before Cloud Key unlock
    await expect(noteCard).not.toContainText('Distressed during morning registration');
    await expect(noteCard).not.toContainText('Pupil stated they had not eaten since yesterday evening.');

    // 6. Attempt to unlock with an expired Cloud Key token first -> must fail fast with renewal error banner
    const expiredKeyRes = await request.post('/api/safeguarding/cloud-key/renew', {
      data: { ttl_seconds: -60 },
    });
    expect(expiredKeyRes.ok()).toBe(true);
    const expiredKeyBody = await expiredKeyRes.json();
    expect(expiredKeyBody.ok).toBe(true);
    expect(typeof expiredKeyBody.cloud_key_token).toBe('string');

    const cloudKeyInput = page.getByTestId('dsl-cloud-key-input');
    await cloudKeyInput.fill(expiredKeyBody.cloud_key_token);

    const decryptBtn = page.getByTestId(`decrypt-safeguarding-note-btn-${noteId}`);
    await decryptBtn.click();

    const vaultErrorBanner = page.getByTestId('dsl-vault-error-banner');
    await expect(vaultErrorBanner).toBeVisible();
    await expect(vaultErrorBanner).toContainText('expired');

    // 7. Click "Renew OAuth Cloud Key" in the UI to generate a fresh short-lived signed Cloud Key and unlock the note
    const renewCloudKeyBtn = page.getByTestId('renew-dsl-cloud-key-btn');
    await expect(renewCloudKeyBtn).toBeVisible();
    await renewCloudKeyBtn.click();

    await expect(cloudKeyInput).not.toHaveValue('');
    await expect(cloudKeyInput).not.toHaveValue(expiredKeyBody.cloud_key_token);

    await decryptBtn.click();

    const decryptedView = page.getByTestId(`decrypted-safeguarding-note-${noteId}`);
    await expect(decryptedView).toBeVisible();
    await expect(decryptedView).toContainText('Distressed during morning registration');
    await expect(decryptedView).toContainText('Pupil stated they had not eaten since yesterday evening.');
    await expect(decryptedView).toContainText('Escalated immediately to DSL via Safeguarding Postbox');
    await expect(decryptedView).toContainText('CREATE_POSTBOX');
    await expect(decryptedView).toContainText('DSL_DECRYPT_VIEW');
  });

  test('luaSafeguardingReadIsolation_pluginDeclaringSafeguardingCreateOrRead_exposesOnlyCreatePostboxAndBlocksDecryptInLua_test', async ({
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded persona state');
    }

    // Even when logged in as DSL (holding both dal.safeguarding.create and dal.safeguarding.read),
    // the Lua plugin sandbox (`uk.org.optimis.test`) must ONLY ever expose `ctx.dal.safeguarding.create_postbox`
    // and never expose `read`, `decrypt`, or `get_by_id` to third-party Lua code.
    const switchDslRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.dsl_staff_id },
    });
    expect(switchDslRes.ok()).toBe(true);

    const rpcRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/submit_safeguarding_postbox', {
      data: {
        args: {
          pupil_id: seeded.pupil_id,
          reference_date: '2026-10-28',
        },
      },
    });
    expect(rpcRes.ok()).toBe(true);
    const rpcBody = await rpcRes.json();
    expect(rpcBody.ok).toBe(true);
    expect(rpcBody.data.status).toBe('POSTBOX_ACCEPTED');
    expect(rpcBody.data.note_id).toMatch(/^sfg_/);
    expect(rpcBody.data.read_back_blocked).toBe(true);
  });
});
