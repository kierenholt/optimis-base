import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

const ROOT_DIR = process.cwd();

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

test.describe('Global Search & Discovery: Landing Page Top-Half Search & Command Palette (Ctrl+K)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let m1Seed: Milestone1SeedResponse | null = null;
  let personaSeed: PersonaSeedResponse | null = null;
  let testPlugId: string = '';
  let behaviourPlugId: string = '';
  let senPlugId: string = '';

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    m1Seed = null;
    personaSeed = null;
    testPlugId = '';
    behaviourPlugId = '';
    senPlugId = '';

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

    // 3. Ensure dev persona switcher is enabled
    const authModeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!authModeRes.ok()) {
      throw new Error(`Failed to enable dev persona switcher: ${authModeRes.status()}`);
    }

    // 4. Seed Persona fixtures and Milestone 1 scoped group/pupil fixtures
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

    // 5. Switch to Admin persona and install + activate test_plugin, behaviour_plugin, and senco_provision_plugin
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

    // 3. Clean up installed plugins, pinned recents, and seeded fixtures
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
      for (const pid of [senPlugId, behaviourPlugId, testPlugId]) {
        if (pid) {
          await request.delete(`/api/admin/plugins/${encodeURIComponent(pid)}`);
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

  test('manifestSchemaAndUiConfig_withViewsAndWidgetsKeywords_exposesSearchSynonymsAndVerbsWithoutReverseDnsLeak_test', async ({
    request,
  }) => {
    // 1. Verify sdk/manifest.schema.json defines keywords: string[] on views[] and widgets[]
    const schemaPath = path.resolve(ROOT_DIR, 'sdk/manifest.schema.json');
    const schema = JSON.parse(fs.readFileSync(schemaPath, 'utf-8'));

    const viewProps = schema.properties?.views?.items?.properties;
    expect(viewProps).toBeDefined();
    expect(viewProps.keywords).toEqual({
      type: 'array',
      description: expect.any(String),
      items: {
        type: 'string',
        minLength: 1,
      },
    });

    const widgetProps = schema.properties?.widgets?.items?.properties;
    expect(widgetProps).toBeDefined();
    expect(widgetProps.keywords).toEqual({
      type: 'array',
      description: expect.any(String),
      items: {
        type: 'string',
        minLength: 1,
      },
    });

    // 2. Verify GET /api/system/ui-config returns keywords on views[] and widgets[] with opaque plug_id
    const uiConfigRes = await request.get('/api/system/ui-config');
    expect(uiConfigRes.ok()).toBe(true);
    const uiConfigBody = await uiConfigRes.json();
    const rawJson = JSON.stringify(uiConfigBody);
    expect(rawJson).not.toContain('uk.org.optimis.');

    const behPlugin = uiConfigBody.plugins.find((p: any) => p.id === behaviourPlugId);
    expect(behPlugin).toBeDefined();
    const behHomeView = behPlugin.views.find((v: any) => v.id === 'home');
    expect(behHomeView).toBeDefined();
    expect(Array.isArray(behHomeView.keywords)).toBe(true);
    expect(behHomeView.keywords).toEqual(expect.arrayContaining(['detention', 'sanction', 'merit']));

    const behWidget = behPlugin.widgets.find((w: any) => w.id === 'behaviour_summary');
    expect(behWidget).toBeDefined();
    expect(Array.isArray(behWidget.keywords)).toBe(true);
    expect(behWidget.keywords).toEqual(expect.arrayContaining(['detention', 'merit', 'pastoral']));
  });

  test('landingPageTopHalfSearch_synonymKeywordAndWidgetAddAction_navigatesToPluginViewAndPlacesWidgetOnDashboard_test', async ({
    page,
  }) => {
    await page.goto('/?reference_date=2026-10-27&academic_year_id=ay_2026_2027');

    // 1. Verify Landing Page Top-Half Search is visible on the Home Dashboard
    const landingSearchPanel = page.getByTestId('landing-page-search-panel');
    await expect(landingSearchPanel).toBeVisible();
    const landingInput = page.getByTestId('landing-search-input');
    await expect(landingInput).toBeVisible();

    // 2. Search by action verb / synonym ("detention") that is only in keywords[], not in title
    await landingInput.fill('detention');
    await expect(page.getByTestId('landing-search-results')).toBeVisible();

    const behViewItem = page.getByTestId(`landing-search-result-view-${behaviourPlugId}-home`);
    await expect(behViewItem).toBeVisible();
    await expect(behViewItem).toContainText('Behaviour & Pastoral Points');

    const behWidgetItem = page.getByTestId('landing-search-result-widget-behaviour_summary');
    await expect(behWidgetItem).toBeVisible();
    await expect(behWidgetItem).toContainText('Behaviour & Merits Summary');

    // 3. Click inline "Add to Dashboard" action on the widget search result
    const addBehWidgetBtn = page.getByTestId('landing-search-add-widget-btn-behaviour_summary');
    await expect(addBehWidgetBtn).toBeVisible();
    await expect(addBehWidgetBtn).toBeEnabled();
    await addBehWidgetBtn.click();

    // Verify widget is placed onto the Home Dashboard Grid and button updates to "Added to Dashboard" (disabled) while search stays open
    await expect(page.getByTestId('placed-widget-card-behaviour_summary')).toBeVisible();
    await expect(addBehWidgetBtn).toBeDisabled();
    await expect(addBehWidgetBtn).toContainText('Added to Dashboard');
    await expect(page.getByTestId('landing-search-results')).toBeVisible();

    // 4. Search by SEN synonym ("ehcp") and click the matched view to navigate to /p/<senPlugId>
    await landingInput.fill('ehcp');
    const senViewItem = page.getByTestId(`landing-search-result-view-${senPlugId}-home`);
    await expect(senViewItem).toBeVisible();
    await senViewItem.click();

    await expect.poll(() => new URL(page.url()).pathname).toContain(`/p/${senPlugId}`);
    await expect(page.getByTestId(`plugin-iframe-${senPlugId}`)).toBeVisible();
  });

  test('globalCommandPaletteCtrlK_scopedPupilGroupAndKeywordSearch_enforcesOwnGroupsIsolationAndDeepLinksToEntityHandler_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // 1. Switch session to Classroom Teacher (holding :own_groups scopes for 7A only)
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: m1Seed.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);

    // 2. Verify backend GET /api/system/search?q=... enforces :own_groups and temporal active dates
    const teacherSearchRes = await request.get(
      '/api/system/search?q=a&reference_date=2026-10-27&academic_year_id=ay_2026_2027'
    );
    expect(teacherSearchRes.ok()).toBe(true);
    const teacherSearchBody = await teacherSearchRes.json();
    expect(teacherSearchBody.ok).toBe(true);

    const teacherPupilIds = (teacherSearchBody.pupils || []).map((p: any) => p.id);
    expect(teacherPupilIds).toContain(m1Seed.active_pupil_id); // Ada Lovelace in 7A
    expect(teacherPupilIds).not.toContain(m1Seed.other_group_pupil_id); // Alan Turing in 7B (:own_groups blocked)
    expect(teacherPupilIds).not.toContain(m1Seed.transferred_pupil_id); // Charles Babbage (off-roll effective_to < reference_date)

    const teacherGroupIds = (teacherSearchBody.groups || []).map((g: any) => g.id);
    expect(teacherGroupIds).toContain(m1Seed.own_group_id);
    expect(teacherGroupIds).not.toContain(m1Seed.other_group_id);

    // 3. Open Host UI as Classroom Teacher and trigger Global Command Palette via Ctrl+K
    await page.goto('/?reference_date=2026-10-27&academic_year_id=ay_2026_2027');
    await expect(page.getByTestId('command-palette-modal')).toHaveCount(0);
    await expect(page.getByTestId('host-search-view')).toHaveCount(0);

    await page.keyboard.press('Control+k');
    const paletteModal = page.getByTestId('command-palette-modal');
    await expect(paletteModal).toBeVisible();
    const paletteInput = page.getByTestId('command-palette-input');
    await expect(paletteInput).toBeFocused();

    // 4. Press Escape to close modal, then click the Narrow Icon Rail Search button -> opens full-page Search view in right-hand pane (/search), NOT a modal
    await page.keyboard.press('Escape');
    await expect(paletteModal).toHaveCount(0);

    await page.getByTestId('nav-rail-search').click();
    expect(new URL(page.url()).pathname).toBe('/search');
    await expect(paletteModal).toHaveCount(0);
    const searchView = page.getByTestId('host-search-view');
    await expect(searchView).toBeVisible();
    await expect(page.getByTestId('landing-page-search-panel')).toBeHidden();
    await expect(page.getByTestId('dashboard-widget-grid-section')).toBeHidden();

    const searchViewInput = page.getByTestId('host-search-input');
    await expect(searchViewInput).toBeVisible();
    const hostSearchTabPlugins = page.getByTestId('host-search-tab-plugins');
    await expect(hostSearchTabPlugins).toBeVisible();
    await expect(hostSearchTabPlugins).toContainText('Pages (');

    // 5. Search for SEN widget via keyword ("differentiation") in full-page Search view and click inline "Add to Dashboard"
    await searchViewInput.fill('differentiation');
    const addSenWidgetBtn = page.getByTestId('host-search-add-widget-btn-sen_differentiation');
    await expect(addSenWidgetBtn).toBeVisible();
    await addSenWidgetBtn.click();
    await expect(addSenWidgetBtn).toBeDisabled();
    await expect(addSenWidgetBtn).toContainText('Added to Dashboard');

    // Switch back to Home view via sidebar Home button to verify widget is placed on Dashboard
    await page.getByTestId('nav-rail-home').click();
    await expect(page.getByTestId('placed-widget-card-sen_differentiation')).toBeVisible();

    // Return to full-page Search view via sidebar Search button
    await page.getByTestId('nav-rail-search').click();
    await expect(searchView).toBeVisible();

    // 6. Search for pupil "Ada" vs "Alan" in full-page Search view as Classroom Teacher (:own_groups)
    await searchViewInput.fill('Alan');
    await expect(page.getByTestId(`host-search-result-pupil-${m1Seed.other_group_pupil_id}`)).toHaveCount(0);

    await searchViewInput.fill('Ada');
    const adaResult = page.getByTestId(`host-search-result-pupil-${m1Seed.active_pupil_id}`);
    await expect(adaResult).toBeVisible();
    await expect(adaResult).toContainText('Ada Lovelace');

    // Clicking Ada Lovelace routes to entity_link_handlers.student (/p/<testPlugId>/student/<active_pupil_id>) and replaces right-hand pane with plugin view
    await adaResult.click();
    await expect(searchView).toHaveCount(0);
    await expect
      .poll(() => new URL(page.url()).pathname)
      .toBe(`/p/${testPlugId}/student/${m1Seed.active_pupil_id}`);
    await expect(page.getByTestId(`plugin-iframe-${testPlugId}`)).toBeVisible();

    // 7. Search for group "10Ma1" via Ctrl+K Command Palette modal and navigate to /p/<testPlugId>/group/<own_group_id>
    await page.keyboard.press('Control+k');
    await expect(paletteModal).toBeVisible();
    await paletteInput.fill('10Ma1');
    const groupResult = page.getByTestId(`command-palette-result-group-${m1Seed.own_group_id}`);
    await expect(groupResult).toBeVisible();
    await expect(groupResult).toContainText('Year 10 Maths Set 1');
    await groupResult.click();
    await expect(paletteModal).toHaveCount(0);
    await expect
      .poll(() => new URL(page.url()).pathname)
      .toBe(`/p/${testPlugId}/group/${m1Seed.own_group_id}`);
  });

  test('fullListButtonAndCommandPaletteTabs_clickingFullListAndSwitchingTabs_filtersViewsWidgetsPupilsAndGroupsByTab_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // 1. Switch session to Classroom Teacher
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: m1Seed.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);

    await page.goto('/?reference_date=2026-10-27&academic_year_id=ay_2026_2027');

    // 2. Verify the button is renamed to "Full list" and opens the modal when clicked
    const fullListBtn = page.getByTestId('landing-open-command-palette-btn');
    await expect(fullListBtn).toBeVisible();
    await expect(fullListBtn).toHaveText('Full list');
    await fullListBtn.click();

    const paletteModal = page.getByTestId('command-palette-modal');
    await expect(paletteModal).toBeVisible();

    // 3. Verify tabs exist: All, Plugins, Widgets, Pupils, Groups (with All selected by default)
    const tabAll = page.getByTestId('command-palette-tab-all');
    const tabPlugins = page.getByTestId('command-palette-tab-plugins');
    const tabWidgets = page.getByTestId('command-palette-tab-widgets');
    const tabPupils = page.getByTestId('command-palette-tab-pupils');
    const tabGroups = page.getByTestId('command-palette-tab-groups');

    await expect(tabAll).toBeVisible();
    await expect(tabPlugins).toBeVisible();
    await expect(tabPlugins).toContainText('Pages (');
    await expect(tabWidgets).toBeVisible();
    await expect(tabPupils).toBeVisible();
    await expect(tabGroups).toBeVisible();
    await expect(tabAll).toHaveAttribute('aria-selected', 'true');

    // Even with an empty search input, All tab shows Views, Widgets, Pupils, and Groups
    const behViewItem = page.getByTestId(`command-palette-result-view-${behaviourPlugId}-home`);
    const behWidgetItem = page.getByTestId('command-palette-result-widget-behaviour_summary');
    const adaPupilItem = page.getByTestId(`command-palette-result-pupil-${m1Seed.active_pupil_id}`);
    const ownGroupItem = page.getByTestId(`command-palette-result-group-${m1Seed.own_group_id}`);

    await expect(behViewItem).toBeVisible();
    await expect(behWidgetItem).toBeVisible();
    await expect(adaPupilItem).toBeVisible();
    await expect(ownGroupItem).toBeVisible();

    // 4. Switch to "Pages" tab -> only plugin views are shown
    await tabPlugins.click();
    await expect(tabPlugins).toHaveAttribute('aria-selected', 'true');
    await expect(behViewItem).toBeVisible();
    await expect(behWidgetItem).toHaveCount(0);
    await expect(adaPupilItem).toHaveCount(0);
    await expect(ownGroupItem).toHaveCount(0);

    // 5. Switch to "Widgets" tab -> only dashboard widgets are shown
    await tabWidgets.click();
    await expect(tabWidgets).toHaveAttribute('aria-selected', 'true');
    await expect(behWidgetItem).toBeVisible();
    await expect(behViewItem).toHaveCount(0);
    await expect(adaPupilItem).toHaveCount(0);
    await expect(ownGroupItem).toHaveCount(0);

    // 6. Switch to "Pupils" tab -> only caller-scoped pupils are shown
    await tabPupils.click();
    await expect(tabPupils).toHaveAttribute('aria-selected', 'true');
    await expect(adaPupilItem).toBeVisible();
    await expect(page.getByTestId(`command-palette-result-pupil-${m1Seed.other_group_pupil_id}`)).toHaveCount(0);
    await expect(behViewItem).toHaveCount(0);
    await expect(behWidgetItem).toHaveCount(0);
    await expect(ownGroupItem).toHaveCount(0);

    // 7. Switch to "Groups" tab -> only caller-scoped groups are shown
    await tabGroups.click();
    await expect(tabGroups).toHaveAttribute('aria-selected', 'true');
    await expect(ownGroupItem).toBeVisible();
    await expect(page.getByTestId(`command-palette-result-group-${m1Seed.other_group_id}`)).toHaveCount(0);
    await expect(behViewItem).toHaveCount(0);
    await expect(behWidgetItem).toHaveCount(0);
    await expect(adaPupilItem).toHaveCount(0);
  });

  test('landingPageSearch_viewPinButtonAndRecordTypeFilter_pinsViewFromSearchResultsAndFiltersByRecordType_test', async ({
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

    await page.goto('/?reference_date=2026-10-27&academic_year_id=ay_2026_2027');

    // 1. Verify record type filter controls exist in the Landing Page Search panel
    const filterAll = page.getByTestId('landing-search-filter-all');
    const filterPlugins = page.getByTestId('landing-search-filter-plugins');
    const filterWidgets = page.getByTestId('landing-search-filter-widgets');
    const filterPupils = page.getByTestId('landing-search-filter-pupils');
    const filterGroups = page.getByTestId('landing-search-filter-groups');

    await expect(filterAll).toBeVisible();
    await expect(filterPlugins).toBeVisible();
    await expect(filterPlugins).toHaveText('Pages');
    await expect(filterWidgets).toBeVisible();
    await expect(filterPupils).toBeVisible();
    await expect(filterGroups).toBeVisible();
    await expect(filterAll).toHaveAttribute('aria-selected', 'true');

    // 2. Search "detention" -> matches both Behaviour view and Behaviour widget
    const landingInput = page.getByTestId('landing-search-input');
    await landingInput.fill('detention');

    const behViewItem = page.getByTestId(`landing-search-result-view-${behaviourPlugId}-home`);
    const behWidgetItem = page.getByTestId('landing-search-result-widget-behaviour_summary');
    await expect(behViewItem).toBeVisible();
    await expect(behWidgetItem).toBeVisible();

    // 3. Verify the returned view item displays a Pin button and clicking it pins/unpins the view to the sidebar
    const behPinBtn = page.getByTestId(`landing-search-pin-view-btn-${behaviourPlugId}-home`);
    await expect(behPinBtn).toBeVisible();
    await expect(behPinBtn).toContainText('Pin');
    await behPinBtn.click();

    await expect(behPinBtn).toContainText('Pinned');
    await expect(page.getByTestId('pinned-recent-item-0')).toBeVisible();
    await expect(page.getByTestId('pinned-recent-item-0')).toHaveAttribute('data-pinned-route', `/p/${behaviourPlugId}`);

    // 4. Filter by record type in Landing Page Search:
    // Switch to "Plugins" filter -> only plugin views are shown, widgets hidden
    await filterPlugins.click();
    await expect(filterPlugins).toHaveAttribute('aria-selected', 'true');
    await expect(behViewItem).toBeVisible();
    await expect(behWidgetItem).toHaveCount(0);

    // Switch to "Widgets" filter -> only widgets are shown, plugin views hidden
    await filterWidgets.click();
    await expect(filterWidgets).toHaveAttribute('aria-selected', 'true');
    await expect(behWidgetItem).toBeVisible();
    await expect(behViewItem).toHaveCount(0);

    // Search "a" (matches Views, Widgets, Pupils, and Groups) and test "Pupils" and "Groups" filters
    await landingInput.fill('a');
    await filterPupils.click();
    await expect(filterPupils).toHaveAttribute('aria-selected', 'true');
    await expect(page.getByTestId(`landing-search-result-pupil-${m1Seed.active_pupil_id}`)).toBeVisible();
    await expect(page.getByTestId(`landing-search-result-group-${m1Seed.own_group_id}`)).toHaveCount(0);
    await expect(behViewItem).toHaveCount(0);

    await filterGroups.click();
    await expect(filterGroups).toHaveAttribute('aria-selected', 'true');
    await expect(page.getByTestId(`landing-search-result-group-${m1Seed.own_group_id}`)).toBeVisible();
    await expect(page.getByTestId(`landing-search-result-pupil-${m1Seed.active_pupil_id}`)).toHaveCount(0);
    await expect(behViewItem).toHaveCount(0);
  });
});
