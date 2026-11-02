# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: core/milestone1_core_dal_scoping.spec.ts >> Milestone 1: Coarse-Grained Core DAL Domain Methods & Dynamic :own_groups Scoping >> getGroupRosterSummary_teacherWithOwnGroupsScope_returnsOnlyActiveOwnGroupPupilsAndExcludesTransferred_test
- Location: tests/e2e/core/milestone1_core_dal_scoping.spec.ts:146:7

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator:  getByTestId('plugin-iframe-uvxcor1o')
Expected: visible
Received: hidden
Timeout:  5000ms

Call log:
  - Expect "toBeVisible" getByTestId('plugin-iframe-uvxcor1o') with timeout 5000ms
  - waiting for getByTestId('plugin-iframe-uvxcor1o')
    14 × locator resolved to <iframe data-lru-active="false" sandbox="allow-scripts" src="/p/uvxcor1o/ui/index.html" title="Plugin Workspace: uvxcor1o" data-testid="plugin-iframe-uvxcor1o"></iframe>
       - unexpected value "hidden"

```

```yaml
- complementary:
  - text: URN 100001
  - heading "Oakridge All-Through Academy" [level=1]
  - text: ay_2026_2027
  - navigation "Primary Icon Rail":
    - button "Home"
    - button "Search"
  - text: Pinned
  - button "Settings"
  - button "Logout"
- main:
  - heading "Global Search & Discovery" [level=2]
  - text: Search plugin views, dashboard widgets, pupils, and teaching groups by name, action verb, or synonym
  - button "Full list"
  - tablist "Search record type filter":
    - tab "All" [selected]
    - tab "Pages"
    - tab "Widgets"
    - tab "Pupils"
    - tab "Groups"
  - searchbox "Search views, widgets, pupils, or groups (e.g. detention, ehcp, register, Ada Lovelace)..."
  - heading "Home Dashboard Widgets" [level=2]
  - button "Edit Dashboard Grid"
  - text: Statutory Attendance Pulse
  - iframe
```

# Test source

```ts
  59  |     if (!installRes.ok()) {
  60  |       throw new Error(`Failed to install test_plugin: ${installRes.status()}`);
  61  |     }
  62  |     testPlugId = (await installRes.json()).plug_id;
  63  |     const activateRes = await request.post('/api/admin/plugins/uk.org.optimis.test/activate', {
  64  |       data: {},
  65  |     });
  66  |     if (!activateRes.ok()) {
  67  |       throw new Error(`Failed to activate uk.org.optimis.test: ${activateRes.status()}`);
  68  |     }
  69  | 
  70  |     // 4. Seed isolated Milestone 1 fixtures explicitly in setup
  71  |     const seedRes = await request.post('/api/test/seed-milestone1-fixtures', {
  72  |       data: {
  73  |         academic_year_id: 'ay_2026_2027',
  74  |         reference_date: '2026-10-27',
  75  |       },
  76  |     });
  77  |     if (!seedRes.ok()) {
  78  |       throw new Error(`Failed to seed Milestone 1 fixtures: ${seedRes.status()}`);
  79  |     }
  80  |     const body = (await seedRes.json()) as Milestone1SeedResponse;
  81  |     if (
  82  |       !body.fixture_id ||
  83  |       !body.own_group_id ||
  84  |       !body.other_group_id ||
  85  |       !body.active_pupil_id ||
  86  |       !body.transferred_pupil_id ||
  87  |       !body.other_group_pupil_id ||
  88  |       !body.teacher_staff_id
  89  |     ) {
  90  |       throw new Error('Seeded Milestone 1 response missing mandatory fixture identifiers');
  91  |     }
  92  |     seeded = body;
  93  |     await request.post('/api/auth/persona', {
  94  |       data: { staff_id: seeded.teacher_staff_id },
  95  |     });
  96  |   });
  97  | 
  98  |   test.afterEach(async ({ request }, testInfo) => {
  99  |     // 1. Fetch Backend Service Logs emitted during this test
  100 |     if (logCursor) {
  101 |       const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
  102 |       if (logsRes.ok()) {
  103 |         const serviceLogs: string[] = (await logsRes.json()).entries;
  104 |         if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
  105 |           console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
  106 |         }
  107 |       }
  108 |     }
  109 | 
  110 |     // 2. Dump tapped Browser Console logs on failure or error
  111 |     if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
  112 |       console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
  113 |       console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
  114 |     }
  115 | 
  116 |     // 3. Remove runtime-installed plugin
  117 |     await request.post('/api/test/auth-mode', {
  118 |       data: { enable_dev_persona_switcher: true },
  119 |     });
  120 |     await request.delete('/api/admin/plugins/uk.org.optimis.test');
  121 | 
  122 |     // 4. Clean up seeded Milestone 1 test data at the end of the test
  123 |     if (seeded?.fixture_id) {
  124 |       const delRes = await request.delete(
  125 |         `/api/test/seed-milestone1-fixtures/${encodeURIComponent(seeded.fixture_id)}`
  126 |       );
  127 |       if (!delRes.ok()) {
  128 |         throw new Error(`Failed to clean up Milestone 1 fixture ${seeded.fixture_id}: ${delRes.status()}`);
  129 |       }
  130 |     }
  131 | 
  132 |     expect(pageErrors).toEqual([]);
  133 |   });
  134 | 
  135 |   function buildWorkspaceUrl(f: Milestone1SeedResponse): string {
  136 |     const params = new URLSearchParams({
  137 |       pupil_id: f.active_pupil_id,
  138 |       group_id: f.own_group_id,
  139 |       other_group_id: f.other_group_id,
  140 |       reference_date: f.reference_date,
  141 |       academic_year_id: f.academic_year_id,
  142 |     });
  143 |     return `/?${params.toString()}`;
  144 |   }
  145 | 
  146 |   test('getGroupRosterSummary_teacherWithOwnGroupsScope_returnsOnlyActiveOwnGroupPupilsAndExcludesTransferred_test', async ({
  147 |     page,
  148 |   }) => {
  149 |     if (!seeded) {
  150 |       throw new Error('Missing mandatory seeded fixture state');
  151 |     }
  152 | 
  153 |     await page.goto(buildWorkspaceUrl(seeded));
  154 | 
  155 |     const hostShell = page.getByTestId('host-shell');
  156 |     await expect(hostShell).toBeVisible();
  157 | 
  158 |     const iframeEl = page.getByTestId(`plugin-iframe-${testPlugId}`);
> 159 |     await expect(iframeEl).toBeVisible();
      |                            ^ Error: expect(locator).toBeVisible() failed
  160 |     await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');
  161 | 
  162 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
  163 |     const loadRosterBtn = pluginFrame.getByTestId('load-own-group-roster-btn');
  164 |     await expect(loadRosterBtn).toBeVisible();
  165 |     await expect(loadRosterBtn).toHaveAttribute('data-mis-rpc', 'get_group_roster_summary');
  166 | 
  167 |     await loadRosterBtn.click();
  168 | 
  169 |     const rosterResult = pluginFrame.getByTestId('roster-result');
  170 |     await expect(rosterResult).toBeVisible();
  171 |     await expect(rosterResult).toContainText(seeded.active_pupil_id);
  172 |     await expect(rosterResult).toContainText('Ada Lovelace');
  173 |     await expect(rosterResult).toContainText('E');
  174 |     await expect(rosterResult).toContainText('Anaphylaxis (Peanut Allergy)');
  175 | 
  176 |     // Transferred pupil (effective_to < reference_date) and other teacher's pupil MUST NOT appear
  177 |     const rosterText = await rosterResult.textContent();
  178 |     expect(rosterText).not.toContain(seeded.transferred_pupil_id);
  179 |     expect(rosterText).not.toContain(seeded.other_group_pupil_id);
  180 |   });
  181 | 
  182 |   test('getGroupRosterSummary_teacherWithOwnGroupsScopeRequestingOtherTeachersGroup_returnsAccessDeniedError_test', async ({
  183 |     page,
  184 |   }) => {
  185 |     if (!seeded) {
  186 |       throw new Error('Missing mandatory seeded fixture state');
  187 |     }
  188 | 
  189 |     await page.goto(buildWorkspaceUrl(seeded));
  190 | 
  191 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
  192 |     const loadOtherGroupBtn = pluginFrame.getByTestId('load-other-group-roster-btn');
  193 |     await expect(loadOtherGroupBtn).toBeVisible();
  194 |     await expect(loadOtherGroupBtn).toHaveAttribute('data-mis-rpc', 'get_group_roster_summary');
  195 | 
  196 |     await loadOtherGroupBtn.click();
  197 | 
  198 |     const errorBanner = pluginFrame.getByTestId('rpc-error-banner');
  199 |     await expect(errorBanner).toBeVisible();
  200 |     await expect(errorBanner).toContainText('own_groups');
  201 | 
  202 |     const errorBannerText = await errorBanner.textContent();
  203 |     expect(errorBannerText).not.toContain(seeded.other_group_pupil_id);
  204 |   });
  205 | 
  206 |   test('getPupilGuardians_pupilWithCourtOrderRestrictedFatherAndConfidentialHousehold_returnsOnlyUnrestrictedPRGuardianAndSafeAddress_test', async ({
  207 |     page,
  208 |   }) => {
  209 |     if (!seeded) {
  210 |       throw new Error('Missing mandatory seeded fixture state');
  211 |     }
  212 | 
  213 |     await page.goto(buildWorkspaceUrl(seeded));
  214 | 
  215 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
  216 |     const loadGuardiansBtn = pluginFrame.getByTestId('load-pupil-guardians-btn');
  217 |     await expect(loadGuardiansBtn).toBeVisible();
  218 |     await expect(loadGuardiansBtn).toHaveAttribute('data-mis-rpc', 'get_pupil_guardians');
  219 | 
  220 |     await loadGuardiansBtn.click();
  221 | 
  222 |     const guardiansResult = pluginFrame.getByTestId('guardians-result');
  223 |     await expect(guardiansResult).toBeVisible();
  224 |     await expect(guardiansResult).toContainText('Anne Byron');
  225 |     await expect(guardiansResult).toContainText('MOTHER');
  226 |     await expect(guardiansResult).toContainText('42 High Street');
  227 | 
  228 |     // Court-order-restricted father and confidential refuge address MUST NEVER leak to UI
  229 |     const guardiansText = await guardiansResult.textContent();
  230 |     expect(guardiansText).not.toContain('George Byron');
  231 |     expect(guardiansText).not.toContain(seeded.restricted_father_person_id);
  232 |     expect(guardiansText).not.toContain('Prohibited Steps Order');
  233 |     expect(guardiansText).not.toContain('10 Safe Haven Way');
  234 |     expect(guardiansText).not.toContain('SW1A 1AA');
  235 |   });
  236 | 
  237 |   test('recordGroupAmAttendance_validAmMarksForOwnGroup_persistsStatutoryAmMarksAndEmitsEvent_test', async ({
  238 |     page,
  239 |   }) => {
  240 |     if (!seeded) {
  241 |       throw new Error('Missing mandatory seeded fixture state');
  242 |     }
  243 | 
  244 |     await page.goto(buildWorkspaceUrl(seeded));
  245 | 
  246 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
  247 |     const recordAttendanceBtn = pluginFrame.getByTestId('record-am-attendance-btn');
  248 |     await expect(recordAttendanceBtn).toBeVisible();
  249 |     await expect(recordAttendanceBtn).toHaveAttribute('data-mis-rpc', 'record_group_am_attendance');
  250 | 
  251 |     await recordAttendanceBtn.click();
  252 | 
  253 |     const attendanceResult = pluginFrame.getByTestId('attendance-result');
  254 |     await expect(attendanceResult).toBeVisible();
  255 |     await expect(attendanceResult).toContainText('SESSION_SAVED:AM');
  256 |     await expect(attendanceResult).toContainText(seeded.active_pupil_id);
  257 |     await expect(attendanceResult).toContainText('attendance.session_marks_recorded');
  258 |   });
  259 | 
```