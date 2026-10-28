import { test, expect } from '@playwright/test';
import { execFileSync } from 'child_process';
import path from 'path';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

const MIS_SERVER_BIN = path.resolve(process.cwd(), 'bin/mis-server');

function runMisServerCli(args: string[]): { stdout: string; stderr: string; exitCode: number } {
  try {
    const stdout = execFileSync(MIS_SERVER_BIN, args, {
      cwd: process.cwd(),
      encoding: 'utf-8',
      env: { ...process.env },
    });
    return { stdout: stdout.trim(), stderr: '', exitCode: 0 };
  } catch (err: any) {
    return {
      stdout: typeof err.stdout === 'string' ? err.stdout.trim() : (err.stdout?.toString('utf-8') || '').trim(),
      stderr: typeof err.stderr === 'string' ? err.stderr.trim() : (err.stderr?.toString('utf-8') || '').trim(),
      exitCode: typeof err.status === 'number' ? err.status : 1,
    };
  }
}

test.describe('Local Break-Glass CLI (./mis-server config ... — Section 7C)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let originalSchoolName: string | null = null;
  const keysToDeleteInCleanup = [
    'sso_oidc_issuer_url',
    'sso_oidc_client_id',
    'sso_provider',
    'sso_redirect_uri',
    'test_break_glass_custom_key',
  ];

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    originalSchoolName = null;

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

    // 3. Ensure dev persona switcher mode is enabled at start of test
    const modeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!modeRes.ok()) {
      throw new Error(`Failed to reset auth-mode in beforeEach: ${modeRes.status()}`);
    }

    // 4. Snapshot original school_name if present in __system_config
    const getSchoolName = runMisServerCli(['config', 'get', 'school_name']);
    if (getSchoolName.exitCode === 0 && getSchoolName.stdout) {
      originalSchoolName = getSchoolName.stdout;
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Restore original school_name or delete it if it wasn't present before
    if (originalSchoolName !== null) {
      runMisServerCli(['config', 'set', 'school_name', originalSchoolName]);
    } else {
      await request.delete('/api/test/system-config/school_name');
    }

    // 2. Clean up any break-glass & SSO keys written during the test
    for (const key of keysToDeleteInCleanup) {
      await request.delete(`/api/test/system-config/${encodeURIComponent(key)}`);
    }

    // 3. Ensure dev persona switcher mode is restored to true after test
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });

    // 4. Fetch Backend Service Logs emitted during this test
    if (logCursor) {
      const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
      if (logsRes.ok()) {
        const serviceLogs: string[] = (await logsRes.json()).entries;
        if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
          console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
        }
      }
    }

    // 5. Dump tapped Browser Console logs on failure or error
    if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
      console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
      console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
    }

    expect(pageErrors).toEqual([]);
  });

  test('configListGetAndSetCli_existingCoreDb_inspectsAndUpdatesSystemConfigReflectedInHostUi_test', async ({
    page,
  }) => {
    // 1. Inspect existing __system_config via `config list` and `config get school_urn`
    const listRes = runMisServerCli(['config', 'list']);
    expect(listRes.exitCode).toBe(0);
    expect(listRes.stdout).toContain('school_urn=100001');

    const getUrnRes = runMisServerCli(['config', 'get', 'school_urn']);
    expect(getUrnRes.exitCode).toBe(0);
    expect(getUrnRes.stdout).toBe('100001');

    // 2. Mutate __system_config via `config set <key> <value>`
    const setCustomRes = runMisServerCli([
      'config',
      'set',
      'test_break_glass_custom_key',
      'break_glass_verified_value_01',
    ]);
    expect(setCustomRes.exitCode).toBe(0);
    expect(setCustomRes.stdout).toContain('test_break_glass_custom_key=break_glass_verified_value_01');

    const getCustomRes = runMisServerCli(['config', 'get', 'test_break_glass_custom_key']);
    expect(getCustomRes.exitCode).toBe(0);
    expect(getCustomRes.stdout).toBe('break_glass_verified_value_01');

    const updatedSchoolName = 'Oakridge Break-Glass CLI Academy';
    const setSchoolNameRes = runMisServerCli(['config', 'set', 'school_name', updatedSchoolName]);
    expect(setSchoolNameRes.exitCode).toBe(0);

    // 3. Verify `config list --json` outputs structured JSON entries including updated_at
    const listJsonRes = runMisServerCli(['config', 'list', '--json']);
    expect(listJsonRes.exitCode).toBe(0);
    const parsedEntries = JSON.parse(listJsonRes.stdout) as Array<{
      key: string;
      value: string;
      updated_at: string;
    }>;
    expect(Array.isArray(parsedEntries)).toBe(true);
    const customEntry = parsedEntries.find((e) => e.key === 'test_break_glass_custom_key');
    expect(customEntry).toBeDefined();
    expect(customEntry?.value).toBe('break_glass_verified_value_01');
    expect(Boolean(customEntry?.updated_at)).toBe(true);

    // 4. Load the Host UI Shell in Playwright and verify the updated school_name in __system_config is rendered
    await page.goto('/');
    await expect(page.getByTestId('host-shell')).toBeVisible();
    await expect(page.getByTestId('host-school-name')).toHaveText(updatedSchoolName);
  });

  test('configRepairSsoCli_misconfiguredSsoAndDevSwitcherDisabled_repairsSsoInCoreDbAndUnblocksSsoLogin_test', async ({
    request,
  }) => {
    // 1. Simulate broken/misconfigured SSO in __system_config (non-HTTPS issuer and missing client_id)
    const setBrokenRes = runMisServerCli([
      'config',
      'set',
      'sso_oidc_issuer_url',
      'http://misconfigured-idp.oakridge.internal/v2.0',
    ]);
    expect(setBrokenRes.exitCode).toBe(0);

    // 2. Switching to real SSO mode (ENABLE_DEV_PERSONA_SWITCHER=false) must fail fast with REAL_SSO_NOT_CONFIGURED
    const brokenModeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: false },
    });
    expect(brokenModeRes.status()).toBe(500);
    const brokenModeBody = await brokenModeRes.json();
    expect(brokenModeBody.ok).toBe(false);
    expect(brokenModeBody.error_code).toContain('REAL_SSO_NOT_CONFIGURED');

    const brokenLoginRes = await request.get('/api/auth/sso/login');
    expect(brokenLoginRes.status()).toBe(500);
    const brokenLoginBody = await brokenLoginRes.json();
    expect(brokenLoginBody.ok).toBe(false);
    expect(brokenLoginBody.error_code).toContain('REAL_SSO_NOT_CONFIGURED');

    // 3. Execute Break-Glass CLI `./mis-server config repair-sso` from the terminal to repair __system_config
    const repairRes = runMisServerCli([
      'config',
      'repair-sso',
      '--provider',
      'ENTRA_ID',
      '--issuer-url',
      'https://login.microsoftonline.com/9314001-oakridge-tenant/v2.0',
      '--client-id',
      'oakridge-entra-oidc-client-001',
      '--redirect-uri',
      'https://mis.oakridge.sch.uk/api/auth/sso/callback',
    ]);
    expect(repairRes.exitCode).toBe(0);
    expect(repairRes.stdout).toContain('sso_oidc_issuer_url=https://login.microsoftonline.com/9314001-oakridge-tenant/v2.0');
    expect(repairRes.stdout).toContain('sso_oidc_client_id=oakridge-entra-oidc-client-001');
    expect(repairRes.stdout).toContain('sso_provider=ENTRA_ID');

    // 4. Verify each repaired key via `config get <key>`
    expect(runMisServerCli(['config', 'get', 'sso_oidc_issuer_url']).stdout).toBe(
      'https://login.microsoftonline.com/9314001-oakridge-tenant/v2.0'
    );
    expect(runMisServerCli(['config', 'get', 'sso_oidc_client_id']).stdout).toBe(
      'oakridge-entra-oidc-client-001'
    );
    expect(runMisServerCli(['config', 'get', 'sso_provider']).stdout).toBe('ENTRA_ID');
    expect(runMisServerCli(['config', 'get', 'sso_redirect_uri']).stdout).toBe(
      'https://mis.oakridge.sch.uk/api/auth/sso/callback'
    );

    // 5. Verify that real SSO mode and /api/auth/sso/login now succeed using the repaired __system_config
    const repairedModeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: false },
    });
    expect(repairedModeRes.status()).toBe(200);
    const repairedModeBody = await repairedModeRes.json();
    expect(repairedModeBody.ok).toBe(true);
    expect(repairedModeBody.enable_dev_persona_switcher).toBe(false);

    const ssoLoginRes = await request.get('/api/auth/sso/login');
    expect(ssoLoginRes.status()).toBe(200);
    const ssoLoginBody = await ssoLoginRes.json();
    expect(ssoLoginBody.ok).toBe(true);
    expect(ssoLoginBody.provider).toBe('ENTRA_ID');
    expect(ssoLoginBody.issuer_url).toBe('https://login.microsoftonline.com/9314001-oakridge-tenant/v2.0');
    expect(ssoLoginBody.client_id).toBe('oakridge-entra-oidc-client-001');
    expect(ssoLoginBody.redirect_uri).toBe('https://mis.oakridge.sch.uk/api/auth/sso/callback');
    expect(ssoLoginBody.authorization_url).toContain(
      'https://login.microsoftonline.com/9314001-oakridge-tenant/v2.0'
    );
    expect(ssoLoginBody.authorization_url).toContain('client_id=oakridge-entra-oidc-client-001');
  });

  test('configCliFailFast_missingOrInvalidArguments_exitsNonZeroWithExplicitError_test', async () => {
    // 1. Missing subcommand
    const noSubRes = runMisServerCli(['config']);
    expect(noSubRes.exitCode).not.toBe(0);
    expect(noSubRes.stderr).toContain('subcommand');

    // 2. Unknown subcommand
    const badSubRes = runMisServerCli(['config', 'invalid-action']);
    expect(badSubRes.exitCode).not.toBe(0);
    expect(badSubRes.stderr).toContain('unknown config subcommand');

    // 3. `config get` without key or with non-existent key
    const getMissingArgRes = runMisServerCli(['config', 'get']);
    expect(getMissingArgRes.exitCode).not.toBe(0);
    expect(getMissingArgRes.stderr).toContain('key');

    const getUnknownKeyRes = runMisServerCli(['config', 'get', 'non_existent_system_config_key_999']);
    expect(getUnknownKeyRes.exitCode).not.toBe(0);
    expect(getUnknownKeyRes.stderr).toContain('missing mandatory __system_config key: non_existent_system_config_key_999');

    // 4. `config set` with missing value
    const setMissingValRes = runMisServerCli(['config', 'set', 'some_key_only']);
    expect(setMissingValRes.exitCode).not.toBe(0);
    expect(setMissingValRes.stderr).toContain('value');

    // 5. `config repair-sso` with non-HTTPS issuer-url or missing client-id
    const repairNonHttpsRes = runMisServerCli([
      'config',
      'repair-sso',
      '--issuer-url',
      'http://insecure-idp.example.org/v2.0',
      '--client-id',
      'client-123',
    ]);
    expect(repairNonHttpsRes.exitCode).not.toBe(0);
    expect(repairNonHttpsRes.stderr).toContain('https://');

    const repairMissingClientRes = runMisServerCli([
      'config',
      'repair-sso',
      '--issuer-url',
      'https://accounts.google.com',
    ]);
    expect(repairMissingClientRes.exitCode).not.toBe(0);
    expect(repairMissingClientRes.stderr).toContain('client-id');
  });
});
