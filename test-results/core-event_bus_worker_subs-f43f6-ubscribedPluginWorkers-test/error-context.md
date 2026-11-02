# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: core/event_bus_worker_subscriptions.spec.ts >> Native In-Memory Event Bus Router, Manifest Worker Subscriptions & Core Event Catalog (Sections 3, 9.1 & 10.2) >> recordAmAttendanceAndSafeguardingPostbox_coreDalCommits_emitsCoreCatalogEventsAndExecutesSubscribedPluginWorkers_test
- Location: tests/e2e/core/event_bus_worker_subscriptions.spec.ts:149:7

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator:  locator('[data-testid="plugin-iframe-uvxcor1o"]').contentFrame().getByTestId('record-am-attendance-btn')
Expected: visible
Received: hidden
Timeout:  5000ms

Call log:
  - Expect "toBeVisible" locator('[data-testid="plugin-iframe-uvxcor1o"]').contentFrame().getByTestId('record-am-attendance-btn') with timeout 5000ms
  - waiting for locator('[data-testid="plugin-iframe-uvxcor1o"]').contentFrame().getByTestId('record-am-attendance-btn')
    14 × locator resolved to <button id="record-am-attendance-btn" data-testid="record-am-attendance-btn" data-mis-rpc="record_group_am_attendance" data-mis-args="{"group_id":"grp_own_7bce7287","pupil_id":"pup_act_7bce7287","reference_date":"2026-10-27","academic_year_id":"ay_2026_2027","staff_id":"stf_001"}">↵        Record AM Attendance↵      </button>
       - unexpected value "hidden"

```

# Page snapshot

```yaml
- generic [ref=e3]:
  - complementary [ref=e4]:
    - generic "Oakridge All-Through Academy • URN 100001 • ay_2026_2027" [ref=e5]:
      - generic [ref=e6]: URN 100001
      - heading "Oakridge All-Through Academy" [level=1] [ref=e7]
      - generic [ref=e8]: ay_2026_2027
    - navigation "Primary Icon Rail" [ref=e9]:
      - button "Home" [ref=e10] [cursor=pointer]
      - button "Search" [ref=e15] [cursor=pointer]
    - generic [ref=e20]:
      - generic [ref=e21]: Pinned
      - button "Staff Absence & Cover Manager" [ref=e23] [cursor=pointer]
      - button "Sparx Maths Partner Sync" [ref=e30] [cursor=pointer]
      - button "Medical Room & First Aid" [ref=e35] [cursor=pointer]
    - generic [ref=e41]:
      - button "Settings" [ref=e42] [cursor=pointer]
      - button "Logout" [ref=e47] [cursor=pointer]
  - main [ref=e53]:
    - generic [ref=e54]:
      - generic [ref=e55]:
        - generic [ref=e56]:
          - generic [ref=e57]:
            - heading "Global Search & Discovery" [level=2] [ref=e58]
            - generic [ref=e63]: Search plugin views, dashboard widgets, pupils, and teaching groups by name, action verb, or synonym
          - button "Full list" [ref=e64] [cursor=pointer]
        - tablist "Search record type filter" [ref=e69]:
          - tab "All" [selected] [ref=e70] [cursor=pointer]
          - tab "Pages" [ref=e75] [cursor=pointer]
          - tab "Widgets" [ref=e80] [cursor=pointer]
          - tab "Pupils" [ref=e87] [cursor=pointer]
          - tab "Groups" [ref=e92] [cursor=pointer]
        - searchbox "Search views, widgets, pupils, or groups (e.g. detention, ehcp, register, Ada Lovelace)..." [ref=e99]
      - generic [ref=e100]:
        - generic [ref=e101]:
          - heading "Home Dashboard Widgets" [level=2] [ref=e103]
          - button "Edit Dashboard Grid" [ref=e111] [cursor=pointer]
        - generic [ref=e119]:
          - generic [ref=e120]: Statutory Attendance Pulse
          - iframe [ref=e126]:
            - generic [ref=f1e2]:
              - generic [ref=f1e3]:
                - generic [ref=f1e8]: THEME:light | PRIMARY:#1d4ed8
                - generic [ref=f1e9]: PUPIL:pup_act_7bce7287 | GROUP:grp_own_7bce7287
              - generic [ref=f1e10]:
                - button "Fetch Pupil via Widget RPC" [ref=f1e11] [cursor=pointer]
                - link "Open Secondary Student Workspace" [ref=f1e12] [cursor=pointer]:
                  - /url: "#widget-student-link"
```

# Test source

```ts
  80  |       !body.active_pupil_id ||
  81  |       !body.teacher_staff_id
  82  |     ) {
  83  |       throw new Error('Seeded fixture response missing mandatory identifiers');
  84  |     }
  85  |     seeded = body;
  86  |   });
  87  | 
  88  |   test.afterEach(async ({ request }, testInfo) => {
  89  |     // 1. Purge any plugin notes / worker event logs created for active_pupil_id
  90  |     if (seeded?.active_pupil_id && testPlugId) {
  91  |       await request.post(`/api/p/${testPlugId}/rpc/purge_pupil_data`, {
  92  |         data: {
  93  |           args: {
  94  |             pupil_id: seeded.active_pupil_id,
  95  |             staff_id: seeded.teacher_staff_id,
  96  |           },
  97  |         },
  98  |       });
  99  |     }
  100 | 
  101 |     // 2. Fetch Backend Service Logs emitted during this test
  102 |     if (logCursor) {
  103 |       const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
  104 |       if (logsRes.ok()) {
  105 |         const serviceLogs: string[] = (await logsRes.json()).entries;
  106 |         if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
  107 |           console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
  108 |         }
  109 |       }
  110 |     }
  111 | 
  112 |     // 3. Dump tapped Browser Console logs on failure or error
  113 |     if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
  114 |       console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
  115 |       console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
  116 |     }
  117 | 
  118 |     // 4. Remove runtime-installed plugin
  119 |     await request.post('/api/test/auth-mode', {
  120 |       data: { enable_dev_persona_switcher: true },
  121 |     });
  122 |     await request.delete('/api/admin/plugins/uk.org.optimis.test');
  123 | 
  124 |     // 5. Clean up seeded fixtures in core.db
  125 |     if (seeded?.fixture_id) {
  126 |       const delRes = await request.delete(
  127 |         `/api/test/seed-milestone1-fixtures/${encodeURIComponent(seeded.fixture_id)}`
  128 |       );
  129 |       if (!delRes.ok()) {
  130 |         throw new Error(`Failed to clean up fixture ${seeded.fixture_id}: ${delRes.status()}`);
  131 |       }
  132 |     }
  133 | 
  134 |     expect(pageErrors).toEqual([]);
  135 |   });
  136 | 
  137 |   function buildWorkspaceUrl(f: Milestone1SeedResponse): string {
  138 |     const params = new URLSearchParams({
  139 |       pupil_id: f.active_pupil_id,
  140 |       group_id: f.own_group_id,
  141 |       other_group_id: f.other_group_id,
  142 |       reference_date: f.reference_date,
  143 |       academic_year_id: f.academic_year_id,
  144 |       staff_id: f.teacher_staff_id,
  145 |     });
  146 |     return `/?${params.toString()}`;
  147 |   }
  148 | 
  149 |   test('recordAmAttendanceAndSafeguardingPostbox_coreDalCommits_emitsCoreCatalogEventsAndExecutesSubscribedPluginWorkers_test', async ({
  150 |     page,
  151 |     request,
  152 |   }) => {
  153 |     if (!seeded) {
  154 |       throw new Error('Missing mandatory seeded fixture state');
  155 |     }
  156 | 
  157 |     // 1. Verify the Standard Core Domain Event Catalog endpoint exposes core events & worker subscriptions
  158 |     const catalogRes = await request.get('/api/events/catalog');
  159 |     expect(catalogRes.ok()).toBe(true);
  160 |     const catalogBody = await catalogRes.json();
  161 |     expect(catalogBody.ok).toBe(true);
  162 |     expect(catalogBody.core_events).toEqual(
  163 |       expect.arrayContaining([
  164 |         'core.pupil.enrolled',
  165 |         'core.group.membership_changed',
  166 |         'core.attendance.session_recorded',
  167 |         'core.safeguarding.postbox_created',
  168 |         'core.gdpr.pupil_purged',
  169 |       ])
  170 |     );
  171 | 
  172 |     // 2. Open Host UI Shell and interact with the sandboxed plugin UI
  173 |     await page.goto(buildWorkspaceUrl(seeded));
  174 |     await expect(page.getByTestId('host-shell')).toBeVisible();
  175 | 
  176 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
  177 | 
  178 |     // 3. Record statutory AM attendance -> commits CoreDAL write -> emits core.attendance.session_recorded
  179 |     const recordAttendanceBtn = pluginFrame.getByTestId('record-am-attendance-btn');
> 180 |     await expect(recordAttendanceBtn).toBeVisible();
      |                                       ^ Error: expect(locator).toBeVisible() failed
  181 |     await recordAttendanceBtn.click();
  182 |     await expect(pluginFrame.getByTestId('attendance-result')).toContainText('SESSION_SAVED:AM');
  183 | 
  184 |     // 4. Submit safeguarding postbox -> commits CoreDAL write -> emits core.safeguarding.postbox_created
  185 |     const submitSafeguardingBtn = pluginFrame.getByTestId('submit-safeguarding-postbox-btn');
  186 |     await expect(submitSafeguardingBtn).toBeVisible();
  187 |     await submitSafeguardingBtn.click();
  188 |     await expect(pluginFrame.getByTestId('safeguarding-postbox-result')).toContainText('POSTBOX_ACCEPTED');
  189 | 
  190 |     // 5. Poll/load worker event logs via the declarative UI button until both async worker executions complete
  191 |     const loadWorkerLogsBtn = pluginFrame.getByTestId('load-worker-event-logs-btn');
  192 |     await expect(loadWorkerLogsBtn).toBeVisible();
  193 |     await expect(loadWorkerLogsBtn).toHaveAttribute('data-mis-rpc', 'list_worker_event_logs');
  194 | 
  195 |     const workerLogsResult = pluginFrame.getByTestId('worker-event-logs-result');
  196 |     await expect(async () => {
  197 |       await loadWorkerLogsBtn.click();
  198 |       await expect(workerLogsResult).toContainText('core.attendance.session_recorded');
  199 |       await expect(workerLogsResult).toContainText('core.safeguarding.postbox_created');
  200 |     }).toPass({ timeout: 5000 });
  201 | 
  202 |     const rawLogsText = await workerLogsResult.textContent();
  203 |     expect(rawLogsText).toBeTruthy();
  204 |     const parsed = JSON.parse(rawLogsText!);
  205 |     expect(parsed.pupil_id).toBe(seeded.active_pupil_id);
  206 |     expect(parsed.logs.length).toBeGreaterThanOrEqual(2);
  207 | 
  208 |     const attLog = parsed.logs.find(
  209 |       (l: { event_name: string }) => l.event_name === 'core.attendance.session_recorded'
  210 |     );
  211 |     expect(attLog).toBeTruthy();
  212 |     expect(attLog.hop).toBe(1);
  213 |     expect(attLog.caller_type).toBe('SYSTEM');
  214 |     expect(attLog.caller_staff_id).toBe('');
  215 | 
  216 |     const sfgLog = parsed.logs.find(
  217 |       (l: { event_name: string }) => l.event_name === 'core.safeguarding.postbox_created'
  218 |     );
  219 |     expect(sfgLog).toBeTruthy();
  220 |     expect(sfgLog.hop).toBe(1);
  221 |     expect(sfgLog.caller_type).toBe('SYSTEM');
  222 |     expect(sfgLog.caller_staff_id).toBe('');
  223 |     // Envelope metadata only — zero plaintext safeguarding narrative in event payload
  224 |     expect(JSON.stringify(sfgLog)).not.toContain('Pupil reported concern during registration');
  225 |   });
  226 | 
  227 |   test('triggerLoopRelay_recursiveWorkerReEmission_enforcesMaxHop3AndDropsHop4_test', async ({
  228 |     page,
  229 |     request,
  230 |   }) => {
  231 |     if (!seeded) {
  232 |       throw new Error('Missing mandatory seeded fixture state');
  233 |     }
  234 | 
  235 |     await page.goto(buildWorkspaceUrl(seeded));
  236 |     await expect(page.getByTestId('host-shell')).toBeVisible();
  237 | 
  238 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
  239 | 
  240 |     const triggerLoopBtn = pluginFrame.getByTestId('trigger-loop-relay-btn');
  241 |     await expect(triggerLoopBtn).toBeVisible();
  242 |     await expect(triggerLoopBtn).toHaveAttribute('data-mis-rpc', 'trigger_loop_relay');
  243 |     await triggerLoopBtn.click();
  244 | 
  245 |     const loadWorkerLogsBtn = pluginFrame.getByTestId('load-worker-event-logs-btn');
  246 |     const workerLogsResult = pluginFrame.getByTestId('worker-event-logs-result');
  247 | 
  248 |     // Wait until hop 3 is processed by the worker goroutine
  249 |     await expect(async () => {
  250 |       await loadWorkerLogsBtn.click();
  251 |       const txt = await workerLogsResult.textContent();
  252 |       expect(txt).toBeTruthy();
  253 |       const data = JSON.parse(txt!);
  254 |       const loopLogs = (data.logs || []).filter(
  255 |         (l: { event_name: string }) => l.event_name === 'test.loop_relay'
  256 |       );
  257 |       expect(loopLogs.length).toBe(3);
  258 |     }).toPass({ timeout: 5000 });
  259 | 
  260 |     const finalTxt = await workerLogsResult.textContent();
  261 |     const finalData = JSON.parse(finalTxt!);
  262 |     const loopLogs = (finalData.logs || []).filter(
  263 |       (l: { event_name: string }) => l.event_name === 'test.loop_relay'
  264 |     );
  265 | 
  266 |     // Must execute for Hop 1, Hop 2, and Hop 3, and drop Hop 4 (Hop > 3)
  267 |     expect(loopLogs.map((l: { hop: number }) => l.hop)).toEqual([1, 2, 3]);
  268 | 
  269 |     // Verify Host Event Bus metrics recorded the dropped event (events_dropped_total >= 1)
  270 |     const catalogRes = await request.get('/api/events/catalog');
  271 |     expect(catalogRes.ok()).toBe(true);
  272 |     const catalogBody = await catalogRes.json();
  273 |     expect(catalogBody.metrics.events_dropped_total).toBeGreaterThanOrEqual(1);
  274 |   });
  275 | });
  276 | 
```