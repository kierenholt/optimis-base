# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: core/http_egress_engine.spec.ts >> Real & Mock ctx.http Egress Engine (Sections 4, 9.2 & 10.1) >> ctxHttpGetAndPost_approvedDomainWithMockEngine_returnsStatusHeadersBodyAndFollowsApprovedRedirect_test
- Location: tests/e2e/core/http_egress_engine.spec.ts:140:7

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator:  locator('[data-testid="plugin-iframe-uvxcor1o"]').contentFrame().getByTestId('egress-sync-get-btn')
Expected: visible
Received: hidden
Timeout:  5000ms

Call log:
  - Expect "toBeVisible" locator('[data-testid="plugin-iframe-uvxcor1o"]').contentFrame().getByTestId('egress-sync-get-btn') with timeout 5000ms
  - waiting for locator('[data-testid="plugin-iframe-uvxcor1o"]').contentFrame().getByTestId('egress-sync-get-btn')
    14 × locator resolved to <button id="egress-sync-get-btn" data-testid="egress-sync-get-btn" data-mis-rpc="sync_external_sparx_get" data-mis-args="{"pupil_id":"pup_act_106b4e89","target_url":"https://sync.sparxmaths.uk/v1/start","staff_id":"stf_t1_106b4e89"}">↵        Sync Sparx GET↵      </button>
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
    - generic [ref=e20]: Pinned
    - generic [ref=e22]:
      - button "Settings" [ref=e23] [cursor=pointer]
      - button "Logout" [ref=e28] [cursor=pointer]
  - main [ref=e34]:
    - generic [ref=e35]:
      - generic [ref=e36]:
        - generic [ref=e37]:
          - generic [ref=e38]:
            - heading "Global Search & Discovery" [level=2] [ref=e39]
            - generic [ref=e44]: Search plugin views, dashboard widgets, pupils, and teaching groups by name, action verb, or synonym
          - button "Full list" [ref=e45] [cursor=pointer]
        - tablist "Search record type filter" [ref=e50]:
          - tab "All" [selected] [ref=e51] [cursor=pointer]
          - tab "Pages" [ref=e56] [cursor=pointer]
          - tab "Widgets" [ref=e61] [cursor=pointer]
          - tab "Pupils" [ref=e68] [cursor=pointer]
          - tab "Groups" [ref=e73] [cursor=pointer]
        - searchbox "Search views, widgets, pupils, or groups (e.g. detention, ehcp, register, Ada Lovelace)..." [ref=e80]
      - generic [ref=e81]:
        - generic [ref=e82]:
          - heading "Home Dashboard Widgets" [level=2] [ref=e84]
          - button "Edit Dashboard Grid" [ref=e92] [cursor=pointer]
        - generic [ref=e100]:
          - generic [ref=e101]: Statutory Attendance Pulse
          - iframe [ref=e107]:
            - generic [ref=f1e2]:
              - generic [ref=f1e3]:
                - generic [ref=f1e8]: THEME:light | PRIMARY:#1d4ed8
                - generic [ref=f1e9]: PUPIL:pup_act_106b4e89 | GROUP:grp_own_106b4e89
              - generic [ref=f1e10]:
                - button "Fetch Pupil via Widget RPC" [ref=f1e11] [cursor=pointer]
                - link "Open Secondary Student Workspace" [ref=f1e12] [cursor=pointer]:
                  - /url: "#widget-student-link"
```

# Test source

```ts
  97  |     await request.delete('/api/test/egress-mock');
  98  |     if (seeded?.active_pupil_id && testPlugId) {
  99  |       await request.post(`/api/p/${testPlugId}/rpc/purge_pupil_data`, {
  100 |         data: { args: { pupil_id: seeded.active_pupil_id } },
  101 |       });
  102 |     }
  103 | 
  104 |     // 2. Fetch Backend Service Logs emitted during this test
  105 |     if (logCursor) {
  106 |       const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
  107 |       if (logsRes.ok()) {
  108 |         const serviceLogs: string[] = (await logsRes.json()).entries;
  109 |         if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
  110 |           console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
  111 |         }
  112 |       }
  113 |     }
  114 | 
  115 |     // 3. Dump tapped Browser Console logs on failure or error
  116 |     if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
  117 |       console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
  118 |       console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
  119 |     }
  120 | 
  121 |     // 4. Remove runtime-installed plugin
  122 |     await request.post('/api/test/auth-mode', {
  123 |       data: { enable_dev_persona_switcher: true },
  124 |     });
  125 |     await request.delete('/api/admin/plugins/uk.org.optimis.test');
  126 | 
  127 |     // 5. Clean up seeded fixtures
  128 |     if (seeded?.fixture_id) {
  129 |       const delRes = await request.delete(
  130 |         `/api/test/seed-milestone1-fixtures/${encodeURIComponent(seeded.fixture_id)}`
  131 |       );
  132 |       if (!delRes.ok()) {
  133 |         throw new Error(`Failed to clean up fixture ${seeded.fixture_id}: ${delRes.status()}`);
  134 |       }
  135 |     }
  136 | 
  137 |     expect(pageErrors).toEqual([]);
  138 |   });
  139 | 
  140 |   test('ctxHttpGetAndPost_approvedDomainWithMockEngine_returnsStatusHeadersBodyAndFollowsApprovedRedirect_test', async ({
  141 |     page,
  142 |     request,
  143 |   }) => {
  144 |     if (!seeded) {
  145 |       throw new Error('Missing mandatory seeded state');
  146 |     }
  147 | 
  148 |     // 1. Register a 302 redirect on an approved domain (https://sync.sparxmaths.uk/v1/start -> https://sync.sparxmaths.uk/v1/final)
  149 |     const mockRedirRes = await request.post('/api/test/egress-mock', {
  150 |       data: {
  151 |         method: 'GET',
  152 |         url: 'https://sync.sparxmaths.uk/v1/start',
  153 |         status: 302,
  154 |         redirect_to: 'https://sync.sparxmaths.uk/v1/final',
  155 |       },
  156 |     });
  157 |     expect(mockRedirRes.ok()).toBe(true);
  158 | 
  159 |     const mockFinalRes = await request.post('/api/test/egress-mock', {
  160 |       data: {
  161 |         method: 'GET',
  162 |         url: 'https://sync.sparxmaths.uk/v1/final',
  163 |         status: 200,
  164 |         headers: { 'X-Sparx-Sync': 'verified-redirect' },
  165 |         body: '{"sync_status":"SPARX_REDIRECT_OK","homework_completion":94}',
  166 |       },
  167 |     });
  168 |     expect(mockFinalRes.ok()).toBe(true);
  169 | 
  170 |     // 2. Register a POST mock rule on https://sync.sparxmaths.uk/v1/push
  171 |     const mockPostRes = await request.post('/api/test/egress-mock', {
  172 |       data: {
  173 |         method: 'POST',
  174 |         url: 'https://sync.sparxmaths.uk/v1/push',
  175 |         status: 201,
  176 |         headers: { 'X-Sparx-Receipt': 'rcpt_901' },
  177 |         body: '{"receipt_id":"rcpt_901","accepted":true}',
  178 |       },
  179 |     });
  180 |     expect(mockPostRes.ok()).toBe(true);
  181 | 
  182 |     // 3. Load Host Shell UI with sandboxed plugin iframe and trigger GET sync via UI button
  183 |     const query = new URLSearchParams({
  184 |       pupil_id: seeded.active_pupil_id,
  185 |       group_id: seeded.own_group_id,
  186 |       other_group_id: seeded.other_group_id,
  187 |       reference_date: seeded.reference_date,
  188 |       academic_year_id: seeded.academic_year_id,
  189 |     });
  190 |     await page.goto(`/?${query.toString()}`);
  191 | 
  192 |     const hostShell = page.getByTestId('host-shell');
  193 |     await expect(hostShell).toBeVisible();
  194 | 
  195 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
  196 |     const syncGetBtn = pluginFrame.getByTestId('egress-sync-get-btn');
> 197 |     await expect(syncGetBtn).toBeVisible();
      |                              ^ Error: expect(locator).toBeVisible() failed
  198 |     await syncGetBtn.click();
  199 | 
  200 |     const syncResult = pluginFrame.getByTestId('egress-sync-result');
  201 |     await expect(syncResult).toContainText('SPARX_REDIRECT_OK');
  202 |     await expect(syncResult).toContainText('https://sync.sparxmaths.uk/v1/final');
  203 | 
  204 |     // 4. Trigger POST sync via UI button and assert status 201 + response payload
  205 |     const syncPostBtn = pluginFrame.getByTestId('egress-sync-post-btn');
  206 |     await expect(syncPostBtn).toBeVisible();
  207 |     await syncPostBtn.click();
  208 | 
  209 |     await expect(syncResult).toContainText('rcpt_901');
  210 |     await expect(syncResult).toContainText('"status":201');
  211 |   });
  212 | 
  213 |   test('ctxHttpGet_redirectToUnapprovedDomain_rejectsRedirectInCheckRedirectAndFailsFast_test', async ({
  214 |     page,
  215 |     request,
  216 |   }) => {
  217 |     if (!seeded) {
  218 |       throw new Error('Missing mandatory seeded state');
  219 |     }
  220 | 
  221 |     // Configure initial approved domain URL to redirect (302) to an unapproved external domain
  222 |     const mockRedirRes = await request.post('/api/test/egress-mock', {
  223 |       data: {
  224 |         method: 'GET',
  225 |         url: 'https://sync.sparxmaths.uk/v1/start',
  226 |         status: 302,
  227 |         redirect_to: 'https://evil.unapproved-domain.org/exfiltrate',
  228 |       },
  229 |     });
  230 |     expect(mockRedirRes.ok()).toBe(true);
  231 | 
  232 |     const query = new URLSearchParams({
  233 |       pupil_id: seeded.active_pupil_id,
  234 |       group_id: seeded.own_group_id,
  235 |       other_group_id: seeded.other_group_id,
  236 |       reference_date: seeded.reference_date,
  237 |       academic_year_id: seeded.academic_year_id,
  238 |     });
  239 |     await page.goto(`/?${query.toString()}`);
  240 | 
  241 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
  242 |     const syncGetBtn = pluginFrame.getByTestId('egress-sync-get-btn');
  243 |     await expect(syncGetBtn).toBeVisible();
  244 |     await syncGetBtn.click();
  245 | 
  246 |     const errorBanner = pluginFrame.getByTestId('rpc-error-banner');
  247 |     await expect(errorBanner).toBeVisible();
  248 |     await expect(errorBanner).toContainText('egress redirect target domain not permitted by manifest.json');
  249 |   });
  250 | 
  251 |   test('ctxHttpPost_undeclaredDomainOrOpenWriteTransaction_rollsBackWriteTxAndFailsFast_test', async ({
  252 |     request,
  253 |   }) => {
  254 |     if (!seeded) {
  255 |       throw new Error('Missing mandatory seeded state');
  256 |     }
  257 | 
  258 |     // 1. Calling ctx.http.post to an undeclared domain must fail fast
  259 |     const unapprovedRes = await request.post(`/api/p/${testPlugId}/rpc/push_external_sparx_post`, {
  260 |       data: {
  261 |         args: {
  262 |           pupil_id: seeded.active_pupil_id,
  263 |           target_url: 'https://unapproved.vendor.io/webhook',
  264 |           payload: '{"event":"test"}',
  265 |           staff_id: seeded.teacher_staff_id,
  266 |         },
  267 |       },
  268 |     });
  269 |     expect(unapprovedRes.status()).toBe(400);
  270 |     const unapprovedBody = await unapprovedRes.json();
  271 |     expect(unapprovedBody.ok).toBe(false);
  272 |     expect(unapprovedBody.error_code).toContain('egress domain not permitted by manifest.json');
  273 | 
  274 |     // 2. Prefix-spoofed domain (https://sync.sparxmaths.uk.attacker.com) must also fail fast
  275 |     const spoofedRes = await request.post(`/api/p/${testPlugId}/rpc/push_external_sparx_post`, {
  276 |       data: {
  277 |         args: {
  278 |           pupil_id: seeded.active_pupil_id,
  279 |           target_url: 'https://sync.sparxmaths.uk.attacker.com/v1/push',
  280 |           payload: '{"event":"test"}',
  281 |           staff_id: seeded.teacher_staff_id,
  282 |         },
  283 |       },
  284 |     });
  285 |     expect(spoofedRes.status()).toBe(400);
  286 |     const spoofedBody = await spoofedRes.json();
  287 |     expect(spoofedBody.ok).toBe(false);
  288 |     expect(spoofedBody.error_code).toContain('egress domain not permitted by manifest.json');
  289 | 
  290 |     // 3. Invoking ctx.http inside an open plugin_db write transaction must halt/rollback the write transaction and fail fast
  291 |     const writeTxRes = await request.post(`/api/p/${testPlugId}/rpc/mutate_then_illegal_egress`, {
  292 |       data: {
  293 |         args: {
  294 |           pupil_id: seeded.active_pupil_id,
  295 |           note_id: 'note_should_rollback_1',
  296 |           note: 'Uncommitted note before illegal HTTP call',
  297 |           target_url: 'https://sync.sparxmaths.uk/v1/push',
```