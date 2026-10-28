import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

interface Milestone1SeedResponse {
  fixture_id: string;
  academic_year_id: string;
  reference_date: string;
  teacher_staff_id: string;
  other_teacher_staff_id: string;
  dsl_staff_id: string;
  own_group_id: string;
  other_group_id: string;
  active_pupil_id: string;
  transferred_pupil_id: string;
  other_group_pupil_id: string;
}

interface PersonaSeedResponse {
  fixture_id: string;
  pupil_id: string;
  admin_staff_id: string;
  dsl_staff_id: string;
  hod_staff_id: string;
  teacher_staff_id: string;
  terminated_staff_id: string;
}

test.describe('Behaviour & Pastoral Points Plugin Vertical Slice (uk.org.optimis.behaviour)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let m1Seed: Milestone1SeedResponse | null = null;
  let personaSeed: PersonaSeedResponse | null = null;

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    m1Seed = null;
    personaSeed = null;

    // 1. Tap Browser Console & Uncaught Page Errors
    page.on('console', (msg) => {
      browserLogs.push(`[browser:${msg.type()}] ${msg.text()}`);
    });
    page.on('pageerror', (err) => {
      pageErrors.push(`[pageerror] ${err.stack || err.message}`);
    });

    // 2. Tap Backend Service Logs (record cursor before test execution)
    const cursorRes = await request.get('/api/test/logs/cursor');
    if (!cursorRes.ok()) {
      throw new Error(`Failed to obtain service log cursor: ${cursorRes.status()}`);
    }
    logCursor = (await cursorRes.json()).cursor;

    // 3. Ensure dev persona switcher is enabled
    const authModeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!authModeRes.ok()) {
      throw new Error(`Failed to enable dev persona switcher: ${authModeRes.status()}`);
    }

    // 4. Seed Admin/DPO persona fixtures and Milestone 1 own_groups / other_groups fixtures
    const pSeedRes = await request.post('/api/test/seed-personas', {
      data: {
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    if (!pSeedRes.ok()) {
      throw new Error(`Failed to seed persona fixtures: ${pSeedRes.status()}`);
    }
    personaSeed = (await pSeedRes.json()) as PersonaSeedResponse;
    if (!personaSeed.fixture_id || !personaSeed.admin_staff_id) {
      throw new Error('Persona seed response missing mandatory identifiers');
    }

    const m1Res = await request.post('/api/test/seed-milestone1-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-27',
      },
    });
    if (!m1Res.ok()) {
      throw new Error(`Failed to seed Milestone 1 fixtures: ${m1Res.status()}`);
    }
    m1Seed = (await m1Res.json()) as Milestone1SeedResponse;
    if (
      !m1Seed.fixture_id ||
      !m1Seed.teacher_staff_id ||
      !m1Seed.own_group_id ||
      !m1Seed.other_group_id ||
      !m1Seed.active_pupil_id ||
      !m1Seed.other_group_pupil_id
    ) {
      throw new Error('Milestone 1 seed response missing mandatory identifiers');
    }

    // 5. Switch to Admin persona to install and activate both test_plugin (student deep-link handler) and behaviour_plugin (uk.org.optimis.behaviour)
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personaSeed.admin_staff_id },
    });
    if (!switchAdminRes.ok()) {
      throw new Error(`Failed to switch to Admin persona in setup: ${switchAdminRes.status()}`);
    }

    const installTestRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'test_plugin', staff_id: personaSeed.admin_staff_id },
    });
    if (!installTestRes.ok()) {
      throw new Error(`Failed to install test_plugin: ${installTestRes.status()}`);
    }
    const activateTestRes = await request.post('/api/admin/plugins/uk.org.optimis.test/activate', {
      data: { staff_id: personaSeed.admin_staff_id },
    });
    if (!activateTestRes.ok()) {
      throw new Error(`Failed to activate uk.org.optimis.test: ${activateTestRes.status()}`);
    }

    const installBehaviourRes = await request.post('/api/admin/plugins/install', {
      data: { folder: 'behaviour_plugin', staff_id: personaSeed.admin_staff_id },
    });
    if (!installBehaviourRes.ok()) {
      throw new Error(
        `Failed to install behaviour_plugin: ${installBehaviourRes.status()} ${await installBehaviourRes.text()}`
      );
    }
    const activateBehaviourRes = await request.post(
      '/api/admin/plugins/uk.org.optimis.behaviour/activate',
      {
        data: { staff_id: personaSeed.admin_staff_id },
      }
    );
    if (!activateBehaviourRes.ok()) {
      throw new Error(`Failed to activate uk.org.optimis.behaviour: ${activateBehaviourRes.status()}`);
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Clean up behaviour_events in plugin_behaviour.db
    if (personaSeed && m1Seed) {
      await request.post('/api/auth/persona', {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      await request.post('/api/admin/plugins/uk.org.optimis.behaviour/activate', {
        data: { staff_id: personaSeed.admin_staff_id },
      });
      for (const pid of [
        m1Seed.active_pupil_id,
        m1Seed.transferred_pupil_id,
        m1Seed.other_group_pupil_id,
      ]) {
        await request.post('/api/plugins/uk.org.optimis.behaviour/rpc/purge_pupil_data', {
          data: { args: { pupil_id: pid } },
        });
      }
    }

    // 2. Fetch Backend Service Logs emitted during this test
    if (logCursor) {
      const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
      if (logsRes.ok()) {
        const serviceLogs: string[] = (await logsRes.json()).entries;
        if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
          console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
        }
      }
    }

    // 3. Dump tapped Browser Console logs on failure or error
    if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
      console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
      console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
    }

    // 4. Remove runtime-installed plugins
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete('/api/admin/plugins/uk.org.optimis.behaviour');
    await request.delete('/api/admin/plugins/uk.org.optimis.test');

    // 5. Clean up seeded fixtures in core.db
    if (m1Seed?.fixture_id) {
      const delM1 = await request.delete(
        `/api/test/seed-milestone1-fixtures/${encodeURIComponent(m1Seed.fixture_id)}`
      );
      if (!delM1.ok()) {
        throw new Error(`Failed to clean up Milestone 1 fixture ${m1Seed.fixture_id}: ${delM1.status()}`);
      }
    }
    if (personaSeed?.fixture_id) {
      const delP = await request.delete(
        `/api/test/seed-personas/${encodeURIComponent(personaSeed.fixture_id)}`
      );
      if (!delP.ok()) {
        throw new Error(`Failed to clean up persona fixture ${personaSeed.fixture_id}: ${delP.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('logBehaviourMeritAndIncident_teacherLogsEventsUpdatesDashboardWidgetDeepLinksStudentAndAdminRedactsSarIncidentNotes_test', async ({
    page,
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded state');
    }

    // 1. Verify GET /api/system/ui-config exposes uk.org.optimis.behaviour and its behaviour_summary widget
    const uiConfigRes = await request.get(
      `/api/system/ui-config?staff_id=${encodeURIComponent(m1Seed.teacher_staff_id)}`
    );
    expect(uiConfigRes.ok()).toBe(true);
    const uiConfigBody = await uiConfigRes.json();
    expect(uiConfigBody.ok).toBe(true);

    const behaviourPlugin = uiConfigBody.plugins.find(
      (p: { id: string }) => p.id === 'uk.org.optimis.behaviour'
    );
    expect(behaviourPlugin).toBeDefined();
    expect(behaviourPlugin.widgets).toEqual([
      expect.objectContaining({
        id: 'behaviour_summary',
        plugin_id: 'uk.org.optimis.behaviour',
        entry: 'ui/dist/widget.html',
        entry_url: '/plugins/uk.org.optimis.behaviour/ui/widget.html',
      }),
    ]);

    // 2. Open Host UI Shell as the Classroom Teacher (who holds dal.pupils.read:own_groups for own_group_id)
    const params = new URLSearchParams({
      pupil_id: m1Seed.active_pupil_id,
      group_id: m1Seed.own_group_id,
      other_group_id: m1Seed.other_group_id,
      reference_date: m1Seed.reference_date,
      academic_year_id: m1Seed.academic_year_id,
      staff_id: m1Seed.teacher_staff_id,
    });
    await page.goto(`/?${params.toString()}`);

    await expect(page.getByTestId('host-shell')).toBeVisible();

    // 3. Place the `behaviour_summary` Mini-Iframe Widget onto the Home Dashboard Grid FIRST so it is mounted before logging events (verifying 1H `mis:plugin:invalidated` live auto-refresh)
    const editModeToggleBtn = page.getByTestId('dashboard-edit-mode-toggle-btn');
    await expect(editModeToggleBtn).toBeVisible();
    await editModeToggleBtn.click();

    const placeBehaviourWidgetBtn = page.getByTestId('place-widget-btn-behaviour_summary');
    await expect(placeBehaviourWidgetBtn).toBeVisible();
    await placeBehaviourWidgetBtn.click();

    await editModeToggleBtn.click();

    const widgetIframeEl = page.getByTestId('widget-iframe-behaviour_summary');
    await expect(widgetIframeEl).toBeVisible();
    await expect(widgetIframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const widgetFrame = page.frameLocator('[data-testid="widget-iframe-behaviour_summary"]');
    await expect(widgetFrame.getByTestId('widget-behaviour-merits')).toHaveText('0');

    // 4. Switch to the Behaviour plugin workspace tab and log a Merit (+3) and an Incident (-2, containing a third-party pupil name in incident_notes)
    const behaviourNavBtn = page.getByTestId('nav-item-plugin-uk.org.optimis.behaviour');
    await expect(behaviourNavBtn).toBeVisible();
    await behaviourNavBtn.click();

    const behaviourIframeEl = page.getByTestId('plugin-iframe-uk.org.optimis.behaviour');
    await expect(behaviourIframeEl).toBeVisible();
    await expect(behaviourIframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const behaviourFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.behaviour"]');

    // Log a Merit (+3)
    const eventTypeSelect = behaviourFrame.getByTestId('behaviour-event-type-select');
    const categoryInput = behaviourFrame.getByTestId('behaviour-category-input');
    const pointsInput = behaviourFrame.getByTestId('behaviour-points-input');
    const notesInput = behaviourFrame.getByTestId('behaviour-incident-notes-input');
    const logBtn = behaviourFrame.getByTestId('log-behaviour-event-btn');

    await expect(eventTypeSelect).toBeVisible();
    await expect(logBtn).toBeVisible();
    await expect(logBtn).toHaveAttribute('data-mis-rpc', 'log_behaviour_event');

    await eventTypeSelect.selectOption('MERIT');
    await categoryInput.fill('Excellence in Mathematics');
    await pointsInput.fill('3');
    await notesInput.fill('Completed advanced polynomial factorisation extension independently.');

    // Verify 1H: RPC args do not leak staff_id (identity is propagated via X-MIS-Staff-ID header by Host UI Shell)
    const rawArgsAttr = (await logBtn.getAttribute('data-mis-args')) || '{}';
    expect(JSON.parse(rawArgsAttr)).not.toHaveProperty('staff_id');

    await logBtn.click();

    await expect(behaviourFrame.getByTestId('behaviour-merit-total')).toHaveText('3');
    await expect(behaviourFrame.getByTestId('behaviour-net-points')).toHaveText('3');

    // Log an Incident (-2) with a third-party pupil name ("Draco Malfoy") in incident_notes
    await eventTypeSelect.selectOption('INCIDENT');
    await categoryInput.fill('Low-Level Disruption');
    await pointsInput.fill('-2');
    await notesInput.fill(
      'Argued loudly with third-party pupil Draco Malfoy over calculator sharing during starter activity.'
    );
    await logBtn.click();

    await expect(behaviourFrame.getByTestId('behaviour-merit-total')).toHaveText('3');
    await expect(behaviourFrame.getByTestId('behaviour-demerit-total')).toHaveText('-2');
    await expect(behaviourFrame.getByTestId('behaviour-net-points')).toHaveText('1');
    await expect(behaviourFrame.getByTestId('behaviour-events-list')).toContainText(
      'Argued loudly with third-party pupil Draco Malfoy'
    );

    // 5. Verify 1H: The mounted `behaviour_summary` Mini-Iframe Widget automatically refreshed via `mis:plugin:invalidated` without clicking Refresh
    await expect(widgetFrame.getByTestId('widget-behaviour-merits')).toHaveText('3');
    await expect(widgetFrame.getByTestId('widget-behaviour-demerits')).toHaveText('-2');
    await expect(widgetFrame.getByTestId('widget-behaviour-net')).toHaveText('1');
    await expect(widgetFrame.getByTestId('widget-behaviour-pupil-name')).toContainText('Ada Lovelace');

    // Click the declarative `<a data-mis-link="student">` inside widget.html and verify it wakes the designated student handler plugin (`uk.org.optimis.test`) in the Keep-Alive LRU Pool
    const widgetStudentLink = widgetFrame.getByTestId('widget-behaviour-student-link');
    await expect(widgetStudentLink).toBeVisible();
    await expect(widgetStudentLink).toHaveAttribute('data-mis-link', 'student');
    await expect(widgetStudentLink).toHaveAttribute('data-pupil-id', m1Seed.active_pupil_id);

    await widgetStudentLink.click();

    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.test')).toBeVisible();
    await expect(page.getByTestId('plugin-iframe-uk.org.optimis.behaviour')).toBeHidden();
    await expect(page.getByTestId('active-entity-context-pill')).toContainText(
      `student:${m1Seed.active_pupil_id}`
    );
    await expect(page.getByTestId('deep-link-navigation-status')).toContainText(
      `student:${m1Seed.active_pupil_id} -> uk.org.optimis.test`
    );

    // 6. Switch to System Admin / DPO persona, run UK GDPR SAR Export, and redact third-party pupil name ("Draco Malfoy") in `incident_notes`
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personaSeed.admin_staff_id },
    });
    expect(switchAdminRes.ok()).toBe(true);
    const adminSessionBody = await switchAdminRes.json();

    await page.context().addCookies([
      {
        name: 'mis_session',
        value: adminSessionBody.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    await page.goto(
      `/?pupil_id=${encodeURIComponent(m1Seed.active_pupil_id)}&reference_date=${encodeURIComponent(
        m1Seed.reference_date
      )}&academic_year_id=${encodeURIComponent(m1Seed.academic_year_id)}`
    );

    const sarPanel = page.getByTestId('sar-redaction-panel');
    await expect(sarPanel).toBeVisible();

    await page.getByTestId('sar-load-export-btn').click();
    await expect(page.getByTestId('sar-export-status')).toHaveText('SAR_LOADED');

    const behaviourSarSection = page.getByTestId(
      'sar-section-uk.org.optimis.behaviour-Plugin Behaviour & Incident Records'
    );
    await expect(behaviourSarSection).toBeVisible();

    const incidentNotesInput1 = page.getByTestId(
      'sar-redact-input-uk.org.optimis.behaviour-Plugin Behaviour & Incident Records-1-incident_notes'
    );
    await expect(incidentNotesInput1).toBeVisible();
    await expect(incidentNotesInput1).toHaveValue(
      'Argued loudly with third-party pupil Draco Malfoy over calculator sharing during starter activity.'
    );

    // Redact the third-party pupil name in `incident_notes`
    await incidentNotesInput1.fill(
      'Argued loudly with third-party pupil [REDACTED THIRD PARTY] over calculator sharing during starter activity.'
    );

    await page.getByTestId('sar-finalize-export-btn').click();
    await expect(page.getByTestId('sar-export-status')).toContainText('SAR_EXPORT_FINALIZED');

    const finalizedJsonEl = page.getByTestId('sar-finalized-json');
    await expect(finalizedJsonEl).toBeVisible();
    const rawExportJson = (await finalizedJsonEl.textContent()) || '';
    expect(rawExportJson).not.toContain('Draco Malfoy');
    expect(rawExportJson).toContain('[REDACTED THIRD PARTY]');
    expect(rawExportJson).toContain('Completed advanced polynomial factorisation extension independently.');
  });

  test('behaviourPluginOwnGroupsAndGdprPurge_teacherBlockedFromOtherGroupPupilAndPurgeRemovesAllRows_test', async ({
    request,
  }) => {
    if (!m1Seed || !personaSeed) {
      throw new Error('Missing mandatory seeded state');
    }

    // 1. Verify strict `:own_groups` scoping (`strict_own_groups: true`): Teacher (`m1Seed.teacher_staff_id`) attempting to log behaviour for a pupil in another teacher's group (`m1Seed.other_group_pupil_id`) under strict_own_groups MUST fail fast
    const unauthorizedLogRes = await request.post(
      `/api/plugins/uk.org.optimis.behaviour/rpc/log_behaviour_event?staff_id=${encodeURIComponent(
        m1Seed.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            pupil_id: m1Seed.other_group_pupil_id,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
            strict_own_groups: true,
            event_type: 'INCIDENT',
            category: 'Homework Incomplete',
            points: -1,
            incident_notes: 'Attempted cross-group write by teacher under strict_own_groups.',
          },
        },
      }
    );
    expect(unauthorizedLogRes.status()).toBe(400);
    const unauthorizedLogBody = await unauthorizedLogRes.json();
    expect(unauthorizedLogBody.ok).toBe(false);
    expect(unauthorizedLogBody.error_code).toContain('access denied under own_groups scope');

    // 1B: Verify Hierarchical Scope Subsumption: HoD (`personaSeed.hod_staff_id`, who holds `dal.pupils.read:all`) calling the `:own_groups` endpoint WITH `strict_own_groups: true` succeeds because `:all` subsumes `:own_groups`
    const hodLogRes = await request.post(
      `/api/plugins/uk.org.optimis.behaviour/rpc/log_behaviour_event?staff_id=${encodeURIComponent(
        personaSeed.hod_staff_id
      )}`,
      {
        data: {
          args: {
            pupil_id: m1Seed.other_group_pupil_id,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
            strict_own_groups: true,
            event_type: 'INCIDENT',
            category: 'Corridor Conduct',
            points: -1,
            incident_notes: 'HoD logged corridor conduct incident via :all scope subsumption.',
          },
        },
      }
    );
    expect(hodLogRes.ok()).toBe(true);
    const hodLogBody = await hodLogRes.json();
    expect(hodLogBody.ok).toBe(true);
    // 1F: Verify UUID v4 primary key generated via ctx.crypto.uuid()
    expect(hodLogBody.data.logged_event_id).toMatch(/^bev_[0-9a-f-]{36}$/);

    // Also verify group-transferred pupil (group_pupil_memberships.effective_to < reference_date) is blocked under `strict_own_groups: true`
    const transferredSummaryRes = await request.post(
      `/api/plugins/uk.org.optimis.behaviour/rpc/get_pupil_behaviour_summary?staff_id=${encodeURIComponent(
        m1Seed.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            pupil_id: m1Seed.transferred_pupil_id,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
            strict_own_groups: true,
          },
        },
      }
    );
    expect(transferredSummaryRes.status()).toBe(400);
    const transferredSummaryBody = await transferredSummaryRes.json();
    expect(transferredSummaryBody.ok).toBe(false);
    expect(transferredSummaryBody.error_code).toContain('access denied under own_groups scope');

    // 2. Teacher logs a valid event for own-group `active_pupil_id`, verifies batch rollup (`1E` `get_by_ids_batch`), then Admin executes auto-bound `export_pupil_data` & `purge_pupil_data` (`1G`) and verifies 0 orphaned records remain
    const logValidRes = await request.post(
      `/api/plugins/uk.org.optimis.behaviour/rpc/log_behaviour_event?staff_id=${encodeURIComponent(
        m1Seed.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            pupil_id: m1Seed.active_pupil_id,
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
            event_type: 'MERIT',
            category: 'Peer Mentoring',
            points: 2,
            incident_notes: 'Helped classmate understand simultaneous equations.',
          },
        },
      }
    );
    expect(logValidRes.ok()).toBe(true);

    // 1E: Verify batch pupil behaviour lookup using `ctx.dal.pupils.get_by_ids_batch`
    const batchSummaryRes = await request.post(
      `/api/plugins/uk.org.optimis.behaviour/rpc/get_pupil_behaviour_summary?staff_id=${encodeURIComponent(
        m1Seed.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            pupil_ids: [m1Seed.active_pupil_id, m1Seed.other_group_pupil_id],
            academic_year_id: m1Seed.academic_year_id,
            reference_date: m1Seed.reference_date,
          },
        },
      }
    );
    expect(batchSummaryRes.ok()).toBe(true);
    const batchSummaryBody = await batchSummaryRes.json();
    expect(batchSummaryBody.ok).toBe(true);
    expect(Array.isArray(batchSummaryBody.data.pupils)).toBe(true);
    expect(batchSummaryBody.data.pupils.length).toBe(2);

    const exportBeforePurgeRes = await request.post(
      `/api/plugins/uk.org.optimis.behaviour/rpc/export_pupil_data?staff_id=${encodeURIComponent(
        personaSeed.admin_staff_id
      )}`,
      {
        data: { args: { pupil_id: m1Seed.active_pupil_id } },
      }
    );
    expect(exportBeforePurgeRes.ok()).toBe(true);
    const exportBeforeBody = await exportBeforePurgeRes.json();
    expect(exportBeforeBody.ok).toBe(true);
    expect(exportBeforeBody.data[0].records.length).toBe(1);
    expect(exportBeforeBody.data[0].redactable_fields).toEqual(['incident_notes']);

    const purgeRes = await request.post(
      `/api/plugins/uk.org.optimis.behaviour/rpc/purge_pupil_data?staff_id=${encodeURIComponent(
        personaSeed.admin_staff_id
      )}`,
      {
        data: { args: { pupil_id: m1Seed.active_pupil_id } },
      }
    );
    expect(purgeRes.ok()).toBe(true);
    const purgeBody = await purgeRes.json();
    expect(purgeBody.ok).toBe(true);
    expect(purgeBody.data.purged_rows).toBe(1);

    const exportAfterPurgeRes = await request.post(
      `/api/plugins/uk.org.optimis.behaviour/rpc/export_pupil_data?staff_id=${encodeURIComponent(
        personaSeed.admin_staff_id
      )}`,
      {
        data: { args: { pupil_id: m1Seed.active_pupil_id } },
      }
    );
    expect(exportAfterPurgeRes.ok()).toBe(true);
    const exportAfterBody = await exportAfterPurgeRes.json();
    expect(exportAfterBody.data[0].records.length).toBe(0);
  });
});
