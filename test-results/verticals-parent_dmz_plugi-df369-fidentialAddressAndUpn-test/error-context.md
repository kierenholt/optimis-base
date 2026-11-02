# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: verticals/parent_dmz_plugin_vertical.spec.ts >> Parent Portal DMZ Sync Plugin Vertical: Inbox Pull Safeguarding Blocks & Sanitized Read-Projection Push (Sections 1, 3, 5 & 9) >> pullDmzInboxAndPushSanitizedProjection_asAttendanceOfficer_acceptsUnrestrictedPrMotherRejectsCourtOrderFatherAndStripsConfidentialAddressAndUpn_test
- Location: tests/e2e/verticals/parent_dmz_plugin_vertical.spec.ts:227:7

# Error details

```
Error: expect(received).toBe(expected) // Object.is equality

Expected: true
Received: false
```

# Test source

```ts
  147 |         staff_id: personas.admin_staff_id,
  148 |       },
  149 |     });
  150 |     if (!installDmzRes.ok()) {
  151 |       throw new Error(`Failed to install parent_dmz_plugin: ${installDmzRes.status()} ${await installDmzRes.text()}`);
  152 |     }
  153 |     dmzPlugId = (await installDmzRes.json()).plug_id;
  154 |     const activateDmzRes = await request.post('/api/admin/plugins/uk.org.optimis.parent.dmz/activate', {
  155 |       data: { staff_id: personas.admin_staff_id },
  156 |     });
  157 |     if (!activateDmzRes.ok()) {
  158 |       throw new Error(`Failed to activate uk.org.optimis.parent.dmz: ${activateDmzRes.status()}`);
  159 |     }
  160 |   });
  161 | 
  162 |   test.afterEach(async ({ request }, testInfo) => {
  163 |     // 1. Restore mock external dependency modes & clear mock DMZ store
  164 |     await request.post('/api/test/external-deps-mode', {
  165 |       data: {
  166 |         enable_real_cloud_kms: false,
  167 |         enable_real_parent_dmz_sync: false,
  168 |         enable_real_mat_sync: false,
  169 |       },
  170 |     });
  171 | 
  172 |     // 2. Purge plugin data created during the test
  173 |     if (personas && m1Seeded && dmzPlugId) {
  174 |       await request.post('/api/auth/persona', {
  175 |         data: { staff_id: personas.admin_staff_id },
  176 |       });
  177 |       await request.post(`/api/p/${dmzPlugId}/rpc/purge_pupil_data`, {
  178 |         data: { args: { pupil_id: m1Seeded.active_pupil_id } },
  179 |       });
  180 |     }
  181 | 
  182 |     // 3. Fetch Backend Service Logs emitted during this test
  183 |     if (logCursor) {
  184 |       const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
  185 |       if (logsRes.ok()) {
  186 |         const serviceLogs: string[] = (await logsRes.json()).entries;
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
  199 |     // 5. Remove runtime-installed plugins
  200 |     await request.post('/api/test/auth-mode', {
  201 |       data: { enable_dev_persona_switcher: true },
  202 |     });
  203 |     await request.delete('/api/admin/plugins/uk.org.optimis.parent.dmz');
  204 |     await request.delete('/api/admin/plugins/uk.org.optimis.test');
  205 | 
  206 |     // 6. Clean up seeded fixtures in core.db
  207 |     if (m1Seeded?.fixture_id) {
  208 |       const delM1 = await request.delete(
  209 |         `/api/test/seed-milestone1-fixtures/${encodeURIComponent(m1Seeded.fixture_id)}`
  210 |       );
  211 |       if (!delM1.ok()) {
  212 |         throw new Error(`Failed to clean up Milestone 1 fixture ${m1Seeded.fixture_id}: ${delM1.status()}`);
  213 |       }
  214 |     }
  215 |     if (personas?.fixture_id) {
  216 |       const delP = await request.delete(
  217 |         `/api/test/seed-personas/${encodeURIComponent(personas.fixture_id)}`
  218 |       );
  219 |       if (!delP.ok()) {
  220 |         throw new Error(`Failed to clean up persona fixture ${personas.fixture_id}: ${delP.status()}`);
  221 |       }
  222 |     }
  223 | 
  224 |     expect(pageErrors).toEqual([]);
  225 |   });
  226 | 
  227 |   test('pullDmzInboxAndPushSanitizedProjection_asAttendanceOfficer_acceptsUnrestrictedPrMotherRejectsCourtOrderFatherAndStripsConfidentialAddressAndUpn_test', async ({
  228 |     page,
  229 |     request,
  230 |   }) => {
  231 |     if (!personas || !m1Seeded) {
  232 |       throw new Error('Missing mandatory seeded fixtures');
  233 |     }
  234 | 
  235 |     // 1. Record a statutory AM attendance session mark for Ada Lovelace so attendance sessions are populated
  236 |     const rpcAttRes = await request.post(`/api/p/${testPlugId}/rpc/record_group_am_attendance`, {
  237 |       data: {
  238 |         args: {
  239 |           group_id: m1Seeded.own_group_id,
  240 |           pupil_id: m1Seeded.active_pupil_id,
  241 |           reference_date: m1Seeded.reference_date,
  242 |           academic_year_id: m1Seeded.academic_year_id,
  243 |           staff_id: m1Seeded.teacher_staff_id,
  244 |         },
  245 |       },
  246 |     });
> 247 |     expect(rpcAttRes.ok()).toBe(true);
      |                            ^ Error: expect(received).toBe(expected) // Object.is equality
  248 | 
  249 |     // 2. Enqueue two untrusted parent absence notices in the Mock DMZ Write-Inbox:
  250 |     //    (a) Unrestricted Mother with Parental Responsibility (Anne Byron) -> MUST BE ACCEPTED
  251 |     //    (b) Court-order-restricted Father (George Byron, Prohibited Steps Order) -> MUST BE REJECTED_SAFEGUARDING_BLOCK
  252 |     const enqueueMomRes = await request.post('/api/test/parent-dmz/enqueue-inbox', {
  253 |       data: {
  254 |         pupil_id: m1Seeded.active_pupil_id,
  255 |         submitted_by_guardian_person_id: m1Seeded.unrestricted_mother_person_id,
  256 |         submission_type: 'ABSENCE_NOTICE',
  257 |         date: m1Seeded.reference_date,
  258 |         session: 'PM',
  259 |         reason: 'Hospital dental appointment at 14:00',
  260 |       },
  261 |     });
  262 |     expect(enqueueMomRes.status()).toBe(201);
  263 | 
  264 |     const enqueueDadRes = await request.post('/api/test/parent-dmz/enqueue-inbox', {
  265 |       data: {
  266 |         pupil_id: m1Seeded.active_pupil_id,
  267 |         submitted_by_guardian_person_id: m1Seeded.restricted_father_person_id,
  268 |         submission_type: 'ABSENCE_NOTICE',
  269 |         date: m1Seeded.reference_date,
  270 |         session: 'AM',
  271 |         reason: 'Unauthorized absence submission by court-order-blocked father',
  272 |       },
  273 |     });
  274 |     expect(enqueueDadRes.status()).toBe(201);
  275 | 
  276 |     // 3. Switch to Attendance Officer persona (SUPPORT role holding dal.pupils.read:all, dal.guardians.read:all, dal.attendance.read:all)
  277 |     const switchAttOfficerRes = await request.post('/api/auth/persona', {
  278 |       data: { staff_id: personas.attendance_officer_staff_id },
  279 |     });
  280 |     expect(switchAttOfficerRes.ok()).toBe(true);
  281 |     const attOfficerSession = await switchAttOfficerRes.json();
  282 |     expect(attOfficerSession.caller.active_roles).toContain('SUPPORT');
  283 |     expect(attOfficerSession.caller.scopes).toContain('dal.pupils.read:all');
  284 |     expect(attOfficerSession.caller.scopes).toContain('dal.guardians.read:all');
  285 |     expect(attOfficerSession.caller.scopes).toContain('dal.attendance.read:all');
  286 | 
  287 |     await page.context().addCookies([
  288 |       {
  289 |         name: 'mis_session',
  290 |         value: attOfficerSession.session_token,
  291 |         url: 'http://127.0.0.1:18080',
  292 |         sameSite: 'Strict',
  293 |       },
  294 |     ]);
  295 | 
  296 |     // 4. Navigate to Host UI Shell and open the Parent Portal DMZ Sync plugin iframe
  297 |     const query = new URLSearchParams({
  298 |       pupil_id: m1Seeded.active_pupil_id,
  299 |       reference_date: m1Seeded.reference_date,
  300 |       academic_year_id: m1Seeded.academic_year_id,
  301 |       staff_id: personas.attendance_officer_staff_id,
  302 |     });
  303 |     await page.goto(`/?${query.toString()}`);
  304 | 
  305 |     await expect(page.getByTestId('host-shell')).toBeVisible();
  306 |     await page.getByTestId('nav-rail-search').click();
  307 |     const dmzNavBtn = page.locator('[data-testid^="command-palette-result-view-"][data-testid$="-home"]').filter({ hasText: 'Parent' }).first();
  308 |     await expect(dmzNavBtn).toBeVisible();
  309 |     await dmzNavBtn.click();
  310 | 
  311 |     const iframeEl = page.getByTestId(`plugin-iframe-${dmzPlugId}`);
  312 |     await expect(iframeEl).toBeVisible();
  313 |     await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');
  314 | 
  315 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${dmzPlugId}"]`);
  316 |     await expect(pluginFrame.getByTestId('parent-dmz-plugin-root')).toBeVisible();
  317 | 
  318 |     // 5. Attendance Officer clicks "Pull Queued DMZ Write-Inbox" inside the sandboxed plugin UI
  319 |     const pullBtn = pluginFrame.getByTestId('pull-dmz-inbox-btn');
  320 |     await expect(pullBtn).toBeVisible();
  321 |     await expect(pullBtn).toHaveAttribute('data-mis-rpc', 'pull_dmz_inbox');
  322 |     await pullBtn.click();
  323 | 
  324 |     // Verify UI displays accepted=1 (unrestricted PR mother) and rejected=1 (court-order-blocked father with REJECTED_SAFEGUARDING_BLOCK)
  325 |     await expect(pluginFrame.getByTestId('dmz-inbox-summary')).toBeVisible();
  326 |     await expect(pluginFrame.getByTestId('dmz-accepted-count')).toHaveText('1');
  327 |     await expect(pluginFrame.getByTestId('dmz-rejected-count')).toHaveText('1');
  328 | 
  329 |     const momStatus = pluginFrame.getByTestId(
  330 |       `dmz-submission-status-${m1Seeded.unrestricted_mother_person_id}`
  331 |     );
  332 |     await expect(momStatus).toBeVisible();
  333 |     await expect(momStatus).toHaveText('ACCEPTED');
  334 | 
  335 |     const dadStatus = pluginFrame.getByTestId(
  336 |       `dmz-submission-status-${m1Seeded.restricted_father_person_id}`
  337 |     );
  338 |     await expect(dadStatus).toBeVisible();
  339 |     await expect(dadStatus).toHaveText('REJECTED_SAFEGUARDING_BLOCK');
  340 | 
  341 |     // 6. Attendance Officer clicks "Push Sanitized Read-Projection" inside the sandboxed plugin UI
  342 |     const pushBtn = pluginFrame.getByTestId('push-dmz-projection-btn');
  343 |     await expect(pushBtn).toBeVisible();
  344 |     await expect(pushBtn).toHaveAttribute('data-mis-rpc', 'push_sanitized_projection');
  345 |     await pushBtn.click();
  346 | 
  347 |     const projCard = pluginFrame.getByTestId('dmz-projection-card');
```