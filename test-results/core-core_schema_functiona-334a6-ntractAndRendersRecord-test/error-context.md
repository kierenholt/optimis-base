# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: core/core_schema_functional.spec.ts >> Optimis Core Schema (45 Tables) End-to-End Functional Invariants >> verifyCoreSchemaInvariants_seededPupilLoadedInHostShell_enforcesDalContractAndRendersRecord_test
- Location: tests/e2e/core/core_schema_functional.spec.ts:108:7

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
  17  |     seededPupilId = '';
  18  |     testPlugId = '';
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
  46  |     const activateRes = await request.post('/api/admin/plugins/uk.org.optimis.test/activate', {
  47  |       data: {},
  48  |     });
  49  |     if (!activateRes.ok()) {
  50  |       throw new Error(`Failed to activate uk.org.optimis.test: ${activateRes.status()}`);
  51  |     }
  52  | 
  53  |     // 4. Seed required test pupil fixture explicitly in setup
  54  |     const seedRes = await request.post('/api/test/seed-pupil', {
  55  |       data: {
  56  |         upn: 'H823400026001',
  57  |         first_name: 'Alan',
  58  |         last_name: 'Turing',
  59  |         academic_year_id: 'ay_2026_2027',
  60  |         effective_from: '2026-09-01',
  61  |       },
  62  |     });
  63  |     if (!seedRes.ok()) {
  64  |       throw new Error(`Failed to seed pupil fixture: ${seedRes.status()}`);
  65  |     }
  66  |     const body = await seedRes.json();
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
  95  |     await request.delete('/api/admin/plugins/uk.org.optimis.test');
  96  | 
  97  |     // 4. Clean up seeded test data at the end of the test
  98  |     if (seededPupilId) {
  99  |       const delRes = await request.delete(`/api/test/seed-pupil/${seededPupilId}`);
  100 |       if (!delRes.ok()) {
  101 |         throw new Error(`Failed to clean up seeded pupil ${seededPupilId}: ${delRes.status()}`);
  102 |       }
  103 |     }
  104 | 
  105 |     expect(pageErrors).toEqual([]);
  106 |   });
  107 | 
  108 |   test('verifyCoreSchemaInvariants_seededPupilLoadedInHostShell_enforcesDalContractAndRendersRecord_test', async ({
  109 |     page,
  110 |   }) => {
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
  127 |     const expectedLogSubstring = `[plugin:uk.org.optimis.test] {"id":"${seededPupilId}","upn":"H823400026001","first_name":"Alan","last_name":"Turing"}`;
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