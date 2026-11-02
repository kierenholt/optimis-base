# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: verticals/staff_cover_plugin_vertical.spec.ts >> Vertical 8: Staff Absence & Dynamic Cover Assignment Plugin (uk.org.optimis.ops.cover) >> assignCoverTeacherAndTakeRegister_coverTeacherInitiallyLacksOwnGroupsAccessThenAssignedCover_grantsDynamicOwnGroupsAccessLoadsRosterAndSubmitsAttendance_test
- Location: tests/e2e/verticals/staff_cover_plugin_vertical.spec.ts:212:7

# Error details

```
Error: apiRequestContext.get: connect ECONNREFUSED 127.0.0.1:18080
Call log:
  - → GET http://127.0.0.1:18080/api/test/logs/cursor
    - user-agent: Playwright/1.64.1 (x64; debian 13) node/20.19
    - accept: */*
    - accept-encoding: gzip,deflate,br

```

```
Error: apiRequestContext.post: connect ECONNREFUSED 127.0.0.1:18080
Call log:
  - → POST http://127.0.0.1:18080/api/test/auth-mode
    - user-agent: Playwright/1.64.1 (x64; debian 13) node/20.19
    - accept: */*
    - accept-encoding: gzip,deflate,br
    - content-type: application/json
    - content-length: 36

```

# Test source

```ts
  86  |     }
  87  |     const personaBody = (await personaRes.json()) as SeededPersonaFixtures;
  88  |     if (!personaBody.fixture_id || !personaBody.admin_staff_id) {
  89  |       throw new Error('Seeded persona fixture response missing mandatory admin_staff_id');
  90  |     }
  91  |     personas = personaBody;
  92  | 
  93  |     // 5. Seed Milestone 1 fixtures (Absent Teacher, Cover Teacher, classes, pupils, and AM/PM timetable slots)
  94  |     const seedRes = await request.post('/api/test/seed-milestone1-fixtures', {
  95  |       data: {
  96  |         academic_year_id: 'ay_2026_2027',
  97  |         reference_date: '2026-10-28',
  98  |         include_second_active_pupil: true,
  99  |       },
  100 |     });
  101 |     if (!seedRes.ok()) {
  102 |       throw new Error(`Failed to seed Milestone 1 fixtures: ${seedRes.status()} ${await seedRes.text()}`);
  103 |     }
  104 |     const seedBody = (await seedRes.json()) as Milestone1CoverSeedResponse;
  105 |     if (
  106 |       !seedBody.fixture_id ||
  107 |       !seedBody.teacher_staff_id ||
  108 |       !seedBody.other_teacher_staff_id ||
  109 |       !seedBody.own_group_id ||
  110 |       !seedBody.other_group_id ||
  111 |       !seedBody.active_pupil_id ||
  112 |       !seedBody.second_active_pupil_id ||
  113 |       !seedBody.transferred_pupil_id ||
  114 |       !seedBody.timetable_slot_am_id
  115 |     ) {
  116 |       throw new Error('Seeded Milestone 1 fixture response missing mandatory staff cover identifiers');
  117 |     }
  118 |     seeded = seedBody;
  119 | 
  120 |     // 6. Runtime-install and activate staff_cover_plugin (uk.org.optimis.ops.cover)
  121 |     const installRes = await request.post('/api/admin/plugins/install', {
  122 |       data: {
  123 |         folder: PLUGIN_FOLDER,
  124 |         staff_id: personas.admin_staff_id,
  125 |       },
  126 |     });
  127 |     if (!installRes.ok()) {
  128 |       throw new Error(
  129 |         `Failed to install ${PLUGIN_FOLDER}: ${installRes.status()} ${await installRes.text()}`
  130 |       );
  131 |     }
  132 |     const installBody = await installRes.json();
  133 |     expect(installBody.ok).toBe(true);
  134 |     expect(installBody.plug_id).toMatch(/^[a-z0-9]{8}$/);
  135 |     expect(installBody.database_alias).toBe(PLUGIN_DB_ALIAS);
  136 |     coverPlugId = installBody.plug_id;
  137 | 
  138 |     const activateRes = await request.post(
  139 |       `/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}/activate`,
  140 |       {
  141 |         data: { staff_id: personas.admin_staff_id },
  142 |       }
  143 |     );
  144 |     if (!activateRes.ok()) {
  145 |       throw new Error(
  146 |         `Failed to activate ${PLUGIN_ID}: ${activateRes.status()} ${await activateRes.text()}`
  147 |       );
  148 |     }
  149 |   });
  150 | 
  151 |   test.afterEach(async ({ request }, testInfo) => {
  152 |     // 1. Purge plugin pupil data created during the test
  153 |     if (personas && seeded && coverPlugId) {
  154 |       await request.post('/api/auth/persona', {
  155 |         data: { staff_id: personas.admin_staff_id },
  156 |       });
  157 |       for (const pupilId of [
  158 |         seeded.active_pupil_id,
  159 |         seeded.second_active_pupil_id,
  160 |         seeded.transferred_pupil_id,
  161 |       ]) {
  162 |         await request.post(`/api/p/${coverPlugId}/rpc/purge_pupil_data`, {
  163 |           data: { args: { pupil_id: pupilId } },
  164 |         });
  165 |       }
  166 |     }
  167 | 
  168 |     // 2. Fetch Backend Service Logs emitted during this test
  169 |     if (logCursor) {
  170 |       const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
  171 |       if (logsRes.ok()) {
  172 |         const serviceLogs: string[] = (await logsRes.json()).entries;
  173 |         if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
  174 |           console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
  175 |         }
  176 |       }
  177 |     }
  178 | 
  179 |     // 3. Dump tapped Browser Console logs on failure or error
  180 |     if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
  181 |       console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
  182 |       console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
  183 |     }
  184 | 
  185 |     // 4. Remove runtime-installed plugin
> 186 |     await request.post('/api/test/auth-mode', {
      |                   ^ Error: apiRequestContext.post: connect ECONNREFUSED 127.0.0.1:18080
  187 |       data: { enable_dev_persona_switcher: true },
  188 |     });
  189 |     await request.delete(`/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}`);
  190 | 
  191 |     // 5. Clean up seeded fixtures in core.db
  192 |     if (seeded?.fixture_id) {
  193 |       const delM1 = await request.delete(
  194 |         `/api/test/seed-milestone1-fixtures/${encodeURIComponent(seeded.fixture_id)}`
  195 |       );
  196 |       if (!delM1.ok()) {
  197 |         throw new Error(`Failed to clean up Milestone 1 fixture ${seeded.fixture_id}: ${delM1.status()}`);
  198 |       }
  199 |     }
  200 |     if (personas?.fixture_id) {
  201 |       const delPersonas = await request.delete(
  202 |         `/api/test/seed-personas/${encodeURIComponent(personas.fixture_id)}`
  203 |       );
  204 |       if (!delPersonas.ok()) {
  205 |         throw new Error(`Failed to clean up persona fixture ${personas.fixture_id}: ${delPersonas.status()}`);
  206 |       }
  207 |     }
  208 | 
  209 |     expect(pageErrors).toEqual([]);
  210 |   });
  211 | 
  212 |   test('assignCoverTeacherAndTakeRegister_coverTeacherInitiallyLacksOwnGroupsAccessThenAssignedCover_grantsDynamicOwnGroupsAccessLoadsRosterAndSubmitsAttendance_test', async ({
  213 |     page,
  214 |     request,
  215 |   }) => {
  216 |     if (!personas || !seeded) {
  217 |       throw new Error('Missing mandatory seeded fixture state');
  218 |     }
  219 | 
  220 |     // =========================================================================
  221 |     // Step 1: Cover Teacher initially lacks :own_groups access to Absent Teacher's class (own_group_id)
  222 |     // =========================================================================
  223 |     const switchCoverRes = await request.post('/api/auth/persona', {
  224 |       data: { staff_id: seeded.other_teacher_staff_id },
  225 |     });
  226 |     expect(switchCoverRes.ok()).toBe(true);
  227 |     const coverSessionBody = await switchCoverRes.json();
  228 |     expect(coverSessionBody.ok).toBe(true);
  229 | 
  230 |     await page.context().addCookies([
  231 |       {
  232 |         name: 'mis_session',
  233 |         value: coverSessionBody.session_token,
  234 |         url: 'http://127.0.0.1:18080',
  235 |         sameSite: 'Strict',
  236 |       },
  237 |     ]);
  238 | 
  239 |     const params = new URLSearchParams({
  240 |       pupil_id: seeded.active_pupil_id,
  241 |       group_id: seeded.own_group_id,
  242 |       absent_staff_id: seeded.teacher_staff_id,
  243 |       cover_staff_id: seeded.other_teacher_staff_id,
  244 |       slot_id: seeded.timetable_slot_am_id,
  245 |       reference_date: seeded.reference_date,
  246 |       academic_year_id: seeded.academic_year_id,
  247 |     });
  248 |     await page.goto(`/?${params.toString()}`);
  249 | 
  250 |     await expect(page.getByTestId('host-shell')).toBeVisible();
  251 | 
  252 |     await page.getByTestId('nav-rail-search').click();
  253 |     const navItem = page.locator('[data-testid^="host-search-result-view-"][data-testid$="-home"]').filter({ hasText: 'Cover' }).first();
  254 |     await expect(navItem).toBeVisible();
  255 |     await navItem.click();
  256 | 
  257 |     const iframeEl = page.getByTestId(`plugin-iframe-${coverPlugId}`);
  258 |     await expect(iframeEl).toBeVisible();
  259 |     await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');
  260 | 
  261 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${coverPlugId}"]`);
  262 |     await expect(pluginFrame.getByTestId('staff-cover-root')).toBeVisible();
  263 | 
  264 |     const loadCoverRosterBtn = pluginFrame.getByTestId('load-cover-roster-btn');
  265 |     await expect(loadCoverRosterBtn).toBeVisible();
  266 |     await expect(loadCoverRosterBtn).toHaveAttribute('data-mis-rpc', 'load_cover_class_roster');
  267 |     await loadCoverRosterBtn.click();
  268 | 
  269 |     const coverErrorBanner = pluginFrame.getByTestId('cover-error-banner');
  270 |     await expect(coverErrorBanner).toBeVisible();
  271 |     await expect(coverErrorBanner).toContainText('access denied under own_groups scope');
  272 | 
  273 |     // Also verify via direct RPC that submitting attendance for own_group_id as Cover Teacher fails before cover assignment
  274 |     const preCoverSubmitRes = await request.post(
  275 |       `/api/p/${coverPlugId}/rpc/submit_cover_attendance_batch`,
  276 |       {
  277 |         data: {
  278 |           args: {
  279 |             group_id: seeded.own_group_id,
  280 |             date: seeded.reference_date,
  281 |             session: 'AM',
  282 |             academic_year_id: seeded.academic_year_id,
  283 |             marks: [{ pupil_id: seeded.active_pupil_id, attendance_code: '/' }],
  284 |           },
  285 |         },
  286 |       }
```