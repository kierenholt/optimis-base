import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

interface SeededMilestone1 {
  fixture_id: string;
  academic_year_id: string;
  reference_date: string;
  teacher_staff_id: string;
  other_teacher_id: string;
  dsl_staff_id: string;
  own_group_id: string;
  other_group_id: string;
  active_pupil_id: string;
}

test.describe('Real & Mock ctx.http Egress Engine (Sections 4, 9.2 & 10.1)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let seeded: SeededMilestone1 | null = null;

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

    // 3. Ensure dev persona switcher, mock egress mode, and runtime-installed test_plugin are active
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

    const modeRes = await request.post('/api/test/egress-mode', {
      data: { enable_real_http_egress: false },
    });
    if (!modeRes.ok()) {
      throw new Error(`Failed to configure egress mode: ${modeRes.status()}`);
    }
    const clearRes = await request.delete('/api/test/egress-mock');
    if (!clearRes.ok()) {
      throw new Error(`Failed to clear mock egress rules: ${clearRes.status()}`);
    }

    // 4. Seed isolated fixture data in setup
    const seedRes = await request.post('/api/test/seed-milestone1-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-27',
      },
    });
    if (!seedRes.ok()) {
      throw new Error(`Failed to seed fixtures: ${seedRes.status()}`);
    }
    seeded = (await seedRes.json()) as SeededMilestone1;
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Restore mock egress mode, clear mock rules, and purge plugin data
    await request.post('/api/test/egress-mode', {
      data: { enable_real_http_egress: false },
    });
    await request.delete('/api/test/egress-mock');
    if (seeded?.active_pupil_id) {
      await request.post('/api/plugins/uk.org.optimis.test/rpc/purge_pupil_data', {
        data: { args: { pupil_id: seeded.active_pupil_id } },
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

    // 4. Remove runtime-installed plugin
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete('/api/admin/plugins/uk.org.optimis.test');

    // 5. Clean up seeded fixtures
    if (seeded?.fixture_id) {
      const delRes = await request.delete(
        `/api/test/seed-milestone1-fixtures/${encodeURIComponent(seeded.fixture_id)}`
      );
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up fixture ${seeded.fixture_id}: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('ctxHttpGetAndPost_approvedDomainWithMockEngine_returnsStatusHeadersBodyAndFollowsApprovedRedirect_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded state');
    }

    // 1. Register a 302 redirect on an approved domain (https://sync.sparxmaths.uk/v1/start -> https://sync.sparxmaths.uk/v1/final)
    const mockRedirRes = await request.post('/api/test/egress-mock', {
      data: {
        method: 'GET',
        url: 'https://sync.sparxmaths.uk/v1/start',
        status: 302,
        redirect_to: 'https://sync.sparxmaths.uk/v1/final',
      },
    });
    expect(mockRedirRes.ok()).toBe(true);

    const mockFinalRes = await request.post('/api/test/egress-mock', {
      data: {
        method: 'GET',
        url: 'https://sync.sparxmaths.uk/v1/final',
        status: 200,
        headers: { 'X-Sparx-Sync': 'verified-redirect' },
        body: '{"sync_status":"SPARX_REDIRECT_OK","homework_completion":94}',
      },
    });
    expect(mockFinalRes.ok()).toBe(true);

    // 2. Register a POST mock rule on https://sync.sparxmaths.uk/v1/push
    const mockPostRes = await request.post('/api/test/egress-mock', {
      data: {
        method: 'POST',
        url: 'https://sync.sparxmaths.uk/v1/push',
        status: 201,
        headers: { 'X-Sparx-Receipt': 'rcpt_901' },
        body: '{"receipt_id":"rcpt_901","accepted":true}',
      },
    });
    expect(mockPostRes.ok()).toBe(true);

    // 3. Load Host Shell UI with sandboxed plugin iframe and trigger GET sync via UI button
    const query = new URLSearchParams({
      pupil_id: seeded.active_pupil_id,
      group_id: seeded.own_group_id,
      other_group_id: seeded.other_group_id,
      reference_date: seeded.reference_date,
      academic_year_id: seeded.academic_year_id,
      staff_id: seeded.teacher_staff_id,
    });
    await page.goto(`/?${query.toString()}`);

    const hostShell = page.getByTestId('host-shell');
    await expect(hostShell).toBeVisible();

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    const syncGetBtn = pluginFrame.getByTestId('egress-sync-get-btn');
    await expect(syncGetBtn).toBeVisible();
    await syncGetBtn.click();

    const syncResult = pluginFrame.getByTestId('egress-sync-result');
    await expect(syncResult).toContainText('SPARX_REDIRECT_OK');
    await expect(syncResult).toContainText('https://sync.sparxmaths.uk/v1/final');

    // 4. Trigger POST sync via UI button and assert status 201 + response payload
    const syncPostBtn = pluginFrame.getByTestId('egress-sync-post-btn');
    await expect(syncPostBtn).toBeVisible();
    await syncPostBtn.click();

    await expect(syncResult).toContainText('rcpt_901');
    await expect(syncResult).toContainText('"status":201');
  });

  test('ctxHttpGet_redirectToUnapprovedDomain_rejectsRedirectInCheckRedirectAndFailsFast_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded state');
    }

    // Configure initial approved domain URL to redirect (302) to an unapproved external domain
    const mockRedirRes = await request.post('/api/test/egress-mock', {
      data: {
        method: 'GET',
        url: 'https://sync.sparxmaths.uk/v1/start',
        status: 302,
        redirect_to: 'https://evil.unapproved-domain.org/exfiltrate',
      },
    });
    expect(mockRedirRes.ok()).toBe(true);

    const query = new URLSearchParams({
      pupil_id: seeded.active_pupil_id,
      group_id: seeded.own_group_id,
      other_group_id: seeded.other_group_id,
      reference_date: seeded.reference_date,
      academic_year_id: seeded.academic_year_id,
      staff_id: seeded.teacher_staff_id,
    });
    await page.goto(`/?${query.toString()}`);

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');
    const syncGetBtn = pluginFrame.getByTestId('egress-sync-get-btn');
    await expect(syncGetBtn).toBeVisible();
    await syncGetBtn.click();

    const errorBanner = pluginFrame.getByTestId('rpc-error-banner');
    await expect(errorBanner).toBeVisible();
    await expect(errorBanner).toContainText('egress redirect target domain not permitted by manifest.json');
  });

  test('ctxHttpPost_undeclaredDomainOrOpenWriteTransaction_rollsBackWriteTxAndFailsFast_test', async ({
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded state');
    }

    // 1. Calling ctx.http.post to an undeclared domain must fail fast
    const unapprovedRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/push_external_sparx_post', {
      data: {
        args: {
          pupil_id: seeded.active_pupil_id,
          target_url: 'https://unapproved.vendor.io/webhook',
          payload: '{"event":"test"}',
          staff_id: seeded.teacher_staff_id,
        },
      },
    });
    expect(unapprovedRes.status()).toBe(400);
    const unapprovedBody = await unapprovedRes.json();
    expect(unapprovedBody.ok).toBe(false);
    expect(unapprovedBody.error_code).toContain('egress domain not permitted by manifest.json');

    // 2. Prefix-spoofed domain (https://sync.sparxmaths.uk.attacker.com) must also fail fast
    const spoofedRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/push_external_sparx_post', {
      data: {
        args: {
          pupil_id: seeded.active_pupil_id,
          target_url: 'https://sync.sparxmaths.uk.attacker.com/v1/push',
          payload: '{"event":"test"}',
          staff_id: seeded.teacher_staff_id,
        },
      },
    });
    expect(spoofedRes.status()).toBe(400);
    const spoofedBody = await spoofedRes.json();
    expect(spoofedBody.ok).toBe(false);
    expect(spoofedBody.error_code).toContain('egress domain not permitted by manifest.json');

    // 3. Invoking ctx.http inside an open plugin_db write transaction must halt/rollback the write transaction and fail fast
    const writeTxRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/mutate_then_illegal_egress', {
      data: {
        args: {
          pupil_id: seeded.active_pupil_id,
          note_id: 'note_should_rollback_1',
          note: 'Uncommitted note before illegal HTTP call',
          target_url: 'https://sync.sparxmaths.uk/v1/push',
          staff_id: seeded.teacher_staff_id,
        },
      },
    });
    expect(writeTxRes.status()).toBe(400);
    const writeTxBody = await writeTxRes.json();
    expect(writeTxBody.ok).toBe(false);
    expect(writeTxBody.error_code).toContain('disallowed inside an open SQLite write transaction');

    // 4. Verify the uncommitted note was rolled back in plugin_test.db
    const listRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/list_pupil_notes', {
      data: {
        args: {
          pupil_id: seeded.active_pupil_id,
          staff_id: seeded.teacher_staff_id,
        },
      },
    });
    expect(listRes.ok()).toBe(true);
    const listBody = await listRes.json();
    expect(listBody.ok).toBe(true);
    expect(listBody.data.count).toBe(0);
  });

  test('ctxHttpGet_upstreamExceedsEgressTimeoutMs_abortsRequestAndFailsFast_test', async ({
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded state');
    }

    // Register a slow mock response that exceeds the request/egress timeout
    const mockSlowRes = await request.post('/api/test/egress-mock', {
      data: {
        method: 'GET',
        url: 'https://sync.sparxmaths.uk/v1/slow',
        status: 200,
        delay_ms: 3000,
        body: '{"too_slow":true}',
      },
    });
    expect(mockSlowRes.ok()).toBe(true);

    const slowRpcRes = await request.post('/api/plugins/uk.org.optimis.test/rpc/sync_external_sparx_get', {
      data: {
        args: {
          pupil_id: seeded.active_pupil_id,
          target_url: 'https://sync.sparxmaths.uk/v1/slow',
          staff_id: seeded.teacher_staff_id,
        },
      },
    });
    expect(slowRpcRes.status()).toBe(400);
    const slowBody = await slowRpcRes.json();
    expect(slowBody.ok).toBe(false);
    expect(slowBody.error_code).toMatch(/EGRESS_TIMEOUT|context deadline exceeded|Client\.Timeout/i);
  });
});
