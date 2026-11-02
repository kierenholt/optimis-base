# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: platform/mvp_plugin.spec.ts >> MVP Plugin Sandboxed UI & Lua DAL Console Print >> getPupilRecord_seededPupilClickedInPluginUI_printsDatabaseRecordToConsole_test
- Location: tests/e2e/platform/mvp_plugin.spec.ts:110:7

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator:  getByTestId('plugin-iframe-uvxcor1o')
Expected: visible
Received: hidden
Timeout:  5000ms

Call log:
  - Expect "toBeVisible" getByTestId('plugin-iframe-uvxcor1o') with timeout 5000ms
  - waiting for getByTestId('plugin-iframe-uvxcor1o')
    14 × locator resolved to <iframe data-lru-active="false" sandbox="allow-scripts" src="/p/uvxcor1o/ui/index.html" title="Plugin Workspace: uvxcor1o" data-testid="plugin-iframe-uvxcor1o"></iframe>
       - unexpected value "hidden"

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
  - button "Staff Absence & Cover Manager"
  - button "Sparx Maths Partner Sync"
  - button "Medical Room & First Aid"
  - button "Settings"
  - button "Logout"
- main:
  - heading "Global Search & Discovery" [level=2]
  - text: Search plugin views, dashboard widgets, pupils, and teaching groups by name, action verb, or synonym
  - button "Full list"
  - tablist "Search record type filter":
    - tab "All" [selected]
    - tab "Pages"
    - tab "Widgets"
    - tab "Pupils"
    - tab "Groups"
  - searchbox "Search views, widgets, pupils, or groups (e.g. detention, ehcp, register, Ada Lovelace)..."
  - heading "Home Dashboard Widgets" [level=2]
  - button "Edit Dashboard Grid"
  - text: Statutory Attendance Pulse
  - iframe
```

# Test source

```ts
  17  |     testPlugId = '';
  18  |     logCursor = '';
  19  | 
  20  |     // 1. Tap Browser Console & Uncaught Page Errors
  21  |     page.on('console', (msg) => {
  22  |       browserLogs.push(`[browser:${msg.type()}] ${msg.text()}`);
  23  |     });
  24  |     page.on('pageerror', (err) => {
  25  |       pageErrors.push(`[pageerror] ${err.stack || err.message}`);
  26  |     });
  27  | 
  28  |     // 2. Tap Backend Service Logs (record cursor before test execution)
  29  |     const cursorRes = await request.get('/api/test/logs/cursor');
  30  |     if (!cursorRes.ok()) {
  31  |       throw new Error(`Failed to obtain service log cursor: ${cursorRes.status()}`);
  32  |     }
  33  |     logCursor = (await cursorRes.json()).cursor;
  34  | 
  35  |     // 3. Install and activate test_plugin at runtime for this test
  36  |     await request.post('/api/test/auth-mode', {
  37  |       data: { enable_dev_persona_switcher: true },
  38  |     });
  39  |     const installRes = await request.post('/api/admin/plugins/install', {
  40  |       data: { folder: 'test_plugin' },
  41  |     });
  42  |     if (!installRes.ok()) {
  43  |       throw new Error(`Failed to install test_plugin: ${installRes.status()}`);
  44  |     }
  45  |     testPlugId = (await installRes.json()).plug_id;
  46  |     const activateRes = await request.post(`/api/admin/plugins/${encodeURIComponent(testPlugId)}/activate`, {
  47  |       data: {},
  48  |     });
  49  |     if (!activateRes.ok()) {
  50  |       throw new Error(`Failed to activate test_plugin (${testPlugId}): ${activateRes.status()}`);
  51  |     }
  52  | 
  53  |     // 4. Seed required test data explicitly during setup
  54  |     const res = await request.post('/api/test/seed-pupil', {
  55  |       data: {
  56  |         upn: 'A123456789012',
  57  |         first_name: 'Ada',
  58  |         last_name: 'Lovelace',
  59  |         academic_year_id: '2026-2027',
  60  |         effective_from: '2026-09-01',
  61  |       },
  62  |     });
  63  |     if (!res.ok()) {
  64  |       throw new Error(`Failed to seed pupil fixture: ${res.status()}`);
  65  |     }
  66  |     const body = await res.json();
  67  |     if (!body.pupil_id) {
  68  |       throw new Error('Seeded pupil response missing mandatory pupil_id');
  69  |     }
  70  |     seededPupilId = body.pupil_id;
  71  |   });
  72  | 
  73  |   test.afterEach(async ({ request }, testInfo) => {
  74  |     // 1. Fetch Backend Service Logs emitted during this test
  75  |     if (logCursor) {
  76  |       const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
  77  |       if (logsRes.ok()) {
  78  |         const serviceLogs: string[] = (await logsRes.json()).entries;
  79  |         if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
  80  |           console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
  81  |         }
  82  |       }
  83  |     }
  84  | 
  85  |     // 2. Dump tapped Browser Console logs on failure or error
  86  |     if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
  87  |       console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
  88  |       console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
  89  |     }
  90  | 
  91  |     // 3. Remove runtime-installed plugin
  92  |     await request.post('/api/test/auth-mode', {
  93  |       data: { enable_dev_persona_switcher: true },
  94  |     });
  95  |     if (testPlugId) {
  96  |       await request.delete(`/api/admin/plugins/${encodeURIComponent(testPlugId)}`);
  97  |     }
  98  | 
  99  |     // 4. Always clean up seeded mock data at the end of the test
  100 |     if (seededPupilId) {
  101 |       const delRes = await request.delete(`/api/test/seed-pupil/${seededPupilId}`);
  102 |       if (!delRes.ok()) {
  103 |         throw new Error(`Failed to clean up seeded pupil ${seededPupilId}: ${delRes.status()}`);
  104 |       }
  105 |     }
  106 | 
  107 |     expect(pageErrors).toEqual([]);
  108 |   });
  109 | 
  110 |   test('getPupilRecord_seededPupilClickedInPluginUI_printsDatabaseRecordToConsole_test', async ({ page }) => {
  111 |     await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}`);
  112 | 
  113 |     const hostShell = page.getByTestId('host-shell');
  114 |     await expect(hostShell).toBeVisible();
  115 | 
  116 |     const iframeEl = page.getByTestId(`plugin-iframe-${testPlugId}`);
> 117 |     await expect(iframeEl).toBeVisible();
      |                            ^ Error: expect(locator).toBeVisible() failed
  118 |     await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');
  119 | 
  120 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
  121 |     const printBtn = pluginFrame.getByTestId('print-record-btn');
  122 |     await expect(printBtn).toBeVisible();
  123 |     await expect(printBtn).toHaveAttribute('data-mis-rpc', 'get_pupil_record');
  124 | 
  125 |     await printBtn.click();
  126 | 
  127 |     const expectedLogSubstring = `[plugin:uk.org.optimis.test] {"id":"${seededPupilId}","upn":"A123456789012","first_name":"Ada","last_name":"Lovelace"}`;
  128 | 
  129 |     await expect
  130 |       .poll(
  131 |         () => browserLogs.some((entry) => entry.includes(expectedLogSubstring)),
  132 |         {
  133 |           message: `Expected browser console logs to contain ${expectedLogSubstring}`,
  134 |           timeout: 5000,
  135 |         }
  136 |       )
  137 |       .toBe(true);
  138 |   });
  139 | });
  140 | 
```