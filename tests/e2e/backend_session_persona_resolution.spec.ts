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

test.describe('Backend Session & Persona Resolution (Section 8A & Auth Flag)', () => {
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

    // 4. Seed isolated staff personas & pupil fixture explicitly in setup
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

    // 3. Ensure dev persona switcher mode is restored to true and remove runtime-installed plugin
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete('/api/admin/plugins/uk.org.optimis.test');

    // 4. Clean up seeded persona & pupil fixtures at the end of the test
    if (seeded?.fixture_id) {
      const delRes = await request.delete(`/api/test/seed-personas/${encodeURIComponent(seeded.fixture_id)}`);
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up persona fixture ${seeded.fixture_id}: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('listAndSelectPersona_devPersonaSwitcherEnabled_resolvesSessionRolesAndScopesFromDatabase_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded persona state');
    }

    // 1. List available active personas from core.db
    const listRes = await request.get('/api/auth/personas');
    expect(listRes.ok()).toBe(true);
    const listBody = await listRes.json();
    expect(listBody.ok).toBe(true);
    expect(Array.isArray(listBody.personas)).toBe(true);

    const staffIds = listBody.personas.map((p: { staff_id: string }) => p.staff_id);
    expect(staffIds).toContain(seeded.admin_staff_id);
    expect(staffIds).toContain(seeded.dsl_staff_id);
    expect(staffIds).toContain(seeded.teacher_staff_id);
    // Terminated staff member must NEVER appear in active persona list
    expect(staffIds).not.toContain(seeded.terminated_staff_id);

    // 2. Switch persona to DSL and verify resolved session state
    const switchDslRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.dsl_staff_id },
    });
    expect(switchDslRes.ok()).toBe(true);
    const setCookieHeader = switchDslRes.headers()['set-cookie'] || '';
    expect(setCookieHeader).toContain('mis_session=');
    expect(setCookieHeader.toLowerCase()).toContain('samesite=strict');

    const dslSessionBody = await switchDslRes.json();
    expect(dslSessionBody.ok).toBe(true);
    expect(dslSessionBody.caller.staff_id).toBe(seeded.dsl_staff_id);
    expect(dslSessionBody.caller.active_roles).toContain('DSL');
    expect(dslSessionBody.caller.scopes).toContain('dal.safeguarding.read');
    expect(dslSessionBody.caller.scopes).toContain('dal.pupils.read:all');

    // 3. Verify GET /api/auth/session resolves the active session from the cookie
    const sessionRes = await request.get('/api/auth/session');
    expect(sessionRes.ok()).toBe(true);
    const sessionBody = await sessionRes.json();
    expect(sessionBody.ok).toBe(true);
    expect(sessionBody.caller.staff_id).toBe(seeded.dsl_staff_id);

    // 4. Verify browser page load with active Admin persona executes RPC using resolved session
    await page.context().addCookies([
      {
        name: 'mis_session',
        value: dslSessionBody.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    await page.goto(`/?pupil_id=${encodeURIComponent(seeded.pupil_id)}`);
    const hostShell = page.getByTestId('host-shell');
    await expect(hostShell).toBeVisible();

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    const printBtn = pluginFrame.getByTestId('print-record-btn');
    await expect(printBtn).toBeVisible();
    await printBtn.click();

    const expectedLogSubstring = `[plugin:uk.org.optimis.test] {"id":"${seeded.pupil_id}"`;
    await expect
      .poll(() => browserLogs.some((entry) => entry.includes(expectedLogSubstring)), {
        message: `Expected browser console logs to contain ${expectedLogSubstring}`,
        timeout: 5000,
      })
      .toBe(true);
  });

  test('executeRpcWithPersonaSession_switchedToPersonaWithoutRequiredScope_failsFastWithUnauthorizedScope_test', async ({
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded persona state');
    }

    // 1. Selecting a terminated or non-existent staff_id must fail fast
    const badRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.terminated_staff_id },
    });
    expect(badRes.status()).toBe(400);

    // 2. Switch to Classroom Teacher persona (only holds :own_groups scopes, NOT dal.pupils.read:all)
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);
    const teacherBody = await switchTeacherRes.json();
    expect(teacherBody.caller.staff_id).toBe(seeded.teacher_staff_id);
    expect(teacherBody.caller.active_roles).toEqual(['TEACHER']);
    expect(teacherBody.caller.scopes).toContain('dal.pupils.read:own_groups');
    expect(teacherBody.caller.scopes).not.toContain('dal.pupils.read:all');

    // 3. Calling get_pupil_record (which requires dal.pupils.read:all in manifest.json) as this Teacher MUST fail fast
    const rpcRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/get_pupil_record', {
      data: {
        args: { pupil_id: seeded.pupil_id },
      },
    });
    expect(rpcRes.status()).toBe(400);
    const rpcBody = await rpcRes.json();
    expect(rpcBody.ok).toBe(false);
    expect(rpcBody.error_code).toContain('caller lacks required DAL scopes');
  });

  test('loadRealSsoWhenDevPersonaSwitcherDisabled_envFlagFalse_failsFastOnRealSsoAndBlocksPersonaEndpoints_test', async ({
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded persona state');
    }

    // 1. Set ENABLE_DEV_PERSONA_SWITCHER=false and verify attempting to load REAL SSO fails fast immediately
    const modeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: false },
    });
    expect(modeRes.status()).toBe(500);
    const modeBody = await modeRes.json();
    expect(modeBody.ok).toBe(false);
    expect(modeBody.error_code).toContain('REAL_SSO_NOT_CONFIGURED');

    // 2. Attempting to initiate real SSO login when ENABLE_DEV_PERSONA_SWITCHER=false must fail fast
    const ssoRes = await request.get('/api/auth/sso/login');
    expect(ssoRes.status()).toBe(500);
    const ssoBody = await ssoRes.json();
    expect(ssoBody.ok).toBe(false);
    expect(ssoBody.error_code).toContain('REAL_SSO_NOT_CONFIGURED');

    // 3. Attempting to list or switch dev personas when ENABLE_DEV_PERSONA_SWITCHER=false must fail fast
    const personasRes = await request.get('/api/auth/personas');
    expect(personasRes.status()).toBe(403);
    const personasBody = await personasRes.json();
    expect(personasBody.ok).toBe(false);
    expect(personasBody.error_code).toBe('DEV_PERSONA_SWITCHER_DISABLED');

    const switchRes = await request.post('/api/auth/persona', {
      data: { staff_id: seeded.admin_staff_id },
    });
    expect(switchRes.status()).toBe(403);
    const switchBody = await switchRes.json();
    expect(switchBody.ok).toBe(false);
    expect(switchBody.error_code).toBe('DEV_PERSONA_SWITCHER_DISABLED');
  });
});
