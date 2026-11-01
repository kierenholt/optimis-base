import { test, expect } from '@playwright/test';
import { spawnSync } from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

const ROOT_DIR = process.cwd();
const MIS_VERIFY_BIN = path.resolve(ROOT_DIR, 'bin/mis-verify');

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
  second_active_pupil_id: string;
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
  attendance_officer_staff_id: string;
  terminated_staff_id: string;
}

function runBinary(
  binPath: string,
  args: string[],
  cwd: string = ROOT_DIR
): { status: number; stdout: string; stderr: string } {
  if (!fs.existsSync(binPath)) {
    throw new Error(`Required CLI binary not found at ${binPath}`);
  }
  const proc = spawnSync(binPath, args, {
    cwd,
    encoding: 'utf-8',
    timeout: 30000,
  });
  return {
    status: proc.status ?? -1,
    stdout: proc.stdout || '',
    stderr: proc.stderr || '',
  };
}

test.describe('Monochrome Lucide Icon Theme Contract: Host Shell, Plugin SDK [data-mis-icon], and mis-verify Gatekeeper', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let m1Seed: Milestone1SeedResponse | null = null;
  let personaSeed: PersonaSeedResponse | null = null;
  let testPlugId: string = '';
  let senPlugId: string = '';
  let behaviourPlugId: string = '';
  let tempDirs: string[] = [];

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    m1Seed = null;
    personaSeed = null;
    testPlugId = '';
    senPlugId = '';
    behaviourPlugId = '';
    tempDirs = [];

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

    // 4. Seed Persona fixtures and Milestone 1 scoped fixtures
    const pSeedRes = await request.post('/api/test/seed-personas', {
      data: {
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    if (!pSeedRes.ok()) {
      throw new Error(`Failed to seed persona fixtures: ${pSeedRes.status()}`);
    }
    personaSeed = (await pSeedRes.json()) as PersonaSeedResponse;

    const m1Res = await request.post('/api/test/seed-milestone1-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-27',
        include_second_active_pupil: true,
      },
    });
    if (!m1Res.ok()) {
      throw new Error(`Failed to seed Milestone 1 fixtures: ${m1Res.status()}`);
    }
    m1Seed = (await m1Res.json()) as Milestone1SeedResponse;

    // 5. Switch to Admin persona and install + activate test_plugin, senco_provision_plugin, and behaviour_plugin
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personaSeed.admin_staff_id },
    });
    if (!switchAdminRes.ok()) {
      throw new Error(`Failed to switch to Admin persona in setup: ${switchAdminRes.status()}`);
    }

    const installTestRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'test_plugin', staff_id: personaSeed.admin_staff_id },
    });
    if (!installTestRes.ok()) {
      throw new Error(`Failed to install test_plugin: ${installTestRes.status()}`);
    }
    testPlugId = (await installTestRes.json()).plug_id;
    await request.post(`/api/admin/plugins/${encodeURIComponent(testPlugId)}/activate`, {
      data: { staff_id: personaSeed.admin_staff_id },
    });

    const installSenRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'senco_provision_plugin', staff_id: personaSeed.admin_staff_id },
    });
    if (!installSenRes.ok()) {
      throw new Error(`Failed to install senco_provision_plugin: ${installSenRes.status()}`);
    }
    senPlugId = (await installSenRes.json()).plug_id;
    await request.post(`/api/admin/plugins/${encodeURIComponent(senPlugId)}/activate`, {
      data: { staff_id: personaSeed.admin_staff_id },
    });

    const installBehRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'behaviour_plugin', staff_id: personaSeed.admin_staff_id },
    });
    if (!installBehRes.ok()) {
      throw new Error(`Failed to install behaviour_plugin: ${installBehRes.status()}`);
    }
    behaviourPlugId = (await installBehRes.json()).plug_id;
    await request.post(`/api/admin/plugins/${encodeURIComponent(behaviourPlugId)}/activate`, {
      data: { staff_id: personaSeed.admin_staff_id },
    });
  });

  test.afterEach(async ({ request }, testInfo) => {
    if (logCursor) {
      const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
      if (logsRes.ok()) {
        const serviceLogs: string[] = (await logsRes.json()).entries;
        if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
          console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
        }
      }
    }

    if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
      console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
      console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
    }

    for (const dir of tempDirs) {
      try {
        fs.rmSync(dir, { recursive: true, force: true });
      } catch {
        // ignore cleanup error
      }
    }

    if (personaSeed) {
      await request.post('/api/auth/persona', {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      await request.post('/api/system/pinned-recents', {
        data: {
          action: 'clear',
          staff_id: personaSeed.admin_staff_id,
        },
      });
      if (m1Seed) {
        await request.post('/api/system/pinned-recents', {
          data: {
            action: 'clear',
            staff_id: m1Seed.teacher_staff_id,
          },
        });
      }
      for (const pid of [behaviourPlugId, senPlugId, testPlugId]) {
        if (pid) {
          await request.delete(
            `/api/admin/plugins/${encodeURIComponent(pid)}?staff_id=${encodeURIComponent(
              personaSeed.admin_staff_id
            )}`
          );
        }
      }
    }
    if (m1Seed) {
      await request.delete(`/api/test/seed-milestone1-fixtures/${encodeURIComponent(m1Seed.fixture_id)}`);
      m1Seed = null;
    }
    if (personaSeed) {
      await request.delete(`/api/test/seed-personas/${encodeURIComponent(personaSeed.fixture_id)}`);
      personaSeed = null;
    }

    expect(pageErrors).toEqual([]);
  });

  test('hostShellMonochromeLucideIcons_sidebarTopbarSearchDashboardAndAdminPanels_rendersMonochromeLucideSvgIconsWithZeroEmojisAndInheritsThemeCurrentColor_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: m1Seed.teacher_staff_id },
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

    const targetSenRoute = `/p/${senPlugId}/student/${encodeURIComponent(
      m1Seed.second_active_pupil_id
    )}?tab=iep`;
    await page.goto(
      `${targetSenRoute}&reference_date=${encodeURIComponent(
        m1Seed.reference_date
      )}&academic_year_id=${encodeURIComponent(m1Seed.academic_year_id)}&staff_id=${encodeURIComponent(
        m1Seed.teacher_staff_id
      )}`
    );

    const hostShell = page.getByTestId('host-shell');
    await expect(hostShell).toBeVisible();

    // 1. Verify Narrow Icon Rail buttons render monochrome Lucide <svg> icons with stroke="currentColor" and fill="none"
    for (const [testId, expectedIcon] of [
      ['nav-rail-home', 'house'],
      ['nav-rail-search', 'search'],
      ['nav-rail-settings', 'settings'],
      ['nav-rail-logout', 'log-out'],
    ] as const) {
      const btn = page.getByTestId(testId);
      await expect(btn).toBeVisible();
      const svg = btn.locator('svg').first();
      await expect(svg).toBeVisible();
      await expect(svg).toHaveAttribute('data-lucide-icon', expectedIcon);
      await expect(svg).toHaveAttribute('stroke', 'currentColor');
      await expect(svg).toHaveAttribute('fill', 'none');
    }

    // 2. Verify Pin Current View button in Narrow Icon Rail renders monochrome Lucide <svg> icon with zero emojis
    const topbarPinBtn = page.getByTestId('topbar-pin-view-btn');
    await expect(topbarPinBtn).toBeVisible();
    await expect(topbarPinBtn).toContainText('Pin Current View');
    await expect(topbarPinBtn.locator('svg').first()).toHaveAttribute('data-lucide-icon', 'pin');
    await expect(topbarPinBtn.locator('svg').first()).toHaveAttribute('stroke', 'currentColor');

    // Pin the route and verify Pinned Recents rail item renders the plugin's declared monochrome Lucide icon
    await topbarPinBtn.click();
    await expect(topbarPinBtn).toContainText('Pinned');

    const pinnedItem0 = page.getByTestId('pinned-recent-item-0');
    await expect(pinnedItem0).toBeVisible();
    const pinnedSvg0 = pinnedItem0.locator('svg').first();
    await expect(pinnedSvg0).toBeVisible();
    await expect(pinnedSvg0).toHaveAttribute('data-lucide-icon', 'heart-pulse');
    await expect(pinnedSvg0).toHaveAttribute('stroke', 'currentColor');
    await expect(pinnedSvg0).toHaveAttribute('fill', 'none');

    const unpinBtn0 = page.getByTestId('unpin-recent-btn-0');
    await expect(unpinBtn0).toBeVisible();
    await expect(unpinBtn0.locator('svg').first()).toHaveAttribute('data-lucide-icon', 'x');

    // 3. Open Global Command Palette and verify monochrome Lucide icons on view results, widget results, and pin controls
    await page.getByTestId('nav-rail-search').click();
    const searchModal = page.getByTestId('command-palette-modal');
    await expect(searchModal).toBeVisible();

    const behSearchView = page.getByTestId(`command-palette-result-view-${behaviourPlugId}-home`);
    await expect(behSearchView).toBeVisible();
    await expect(behSearchView.locator('svg').first()).toHaveAttribute('data-lucide-icon', 'award');

    const searchPinBehBtn = page.getByTestId(`search-pin-view-btn-${behaviourPlugId}-home`);
    await expect(searchPinBehBtn).toBeVisible();
    await expect(searchPinBehBtn.locator('svg').first()).toHaveAttribute('data-lucide-icon', 'pin');
    const searchPinText = await searchPinBehBtn.innerText();
    expect(searchPinText).not.toMatch(/[★☆📌]/);

    // Verify conventional close button with an x icon (and no "Esc" text) on Command Palette
    const closePaletteBtn = page.getByTestId('command-palette-close-btn');
    await expect(closePaletteBtn).toBeVisible();
    await expect(closePaletteBtn).toHaveAttribute('aria-label', 'Close');
    await expect(closePaletteBtn).toHaveText('');
    await expect(closePaletteBtn.locator('svg').first()).toHaveAttribute('data-lucide-icon', 'x');
    await expect(closePaletteBtn.locator('svg').first()).toHaveAttribute('stroke', 'currentColor');

    // Close Command Palette via the conventional x close button
    await closePaletteBtn.click();
    await expect(searchModal).toHaveCount(0);

    // 4. Verify DashboardWidgetGridPanel edit mode controls render monochrome Lucide <svg> icons
    const editToggleBtn = page.getByTestId('dashboard-edit-mode-toggle-btn');
    await expect(editToggleBtn.locator('svg').first()).toBeVisible();
    await expect(editToggleBtn.locator('svg').first()).toHaveAttribute('stroke', 'currentColor');

    // 5. Switch to DSL persona and verify SafeguardingVaultPanel renders monochrome Lucide icons
    await page.goto(`/?__persona=${encodeURIComponent(personaSeed.dsl_staff_id)}`);
    const dslPanel = page.getByTestId('dsl-safeguarding-vault-panel');
    await expect(dslPanel).toBeVisible();
    await expect(dslPanel.locator('svg').first()).toHaveAttribute('data-lucide-icon', 'shield-alert');
    await expect(dslPanel.locator('svg').first()).toHaveAttribute('stroke', 'currentColor');

    // 6. Switch to Admin persona and verify GdprSarPanel and AdminDiagnosticsPanel render monochrome Lucide icons
    await page.goto(`/?__persona=${encodeURIComponent(personaSeed.admin_staff_id)}`);
    const sarPanel = page.getByTestId('sar-redaction-panel');
    await expect(sarPanel).toBeVisible();
    await expect(sarPanel.locator('svg').first()).toHaveAttribute('data-lucide-icon', 'file-check');

    const diagPanel = page.getByTestId('admin-diagnostics-panel');
    await expect(diagPanel).toBeVisible();
    const downloadDiagBtn = page.getByTestId('download-plugin-diagnostic-bundle-btn');
    await expect(downloadDiagBtn.locator('svg').first()).toHaveAttribute('data-lucide-icon', 'download');

    // 7. Assert zero emojis or unicode star/pin glyphs exist anywhere in the Host Shell DOM text
    const hostTextContent = await hostShell.innerText();
    expect(hostTextContent).not.toMatch(/[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}★☆]/u);

    // 8. Verify theme inheritance: toggling Light <-> High-Contrast updates icon computed color via currentColor
    const lightNavColor = await page
      .getByTestId('nav-rail-search')
      .locator('svg')
      .first()
      .evaluate((el) => getComputedStyle(el).color);
    expect(lightNavColor).toBe('rgb(248, 250, 252)'); // #f8fafc (--mis-text-sidebar in light theme)

    await page.getByTestId('nav-rail-settings').click();
    await page.getByTestId('host-settings-modal').getByTestId('theme-mode-toggle-btn').click();
    const closeSettingsBtn = page.getByTestId('close-settings-modal-btn');
    await expect(closeSettingsBtn).toBeVisible();
    await expect(closeSettingsBtn).toHaveAttribute('aria-label', 'Close');
    await expect(closeSettingsBtn).toHaveText('');
    await expect(closeSettingsBtn.locator('svg').first()).toHaveAttribute('data-lucide-icon', 'x');
    await closeSettingsBtn.click();

    await expect(hostShell).toHaveAttribute('data-mis-theme', 'high-contrast');
    await page.getByTestId('nav-rail-home').click();
    const hcHomeColor = await page
      .getByTestId('nav-rail-home')
      .locator('svg')
      .first()
      .evaluate((el) => getComputedStyle(el).color);
    expect(hcHomeColor).toBe('rgb(0, 0, 0)'); // #000000 (--mis-brand-primary-fg on active home button in high-contrast)
  });

  test('pluginSdkAndMisVerifyMonochromeLucideIcons_manifestIconsDataMisIconAndEmojiLint_rendersPluginIframeLucideIconsAndRejectsInvalidIconsOrEmojisInVerify_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // 1. Verify GET /api/system/ui-config exposes declared Lucide icon names on plugins, views, and widgets
    const uiConfigRes = await request.get(
      `/api/system/ui-config?staff_id=${encodeURIComponent(personaSeed.admin_staff_id)}`
    );
    expect(uiConfigRes.ok()).toBe(true);
    const uiConfigBody = await uiConfigRes.json();
    const behPlugin = (uiConfigBody.plugins as Array<Record<string, unknown>>).find(
      (p) => p.id === behaviourPlugId
    );
    expect(behPlugin).toBeTruthy();
    expect(behPlugin?.icon).toBe('award');

    const senPlugin = (uiConfigBody.plugins as Array<Record<string, unknown>>).find(
      (p) => p.id === senPlugId
    );
    expect(senPlugin).toBeTruthy();
    expect(senPlugin?.icon).toBe('heart-pulse');

    // 2. Navigate to Behaviour plugin workspace and verify [data-mis-icon] inside sandboxed plugin iframe is hydrated with a monochrome Lucide <svg>
    await page.goto(
      `/p/${behaviourPlugId}?pupil_id=${encodeURIComponent(
        m1Seed.active_pupil_id
      )}&group_id=${encodeURIComponent(m1Seed.own_group_id)}&__persona=${encodeURIComponent(
        m1Seed.teacher_staff_id
      )}`
    );

    const behFrame = page.frameLocator(`[data-testid="plugin-iframe-${behaviourPlugId}"]`);
    const pluginIconSlot = behFrame.locator('[data-mis-icon]').first();
    await expect(pluginIconSlot).toBeVisible();
    await expect(pluginIconSlot).toHaveAttribute('data-mis-icon', 'award');

    const iframeSvg = pluginIconSlot.locator('svg').first();
    await expect(iframeSvg).toBeVisible();
    await expect(iframeSvg).toHaveAttribute('data-lucide-icon', 'award');
    await expect(iframeSvg).toHaveAttribute('stroke', 'currentColor');
    await expect(iframeSvg).toHaveAttribute('fill', 'none');

    // Verify zero emojis inside the plugin iframe
    const iframeBodyText = await behFrame.locator('body').innerText();
    expect(iframeBodyText).not.toMatch(/[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}]/u);

    // 3. Verify bin/mis-verify passes on behaviour_plugin and fails fast on unknown Lucide icon or emoji in plugin UI
    const passRes = runBinary(MIS_VERIFY_BIN, [path.resolve(ROOT_DIR, 'plugins/behaviour_plugin')]);
    expect(passRes.status).toBe(0);

    const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'optimis-lucide-verify-e2e-'));
    tempDirs.push(tmpRoot);

    // Failure A: Unknown Lucide icon name in manifest.json
    const badIconDir = path.join(tmpRoot, 'bad_icon_plugin');
    fs.cpSync(path.resolve(ROOT_DIR, 'plugins/behaviour_plugin'), badIconDir, { recursive: true });
    const manifestPath = path.join(badIconDir, 'manifest.json');
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf-8'));
    manifest.icon = 'not-a-valid-lucide-icon-xyz';
    fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));

    const badIconRes = runBinary(MIS_VERIFY_BIN, [badIconDir]);
    expect(badIconRes.status).not.toBe(0);
    expect(badIconRes.stdout + badIconRes.stderr).toContain('UNKNOWN_LUCIDE_ICON');

    // Failure B: Disallowed emoji character in plugin UI HTML
    const emojiUiDir = path.join(tmpRoot, 'emoji_ui_plugin');
    fs.cpSync(path.resolve(ROOT_DIR, 'plugins/behaviour_plugin'), emojiUiDir, { recursive: true });
    fs.appendFileSync(
      path.join(emojiUiDir, 'ui/dist/index.html'),
      '\n<div>🏠 Non-monochrome emoji icon</div>\n'
    );

    const emojiUiRes = runBinary(MIS_VERIFY_BIN, [emojiUiDir]);
    expect(emojiUiRes.status).not.toBe(0);
    expect(emojiUiRes.stdout + emojiUiRes.stderr).toContain('NON_MONOCHROME_ICON');
  });
});
