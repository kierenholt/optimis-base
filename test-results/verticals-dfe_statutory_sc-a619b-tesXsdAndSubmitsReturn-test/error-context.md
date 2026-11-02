# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: verticals/dfe_statutory_school_census.spec.ts >> DfE Statutory School Census Plugin, Host XML/XSD Validator & Mock COLLECT Service (Milestone 2B) >> generateValidateAndSubmitSchoolCensus_withMockCollectServiceEnabled_rendersRollupsValidatesXsdAndSubmitsReturn_test
- Location: tests/e2e/verticals/dfe_statutory_school_census.spec.ts:188:7

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Census' }).first()
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Census' }).first() with timeout 5000ms
  - waiting for locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Census' }).first()

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
      - tab "All (13)" [selected]
      - tab "Pages (10)"
      - tab "Widgets (0)"
      - tab "Pupils (3)"
      - tab "Groups (0)"
    - text: Views (10)
    - button "DfE Statutory School Census DfE Statutory School Census • /p/gtlnxgix (5 methods) Open View"
    - button "Pin DfE Statutory School Census": Pin
    - button "Educational Visits & Trip Consent Educational Visits & Trip Consent • /p/331qo66q (4 methods) Open View"
    - button "Pin Educational Visits & Trip Consent": Pin
    - button "Google Classroom LMS Sync Google Classroom LMS Sync • /p/k252w5cv (3 methods) Open View"
    - button "Pin Google Classroom LMS Sync": Pin
    - button "Medical Room & First Aid Medical Room & First Aid • /p/2l19fq2l (4 methods) Open View"
    - button "Pin Medical Room & First Aid": Pin
    - button "Pupil Medical & First Aid Profile Medical Room & First Aid • /p/2l19fq2l/student/:pupil_id (4 methods) Open View"
    - button "Pin Pupil Medical & First Aid Profile": Pin
    - button "Parent Portal DMZ Sync Parent Portal DMZ Sync • /p/jqzsh82o (3 methods) Open View"
    - button "Pin Parent Portal DMZ Sync": Pin
    - button "Pupil DMZ Sync History Parent Portal DMZ Sync • /p/jqzsh82o/student/:pupil_id (3 methods) Open View"
    - button "Pin Pupil DMZ Sync History": Pin
    - button "Sparx Maths Partner Sync Sparx Maths Partner Sync • /p/x8ijanmh (4 methods) Open View"
    - button "Pin Sparx Maths Partner Sync": Pin
    - button "Staff Absence & Cover Manager Staff Absence & Cover Manager • /p/kj7wyoj1 (6 methods) Open View"
    - button "Pin Staff Absence & Cover Manager": Pin
    - button "Timetable & Statutory Attendance Timetable & Statutory Attendance • /p/e454uy6j (3 methods) Open View"
    - button "Pin Timetable & Statutory Attendance": Pin
    - text: Pupils (3)
    - button "Maya Patel UPN H823457308101 • Year 10 Open Student"
    - button "Chloe Sterling UPN H823457308103 • Year 10 Open Student"
    - button "Noah Williams UPN H823457308102 • Year 10 Open Student"
```

# Test source

```ts
  126 |   });
  127 | 
  128 |   test.afterEach(async ({ request }, testInfo) => {
  129 |     // 1. Restore Mock DfE COLLECT service flag and clear submissions
  130 |     await request.post('/api/test/census-service-mode', {
  131 |       data: { enable_mock_dfe_census_service: true },
  132 |     });
  133 |     await request.delete('/api/test/census-submissions');
  134 | 
  135 |     // 2. Purge plugin census snapshots
  136 |     if (seeded && censusPlugId) {
  137 |       await request.post('/api/auth/persona', {
  138 |         data: { staff_id: seeded.admin_staff_id },
  139 |       });
  140 |       for (const pupilId of [
  141 |         seeded.pupil_sen_k_fsm_id,
  142 |         seeded.pupil_ehcp_lac_id,
  143 |         seeded.pupil_no_sen_id,
  144 |         seeded.off_roll_pupil_id,
  145 |       ]) {
  146 |         await request.post(`/api/p/${censusPlugId}/rpc/purge_pupil_data`, {
  147 |           data: { args: { pupil_id: pupilId } },
  148 |         });
  149 |       }
  150 |     }
  151 | 
  152 |     // 3. Fetch Backend Service Logs emitted during this test
  153 |     if (logCursor) {
  154 |       const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
  155 |       if (logsRes.ok()) {
  156 |         const serviceLogs: string[] = (await logsRes.json()).entries;
  157 |         if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
  158 |           console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
  159 |         }
  160 |       }
  161 |     }
  162 | 
  163 |     // 4. Dump tapped Browser Console logs on failure or error
  164 |     if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
  165 |       console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
  166 |       console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
  167 |     }
  168 | 
  169 |     // 5. Remove runtime-installed dfe_census_plugin
  170 |     await request.post('/api/test/auth-mode', {
  171 |       data: { enable_dev_persona_switcher: true },
  172 |     });
  173 |     await request.delete('/api/admin/plugins/uk.gov.dfe.census');
  174 | 
  175 |     // 6. Clean up seeded census fixtures in core.db
  176 |     if (seeded?.fixture_id) {
  177 |       const delRes = await request.delete(
  178 |         `/api/test/seed-census-fixtures/${encodeURIComponent(seeded.fixture_id)}`
  179 |       );
  180 |       if (!delRes.ok()) {
  181 |         throw new Error(`Failed to clean up census fixture ${seeded.fixture_id}: ${delRes.status()}`);
  182 |       }
  183 |     }
  184 | 
  185 |     expect(pageErrors).toEqual([]);
  186 |   });
  187 | 
  188 |   test('generateValidateAndSubmitSchoolCensus_withMockCollectServiceEnabled_rendersRollupsValidatesXsdAndSubmitsReturn_test', async ({
  189 |     page,
  190 |     request,
  191 |   }) => {
  192 |     if (!seeded) {
  193 |       throw new Error('Missing mandatory seeded census state');
  194 |     }
  195 | 
  196 |     // 1. Switch to Admin persona (who holds dal.census.read:all) and attach session cookie
  197 |     const switchAdminRes = await request.post('/api/auth/persona', {
  198 |       data: { staff_id: seeded.admin_staff_id },
  199 |     });
  200 |     expect(switchAdminRes.ok()).toBe(true);
  201 |     const adminSessionBody = await switchAdminRes.json();
  202 | 
  203 |     await page.context().addCookies([
  204 |       {
  205 |         name: 'mis_session',
  206 |         value: adminSessionBody.session_token,
  207 |         url: 'http://127.0.0.1:18080',
  208 |         sameSite: 'Strict',
  209 |       },
  210 |     ]);
  211 | 
  212 |     // 2. Navigate to Host UI Shell and open the DfE Statutory School Census plugin
  213 |     const query = new URLSearchParams({
  214 |       pupil_id: seeded.pupil_sen_k_fsm_id,
  215 |       reference_date: seeded.census_date,
  216 |       academic_year_id: seeded.academic_year_id,
  217 |       staff_id: seeded.admin_staff_id,
  218 |     });
  219 |     await page.goto(`/?${query.toString()}`);
  220 | 
  221 |     const hostShell = page.getByTestId('host-shell');
  222 |     await expect(hostShell).toBeVisible();
  223 | 
  224 |     await page.getByTestId('nav-rail-search').click();
  225 |     const censusNavBtn = page.locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Census' }).first();
> 226 |     await expect(censusNavBtn).toBeVisible();
      |                                ^ Error: expect(locator).toBeVisible() failed
  227 |     await censusNavBtn.click();
  228 | 
  229 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${censusPlugId}"]`);
  230 |     const termSelect = pluginFrame.getByTestId('census-term-select');
  231 |     const dateInput = pluginFrame.getByTestId('census-date-input');
  232 |     const generateBtn = pluginFrame.getByTestId('generate-census-btn');
  233 | 
  234 |     await expect(termSelect).toBeVisible();
  235 |     await expect(dateInput).toBeVisible();
  236 |     await expect(generateBtn).toBeVisible();
  237 |     await expect(generateBtn).toHaveAttribute('data-mis-rpc', 'generate_and_validate_census_return');
  238 | 
  239 |     // 3. Trigger Statutory School Census Generation & XSD Validation inside the sandboxed iframe
  240 |     await generateBtn.click();
  241 | 
  242 |     const summaryCards = pluginFrame.getByTestId('census-summary-cards');
  243 |     await expect(summaryCards).toBeVisible();
  244 | 
  245 |     const xsdStatus = pluginFrame.getByTestId('census-xsd-status');
  246 |     await expect(xsdStatus).toHaveText('XSD_VALID');
  247 | 
  248 |     const onRollCount = pluginFrame.getByTestId('census-on-roll-count');
  249 |     await expect(onRollCount).toBeVisible();
  250 |     const onRollValue = Number((await onRollCount.textContent())?.trim());
  251 |     expect(onRollValue).toBeGreaterThanOrEqual(3);
  252 | 
  253 |     const senBreakdown = pluginFrame.getByTestId('census-sen-breakdown');
  254 |     await expect(senBreakdown).toContainText('K:');
  255 |     await expect(senBreakdown).toContainText('E:');
  256 | 
  257 |     const fsmCount = pluginFrame.getByTestId('census-fsm-count');
  258 |     await expect(fsmCount).toBeVisible();
  259 |     expect(Number((await fsmCount.textContent())?.trim())).toBeGreaterThanOrEqual(2);
  260 | 
  261 |     const lacCount = pluginFrame.getByTestId('census-lac-count');
  262 |     await expect(lacCount).toBeVisible();
  263 |     expect(Number((await lacCount.textContent())?.trim())).toBeGreaterThanOrEqual(1);
  264 | 
  265 |     const attendanceSummary = pluginFrame.getByTestId('census-attendance-summary');
  266 |     await expect(attendanceSummary).toBeVisible();
  267 | 
  268 |     // 4. Inspect generated DfE CBDS XML payload in the UI preview
  269 |     const xmlPreview = pluginFrame.getByTestId('census-xml-preview');
  270 |     await expect(xmlPreview).toBeVisible();
  271 |     const xmlText = (await xmlPreview.textContent()) || '';
  272 | 
  273 |     expect(xmlText).toContain('<?xml version="1.0" encoding="UTF-8"?>');
  274 |     expect(xmlText).toContain('<Collection>SchoolCensus</Collection>');
  275 |     expect(xmlText).toContain('<Term>AUTUMN</Term>');
  276 |     expect(xmlText).toContain('<CensusDate>2026-10-01</CensusDate>');
  277 |     expect(xmlText).toContain('<SENProvision>K</SENProvision>');
  278 |     expect(xmlText).toContain('<SENProvision>E</SENProvision>');
  279 |     expect(xmlText).toContain('<FSMPeriods>');
  280 |     expect(xmlText).toContain('<LookedAfter>1</LookedAfter>');
  281 |     expect(xmlText).toContain('<PossibleSessions>');
  282 |     expect(xmlText).toContain('<SessionsPresent>');
  283 |     expect(xmlText).toContain('<AuthorisedAbsenceSessions>');
  284 |     expect(xmlText).toContain('<UnauthorisedAbsenceSessions>');
  285 | 
  286 |     // Verify off-roll pupil (effective_to = 2026-09-15 < census_date 2026-10-01) is strictly excluded from the generated return
  287 |     const listReturnsRes = await request.post(`/api/p/${censusPlugId}/rpc/list_census_returns`, {
  288 |       data: { args: { academic_year_id: seeded.academic_year_id } },
  289 |     });
  290 |     expect(listReturnsRes.ok()).toBe(true);
  291 |     const listReturnsBody = await listReturnsRes.json();
  292 |     expect(listReturnsBody.ok).toBe(true);
  293 |     expect(listReturnsBody.data.returns.length).toBeGreaterThanOrEqual(1);
  294 | 
  295 |     const exportOffRollRes = await request.post(`/api/p/${censusPlugId}/rpc/export_pupil_data`, {
  296 |       data: { args: { pupil_id: seeded.off_roll_pupil_id } },
  297 |     });
  298 |     expect(exportOffRollRes.ok()).toBe(true);
  299 |     const exportOffRollBody = await exportOffRollRes.json();
  300 |     expect(exportOffRollBody.ok).toBe(true);
  301 |     expect(exportOffRollBody.data.sections[0].records.length).toBe(0);
  302 | 
  303 |     // 5. Submit validated census return to the Mock DfE COLLECT service via UI button
  304 |     const submitBtn = pluginFrame.getByTestId('submit-census-collect-btn');
  305 |     await expect(submitBtn).toBeVisible();
  306 |     await expect(submitBtn).toHaveAttribute('data-mis-rpc', 'submit_census_return');
  307 |     await submitBtn.click();
  308 | 
  309 |     const collectReceipt = pluginFrame.getByTestId('census-collect-receipt');
  310 |     await expect(collectReceipt).toBeVisible();
  311 |     await expect(collectReceipt).toContainText('COLLECT_ACCEPTED');
  312 |     await expect(collectReceipt).toContainText('collect_sub_');
  313 | 
  314 |     // 6. Verify the Mock DfE COLLECT service recorded the outbound XML submission
  315 |     const submissionsRes = await request.get('/api/test/census-submissions');
  316 |     expect(submissionsRes.ok()).toBe(true);
  317 |     const submissionsBody = await submissionsRes.json();
  318 |     expect(submissionsBody.ok).toBe(true);
  319 |     expect(submissionsBody.submissions.length).toBe(1);
  320 |     expect(submissionsBody.submissions[0].status).toBe('COLLECT_ACCEPTED');
  321 |     expect(submissionsBody.submissions[0].term_code).toBe('AUTUMN');
  322 |     expect(submissionsBody.submissions[0].census_date).toBe('2026-10-01');
  323 |     expect(submissionsBody.submissions[0].xml_payload).toContain('<Collection>SchoolCensus</Collection>');
  324 |   });
  325 | 
  326 |   test('submitSchoolCensus_whenMockCollectFeatureFlagDisabledOrInvalidXsdOrUnauthorizedTeacher_failsFast_test', async ({
```