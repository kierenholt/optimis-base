# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: platform/plugin_attached_db.spec.ts >> Per-Plugin Attached SQLite (plugin_test.db) & Migration Snapshots E2E >> saveAndQueryPluginNote_seededPupilInSandboxedPlugin_persistsToAttachedPluginDBAndEnforcesPragmas_test
- Location: tests/e2e/platform/plugin_attached_db.spec.ts:119:7

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
  53  |     // 4. Seed required test pupil explicitly during setup
  54  |     const res = await request.post('/api/test/seed-pupil', {
  55  |       data: {
  56  |         upn: 'P987654321001',
  57  |         first_name: 'Grace',
  58  |         last_name: 'Hopper',
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
  74  |     // 1. Purge any plugin_test.db rows created for seededPupilId during the test
  75  |     if (seededPupilId && testPlugId) {
  76  |       await request.post(`/api/p/${encodeURIComponent(testPlugId)}/rpc/purge_pupil_data`, {
  77  |         data: {
  78  |           args: { pupil_id: seededPupilId },
  79  |         },
  80  |       });
  81  |     }
  82  | 
  83  |     // 2. Fetch Backend Service Logs emitted during this test
  84  |     if (logCursor) {
  85  |       const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
  86  |       if (logsRes.ok()) {
  87  |         const serviceLogs: string[] = (await logsRes.json()).entries;
  88  |         if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
  89  |           console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
  90  |         }
  91  |       }
  92  |     }
  93  | 
  94  |     // 3. Dump tapped Browser Console logs on failure or error
  95  |     if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
  96  |       console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
  97  |       console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
  98  |     }
  99  | 
  100 |     // 4. Remove runtime-installed plugin
  101 |     await request.post('/api/test/auth-mode', {
  102 |       data: { enable_dev_persona_switcher: true },
  103 |     });
  104 |     if (testPlugId) {
  105 |       await request.delete(`/api/admin/plugins/${encodeURIComponent(testPlugId)}`);
  106 |     }
  107 | 
  108 |     // 5. Clean up seeded core pupil record at the end of the test
  109 |     if (seededPupilId) {
  110 |       const delRes = await request.delete(`/api/test/seed-pupil/${seededPupilId}`);
  111 |       if (!delRes.ok()) {
  112 |         throw new Error(`Failed to clean up seeded pupil ${seededPupilId}: ${delRes.status()}`);
  113 |       }
  114 |     }
  115 | 
  116 |     expect(pageErrors).toEqual([]);
  117 |   });
  118 | 
  119 |   test('saveAndQueryPluginNote_seededPupilInSandboxedPlugin_persistsToAttachedPluginDBAndEnforcesPragmas_test', async ({
  120 |     page,
  121 |     request,
  122 |   }) => {
  123 |     await page.goto(`/?pupil_id=${encodeURIComponent(seededPupilId)}`);
  124 | 
  125 |     const hostShell = page.getByTestId('host-shell');
  126 |     await expect(hostShell).toBeVisible();
  127 | 
  128 |     const iframeEl = page.getByTestId(`plugin-iframe-${testPlugId}`);
> 129 |     await expect(iframeEl).toBeVisible();
      |                            ^ Error: expect(locator).toBeVisible() failed
  130 |     await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');
  131 | 
  132 |     const pluginFrame = page.frameLocator(`[data-testid="plugin-iframe-${testPlugId}"]`);
  133 | 
  134 |     const saveNoteBtn = pluginFrame.getByTestId('save-plugin-note-btn');
  135 |     await expect(saveNoteBtn).toBeVisible();
  136 |     await expect(saveNoteBtn).toHaveAttribute('data-mis-rpc', 'save_pupil_note');
  137 | 
  138 |     await saveNoteBtn.click();
  139 | 
  140 |     const noteStatus = pluginFrame.getByTestId('plugin-note-status');
  141 |     await expect(noteStatus).toHaveText('Persisted 1 note(s) in plugin_test.db: Excellent progress in algebra');
  142 | 
  143 |     const loadNotesBtn = pluginFrame.getByTestId('load-plugin-notes-btn');
  144 |     await expect(loadNotesBtn).toBeVisible();
  145 |     await expect(loadNotesBtn).toHaveAttribute('data-mis-rpc', 'list_pupil_notes');
  146 | 
  147 |     await loadNotesBtn.click();
  148 | 
  149 |     const expectedLogSubstring = `[plugin:uk.org.optimis.test:plugin_db] {"pupil_id":"${seededPupilId}","count":1,"first_note":"Excellent progress in algebra","db_alias":"plugin_test","max_page_count":16384}`;
  150 | 
  151 |     await expect
  152 |       .poll(
  153 |         () => browserLogs.some((entry) => entry.includes(expectedLogSubstring)),
  154 |         {
  155 |           message: `Expected browser console logs to contain ${expectedLogSubstring}`,
  156 |           timeout: 5000,
  157 |         }
  158 |       )
  159 |       .toBe(true);
  160 | 
  161 |     // Verify GDPR export_pupil_data and purge_pupil_data against the attached plugin_test.db
  162 |     const exportRes = await request.post(`/api/p/${encodeURIComponent(testPlugId)}/rpc/export_pupil_data`, {
  163 |       data: {
  164 |         args: { pupil_id: seededPupilId },
  165 |       },
  166 |     });
  167 |     expect(exportRes.ok()).toBe(true);
  168 |     const exportBody = await exportRes.json();
  169 |     expect(exportBody.ok).toBe(true);
  170 |     expect(Array.isArray(exportBody.data)).toBe(true);
  171 |     expect(exportBody.data.length).toBe(1);
  172 |     expect(exportBody.data[0].title).toBe('Plugin Pupil Notes');
  173 |     expect(exportBody.data[0].records.length).toBe(1);
  174 |     expect(exportBody.data[0].records[0].note).toBe('Excellent progress in algebra');
  175 | 
  176 |     const purgeRes = await request.post(`/api/p/${encodeURIComponent(testPlugId)}/rpc/purge_pupil_data`, {
  177 |       data: {
  178 |         args: { pupil_id: seededPupilId },
  179 |       },
  180 |     });
  181 |     expect(purgeRes.ok()).toBe(true);
  182 |     const purgeBody = await purgeRes.json();
  183 |     expect(purgeBody.ok).toBe(true);
  184 | 
  185 |     const listAfterPurgeRes = await request.post(`/api/p/${encodeURIComponent(testPlugId)}/rpc/list_pupil_notes`, {
  186 |       data: {
  187 |         args: { pupil_id: seededPupilId },
  188 |       },
  189 |     });
  190 |     expect(listAfterPurgeRes.ok()).toBe(true);
  191 |     const listAfterPurgeBody = await listAfterPurgeRes.json();
  192 |     expect(listAfterPurgeBody.ok).toBe(true);
  193 |     expect(listAfterPurgeBody.data.count).toBe(0);
  194 |   });
  195 | });
  196 | 
```