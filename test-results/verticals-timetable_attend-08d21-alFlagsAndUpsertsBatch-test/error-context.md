# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: verticals/timetable_attendance_plugin_vertical.spec.ts >> Vertical 1: Timetable & Statutory Session Attendance Plugin (uk.org.optimis.timetable.attendance) >> loadTimetableResolveSessionViewRosterAndSubmitAttendance_teacherWithOwnGroupsScope_resolvesAmPmLoadsSenMedicalFlagsAndUpsertsBatch_test
- Location: tests/e2e/verticals/timetable_attendance_plugin_vertical.spec.ts:156:7

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Timetable' }).first()
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Timetable' }).first() with timeout 5000ms
  - waiting for locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Timetable' }).first()

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
      - tab "All (7)" [selected]
      - tab "Pages (4)"
      - tab "Widgets (0)"
      - tab "Pupils (2)"
      - tab "Groups (1)"
    - text: Views (4)
    - button "Educational Visits & Trip Consent Educational Visits & Trip Consent • /p/331qo66q (4 methods) Open View"
    - button "Pin Educational Visits & Trip Consent": Pin
    - button "Group Trip Consent & Manifest Educational Visits & Trip Consent • /p/331qo66q/group/:group_id (4 methods) Open View"
    - button "Pin Group Trip Consent & Manifest": Pin
    - button "Timetable & Statutory Attendance Timetable & Statutory Attendance • /p/e454uy6j (3 methods) Open View"
    - button "Pin Timetable & Statutory Attendance": Pin
    - button "Class Session Register Timetable & Statutory Attendance • /p/e454uy6j/group/:group_id (3 methods) Open View"
    - button "Pin Class Session Register": Pin
    - text: Pupils (2)
    - button "Ada Lovelace UPN U148D23592001 • Year 10 Open Student"
    - button "Grace Turing UPN U448D23592004 • Year 10 Open Student"
    - text: Groups (1)
    - button "10Ma1_48d2 — Year 10 Maths Set 1 TEACHING_CLASS Open Group"
```

# Test source

```ts
  94  |         include_second_active_pupil: true,
  95  |       },
  96  |     });
  97  |     if (!seedRes.ok()) {
  98  |       throw new Error(`Failed to seed fixtures: ${seedRes.status()} ${await seedRes.text()}`);
  99  |     }
  100 |     const body = (await seedRes.json()) as Milestone1TimetableSeedResponse;
  101 |     if (
  102 |       !body.fixture_id ||
  103 |       !body.teacher_staff_id ||
  104 |       !body.other_teacher_staff_id ||
  105 |       !body.own_group_id ||
  106 |       !body.other_group_id ||
  107 |       !body.active_pupil_id ||
  108 |       !body.second_active_pupil_id ||
  109 |       !body.transferred_pupil_id ||
  110 |       !body.other_group_pupil_id ||
  111 |       !body.timetable_slot_am_id ||
  112 |       !body.timetable_slot_pm_id
  113 |     ) {
  114 |       throw new Error('Seeded fixture response missing mandatory timetable/attendance identifiers');
  115 |     }
  116 |     seeded = body;
  117 |   });
  118 | 
  119 |   test.afterEach(async ({ request }, testInfo) => {
  120 |     // 1. Fetch Backend Service Logs emitted during this test
  121 |     if (logCursor) {
  122 |       const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
  123 |       if (logsRes.ok()) {
  124 |         const serviceLogs: string[] = (await logsRes.json()).entries;
  125 |         if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
  126 |           console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
  127 |         }
  128 |       }
  129 |     }
  130 | 
  131 |     // 2. Dump tapped Browser Console logs on failure or error
  132 |     if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
  133 |       console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
  134 |       console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
  135 |     }
  136 | 
  137 |     // 3. Remove runtime-installed plugin
  138 |     await request.post('/api/test/auth-mode', {
  139 |       data: { enable_dev_persona_switcher: true },
  140 |     });
  141 |     await request.delete(`/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}`);
  142 | 
  143 |     // 4. Clean up seeded fixtures in core.db
  144 |     if (seeded?.fixture_id) {
  145 |       const delRes = await request.delete(
  146 |         `/api/test/seed-milestone1-fixtures/${encodeURIComponent(seeded.fixture_id)}`
  147 |       );
  148 |       if (!delRes.ok()) {
  149 |         throw new Error(`Failed to clean up fixture ${seeded.fixture_id}: ${delRes.status()}`);
  150 |       }
  151 |     }
  152 | 
  153 |     expect(pageErrors).toEqual([]);
  154 |   });
  155 | 
  156 |   test('loadTimetableResolveSessionViewRosterAndSubmitAttendance_teacherWithOwnGroupsScope_resolvesAmPmLoadsSenMedicalFlagsAndUpsertsBatch_test', async ({
  157 |     page,
  158 |     request,
  159 |   }) => {
  160 |     if (!seeded) {
  161 |       throw new Error('Missing mandatory seeded fixture state');
  162 |     }
  163 | 
  164 |     // 1. Switch to Teacher persona (holds :own_groups scopes for groups, pupils, and attendance)
  165 |     const switchRes = await request.post('/api/auth/persona', {
  166 |       data: { staff_id: seeded.teacher_staff_id },
  167 |     });
  168 |     expect(switchRes.ok()).toBe(true);
  169 |     const sessionBody = await switchRes.json();
  170 |     expect(sessionBody.ok).toBe(true);
  171 | 
  172 |     await page.context().addCookies([
  173 |       {
  174 |         name: 'mis_session',
  175 |         value: sessionBody.session_token,
  176 |         url: 'http://127.0.0.1:18080',
  177 |         sameSite: 'Strict',
  178 |       },
  179 |     ]);
  180 | 
  181 |     const params = new URLSearchParams({
  182 |       pupil_id: seeded.active_pupil_id,
  183 |       group_id: seeded.own_group_id,
  184 |       other_group_id: seeded.other_group_id,
  185 |       reference_date: seeded.reference_date,
  186 |       academic_year_id: seeded.academic_year_id,
  187 |     });
  188 |     await page.goto(`/?${params.toString()}`);
  189 | 
  190 |     await expect(page.getByTestId('host-shell')).toBeVisible();
  191 | 
  192 |     await page.getByTestId('nav-rail-search').click();
  193 |     const navItem = page.locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Timetable' }).first();
> 194 |     await expect(navItem).toBeVisible();
      |                           ^ Error: expect(locator).toBeVisible() failed
  195 |     await navItem.click();
  196 | 
  197 |     const iframeEl = page.getByTestId(`plugin-iframe-${timetablePlugId}`);
  198 |     await expect(iframeEl).toBeVisible();
  199 |     await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');
  200 | 
  201 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${timetablePlugId}"]`);
  202 |     await expect(pluginFrame.getByTestId('timetable-attendance-root')).toBeVisible();
  203 | 
  204 |     // =========================================================================
  205 |     // Step 1: Teacher loads rolling timetable (get_staff_timetable_slots) -> resolves AM/PM statutory session
  206 |     // =========================================================================
  207 |     const cycleWeekSelect = pluginFrame.getByTestId('timetable-cycle-week-select');
  208 |     const daySelect = pluginFrame.getByTestId('timetable-day-select');
  209 |     const loadTimetableBtn = pluginFrame.getByTestId('load-timetable-btn');
  210 | 
  211 |     await expect(cycleWeekSelect).toBeVisible();
  212 |     await expect(daySelect).toBeVisible();
  213 |     await expect(loadTimetableBtn).toBeVisible();
  214 |     await expect(loadTimetableBtn).toHaveAttribute('data-mis-rpc', 'load_staff_timetable');
  215 | 
  216 |     // Verify Week 2 resolves statutory PM session first
  217 |     await cycleWeekSelect.selectOption('2');
  218 |     await loadTimetableBtn.click();
  219 | 
  220 |     const resolvedSessionBadge = pluginFrame.getByTestId('resolved-statutory-session-badge');
  221 |     await expect(resolvedSessionBadge).toBeVisible();
  222 |     await expect(resolvedSessionBadge).toHaveText('PM');
  223 |     await expect(pluginFrame.getByTestId(`timetable-slot-card-${seeded.timetable_slot_pm_id}`)).toBeVisible();
  224 | 
  225 |     // Switch back to Week 1 and verify it resolves statutory AM session for OwnGroupID
  226 |     await cycleWeekSelect.selectOption('1');
  227 |     await loadTimetableBtn.click();
  228 | 
  229 |     await expect(resolvedSessionBadge).toHaveText('AM');
  230 |     const amSlotCard = pluginFrame.getByTestId(`timetable-slot-card-${seeded.timetable_slot_am_id}`);
  231 |     await expect(amSlotCard).toBeVisible();
  232 |     await expect(amSlotCard).toContainText('P1_AM');
  233 |     await expect(amSlotCard).toContainText('AM');
  234 | 
  235 |     // =========================================================================
  236 |     // Step 2: Teacher views class roster with SEN/Medical flags (list_by_group + list_sen_and_medical_batch)
  237 |     // =========================================================================
  238 |     const loadRosterBtn = pluginFrame.getByTestId('load-roster-btn');
  239 |     await expect(loadRosterBtn).toBeVisible();
  240 |     await expect(loadRosterBtn).toHaveAttribute('data-mis-rpc', 'load_slot_roster_with_flags');
  241 |     await loadRosterBtn.click();
  242 | 
  243 |     const rosterTable = pluginFrame.getByTestId('class-roster-table');
  244 |     await expect(rosterTable).toBeVisible();
  245 | 
  246 |     // Active Pupil 1 (Ada Lovelace): SEN 'E' (ASD) + Medical 'Anaphylaxis (Peanut Allergy)' (IHCP)
  247 |     const rowPupil1 = pluginFrame.getByTestId(`roster-row-${seeded.active_pupil_id}`);
  248 |     await expect(rowPupil1).toBeVisible();
  249 |     await expect(rowPupil1).toContainText('Ada Lovelace');
  250 |     await expect(rowPupil1).toContainText('[Y10]');
  251 |     await expect(pluginFrame.getByTestId(`sen-flag-${seeded.active_pupil_id}`)).toContainText('E (ASD)');
  252 |     await expect(pluginFrame.getByTestId(`medical-flag-${seeded.active_pupil_id}`)).toContainText(
  253 |       'Anaphylaxis (Peanut Allergy)'
  254 |     );
  255 |     await expect(pluginFrame.getByTestId(`medical-flag-${seeded.active_pupil_id}`)).toContainText('IHCP');
  256 | 
  257 |     // Active Pupil 2 (Grace Turing): SEN 'K' (SPLD)
  258 |     const rowPupil2 = pluginFrame.getByTestId(`roster-row-${seeded.second_active_pupil_id}`);
  259 |     await expect(rowPupil2).toBeVisible();
  260 |     await expect(rowPupil2).toContainText('Grace Turing');
  261 |     await expect(rowPupil2).toContainText('[Y10]');
  262 |     await expect(pluginFrame.getByTestId(`sen-flag-${seeded.second_active_pupil_id}`)).toContainText('K (SPLD)');
  263 | 
  264 |     // Transferred pupil (effective_to < reference_date) and other teacher's pupil MUST NOT appear
  265 |     await expect(pluginFrame.getByTestId(`roster-row-${seeded.transferred_pupil_id}`)).toHaveCount(0);
  266 |     await expect(pluginFrame.getByTestId(`roster-row-${seeded.other_group_pupil_id}`)).toHaveCount(0);
  267 | 
  268 |     // =========================================================================
  269 |     // Step 3: Teacher submits batch attendance (upsert_session_batch)
  270 |     // =========================================================================
  271 |     const markSelect1 = pluginFrame.getByTestId(`attendance-code-select-${seeded.active_pupil_id}`);
  272 |     const markSelect2 = pluginFrame.getByTestId(`attendance-code-select-${seeded.second_active_pupil_id}`);
  273 |     const minutesLateInput2 = pluginFrame.getByTestId(`minutes-late-input-${seeded.second_active_pupil_id}`);
  274 |     const commentInput2 = pluginFrame.getByTestId(`attendance-comment-input-${seeded.second_active_pupil_id}`);
  275 | 
  276 |     await markSelect1.selectOption('/');
  277 |     await markSelect2.selectOption('L');
  278 |     await minutesLateInput2.fill('10');
  279 |     await commentInput2.fill('Northern Line delay');
  280 | 
  281 |     const submitBatchBtn = pluginFrame.getByTestId('submit-attendance-batch-btn');
  282 |     await expect(submitBatchBtn).toBeVisible();
  283 |     await expect(submitBatchBtn).toHaveAttribute('data-mis-rpc', 'submit_session_attendance_batch');
  284 |     await submitBatchBtn.click();
  285 | 
  286 |     const submissionReceipt = pluginFrame.getByTestId('attendance-submission-receipt');
  287 |     await expect(submissionReceipt).toBeVisible();
  288 |     await expect(submissionReceipt).toContainText('BATCH_SAVED:AM');
  289 |     await expect(submissionReceipt).toContainText('Upserted: 2');
  290 |     await expect(submissionReceipt).toContainText('ars_');
  291 |     await expect(submissionReceipt).toContainText('Timestamp:');
  292 | 
  293 |     // Verify persisted attendance marks are reflected on the roster rows
  294 |     await expect(pluginFrame.getByTestId(`persisted-mark-badge-${seeded.active_pupil_id}`)).toContainText('/');
```