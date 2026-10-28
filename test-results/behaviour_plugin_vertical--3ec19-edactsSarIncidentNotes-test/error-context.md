# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: behaviour_plugin_vertical.spec.ts >> Behaviour & Pastoral Points Plugin Vertical Slice (uk.org.optimis.behaviour) >> logBehaviourMeritAndIncident_teacherLogsEventsUpdatesDashboardWidgetDeepLinksStudentAndAdminRedactsSarIncidentNotes_test
- Location: tests/e2e/behaviour_plugin_vertical.spec.ts:208:7

# Error details

```
Error: expect(locator).toHaveText(expected) failed

Locator:  locator('[data-testid="plugin-iframe-uk.org.optimis.behaviour"]').contentFrame().getByTestId('behaviour-merit-total')
Expected: "3"
Received: "0"
Timeout:  5000ms

Call log:
  - Expect "toHaveText" locator('[data-testid="plugin-iframe-uk.org.optimis.behaviour"]').contentFrame().getByTestId('behaviour-merit-total') with timeout 5000ms
  - waiting for locator('[data-testid="plugin-iframe-uk.org.optimis.behaviour"]').contentFrame().getByTestId('behaviour-merit-total')
    14 × locator resolved to <div class="card-value" id="behaviour-merit-total" data-testid="behaviour-merit-total">0</div>
       - unexpected value "0"

```

```yaml
- text: "0"
```

# Test source

```ts
  178 |     }
  179 | 
  180 |     // 4. Remove runtime-installed plugins
  181 |     await request.post('/api/test/auth-mode', {
  182 |       data: { enable_dev_persona_switcher: true },
  183 |     });
  184 |     await request.delete('/api/admin/plugins/uk.org.optimis.behaviour');
  185 |     await request.delete('/api/admin/plugins/uk.org.optimis.test');
  186 | 
  187 |     // 5. Clean up seeded fixtures in core.db
  188 |     if (m1Seed?.fixture_id) {
  189 |       const delM1 = await request.delete(
  190 |         `/api/test/seed-milestone1-fixtures/${encodeURIComponent(m1Seed.fixture_id)}`
  191 |       );
  192 |       if (!delM1.ok()) {
  193 |         throw new Error(`Failed to clean up Milestone 1 fixture ${m1Seed.fixture_id}: ${delM1.status()}`);
  194 |       }
  195 |     }
  196 |     if (personaSeed?.fixture_id) {
  197 |       const delP = await request.delete(
  198 |         `/api/test/seed-personas/${encodeURIComponent(personaSeed.fixture_id)}`
  199 |       );
  200 |       if (!delP.ok()) {
  201 |         throw new Error(`Failed to clean up persona fixture ${personaSeed.fixture_id}: ${delP.status()}`);
  202 |       }
  203 |     }
  204 | 
  205 |     expect(pageErrors).toEqual([]);
  206 |   });
  207 | 
  208 |   test('logBehaviourMeritAndIncident_teacherLogsEventsUpdatesDashboardWidgetDeepLinksStudentAndAdminRedactsSarIncidentNotes_test', async ({
  209 |     page,
  210 |     request,
  211 |   }) => {
  212 |     if (!m1Seed || !personaSeed) {
  213 |       throw new Error('Missing mandatory seeded state');
  214 |     }
  215 | 
  216 |     // 1. Verify GET /api/system/ui-config exposes uk.org.optimis.behaviour and its behaviour_summary widget
  217 |     const uiConfigRes = await request.get(
  218 |       `/api/system/ui-config?staff_id=${encodeURIComponent(m1Seed.teacher_staff_id)}`
  219 |     );
  220 |     expect(uiConfigRes.ok()).toBe(true);
  221 |     const uiConfigBody = await uiConfigRes.json();
  222 |     expect(uiConfigBody.ok).toBe(true);
  223 | 
  224 |     const behaviourPlugin = uiConfigBody.plugins.find(
  225 |       (p: { id: string }) => p.id === 'uk.org.optimis.behaviour'
  226 |     );
  227 |     expect(behaviourPlugin).toBeDefined();
  228 |     expect(behaviourPlugin.widgets).toEqual([
  229 |       expect.objectContaining({
  230 |         id: 'behaviour_summary',
  231 |         plugin_id: 'uk.org.optimis.behaviour',
  232 |         entry: 'ui/dist/widget.html',
  233 |         entry_url: '/plugins/uk.org.optimis.behaviour/ui/widget.html',
  234 |       }),
  235 |     ]);
  236 | 
  237 |     // 2. Open Host UI Shell as the Classroom Teacher (who holds dal.pupils.read:own_groups for own_group_id)
  238 |     const params = new URLSearchParams({
  239 |       pupil_id: m1Seed.active_pupil_id,
  240 |       group_id: m1Seed.own_group_id,
  241 |       other_group_id: m1Seed.other_group_id,
  242 |       reference_date: m1Seed.reference_date,
  243 |       academic_year_id: m1Seed.academic_year_id,
  244 |       staff_id: m1Seed.teacher_staff_id,
  245 |     });
  246 |     await page.goto(`/?${params.toString()}`);
  247 | 
  248 |     await expect(page.getByTestId('host-shell')).toBeVisible();
  249 | 
  250 |     // 3. Switch to the Behaviour plugin workspace tab and log a Merit (+3) and an Incident (-2, containing a third-party pupil name in incident_notes)
  251 |     const behaviourNavBtn = page.getByTestId('nav-item-plugin-uk.org.optimis.behaviour');
  252 |     await expect(behaviourNavBtn).toBeVisible();
  253 |     await behaviourNavBtn.click();
  254 | 
  255 |     const behaviourIframeEl = page.getByTestId('plugin-iframe-uk.org.optimis.behaviour');
  256 |     await expect(behaviourIframeEl).toBeVisible();
  257 |     await expect(behaviourIframeEl).toHaveAttribute('sandbox', 'allow-scripts');
  258 | 
  259 |     const behaviourFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.behaviour"]');
  260 | 
  261 |     // Log a Merit (+3)
  262 |     const eventTypeSelect = behaviourFrame.getByTestId('behaviour-event-type-select');
  263 |     const categoryInput = behaviourFrame.getByTestId('behaviour-category-input');
  264 |     const pointsInput = behaviourFrame.getByTestId('behaviour-points-input');
  265 |     const notesInput = behaviourFrame.getByTestId('behaviour-incident-notes-input');
  266 |     const logBtn = behaviourFrame.getByTestId('log-behaviour-event-btn');
  267 | 
  268 |     await expect(eventTypeSelect).toBeVisible();
  269 |     await expect(logBtn).toBeVisible();
  270 |     await expect(logBtn).toHaveAttribute('data-mis-rpc', 'log_behaviour_event');
  271 | 
  272 |     await eventTypeSelect.selectOption('MERIT');
  273 |     await categoryInput.fill('Excellence in Mathematics');
  274 |     await pointsInput.fill('3');
  275 |     await notesInput.fill('Completed advanced polynomial factorisation extension independently.');
  276 |     await logBtn.click();
  277 | 
> 278 |     await expect(behaviourFrame.getByTestId('behaviour-merit-total')).toHaveText('3');
      |                                                                       ^ Error: expect(locator).toHaveText(expected) failed
  279 |     await expect(behaviourFrame.getByTestId('behaviour-net-points')).toHaveText('3');
  280 | 
  281 |     // Log an Incident (-2) with a third-party pupil name ("Draco Malfoy") in incident_notes
  282 |     await eventTypeSelect.selectOption('INCIDENT');
  283 |     await categoryInput.fill('Low-Level Disruption');
  284 |     await pointsInput.fill('-2');
  285 |     await notesInput.fill(
  286 |       'Argued loudly with third-party pupil Draco Malfoy over calculator sharing during starter activity.'
  287 |     );
  288 |     await logBtn.click();
  289 | 
  290 |     await expect(behaviourFrame.getByTestId('behaviour-merit-total')).toHaveText('3');
  291 |     await expect(behaviourFrame.getByTestId('behaviour-demerit-total')).toHaveText('-2');
  292 |     await expect(behaviourFrame.getByTestId('behaviour-net-points')).toHaveText('1');
  293 |     await expect(behaviourFrame.getByTestId('behaviour-events-list')).toContainText(
  294 |       'Argued loudly with third-party pupil Draco Malfoy'
  295 |     );
  296 | 
  297 |     // 4. Place the `behaviour_summary` Mini-Iframe Widget onto the Home Dashboard Grid and verify live points + student deep-link
  298 |     const editModeToggleBtn = page.getByTestId('dashboard-edit-mode-toggle-btn');
  299 |     await expect(editModeToggleBtn).toBeVisible();
  300 |     await editModeToggleBtn.click();
  301 | 
  302 |     const placeBehaviourWidgetBtn = page.getByTestId('place-widget-btn-behaviour_summary');
  303 |     await expect(placeBehaviourWidgetBtn).toBeVisible();
  304 |     await placeBehaviourWidgetBtn.click();
  305 | 
  306 |     await editModeToggleBtn.click();
  307 | 
  308 |     const widgetIframeEl = page.getByTestId('widget-iframe-behaviour_summary');
  309 |     await expect(widgetIframeEl).toBeVisible();
  310 |     await expect(widgetIframeEl).toHaveAttribute('sandbox', 'allow-scripts');
  311 | 
  312 |     const widgetFrame = page.frameLocator('[data-testid="widget-iframe-behaviour_summary"]');
  313 |     const widgetRefreshBtn = widgetFrame.getByTestId('widget-behaviour-refresh-btn');
  314 |     await expect(widgetRefreshBtn).toBeVisible();
  315 |     await widgetRefreshBtn.click();
  316 | 
  317 |     await expect(widgetFrame.getByTestId('widget-behaviour-merits')).toHaveText('3');
  318 |     await expect(widgetFrame.getByTestId('widget-behaviour-demerits')).toHaveText('-2');
  319 |     await expect(widgetFrame.getByTestId('widget-behaviour-net')).toHaveText('1');
  320 |     await expect(widgetFrame.getByTestId('widget-behaviour-pupil-name')).toContainText('Ada Lovelace');
  321 | 
  322 |     // Click the declarative `<a data-mis-link="student">` inside widget.html and verify it wakes the designated student handler plugin (`uk.org.optimis.test`) in the Keep-Alive LRU Pool
  323 |     const widgetStudentLink = widgetFrame.getByTestId('widget-behaviour-student-link');
  324 |     await expect(widgetStudentLink).toBeVisible();
  325 |     await expect(widgetStudentLink).toHaveAttribute('data-mis-link', 'student');
  326 |     await expect(widgetStudentLink).toHaveAttribute('data-pupil-id', m1Seed.active_pupil_id);
  327 | 
  328 |     await widgetStudentLink.click();
  329 | 
  330 |     await expect(page.getByTestId('plugin-iframe-uk.org.optimis.test')).toBeVisible();
  331 |     await expect(page.getByTestId('plugin-iframe-uk.org.optimis.behaviour')).toBeHidden();
  332 |     await expect(page.getByTestId('active-entity-context-pill')).toContainText(
  333 |       `student:${m1Seed.active_pupil_id}`
  334 |     );
  335 |     await expect(page.getByTestId('deep-link-navigation-status')).toContainText(
  336 |       `student:${m1Seed.active_pupil_id} -> uk.org.optimis.test`
  337 |     );
  338 | 
  339 |     // 5. Switch to System Admin / DPO persona, run UK GDPR SAR Export, and redact third-party pupil name ("Draco Malfoy") in `incident_notes`
  340 |     const switchAdminRes = await request.post('/api/auth/persona', {
  341 |       data: { staff_id: personaSeed.admin_staff_id },
  342 |     });
  343 |     expect(switchAdminRes.ok()).toBe(true);
  344 |     const adminSessionBody = await switchAdminRes.json();
  345 | 
  346 |     await page.context().addCookies([
  347 |       {
  348 |         name: 'mis_session',
  349 |         value: adminSessionBody.session_token,
  350 |         url: 'http://127.0.0.1:18080',
  351 |         sameSite: 'Strict',
  352 |       },
  353 |     ]);
  354 | 
  355 |     await page.goto(
  356 |       `/?pupil_id=${encodeURIComponent(m1Seed.active_pupil_id)}&reference_date=${encodeURIComponent(
  357 |         m1Seed.reference_date
  358 |       )}&academic_year_id=${encodeURIComponent(m1Seed.academic_year_id)}`
  359 |     );
  360 | 
  361 |     const sarPanel = page.getByTestId('sar-redaction-panel');
  362 |     await expect(sarPanel).toBeVisible();
  363 | 
  364 |     await page.getByTestId('sar-load-export-btn').click();
  365 |     await expect(page.getByTestId('sar-export-status')).toHaveText('SAR_LOADED');
  366 | 
  367 |     const behaviourSarSection = page.getByTestId(
  368 |       'sar-section-uk.org.optimis.behaviour-Plugin Behaviour & Incident Records'
  369 |     );
  370 |     await expect(behaviourSarSection).toBeVisible();
  371 | 
  372 |     const incidentNotesInput1 = page.getByTestId(
  373 |       'sar-redact-input-uk.org.optimis.behaviour-Plugin Behaviour & Incident Records-1-incident_notes'
  374 |     );
  375 |     await expect(incidentNotesInput1).toBeVisible();
  376 |     await expect(incidentNotesInput1).toHaveValue(
  377 |       'Argued loudly with third-party pupil Draco Malfoy over calculator sharing during starter activity.'
  378 |     );
```