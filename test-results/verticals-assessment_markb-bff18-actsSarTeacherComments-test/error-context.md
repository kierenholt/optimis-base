# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: verticals/assessment_markbook_plugin_vertical.spec.ts >> Assessment & Markbook Plugin Vertical Slice: MAT Grading Schema Sync, Teacher (:own_groups) vs HoD (:all) Attainment Rollup & SAR Teacher Comment Redaction (uk.org.optimis.assessment.markbook) >> markbookLifecycleAndCrossSetRollup_pullMatSchemaTeacherSavesOwnSetGradesSwitchesToHodViewsCrossSetRollupAndAdminRedactsSarTeacherComments_test
- Location: tests/e2e/verticals/assessment_markbook_plugin_vertical.spec.ts:226:7

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Markbook' }).first()
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Markbook' }).first() with timeout 5000ms
  - waiting for locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Markbook' }).first()

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
  - region "Global Search and Discovery":
    - heading "Global Search & Discovery" [level=2]
    - text: Search plugin views, dashboard widgets, pupils, and teaching groups across your school workspace
    - searchbox "Type a command, synonym, pupil name, or group code..."
    - tablist "Search category filter tabs":
      - tab "All (15)" [selected]
      - tab "Pages (12)"
      - tab "Widgets (0)"
      - tab "Pupils (2)"
      - tab "Groups (1)"
    - text: Views (12)
    - button "Department Assessment & Markbook Department Assessment & Markbook • /p/dkifq1it (3 methods) Open View"
    - button "Pin Department Assessment & Markbook": Pin
    - button "Class Markbook Department Assessment & Markbook • /p/dkifq1it/group/:group_id (3 methods) Open View"
    - button "Pin Class Markbook": Pin
    - button "Educational Visits & Trip Consent Educational Visits & Trip Consent • /p/331qo66q (4 methods) Open View"
    - button "Pin Educational Visits & Trip Consent": Pin
    - button "Group Trip Consent & Manifest Educational Visits & Trip Consent • /p/331qo66q/group/:group_id (4 methods) Open View"
    - button "Pin Group Trip Consent & Manifest": Pin
    - button "Google Classroom LMS Sync Google Classroom LMS Sync • /p/k252w5cv (1 methods) Open View"
    - button "Pin Google Classroom LMS Sync": Pin
    - button "Course Roster Sync Google Classroom LMS Sync • /p/k252w5cv/group/:group_id (1 methods) Open View"
    - button "Pin Course Roster Sync": Pin
    - button "Sparx Maths Partner Sync Sparx Maths Partner Sync • /p/x8ijanmh (3 methods) Open View"
    - button "Pin Sparx Maths Partner Sync": Pin
    - button "Group Homework Sync Sparx Maths Partner Sync • /p/x8ijanmh/group/:group_id (3 methods) Open View"
    - button "Pin Group Homework Sync": Pin
    - button "Staff Absence & Cover Manager Staff Absence & Cover Manager • /p/kj7wyoj1 (3 methods) Open View"
    - button "Pin Staff Absence & Cover Manager": Pin
    - button "Cover Class Roster Staff Absence & Cover Manager • /p/kj7wyoj1/group/:group_id (3 methods) Open View"
    - button "Pin Cover Class Roster": Pin
    - button "Timetable & Statutory Attendance Timetable & Statutory Attendance • /p/e454uy6j (3 methods) Open View"
    - button "Pin Timetable & Statutory Attendance": Pin
    - button "Class Session Register Timetable & Statutory Attendance • /p/e454uy6j/group/:group_id (3 methods) Open View"
    - button "Pin Class Session Register": Pin
    - text: Pupils (2)
    - button "Ada Lovelace UPN U1715A1327001 • Year 10 Open Student"
    - button "Grace Turing UPN U4715A1327004 • Year 10 Open Student"
    - text: Groups (1)
    - button "10Ma1_715a — Year 10 Maths Set 1 TEACHING_CLASS Open Group"
```

# Test source

```ts
  187 |         if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
  188 |           console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
  189 |         }
  190 |       }
  191 |     }
  192 | 
  193 |     // 4. Dump tapped Browser Console logs on failure or error
  194 |     if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
  195 |       console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
  196 |       console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
  197 |     }
  198 | 
  199 |     // 5. Remove runtime-installed plugin
  200 |     await request.post('/api/test/auth-mode', {
  201 |       data: { enable_dev_persona_switcher: true },
  202 |     });
  203 |     await request.delete('/api/admin/plugins/uk.org.optimis.assessment.markbook');
  204 | 
  205 |     // 6. Clean up seeded fixtures in core.db
  206 |     if (m1Seeded?.fixture_id) {
  207 |       const delM1 = await request.delete(
  208 |         `/api/test/seed-milestone1-fixtures/${encodeURIComponent(m1Seeded.fixture_id)}`
  209 |       );
  210 |       if (!delM1.ok()) {
  211 |         throw new Error(`Failed to clean up Milestone 1 fixture ${m1Seeded.fixture_id}: ${delM1.status()}`);
  212 |       }
  213 |     }
  214 |     if (personas?.fixture_id) {
  215 |       const delP = await request.delete(
  216 |         `/api/test/seed-personas/${encodeURIComponent(personas.fixture_id)}`
  217 |       );
  218 |       if (!delP.ok()) {
  219 |         throw new Error(`Failed to clean up persona fixture ${personas.fixture_id}: ${delP.status()}`);
  220 |       }
  221 |     }
  222 | 
  223 |     expect(pageErrors).toEqual([]);
  224 |   });
  225 | 
  226 |   test('markbookLifecycleAndCrossSetRollup_pullMatSchemaTeacherSavesOwnSetGradesSwitchesToHodViewsCrossSetRollupAndAdminRedactsSarTeacherComments_test', async ({
  227 |     page,
  228 |     request,
  229 |   }) => {
  230 |     if (!personas || !m1Seeded) {
  231 |       throw new Error('Missing mandatory seeded fixtures');
  232 |     }
  233 | 
  234 |     // 1. Seed Mock MAT Central Sync Feed and pull MAT grading schema into `__system_config`
  235 |     const seedMatRes = await request.post('/api/test/mat/seed-feed', {
  236 |       data: {
  237 |         mat_id: 'MAT_OPTIMIS_TRUST_01',
  238 |         grading_schema_version: 'KS4_9_TO_1_AND_PRIMARY_WT_EXS_GDS_V2026',
  239 |         behaviour_taxonomy: ['MERIT_EXCELLENCE', 'RESTORATIVE_CONVERSATION'],
  240 |         approved_plugins: [{ id: 'uk.org.optimis.assessment.markbook', version: '0.1.0' }],
  241 |       },
  242 |     });
  243 |     expect(seedMatRes.status()).toBe(201);
  244 | 
  245 |     const pullPoliciesRes = await request.post('/api/sync/mat/pull-policies', {
  246 |       data: { staff_id: personas.admin_staff_id },
  247 |     });
  248 |     expect(pullPoliciesRes.status()).toBe(200);
  249 |     const pullPoliciesBody = await pullPoliciesRes.json();
  250 |     expect(pullPoliciesBody.ok).toBe(true);
  251 |     expect(pullPoliciesBody.policy_bundle.grading_schema_version).toBe(
  252 |       'KS4_9_TO_1_AND_PRIMARY_WT_EXS_GDS_V2026'
  253 |     );
  254 | 
  255 |     // 2. Switch to Classroom Teacher (`m1Seeded.teacher_staff_id`, holding `:own_groups` for `own_group_id` Year 10 Maths Set 1)
  256 |     const switchTeacherRes = await request.post('/api/auth/persona', {
  257 |       data: { staff_id: m1Seeded.teacher_staff_id },
  258 |     });
  259 |     expect(switchTeacherRes.ok()).toBe(true);
  260 |     const teacherSession = await switchTeacherRes.json();
  261 |     expect(teacherSession.caller.active_roles).toContain('TEACHER');
  262 |     expect(teacherSession.caller.scopes).toContain('dal.groups.read:own_groups');
  263 |     expect(teacherSession.caller.scopes).toContain('dal.pupils.read:own_groups');
  264 |     expect(teacherSession.caller.scopes).not.toContain('dal.groups.read:all');
  265 | 
  266 |     await page.context().addCookies([
  267 |       {
  268 |         name: 'mis_session',
  269 |         value: teacherSession.session_token,
  270 |         url: 'http://127.0.0.1:18080',
  271 |         sameSite: 'Strict',
  272 |       },
  273 |     ]);
  274 | 
  275 |     const query = new URLSearchParams({
  276 |       pupil_id: m1Seeded.active_pupil_id,
  277 |       group_id: m1Seeded.own_group_id,
  278 |       other_group_id: m1Seeded.other_group_id,
  279 |       reference_date: m1Seeded.reference_date,
  280 |       academic_year_id: m1Seeded.academic_year_id,
  281 |     });
  282 |     await page.goto(`/?${query.toString()}`);
  283 | 
  284 |     await expect(page.getByTestId('host-shell')).toBeVisible();
  285 |     await page.getByTestId('nav-rail-search').click();
  286 |     const markbookNavBtn = page.locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Markbook' }).first();
> 287 |     await expect(markbookNavBtn).toBeVisible();
      |                                  ^ Error: expect(locator).toBeVisible() failed
  288 |     await markbookNavBtn.click();
  289 | 
  290 |     const iframeEl = page.getByTestId(`plugin-iframe-${markbookPlugId}`);
  291 |     await expect(iframeEl).toBeVisible();
  292 |     await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');
  293 | 
  294 |     const pluginFrame = page.frameLocator(
  295 |       `[data-testid="plugin-iframe-${markbookPlugId}"]`
  296 |     );
  297 |     await expect(pluginFrame.getByTestId('markbook-plugin-root')).toBeVisible();
  298 | 
  299 |     // Verify MAT grading schema version synced into `__system_config` is surfaced in the markbook workspace
  300 |     await expect(pluginFrame.getByTestId('markbook-mat-schema-badge')).toHaveText(
  301 |       'KS4_9_TO_1_AND_PRIMARY_WT_EXS_GDS_V2026'
  302 |     );
  303 | 
  304 |     // Teacher (:own_groups) creates a markbook column for their own set (`own_group_id` Year 10 Maths Set 1)
  305 |     const colTitleInput = pluginFrame.getByTestId('markbook-column-title-input');
  306 |     const colScaleSelect = pluginFrame.getByTestId('markbook-column-scale-select');
  307 |     const createColBtn = pluginFrame.getByTestId('create-markbook-column-btn');
  308 | 
  309 |     await expect(createColBtn).toBeVisible();
  310 |     await expect(createColBtn).toHaveAttribute('data-mis-rpc', 'create_markbook_column');
  311 | 
  312 |     await colTitleInput.fill('Autumn Half-Term Algebra Assessment');
  313 |     await colScaleSelect.selectOption('9-1');
  314 |     await createColBtn.click();
  315 | 
  316 |     await expect(pluginFrame.getByTestId('markbook-active-column-title')).toContainText(
  317 |       'Autumn Half-Term Algebra Assessment'
  318 |     );
  319 |     await expect(pluginFrame.getByTestId('markbook-active-column-scale')).toHaveText('9-1');
  320 | 
  321 |     // Teacher enters grade '9' and a teacher comment containing a third-party pupil name ("Neville Longbottom") for Ada Lovelace (`active_pupil_id`)
  322 |     const gradeSelect = pluginFrame.getByTestId(`markbook-grade-select-${m1Seeded.active_pupil_id}`);
  323 |     const commentInput = pluginFrame.getByTestId(
  324 |       `markbook-comment-input-${m1Seeded.active_pupil_id}`
  325 |     );
  326 |     const saveGradesBtn = pluginFrame.getByTestId('save-pupil-grades-btn');
  327 | 
  328 |     await expect(gradeSelect).toBeVisible();
  329 |     await expect(saveGradesBtn).toBeVisible();
  330 |     await expect(saveGradesBtn).toHaveAttribute('data-mis-rpc', 'save_pupil_grades');
  331 | 
  332 |     await gradeSelect.selectOption('9');
  333 |     await commentInput.fill(
  334 |       'Scored 96% on higher-tier quadratics paper; peer-checked extension task with third-party pupil Neville Longbottom.'
  335 |     );
  336 |     await saveGradesBtn.click();
  337 | 
  338 |     await expect(pluginFrame.getByTestId('markbook-saved-count-badge')).toContainText('1');
  339 |     await expect(
  340 |       pluginFrame.getByTestId(`markbook-saved-grade-${m1Seeded.active_pupil_id}`)
  341 |     ).toHaveText('9');
  342 | 
  343 |     // Verify Teacher (:own_groups) is blocked from Set 2 (`other_group_id`) via UI / RPC
  344 |     const groupSelect = pluginFrame.getByTestId('markbook-group-select');
  345 |     await groupSelect.selectOption(m1Seeded.other_group_id);
  346 |     await createColBtn.click();
  347 | 
  348 |     const errorBanner = pluginFrame.getByTestId('markbook-error-banner');
  349 |     await expect(errorBanner).toBeVisible();
  350 |     await expect(errorBanner).toContainText('access denied under own_groups scope');
  351 | 
  352 |     // 3. Switch to Head of Department (`personas.hod_staff_id`, holding `:all` scopes) via POST /api/auth/persona
  353 |     const hodSwitchRes = await request.post('/api/auth/persona', {
  354 |       data: { staff_id: personas.hod_staff_id },
  355 |     });
  356 |     expect(hodSwitchRes.ok()).toBe(true);
  357 |     const hodSession = await hodSwitchRes.json();
  358 |     await page.context().addCookies([
  359 |       {
  360 |         name: 'mis_session',
  361 |         value: hodSession.session_token,
  362 |         url: 'http://127.0.0.1:18080',
  363 |         sameSite: 'Strict',
  364 |       },
  365 |     ]);
  366 |     await page.goto(`/p/${markbookPlugId}?${query.toString()}`);
  367 | 
  368 |     await expect(pluginFrame.getByTestId('markbook-caller-role-badge')).toContainText('HOD');
  369 | 
  370 |     // As HoD (`:all`), select Set 2 (`other_group_id`), create a markbook column, and save grade '6' for Alan Turing (`other_group_pupil_id`)
  371 |     await groupSelect.selectOption(m1Seeded.other_group_id);
  372 |     await colTitleInput.fill('Autumn Half-Term Algebra Assessment');
  373 |     await colScaleSelect.selectOption('9-1');
  374 |     await createColBtn.click();
  375 | 
  376 |     await expect(
  377 |       pluginFrame.getByTestId(`markbook-grade-select-${m1Seeded.other_group_pupil_id}`)
  378 |     ).toBeVisible();
  379 |     await pluginFrame
  380 |       .getByTestId(`markbook-grade-select-${m1Seeded.other_group_pupil_id}`)
  381 |       .selectOption('6');
  382 |     await pluginFrame
  383 |       .getByTestId(`markbook-comment-input-${m1Seeded.other_group_pupil_id}`)
  384 |       .fill('Solid foundation algebra performance in Set 2.');
  385 |     await saveGradesBtn.click();
  386 | 
  387 |     await expect(
```