import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

interface Milestone1SeedResponse {
  fixture_id: string;
  academic_year_id: string;
  reference_date: string;
  teacher_staff_id: string;
  other_teacher_staff_id: string;
  own_group_id: string;
  other_group_id: string;
  active_pupil_id: string;
  transferred_pupil_id: string;
  other_group_pupil_id: string;
}

test.describe('Native In-Memory Event Bus Router, Manifest Worker Subscriptions & Core Event Catalog (Sections 3, 9.1 & 10.2)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let seeded: Milestone1SeedResponse | null = null;

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    seeded = null;

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

    // 3. Seed isolated test fixtures explicitly in setup
    const seedRes = await request.post('/api/test/seed-milestone1-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-27',
      },
    });
    if (!seedRes.ok()) {
      throw new Error(`Failed to seed Milestone 1 fixtures: ${seedRes.status()}`);
    }
    const body = (await seedRes.json()) as Milestone1SeedResponse;
    if (
      !body.fixture_id ||
      !body.own_group_id ||
      !body.active_pupil_id ||
      !body.teacher_staff_id
    ) {
      throw new Error('Seeded fixture response missing mandatory identifiers');
    }
    seeded = body;
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Purge any plugin notes / worker event logs created for active_pupil_id
    if (seeded?.active_pupil_id) {
      await request.post('/api/plugins/uk.org.optimis.test/rpc/purge_pupil_data', {
        data: {
          args: {
            pupil_id: seeded.active_pupil_id,
            staff_id: seeded.teacher_staff_id,
          },
        },
      });
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

    // 4. Clean up seeded fixtures in core.db
    if (seeded?.fixture_id) {
      const delRes = await request.delete(
        `/api/test/seed-milestone1-fixtures/${encodeURIComponent(seeded.fixture_id)}`
      );
      if (!delRes.ok()) {
        throw new Error(`Failed to clean up fixture ${seeded.fixture_id}: ${delRes.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  function buildWorkspaceUrl(f: Milestone1SeedResponse): string {
    const params = new URLSearchParams({
      pupil_id: f.active_pupil_id,
      group_id: f.own_group_id,
      other_group_id: f.other_group_id,
      reference_date: f.reference_date,
      academic_year_id: f.academic_year_id,
      staff_id: f.teacher_staff_id,
    });
    return `/?${params.toString()}`;
  }

  test('recordAmAttendanceAndSafeguardingPostbox_coreDalCommits_emitsCoreCatalogEventsAndExecutesSubscribedPluginWorkers_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    // 1. Verify the Standard Core Domain Event Catalog endpoint exposes core events & worker subscriptions
    const catalogRes = await request.get('/api/events/catalog');
    expect(catalogRes.ok()).toBe(true);
    const catalogBody = await catalogRes.json();
    expect(catalogBody.ok).toBe(true);
    expect(catalogBody.core_events).toEqual(
      expect.arrayContaining([
        'core.pupil.enrolled',
        'core.group.membership_changed',
        'core.attendance.session_recorded',
        'core.safeguarding.postbox_created',
        'core.gdpr.pupil_purged',
      ])
    );

    // 2. Open Host UI Shell and interact with the sandboxed plugin UI
    await page.goto(buildWorkspaceUrl(seeded));
    await expect(page.getByTestId('host-shell')).toBeVisible();

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');

    // 3. Record statutory AM attendance -> commits CoreDAL write -> emits core.attendance.session_recorded
    const recordAttendanceBtn = pluginFrame.getByTestId('record-am-attendance-btn');
    await expect(recordAttendanceBtn).toBeVisible();
    await recordAttendanceBtn.click();
    await expect(pluginFrame.getByTestId('attendance-result')).toContainText('SESSION_SAVED:AM');

    // 4. Submit safeguarding postbox -> commits CoreDAL write -> emits core.safeguarding.postbox_created
    const submitSafeguardingBtn = pluginFrame.getByTestId('submit-safeguarding-postbox-btn');
    await expect(submitSafeguardingBtn).toBeVisible();
    await submitSafeguardingBtn.click();
    await expect(pluginFrame.getByTestId('safeguarding-postbox-result')).toContainText('POSTBOX_ACCEPTED');

    // 5. Poll/load worker event logs via the declarative UI button until both async worker executions complete
    const loadWorkerLogsBtn = pluginFrame.getByTestId('load-worker-event-logs-btn');
    await expect(loadWorkerLogsBtn).toBeVisible();
    await expect(loadWorkerLogsBtn).toHaveAttribute('data-mis-rpc', 'list_worker_event_logs');

    const workerLogsResult = pluginFrame.getByTestId('worker-event-logs-result');
    await expect(async () => {
      await loadWorkerLogsBtn.click();
      await expect(workerLogsResult).toContainText('core.attendance.session_recorded');
      await expect(workerLogsResult).toContainText('core.safeguarding.postbox_created');
    }).toPass({ timeout: 5000 });

    const rawLogsText = await workerLogsResult.textContent();
    expect(rawLogsText).toBeTruthy();
    const parsed = JSON.parse(rawLogsText!);
    expect(parsed.pupil_id).toBe(seeded.active_pupil_id);
    expect(parsed.logs.length).toBeGreaterThanOrEqual(2);

    const attLog = parsed.logs.find(
      (l: { event_name: string }) => l.event_name === 'core.attendance.session_recorded'
    );
    expect(attLog).toBeTruthy();
    expect(attLog.hop).toBe(1);
    expect(attLog.caller_type).toBe('SYSTEM');
    expect(attLog.caller_staff_id).toBe('');

    const sfgLog = parsed.logs.find(
      (l: { event_name: string }) => l.event_name === 'core.safeguarding.postbox_created'
    );
    expect(sfgLog).toBeTruthy();
    expect(sfgLog.hop).toBe(1);
    expect(sfgLog.caller_type).toBe('SYSTEM');
    expect(sfgLog.caller_staff_id).toBe('');
    // Envelope metadata only — zero plaintext safeguarding narrative in event payload
    expect(JSON.stringify(sfgLog)).not.toContain('Pupil reported concern during registration');
  });

  test('triggerLoopRelay_recursiveWorkerReEmission_enforcesMaxHop3AndDropsHop4_test', async ({
    page,
    request,
  }) => {
    if (!seeded) {
      throw new Error('Missing mandatory seeded fixture state');
    }

    await page.goto(buildWorkspaceUrl(seeded));
    await expect(page.getByTestId('host-shell')).toBeVisible();

    const pluginFrame = page.frameLocator('[data-testid="plugin-iframe-uk.org.optimis.test"]');

    const triggerLoopBtn = pluginFrame.getByTestId('trigger-loop-relay-btn');
    await expect(triggerLoopBtn).toBeVisible();
    await expect(triggerLoopBtn).toHaveAttribute('data-mis-rpc', 'trigger_loop_relay');
    await triggerLoopBtn.click();

    const loadWorkerLogsBtn = pluginFrame.getByTestId('load-worker-event-logs-btn');
    const workerLogsResult = pluginFrame.getByTestId('worker-event-logs-result');

    // Wait until hop 3 is processed by the worker goroutine
    await expect(async () => {
      await loadWorkerLogsBtn.click();
      const txt = await workerLogsResult.textContent();
      expect(txt).toBeTruthy();
      const data = JSON.parse(txt!);
      const loopLogs = (data.logs || []).filter(
        (l: { event_name: string }) => l.event_name === 'test.loop_relay'
      );
      expect(loopLogs.length).toBe(3);
    }).toPass({ timeout: 5000 });

    const finalTxt = await workerLogsResult.textContent();
    const finalData = JSON.parse(finalTxt!);
    const loopLogs = (finalData.logs || []).filter(
      (l: { event_name: string }) => l.event_name === 'test.loop_relay'
    );

    // Must execute for Hop 1, Hop 2, and Hop 3, and drop Hop 4 (Hop > 3)
    expect(loopLogs.map((l: { hop: number }) => l.hop)).toEqual([1, 2, 3]);

    // Verify Host Event Bus metrics recorded the dropped event (events_dropped_total >= 1)
    const catalogRes = await request.get('/api/events/catalog');
    expect(catalogRes.ok()).toBe(true);
    const catalogBody = await catalogRes.json();
    expect(catalogBody.metrics.events_dropped_total).toBeGreaterThanOrEqual(1);
  });
});
