# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: verticals/staff_cover_plugin_vertical.spec.ts >> Vertical 8: Staff Absence & Dynamic Cover Assignment Plugin (uk.org.optimis.ops.cover) >> assignCoverTeacherAndTakeRegister_coverTeacherInitiallyLacksOwnGroupsAccessThenAssignedCover_grantsDynamicOwnGroupsAccessLoadsRosterAndSubmitsAttendance_test
- Location: tests/e2e/verticals/staff_cover_plugin_vertical.spec.ts:212:7

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Cover' }).first()
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Cover' }).first() with timeout 5000ms
  - waiting for locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Cover' }).first()

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
      - tab "All (9)" [selected]
      - tab "Pages (6)"
      - tab "Widgets (0)"
      - tab "Pupils (2)"
      - tab "Groups (1)"
    - text: Views (6)
    - button "Educational Visits & Trip Consent Educational Visits & Trip Consent • /p/331qo66q (4 methods) Open View"
    - button "Pin Educational Visits & Trip Consent": Pin
    - button "Group Trip Consent & Manifest Educational Visits & Trip Consent • /p/331qo66q/group/:group_id (4 methods) Open View"
    - button "Pin Group Trip Consent & Manifest": Pin
    - button "Staff Absence & Cover Manager Staff Absence & Cover Manager • /p/kj7wyoj1 (3 methods) Open View"
    - button "Pin Staff Absence & Cover Manager": Pin
    - button "Cover Class Roster Staff Absence & Cover Manager • /p/kj7wyoj1/group/:group_id (3 methods) Open View"
    - button "Pin Cover Class Roster": Pin
    - button "Timetable & Statutory Attendance Timetable & Statutory Attendance • /p/e454uy6j (3 methods) Open View"
    - button "Pin Timetable & Statutory Attendance": Pin
    - button "Class Session Register Timetable & Statutory Attendance • /p/e454uy6j/group/:group_id (3 methods) Open View"
    - button "Pin Class Session Register": Pin
    - text: Pupils (2)
    - button "Charles Babbage UPN U2EAE79AB5002 • Year 10 Open Student"
    - button "Alan Turing UPN U3EAE79AB5003 • Year 10 Open Student"
    - text: Groups (1)
    - button "10Ma2_eae7 — Year 10 Maths Set 2 TEACHING_CLASS Open Group"
```

# Test source

```ts
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
  186 |     await request.post('/api/test/auth-mode', {
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
  253 |     const navItem = page.locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Cover' }).first();
> 254 |     await expect(navItem).toBeVisible();
      |                           ^ Error: expect(locator).toBeVisible() failed
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
  287 |     );
  288 |     expect(preCoverSubmitRes.status()).toBe(400);
  289 |     const preCoverSubmitBody = await preCoverSubmitRes.json();
  290 |     expect(preCoverSubmitBody.ok).toBe(false);
  291 |     expect(preCoverSubmitBody.error_code).toContain('access denied under own_groups scope');
  292 | 
  293 |     // =========================================================================
  294 |     // Step 2: Cover Manager (personas.admin_staff_id) records teacher absence and assigns Cover Teacher to Absent Teacher's timetable slot
  295 |     // =========================================================================
  296 |     const switchAdminRes = await request.post('/api/auth/persona', {
  297 |       data: { staff_id: personas.admin_staff_id },
  298 |     });
  299 |     expect(switchAdminRes.ok()).toBe(true);
  300 |     const adminSessionBody = await switchAdminRes.json();
  301 |     await page.context().addCookies([
  302 |       {
  303 |         name: 'mis_session',
  304 |         value: adminSessionBody.session_token,
  305 |         url: 'http://127.0.0.1:18080',
  306 |         sameSite: 'Strict',
  307 |       },
  308 |     ]);
  309 |     await page.goto(`/p/${coverPlugId}?${params.toString()}`);
  310 |     await expect(pluginFrame.getByTestId('cover-active-persona-badge')).toContainText(personas.admin_staff_id);
  311 | 
  312 |     const loadAbsentTimetableBtn = pluginFrame.getByTestId('load-absent-timetable-btn');
  313 |     await expect(loadAbsentTimetableBtn).toBeVisible();
  314 |     await expect(loadAbsentTimetableBtn).toHaveAttribute('data-mis-rpc', 'load_absent_teacher_timetable');
  315 |     await loadAbsentTimetableBtn.click();
  316 | 
  317 |     const absentSlotCard = pluginFrame.getByTestId(`absent-slot-card-${seeded.timetable_slot_am_id}`);
  318 |     await expect(absentSlotCard).toBeVisible();
  319 |     await expect(absentSlotCard).toContainText('P1_AM');
  320 |     await expect(absentSlotCard).toContainText('AM');
  321 |     await expect(absentSlotCard).toContainText('Free Cover Staff:');
  322 | 
  323 |     const absenceReasonInput = pluginFrame.getByTestId('absence-reason-input');
  324 |     const coverNotesInput = pluginFrame.getByTestId('cover-notes-input');
  325 |     await expect(absenceReasonInput).toBeVisible();
  326 |     await expect(coverNotesInput).toBeVisible();
  327 |     await absenceReasonInput.fill('Flu / Medical Absence');
  328 |     await coverNotesInput.fill('Complete quadratic equations worksheet on desk');
  329 | 
  330 |     const assignCoverBtn = pluginFrame.getByTestId('assign-cover-btn');
  331 |     await expect(assignCoverBtn).toBeVisible();
  332 |     await expect(assignCoverBtn).toHaveAttribute('data-mis-rpc', 'record_absence_and_assign_cover');
  333 |     await assignCoverBtn.click();
  334 | 
  335 |     const assignmentReceipt = pluginFrame.getByTestId('cover-assignment-receipt');
  336 |     await expect(assignmentReceipt).toBeVisible();
  337 |     await expect(assignmentReceipt).toContainText('COVER_ASSIGNED');
  338 |     await expect(assignmentReceipt).toContainText(seeded.teacher_staff_id);
  339 |     await expect(assignmentReceipt).toContainText(seeded.other_teacher_staff_id);
  340 |     await expect(assignmentReceipt).toContainText(seeded.own_group_id);
  341 |     await expect(assignmentReceipt).toContainText('COVER_TEACHER');
  342 | 
  343 |     // =========================================================================
  344 |     // Step 3: Switch back to Cover Teacher persona (seeded.other_teacher_staff_id) -> :own_groups now permits loading the class roster and submitting the statutory attendance register
  345 |     // =========================================================================
  346 |     const switchCoverAgainRes = await request.post('/api/auth/persona', {
  347 |       data: { staff_id: seeded.other_teacher_staff_id },
  348 |     });
  349 |     expect(switchCoverAgainRes.ok()).toBe(true);
  350 |     const coverAgainSessionBody = await switchCoverAgainRes.json();
  351 |     await page.context().addCookies([
  352 |       {
  353 |         name: 'mis_session',
  354 |         value: coverAgainSessionBody.session_token,
```