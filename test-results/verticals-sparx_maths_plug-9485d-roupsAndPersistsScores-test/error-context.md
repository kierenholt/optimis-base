# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: verticals/sparx_maths_plugin_vertical.spec.ts >> Sparx Maths Third-Party Partner Plugin Vertical: Two-Phase Read/Egress -> WriteDB Sync & Diagnostic API Key Scrubbing (Sections 4, 8 Gap 1, 9 & 10) >> configureVendorApiKeyAndTriggerTwoPhaseSync_adminConfiguresSealedKeyAndTeacherSyncsOwnGroupsAndPersistsScores_test
- Location: tests/e2e/verticals/sparx_maths_plugin_vertical.spec.ts:232:7

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Sparx' }).first()
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Sparx' }).first() with timeout 5000ms
  - waiting for locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Sparx' }).first()

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
      - tab "All (14)" [selected]
      - tab "Pages (8)"
      - tab "Widgets (0)"
      - tab "Pupils (4)"
      - tab "Groups (2)"
    - text: Views (8)
    - button "Educational Visits & Trip Consent Educational Visits & Trip Consent • /p/331qo66q (4 methods) Open View"
    - button "Pin Educational Visits & Trip Consent": Pin
    - button "Group Trip Consent & Manifest Educational Visits & Trip Consent • /p/331qo66q/group/:group_id (4 methods) Open View"
    - button "Pin Group Trip Consent & Manifest": Pin
    - button "Sparx Maths Partner Sync Sparx Maths Partner Sync • /p/x8ijanmh (4 methods) Open View"
    - button "Pin Sparx Maths Partner Sync": Pin
    - button "Group Homework Sync Sparx Maths Partner Sync • /p/x8ijanmh/group/:group_id (4 methods) Open View"
    - button "Pin Group Homework Sync": Pin
    - button "Staff Absence & Cover Manager Staff Absence & Cover Manager • /p/kj7wyoj1 (6 methods) Open View"
    - button "Pin Staff Absence & Cover Manager": Pin
    - button "Cover Class Roster Staff Absence & Cover Manager • /p/kj7wyoj1/group/:group_id (6 methods) Open View"
    - button "Pin Cover Class Roster": Pin
    - button "Timetable & Statutory Attendance Timetable & Statutory Attendance • /p/e454uy6j (3 methods) Open View"
    - button "Pin Timetable & Statutory Attendance": Pin
    - button "Class Session Register Timetable & Statutory Attendance • /p/e454uy6j/group/:group_id (3 methods) Open View"
    - button "Pin Class Session Register": Pin
    - text: Pupils (4)
    - button "Charles Babbage UPN U2B3F9DB31002 • Year 10 Open Student"
    - button "Ada Lovelace UPN U1B3F9DB31001 • Year 10 Open Student"
    - button "Ada Lovelace UPN PC88D5A61D401 • Year 7 Open Student"
    - button "Alan Turing UPN U3B3F9DB31003 • Year 10 Open Student"
    - text: Groups (2)
    - button "10Ma1_b3f9 — Year 10 Maths Set 1 TEACHING_CLASS Open Group"
    - button "10Ma2_b3f9 — Year 10 Maths Set 2 TEACHING_CLASS Open Group"
```

# Test source

```ts
  216 |       if (!delM1.ok()) {
  217 |         throw new Error(`Failed to clean up Milestone 1 fixture ${m1Seeded.fixture_id}: ${delM1.status()}`);
  218 |       }
  219 |     }
  220 |     if (personas?.fixture_id) {
  221 |       const delP = await request.delete(
  222 |         `/api/test/seed-personas/${encodeURIComponent(personas.fixture_id)}`
  223 |       );
  224 |       if (!delP.ok()) {
  225 |         throw new Error(`Failed to clean up persona fixture ${personas.fixture_id}: ${delP.status()}`);
  226 |       }
  227 |     }
  228 | 
  229 |     expect(pageErrors).toEqual([]);
  230 |   });
  231 | 
  232 |   test('configureVendorApiKeyAndTriggerTwoPhaseSync_adminConfiguresSealedKeyAndTeacherSyncsOwnGroupsAndPersistsScores_test', async ({
  233 |     page,
  234 |     request,
  235 |   }) => {
  236 |     if (!personas || !m1Seeded) {
  237 |       throw new Error('Missing mandatory seeded fixtures');
  238 |     }
  239 | 
  240 |     const vendorApiKey = 'spx_live_secret_9f8e7d6c5b4a3210_do_not_leak';
  241 | 
  242 |     // 1. Register mock egress response for https://sync.sparxmaths.uk/v1/homework/sync
  243 |     //    Returning homework completion scores for Ada Lovelace (active_pupil_id in own_group_id)
  244 |     const mockRuleRes = await request.post('/api/test/egress-mock', {
  245 |       data: {
  246 |         method: 'POST',
  247 |         url: SPARX_SYNC_URL,
  248 |         status: 200,
  249 |         headers: { 'Content-Type': 'application/json' },
  250 |         body: JSON.stringify({
  251 |           sync_batch_id: 'sparx_batch_20261028_01',
  252 |           homework_week: '2026-W44',
  253 |           scores: [
  254 |             {
  255 |               pupil_id: m1Seeded.active_pupil_id,
  256 |               homework_week: '2026-W44',
  257 |               completion_rate: 96,
  258 |               xp_earned: 420,
  259 |               bookwork_accuracy: 92,
  260 |               teacher_note: 'Ada Lovelace completed compulsory and XP Boost tasks ahead of deadline',
  261 |             },
  262 |           ],
  263 |         }),
  264 |       },
  265 |     });
  266 |     expect(mockRuleRes.status()).toBe(201);
  267 | 
  268 |     // 2. Verify RBAC (1D): Regular Classroom Teacher without ADMIN/HOD role or plugin.config.write scope
  269 |     //    cannot call configure_vendor_credentials (403 Forbidden)
  270 |     const switchTeacherFirstRes = await request.post('/api/auth/persona', {
  271 |       data: { staff_id: m1Seeded.teacher_staff_id },
  272 |     });
  273 |     expect(switchTeacherFirstRes.ok()).toBe(true);
  274 | 
  275 |     const teacherUnauthCfgRes = await request.post(
  276 |       `/api/p/${sparxPlugId}/rpc/configure_vendor_credentials`,
  277 |       {
  278 |         data: {
  279 |           args: {
  280 |             api_key: vendorApiKey,
  281 |             sync_endpoint_url: SPARX_SYNC_URL,
  282 |           },
  283 |         },
  284 |       }
  285 |     );
  286 |     expect(teacherUnauthCfgRes.status()).toBe(403);
  287 | 
  288 |     // 3. Authenticate as System Admin (holding ADMIN role + plugin.config.write) to configure sealed vendor API key via UI
  289 |     const switchAdminRes = await request.post('/api/auth/persona', {
  290 |       data: { staff_id: personas.admin_staff_id },
  291 |     });
  292 |     expect(switchAdminRes.ok()).toBe(true);
  293 |     const adminSession = await switchAdminRes.json();
  294 | 
  295 |     await page.context().addCookies([
  296 |       {
  297 |         name: 'mis_session',
  298 |         value: adminSession.session_token,
  299 |         url: 'http://127.0.0.1:18080',
  300 |         sameSite: 'Strict',
  301 |       },
  302 |     ]);
  303 | 
  304 |     const adminQuery = new URLSearchParams({
  305 |       group_id: m1Seeded.own_group_id,
  306 |       pupil_id: m1Seeded.active_pupil_id,
  307 |       reference_date: m1Seeded.reference_date,
  308 |       academic_year_id: m1Seeded.academic_year_id,
  309 |       staff_id: personas.admin_staff_id,
  310 |     });
  311 |     await page.goto(`/?${adminQuery.toString()}`);
  312 | 
  313 |     await expect(page.getByTestId('host-shell')).toBeVisible();
  314 |     await page.getByTestId('nav-rail-search').click();
  315 |     const sparxNavBtn = page.locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Sparx' }).first();
> 316 |     await expect(sparxNavBtn).toBeVisible();
      |                               ^ Error: expect(locator).toBeVisible() failed
  317 |     await sparxNavBtn.click();
  318 | 
  319 |     const iframeEl = page.getByTestId(`plugin-iframe-${sparxPlugId}`);
  320 |     await expect(iframeEl).toBeVisible();
  321 |     await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');
  322 | 
  323 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${sparxPlugId}"]`);
  324 |     await expect(pluginFrame.getByTestId('sparx-maths-plugin-root')).toBeVisible();
  325 | 
  326 |     // 4. Phase 1 Setup: Configure vendor API key in plugin_sparx.db (sealed via ctx.crypto.seal_secret & committed before HTTP call)
  327 |     const apiKeyInput = pluginFrame.getByTestId('sparx-api-key-input');
  328 |     await expect(apiKeyInput).toBeVisible();
  329 |     await apiKeyInput.fill(vendorApiKey);
  330 | 
  331 |     const saveKeyBtn = pluginFrame.getByTestId('save-sparx-config-btn');
  332 |     await expect(saveKeyBtn).toBeVisible();
  333 |     await expect(saveKeyBtn).toHaveAttribute('data-mis-rpc', 'configure_vendor_credentials');
  334 |     await saveKeyBtn.click();
  335 | 
  336 |     const configStatusBadge = pluginFrame.getByTestId('sparx-config-status');
  337 |     await expect(configStatusBadge).toBeVisible();
  338 |     await expect(configStatusBadge).toContainText('CONFIGURED');
  339 |     await expect(pluginFrame.getByTestId('sparx-key-preview')).toContainText('spx_...3210');
  340 |     await expect(pluginFrame.getByTestId('sparx-key-preview')).not.toContainText(vendorApiKey);
  341 | 
  342 |     // Verify get_vendor_config confirms the key is stored sealed (enc:v1:...) and never exposes plaintext
  343 |     const getCfgRes = await request.post(`/api/p/${sparxPlugId}/rpc/get_vendor_config`, {
  344 |       data: { args: {} },
  345 |     });
  346 |     expect(getCfgRes.ok()).toBe(true);
  347 |     const getCfgBody = await getCfgRes.json();
  348 |     expect(getCfgBody.data.status).toBe('CONFIGURED');
  349 |     expect(getCfgBody.data.is_sealed_envelope).toBe(true);
  350 |     expect(JSON.stringify(getCfgBody)).not.toContain(vendorApiKey);
  351 | 
  352 |     // 5. Switch to Classroom Teacher (assigned to own_group_id, holding :own_groups scopes) to trigger daily class sync
  353 |     const switchTeacherRes = await request.post('/api/auth/persona', {
  354 |       data: { staff_id: m1Seeded.teacher_staff_id },
  355 |     });
  356 |     expect(switchTeacherRes.ok()).toBe(true);
  357 |     const teacherSession = await switchTeacherRes.json();
  358 | 
  359 |     await page.context().addCookies([
  360 |       {
  361 |         name: 'mis_session',
  362 |         value: teacherSession.session_token,
  363 |         url: 'http://127.0.0.1:18080',
  364 |         sameSite: 'Strict',
  365 |       },
  366 |     ]);
  367 | 
  368 |     const teacherQuery = new URLSearchParams({
  369 |       group_id: m1Seeded.own_group_id,
  370 |       pupil_id: m1Seeded.active_pupil_id,
  371 |       reference_date: m1Seeded.reference_date,
  372 |       academic_year_id: m1Seeded.academic_year_id,
  373 |       staff_id: m1Seeded.teacher_staff_id,
  374 |     });
  375 |     await page.goto(`/?${teacherQuery.toString()}`);
  376 |     await page.getByTestId('nav-rail-search').click();
  377 |     await page.locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Sparx' }).first().click();
  378 | 
  379 |     // 6. Phase 2 Sync: Trigger two-phase Read/Egress -> WriteDB sync via UI button
  380 |     //    Opens sealed key via ctx.crypto.open_secret, fetches :own_groups roster (Ada Lovelace in own_group_id;
  381 |     //    excludes transferred Alan Turing & other-group Grace Hopper), posts outbound to
  382 |     //    https://sync.sparxmaths.uk/v1/homework/sync with zero open SQLite write transaction,
  383 |     //    and persists returned homework completion scores into plugin_sparx.db via ctx.plugin_db.exec_batch.
  384 |     const syncBtn = pluginFrame.getByTestId('trigger-sparx-sync-btn');
  385 |     await expect(syncBtn).toBeVisible();
  386 |     await expect(syncBtn).toHaveAttribute('data-mis-rpc', 'sync_group_homework');
  387 |     await syncBtn.click();
  388 | 
  389 |     const syncSummary = pluginFrame.getByTestId('sparx-sync-summary');
  390 |     await expect(syncSummary).toBeVisible();
  391 |     await expect(pluginFrame.getByTestId('sparx-sync-status-badge')).toHaveText('SYNCED');
  392 |     await expect(pluginFrame.getByTestId('sparx-sync-batch-id')).toHaveText('sparx_batch_20261028_01');
  393 |     await expect(pluginFrame.getByTestId('sparx-synced-roster-count')).toHaveText('1');
  394 |     await expect(pluginFrame.getByTestId('sparx-persisted-scores-count')).toHaveText('1');
  395 |     await expect(pluginFrame.getByTestId('sparx-two-phase-tx-badge')).toHaveText('READ_EGRESS_THEN_WRITEDB_OK');
  396 | 
  397 |     // Verify persisted score row in the UI table
  398 |     const scoreRow = pluginFrame.getByTestId(`sparx-score-row-${m1Seeded.active_pupil_id}`);
  399 |     await expect(scoreRow).toBeVisible();
  400 |     await expect(scoreRow).toContainText('Ada Lovelace');
  401 |     await expect(scoreRow).toContainText('2026-W44');
  402 |     await expect(
  403 |       pluginFrame.getByTestId(`sparx-score-completion-${m1Seeded.active_pupil_id}`)
  404 |     ).toHaveText('96%');
  405 |     await expect(
  406 |       pluginFrame.getByTestId(`sparx-score-xp-${m1Seeded.active_pupil_id}`)
  407 |     ).toHaveText('420');
  408 |     await expect(
  409 |       pluginFrame.getByTestId(`sparx-score-bookwork-${m1Seeded.active_pupil_id}`)
  410 |     ).toHaveText('92%');
  411 | 
  412 |     // Verify transferred pupil (Alan Turing) and other-group pupil (Grace Hopper) were NOT included in the :own_groups sync
  413 |     await expect(
  414 |       pluginFrame.getByTestId(`sparx-score-row-${m1Seeded.transferred_pupil_id}`)
  415 |     ).toHaveCount(0);
  416 |     await expect(
```