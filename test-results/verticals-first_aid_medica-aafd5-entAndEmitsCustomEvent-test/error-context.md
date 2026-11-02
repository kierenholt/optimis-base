# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: verticals/first_aid_medical_plugin_vertical.spec.ts >> Vertical 6: Medical Room & First Aid Plugin (uk.org.optimis.medical.firstaid) >> selectInjuredPupilViewCriticalMedicalAlertsAndSafeContactsAndLogTreatment_asFirstAider_rendersCriticalIhcpExcludesCourtOrderFatherAndRefugeAddressPersistsTreatmentAndEmitsCustomEvent_test
- Location: tests/e2e/verticals/first_aid_medical_plugin_vertical.spec.ts:178:7

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'First Aid' }).first()
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'First Aid' }).first() with timeout 5000ms
  - waiting for locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'First Aid' }).first()

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
      - tab "All (8)" [selected]
      - tab "Pages (4)"
      - tab "Widgets (0)"
      - tab "Pupils (4)"
      - tab "Groups (0)"
    - text: Views (4)
    - button "Medical Room & First Aid Medical Room & First Aid • /p/2l19fq2l (3 methods) Open View"
    - button "Pin Medical Room & First Aid": Pin
    - button "Pupil Medical & First Aid Profile Medical Room & First Aid • /p/2l19fq2l/student/:pupil_id (3 methods) Open View"
    - button "Pin Pupil Medical & First Aid Profile": Pin
    - button "Parent Portal DMZ Sync Parent Portal DMZ Sync • /p/jqzsh82o (3 methods) Open View"
    - button "Pin Parent Portal DMZ Sync": Pin
    - button "Pupil DMZ Sync History Parent Portal DMZ Sync • /p/jqzsh82o/student/:pupil_id (3 methods) Open View"
    - button "Pin Pupil DMZ Sync History": Pin
    - text: Pupils (4)
    - button "Charles Babbage UPN U2A9962D5F002 • Year 10 Open Student"
    - button "Ada Lovelace UPN U1A9962D5F001 • Year 10 Open Student"
    - button "Ada Lovelace UPN PCABF9B1EC701 • Year 7 Open Student"
    - button "Alan Turing UPN U3A9962D5F003 • Year 10 Open Student"
```

# Test source

```ts
  121 |   });
  122 | 
  123 |   test.afterEach(async ({ request }, testInfo) => {
  124 |     // 1. Purge plugin data created during the test
  125 |     if (personas && m1Seeded && faPlugId) {
  126 |       await request.post('/api/auth/persona', {
  127 |         data: { staff_id: personas.admin_staff_id },
  128 |       });
  129 |       await request.post(`/api/p/${faPlugId}/rpc/purge_pupil_data`, {
  130 |         data: { args: { pupil_id: m1Seeded.active_pupil_id } },
  131 |       });
  132 |     }
  133 | 
  134 |     // 2. Fetch Backend Service Logs emitted during this test
  135 |     if (logCursor) {
  136 |       const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
  137 |       if (logsRes.ok()) {
  138 |         const serviceLogs: string[] = (await logsRes.json()).entries;
  139 |         if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
  140 |           console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
  141 |         }
  142 |       }
  143 |     }
  144 | 
  145 |     // 3. Dump tapped Browser Console logs on failure or error
  146 |     if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
  147 |       console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
  148 |       console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
  149 |     }
  150 | 
  151 |     // 4. Remove runtime-installed plugin
  152 |     await request.post('/api/test/auth-mode', {
  153 |       data: { enable_dev_persona_switcher: true },
  154 |     });
  155 |     await request.delete('/api/admin/plugins/uk.org.optimis.medical.firstaid');
  156 | 
  157 |     // 5. Clean up seeded fixtures in core.db
  158 |     if (m1Seeded?.fixture_id) {
  159 |       const delM1 = await request.delete(
  160 |         `/api/test/seed-milestone1-fixtures/${encodeURIComponent(m1Seeded.fixture_id)}`
  161 |       );
  162 |       if (!delM1.ok()) {
  163 |         throw new Error(`Failed to clean up Milestone 1 fixture ${m1Seeded.fixture_id}: ${delM1.status()}`);
  164 |       }
  165 |     }
  166 |     if (personas?.fixture_id) {
  167 |       const delP = await request.delete(
  168 |         `/api/test/seed-personas/${encodeURIComponent(personas.fixture_id)}`
  169 |       );
  170 |       if (!delP.ok()) {
  171 |         throw new Error(`Failed to clean up persona fixture ${personas.fixture_id}: ${delP.status()}`);
  172 |       }
  173 |     }
  174 | 
  175 |     expect(pageErrors).toEqual([]);
  176 |   });
  177 | 
  178 |   test('selectInjuredPupilViewCriticalMedicalAlertsAndSafeContactsAndLogTreatment_asFirstAider_rendersCriticalIhcpExcludesCourtOrderFatherAndRefugeAddressPersistsTreatmentAndEmitsCustomEvent_test', async ({
  179 |     page,
  180 |     request,
  181 |   }) => {
  182 |     if (!personas || !m1Seeded) {
  183 |       throw new Error('Missing mandatory seeded fixtures');
  184 |     }
  185 | 
  186 |     const firstAiderStaffId = personas.school_nurse_staff_id || personas.attendance_officer_staff_id;
  187 |     expect(firstAiderStaffId).toBeTruthy();
  188 | 
  189 |     // 1. Switch to School Nurse / First Aider persona (SUPPORT role holding dal.pupils.read:all, dal.medical.read:all, dal.guardians.read:all)
  190 |     const switchFirstAiderRes = await request.post('/api/auth/persona', {
  191 |       data: { staff_id: firstAiderStaffId },
  192 |     });
  193 |     expect(switchFirstAiderRes.ok()).toBe(true);
  194 |     const firstAiderSession = await switchFirstAiderRes.json();
  195 |     expect(firstAiderSession.caller.active_roles).toContain('SUPPORT');
  196 |     expect(firstAiderSession.caller.scopes).toContain('dal.pupils.read:all');
  197 |     expect(firstAiderSession.caller.scopes).toContain('dal.medical.read:all');
  198 |     expect(firstAiderSession.caller.scopes).toContain('dal.guardians.read:all');
  199 | 
  200 |     await page.context().addCookies([
  201 |       {
  202 |         name: 'mis_session',
  203 |         value: firstAiderSession.session_token,
  204 |         url: 'http://127.0.0.1:18080',
  205 |         sameSite: 'Strict',
  206 |       },
  207 |     ]);
  208 | 
  209 |     // 2. Open Host UI Shell and navigate to Medical Room & First Aid plugin iframe
  210 |     const query = new URLSearchParams({
  211 |       pupil_id: m1Seeded.active_pupil_id,
  212 |       reference_date: m1Seeded.reference_date,
  213 |       academic_year_id: m1Seeded.academic_year_id,
  214 |       staff_id: firstAiderStaffId,
  215 |     });
  216 |     await page.goto(`/?${query.toString()}`);
  217 | 
  218 |     await expect(page.getByTestId('host-shell')).toBeVisible();
  219 |     await page.getByTestId('nav-rail-search').click();
  220 |     const navBtn = page.locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'First Aid' }).first();
> 221 |     await expect(navBtn).toBeVisible();
      |                          ^ Error: expect(locator).toBeVisible() failed
  222 |     await navBtn.click();
  223 | 
  224 |     const iframeEl = page.getByTestId(`plugin-iframe-${faPlugId}`);
  225 |     await expect(iframeEl).toBeVisible();
  226 |     await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');
  227 | 
  228 |     const frame = page.frameLocator(`[data-testid="plugin-iframe-${faPlugId}"]`);
  229 |     await expect(frame.getByTestId('first-aid-plugin-root')).toBeVisible();
  230 | 
  231 |     // 3. Step 1 of Journey: First Aider selects injured pupil in UI
  232 |     const pupilSelect = frame.getByTestId('first-aid-pupil-select');
  233 |     await expect(pupilSelect).toBeVisible();
  234 |     await pupilSelect.selectOption(m1Seeded.active_pupil_id);
  235 | 
  236 |     const loadProfileBtn = frame.getByTestId('load-pupil-medical-profile-btn');
  237 |     await expect(loadProfileBtn).toHaveAttribute('data-mis-rpc', 'get_pupil_first_aid_profile');
  238 |     await loadProfileBtn.click();
  239 | 
  240 |     // 4. Step 2 of Journey: Plugin renders CRITICAL anaphylaxis/IHCP badge + clinical medication/location/trigger details + emergency contacts
  241 |     const criticalAlertBadge = frame.getByTestId('critical-medical-alert-badge');
  242 |     await expect(criticalAlertBadge).toBeVisible();
  243 |     await expect(criticalAlertBadge).toContainText('CRITICAL');
  244 |     await expect(criticalAlertBadge).toContainText('Anaphylaxis (Peanut Allergy)');
  245 |     await expect(criticalAlertBadge).toContainText('IHCP');
  246 |     await expect(criticalAlertBadge).toContainText('Administer EpiPen immediately and call 999');
  247 |     await expect(criticalAlertBadge).toContainText('EpiPen 0.3mg IM Auto-Injector');
  248 |     await expect(criticalAlertBadge).toContainText('Medical Room Red Cabinet #2');
  249 |     await expect(criticalAlertBadge).toContainText('Peanuts, tree nuts');
  250 |     await expect(criticalAlertBadge).toContainText('PARENTAL_CONSENT_VERIFIED');
  251 | 
  252 |     const emergencyContactsCard = frame.getByTestId('emergency-contacts-card');
  253 |     await expect(emergencyContactsCard).toBeVisible();
  254 |     await expect(emergencyContactsCard).toContainText('Anne Byron');
  255 |     await expect(emergencyContactsCard).toContainText('MOTHER');
  256 |     await expect(emergencyContactsCard).toContainText('42 High Street');
  257 |     await expect(emergencyContactsCard).toContainText('AUTHORISED_TO_COLLECT');
  258 | 
  259 |     // Refuge guardian (Grandparent Judith Milbanke) is shown as emergency contact #2 with her refuge address strictly redacted
  260 |     await expect(emergencyContactsCard).toContainText('Judith Milbanke');
  261 |     await expect(emergencyContactsCard).toContainText('GRANDPARENT');
  262 |     await expect(emergencyContactsCard).toContainText('CONFIDENTIAL_ADDRESS_REDACTED');
  263 | 
  264 |     // Strictly assert court-order-blocked father & confidential refuge address are excluded from the entire iframe DOM
  265 |     const iframeHtmlContent = await frame.locator('body').innerHTML();
  266 |     expect(iframeHtmlContent).not.toContain('George Byron');
  267 |     expect(iframeHtmlContent).not.toContain(m1Seeded.restricted_father_person_id);
  268 |     expect(iframeHtmlContent).not.toContain('Prohibited Steps Order');
  269 |     expect(iframeHtmlContent).not.toContain('10 Safe Haven Way');
  270 |     expect(iframeHtmlContent).not.toContain('Confidential Refuge');
  271 |     expect(iframeHtmlContent).not.toContain('SW1A 1AA');
  272 | 
  273 |     // 5. Step 3 of Journey: Log treatment in `plugin_first_aid.db` and verify post-commit custom event + GDPR SAR hooks
  274 |     await frame.getByTestId('incident-location-input').fill('Main Playground - Netball Court');
  275 |     await frame
  276 |       .getByTestId('injury-summary-input')
  277 |       .fill('Grazed right knee and suspected wasp sting near wrist');
  278 |     await frame
  279 |       .getByTestId('treatment-given-input')
  280 |       .fill('Cleaned wound with sterile saline, applied cold compress, monitored 20 mins for allergic reaction');
  281 |     await frame.getByTestId('outcome-select').selectOption('RETURNED_TO_CLASS');
  282 | 
  283 |     const logTreatmentBtn = frame.getByTestId('log-first-aid-treatment-btn');
  284 |     await expect(logTreatmentBtn).toHaveAttribute('data-mis-rpc', 'log_first_aid_treatment');
  285 |     await logTreatmentBtn.click();
  286 | 
  287 |     const receiptBanner = frame.getByTestId('treatment-receipt-banner');
  288 |     await expect(receiptBanner).toBeVisible();
  289 |     await expect(receiptBanner).toContainText('LOGGED');
  290 |     await expect(receiptBanner).toContainText('plugin.uk.org.optimis.medical.firstaid.logged');
  291 | 
  292 |     const treatmentHistoryList = frame.getByTestId('treatment-history-list');
  293 |     await expect(treatmentHistoryList).toBeVisible();
  294 |     await expect(treatmentHistoryList).toContainText('Main Playground - Netball Court');
  295 |     await expect(treatmentHistoryList).toContainText('Grazed right knee and suspected wasp sting near wrist');
  296 |     await expect(treatmentHistoryList).toContainText('Cleaned wound with sterile saline');
  297 |     await expect(treatmentHistoryList).toContainText('RETURNED_TO_CLASS');
  298 | 
  299 |     // Reload profile to verify post-commit custom event worker persisted audit confirmation in `plugin_first_aid.db`
  300 |     await loadProfileBtn.click();
  301 |     const eventDispatchBadge = frame.getByTestId('first-aid-event-dispatch-badge');
  302 |     await expect(eventDispatchBadge).toBeVisible();
  303 |     await expect(eventDispatchBadge).toContainText('plugin.uk.org.optimis.medical.firstaid.logged');
  304 |     await expect(eventDispatchBadge).toContainText('DISPATCHED_POST_COMMIT');
  305 |   });
  306 | 
  307 |   test('firstAidMedicalPlugin_unauthorizedTeacherWithoutAllMedicalAndGuardianScopesOrMissingParamsAndGdprSarHooks_failsFastAndPurgesCleanly_test', async ({
  308 |     request,
  309 |   }) => {
  310 |     if (!personas || !m1Seeded) {
  311 |       throw new Error('Missing mandatory seeded fixtures');
  312 |     }
  313 | 
  314 |     const firstAiderStaffId = personas.school_nurse_staff_id || personas.attendance_officer_staff_id;
  315 | 
  316 |     // 1. RBAC Verification: Classroom Teacher (lacking dal.medical.read:all & dal.guardians.read:all) is rejected with 403
  317 |     const switchTeacherRes = await request.post('/api/auth/persona', {
  318 |       data: { staff_id: personas.teacher_staff_id },
  319 |     });
  320 |     expect(switchTeacherRes.ok()).toBe(true);
  321 | 
```