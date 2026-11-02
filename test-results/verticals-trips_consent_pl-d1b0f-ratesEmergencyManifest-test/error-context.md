# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: verticals/trips_consent_plugin_vertical.spec.ts >> Vertical 7: Educational Visits, PR Consent Verification & Trip Emergency Pack Plugin (uk.org.optimis.ops.trips) >> createTripVerifyPrConsentAndGenerateEmergencyPack_teacherCreatesGroupTripRejectsNonPrAndCourtOrderGuardiansAcceptsUnrestrictedPrMotherAndGeneratesEmergencyManifest_test
- Location: tests/e2e/verticals/trips_consent_plugin_vertical.spec.ts:223:7

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Trip' }).first()
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Trip' }).first() with timeout 5000ms
  - waiting for locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Trip' }).first()

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
      - tab "All (4)" [selected]
      - tab "Pages (2)"
      - tab "Widgets (0)"
      - tab "Pupils (1)"
      - tab "Groups (1)"
    - text: Views (2)
    - button "Educational Visits & Trip Consent Educational Visits & Trip Consent • /p/331qo66q (4 methods) Open View"
    - button "Pin Educational Visits & Trip Consent": Pin
    - button "Group Trip Consent & Manifest Educational Visits & Trip Consent • /p/331qo66q/group/:group_id (4 methods) Open View"
    - button "Pin Group Trip Consent & Manifest": Pin
    - text: Pupils (1)
    - button "Ada Lovelace UPN U163093CEA001 • Year 10 Open Student"
    - text: Groups (1)
    - button "10Ma1_6309 — Year 10 Maths Set 1 TEACHING_CLASS Open Group"
```

# Test source

```ts
  150 |   });
  151 | 
  152 |   test.afterEach(async ({ request }, testInfo) => {
  153 |     // 1. Clean up trip consent records in plugin_trips.db
  154 |     if (personaSeed && m1Seed && tripsPlugId) {
  155 |       await request.post('/api/auth/persona', {
  156 |         data: { staff_id: personaSeed.admin_staff_id },
  157 |       });
  158 |       await request.post(`/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}/activate`, {
  159 |         data: { staff_id: personaSeed.admin_staff_id },
  160 |       });
  161 |       for (const pid of [
  162 |         m1Seed.active_pupil_id,
  163 |         m1Seed.transferred_pupil_id,
  164 |         m1Seed.other_group_pupil_id,
  165 |       ]) {
  166 |         await request.post(`/api/p/${tripsPlugId}/rpc/purge_pupil_data`, {
  167 |           data: { args: { pupil_id: pid } },
  168 |         });
  169 |       }
  170 |       await request.put('/api/system/user-settings', {
  171 |         data: {
  172 |           staff_id: m1Seed.teacher_staff_id,
  173 |           base_font_size: 'medium',
  174 |           theme_mode: 'light',
  175 |         },
  176 |       });
  177 |     }
  178 | 
  179 |     // 2. Fetch Backend Service Logs emitted during this test
  180 |     if (logCursor) {
  181 |       const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
  182 |       if (logsRes.ok()) {
  183 |         const serviceLogs: string[] = (await logsRes.json()).entries;
  184 |         if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
  185 |           console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
  186 |         }
  187 |       }
  188 |     }
  189 | 
  190 |     // 3. Dump tapped Browser Console logs on failure or error
  191 |     if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
  192 |       console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
  193 |       console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
  194 |     }
  195 | 
  196 |     // 4. Remove runtime-installed plugin
  197 |     await request.post('/api/test/auth-mode', {
  198 |       data: { enable_dev_persona_switcher: true },
  199 |     });
  200 |     await request.delete(`/api/admin/plugins/${encodeURIComponent(PLUGIN_ID)}`);
  201 | 
  202 |     // 5. Clean up seeded fixtures in core.db
  203 |     if (m1Seed?.fixture_id) {
  204 |       const delM1 = await request.delete(
  205 |         `/api/test/seed-milestone1-fixtures/${encodeURIComponent(m1Seed.fixture_id)}`
  206 |       );
  207 |       if (!delM1.ok()) {
  208 |         throw new Error(`Failed to clean up Milestone 1 fixture ${m1Seed.fixture_id}: ${delM1.status()}`);
  209 |       }
  210 |     }
  211 |     if (personaSeed?.fixture_id) {
  212 |       const delP = await request.delete(
  213 |         `/api/test/seed-personas/${encodeURIComponent(personaSeed.fixture_id)}`
  214 |       );
  215 |       if (!delP.ok()) {
  216 |         throw new Error(`Failed to clean up persona fixture ${personaSeed.fixture_id}: ${delP.status()}`);
  217 |       }
  218 |     }
  219 | 
  220 |     expect(pageErrors).toEqual([]);
  221 |   });
  222 | 
  223 |   test('createTripVerifyPrConsentAndGenerateEmergencyPack_teacherCreatesGroupTripRejectsNonPrAndCourtOrderGuardiansAcceptsUnrestrictedPrMotherAndGeneratesEmergencyManifest_test', async ({
  224 |     page,
  225 |     request,
  226 |   }) => {
  227 |     if (!m1Seed || !personaSeed) {
  228 |       throw new Error('Missing mandatory seeded state');
  229 |     }
  230 | 
  231 |     const nonPrGrandparentPersonId = `per_gp_${m1Seed.fixture_id}`;
  232 | 
  233 |     // Open Host UI Shell as the Trip Leader / Classroom Teacher (m1Seed.teacher_staff_id)
  234 |     await request.post('/api/auth/persona', {
  235 |       data: { staff_id: m1Seed.teacher_staff_id },
  236 |     });
  237 |     const params = new URLSearchParams({
  238 |       pupil_id: m1Seed.active_pupil_id,
  239 |       group_id: m1Seed.own_group_id,
  240 |       other_group_id: m1Seed.other_group_id,
  241 |       reference_date: m1Seed.reference_date,
  242 |       academic_year_id: m1Seed.academic_year_id,
  243 |     });
  244 |     await page.goto(`/?${params.toString()}`);
  245 | 
  246 |     await expect(page.getByTestId('host-shell')).toBeVisible();
  247 | 
  248 |     await page.getByTestId('nav-rail-search').click();
  249 |     const tripsNavBtn = page.locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Trip' }).first();
> 250 |     await expect(tripsNavBtn).toBeVisible();
      |                               ^ Error: expect(locator).toBeVisible() failed
  251 |     await tripsNavBtn.click();
  252 | 
  253 |     const tripsIframeEl = page.getByTestId(`plugin-iframe-${tripsPlugId}`);
  254 |     await expect(tripsIframeEl).toBeVisible();
  255 |     await expect(tripsIframeEl).toHaveAttribute('sandbox', 'allow-scripts');
  256 | 
  257 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${tripsPlugId}"]`);
  258 |     await expect(pluginFrame.getByTestId('trips-consent-plugin-root')).toBeVisible();
  259 | 
  260 |     // =========================================================================
  261 |     // Step 1: Teacher creates an educational visit for their group in plugin_trips.db
  262 |     // =========================================================================
  263 |     const tripNameInput = pluginFrame.getByTestId('trip-name-input');
  264 |     const tripDestinationInput = pluginFrame.getByTestId('trip-destination-input');
  265 |     const tripDateInput = pluginFrame.getByTestId('trip-date-input');
  266 |     const createTripBtn = pluginFrame.getByTestId('create-trip-btn');
  267 | 
  268 |     await expect(tripNameInput).toBeVisible();
  269 |     await expect(createTripBtn).toBeVisible();
  270 |     await expect(createTripBtn).toHaveAttribute('data-mis-rpc', 'create_trip');
  271 | 
  272 |     await tripNameInput.fill('Science Museum Astrophysics Visit');
  273 |     await tripDestinationInput.fill('Science Museum, South Kensington');
  274 |     await tripDateInput.fill('2026-11-15');
  275 |     await createTripBtn.click();
  276 | 
  277 |     const activeTripCard = pluginFrame.getByTestId('active-trip-card');
  278 |     await expect(activeTripCard).toBeVisible();
  279 |     await expect(pluginFrame.getByTestId('active-trip-name')).toHaveText('Science Museum Astrophysics Visit');
  280 |     await expect(pluginFrame.getByTestId('active-trip-destination')).toHaveText(
  281 |       'Science Museum, South Kensington'
  282 |     );
  283 |     await expect(pluginFrame.getByTestId('active-trip-eligible-count')).toHaveText('1');
  284 | 
  285 |     // =========================================================================
  286 |     // Step 2a: Attempt to record consent from a non-PR adult (Grandparent has_parental_responsibility=0)
  287 |     //          -> rejected (GUARDIAN_NOT_AUTHORIZED_FOR_CONSENT)
  288 |     // =========================================================================
  289 |     const guardianIdInput = pluginFrame.getByTestId('consent-guardian-id-input');
  290 |     const dietaryInput = pluginFrame.getByTestId('consent-dietary-notes-input');
  291 |     const medicalNotesInput = pluginFrame.getByTestId('consent-medical-notes-input');
  292 |     const recordConsentBtn = pluginFrame.getByTestId('record-consent-btn');
  293 | 
  294 |     await expect(guardianIdInput).toBeVisible();
  295 |     await expect(recordConsentBtn).toBeVisible();
  296 |     await expect(recordConsentBtn).toHaveAttribute('data-mis-rpc', 'record_trip_consent');
  297 | 
  298 |     await guardianIdInput.fill(nonPrGrandparentPersonId);
  299 |     await dietaryInput.fill('No nuts');
  300 |     await medicalNotesInput.fill('EpiPen in bag');
  301 |     await recordConsentBtn.click();
  302 | 
  303 |     const errorBanner = pluginFrame.getByTestId('trips-error-banner');
  304 |     await expect(errorBanner).toBeVisible();
  305 |     await expect(errorBanner).toContainText('GUARDIAN_NOT_AUTHORIZED_FOR_CONSENT');
  306 | 
  307 |     // =========================================================================
  308 |     // Step 2b: Attempt to record consent from a court-order-restricted adult (Father court_order_restricted=1)
  309 |     //          -> rejected (GUARDIAN_NOT_AUTHORIZED_FOR_CONSENT)
  310 |     // =========================================================================
  311 |     await guardianIdInput.fill(m1Seed.restricted_father_person_id);
  312 |     await recordConsentBtn.click();
  313 | 
  314 |     await expect(errorBanner).toBeVisible();
  315 |     await expect(errorBanner).toContainText('GUARDIAN_NOT_AUTHORIZED_FOR_CONSENT');
  316 |     const errorText = (await errorBanner.textContent()) || '';
  317 |     expect(errorText).not.toContain('Prohibited Steps Order');
  318 | 
  319 |     // =========================================================================
  320 |     // Step 3a: Record consent from unrestricted PR mother (Anne Byron, MOTHER, PR=1)
  321 |     // =========================================================================
  322 |     await guardianIdInput.fill(m1Seed.unrestricted_mother_person_id);
  323 |     await dietaryInput.fill('Strict peanut-free and tree-nut-free packed lunch');
  324 |     await medicalNotesInput.fill('Carries 2x EpiPen auto-injectors in red medical pouch');
  325 |     await recordConsentBtn.click();
  326 | 
  327 |     await expect(errorBanner).toBeHidden();
  328 |     const consentStatusBadge = pluginFrame.getByTestId('consent-status-badge');
  329 |     await expect(consentStatusBadge).toBeVisible();
  330 |     await expect(consentStatusBadge).toHaveText('CONSENTED');
  331 | 
  332 |     const consentedGuardianSummary = pluginFrame.getByTestId('consented-guardian-summary');
  333 |     await expect(consentedGuardianSummary).toBeVisible();
  334 |     await expect(consentedGuardianSummary).toContainText('Anne Byron');
  335 |     await expect(consentedGuardianSummary).toContainText('MOTHER');
  336 | 
  337 |     // =========================================================================
  338 |     // Step 3b: Generate Trip Emergency Manifest with medical/dietary/SEN flags
  339 |     // =========================================================================
  340 |     const generatePackBtn = pluginFrame.getByTestId('generate-emergency-pack-btn');
  341 |     await expect(generatePackBtn).toBeVisible();
  342 |     await expect(generatePackBtn).toHaveAttribute('data-mis-rpc', 'generate_trip_emergency_pack');
  343 |     await generatePackBtn.click();
  344 | 
  345 |     const emergencyPackSection = pluginFrame.getByTestId('trip-emergency-pack');
  346 |     await expect(emergencyPackSection).toBeVisible();
  347 |     await expect(pluginFrame.getByTestId('emergency-pack-consented-count')).toHaveText('1');
  348 | 
  349 |     const manifestRow = pluginFrame.getByTestId(`emergency-manifest-row-${m1Seed.active_pupil_id}`);
  350 |     await expect(manifestRow).toBeVisible();
```