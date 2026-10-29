import { test, expect } from '@playwright/test';

// All E2E tests must run sequentially
test.describe.configure({ mode: 'serial' });

interface SeededPersonaFixtures {
  fixture_id: string;
  pupil_id: string;
  admin_staff_id: string;
  dsl_staff_id: string;
  hod_staff_id: string;
  teacher_staff_id: string;
  attendance_officer_staff_id: string;
  terminated_staff_id: string;
}

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
  second_active_pupil_id?: string;
  transferred_pupil_id: string;
  other_group_pupil_id: string;
}

test.describe('Assessment & Markbook Plugin Vertical Slice: MAT Grading Schema Sync, Teacher (:own_groups) vs HoD (:all) Attainment Rollup & SAR Teacher Comment Redaction (uk.org.optimis.assessment.markbook)', () => {
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string = '';
  let personas: SeededPersonaFixtures | null = null;
  let m1Seeded: Milestone1SeedResponse | null = null;

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];
    logCursor = '';
    personas = null;
    m1Seeded = null;

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

    // 3. Ensure dev persona switcher and mock external dependency modes are active
    const authModeRes = await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    if (!authModeRes.ok()) {
      throw new Error(`Failed to configure auth mode: ${authModeRes.status()}`);
    }

    const extModeRes = await request.post('/api/test/external-deps-mode', {
      data: {
        enable_real_cloud_kms: false,
        enable_real_parent_dmz_sync: false,
        enable_real_mat_sync: false,
      },
    });
    if (!extModeRes.ok()) {
      throw new Error(`Failed to configure external deps mode: ${extModeRes.status()}`);
    }

    // 4. Seed isolated persona fixtures (Admin, HoD, Teacher) and Milestone 1 two-set fixtures (with TEACHER role on m1Seeded.teacher_staff_id)
    const personaRes = await request.post('/api/test/seed-personas', {
      data: {
        academic_year_id: 'ay_2026_2027',
        effective_from: '2026-09-01',
      },
    });
    if (!personaRes.ok()) {
      throw new Error(`Failed to seed persona fixtures: ${personaRes.status()}`);
    }
    personas = (await personaRes.json()) as SeededPersonaFixtures;
    if (!personas.fixture_id || !personas.admin_staff_id || !personas.hod_staff_id) {
      throw new Error('Seeded persona fixture response missing mandatory identifiers');
    }

    const m1Res = await request.post('/api/test/seed-milestone1-fixtures', {
      data: {
        academic_year_id: 'ay_2026_2027',
        reference_date: '2026-10-28',
        include_second_active_pupil: true,
      },
    });
    if (!m1Res.ok()) {
      throw new Error(`Failed to seed Milestone 1 fixtures: ${m1Res.status()}`);
    }
    m1Seeded = (await m1Res.json()) as Milestone1SeedResponse;
    if (
      !m1Seeded.fixture_id ||
      !m1Seeded.teacher_staff_id ||
      !m1Seeded.own_group_id ||
      !m1Seeded.other_group_id ||
      !m1Seeded.active_pupil_id ||
      !m1Seeded.other_group_pupil_id
    ) {
      throw new Error('Milestone 1 seed response missing mandatory identifiers');
    }

    // 5. Switch to Admin persona to runtime-install and activate assessment_markbook_plugin (uk.org.optimis.assessment.markbook)
    const switchAdminRes = await request.post('/api/auth/persona', {
      data: { staff_id: personas.admin_staff_id },
    });
    if (!switchAdminRes.ok()) {
      throw new Error(`Failed to switch to Admin persona in setup: ${switchAdminRes.status()}`);
    }

    const installMarkbookRes = await request.post('/api/admin/plugins/install', {
      data: {
        folder: 'assessment_markbook_plugin',
        staff_id: personas.admin_staff_id,
      },
    });
    if (!installMarkbookRes.ok()) {
      throw new Error(
        `Failed to install assessment_markbook_plugin: ${installMarkbookRes.status()} ${await installMarkbookRes.text()}`
      );
    }

    const activateMarkbookRes = await request.post(
      '/api/admin/plugins/uk.org.optimis.assessment.markbook/activate',
      {
        data: { staff_id: personas.admin_staff_id },
      }
    );
    if (!activateMarkbookRes.ok()) {
      throw new Error(
        `Failed to activate uk.org.optimis.assessment.markbook: ${activateMarkbookRes.status()}`
      );
    }
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Purge plugin data created during the test
    if (personas && m1Seeded) {
      await request.post('/api/auth/persona', {
        data: { staff_id: personas.admin_staff_id },
      });
      for (const pid of [
        m1Seeded.active_pupil_id,
        m1Seeded.second_active_pupil_id,
        m1Seeded.transferred_pupil_id,
        m1Seeded.other_group_pupil_id,
      ]) {
        if (pid) {
          await request.post('/api/plugins/uk.org.optimis.assessment.markbook/rpc/purge_pupil_data', {
            data: { args: { pupil_id: pid } },
          });
        }
      }
    }

    // 2. Restore mock external dependency modes & clear MAT policy state
    await request.post('/api/test/external-deps-mode', {
      data: {
        enable_real_cloud_kms: false,
        enable_real_parent_dmz_sync: false,
        enable_real_mat_sync: false,
      },
    });

    // 3. Fetch Backend Service Logs emitted during this test
    if (logCursor) {
      const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
      if (logsRes.ok()) {
        const serviceLogs: string[] = (await logsRes.json()).entries;
        if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
          console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
        }
      }
    }

    // 4. Dump tapped Browser Console logs on failure or error
    if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
      console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
      console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
    }

    // 5. Remove runtime-installed plugin
    await request.post('/api/test/auth-mode', {
      data: { enable_dev_persona_switcher: true },
    });
    await request.delete('/api/admin/plugins/uk.org.optimis.assessment.markbook');

    // 6. Clean up seeded fixtures in core.db
    if (m1Seeded?.fixture_id) {
      const delM1 = await request.delete(
        `/api/test/seed-milestone1-fixtures/${encodeURIComponent(m1Seeded.fixture_id)}`
      );
      if (!delM1.ok()) {
        throw new Error(`Failed to clean up Milestone 1 fixture ${m1Seeded.fixture_id}: ${delM1.status()}`);
      }
    }
    if (personas?.fixture_id) {
      const delP = await request.delete(
        `/api/test/seed-personas/${encodeURIComponent(personas.fixture_id)}`
      );
      if (!delP.ok()) {
        throw new Error(`Failed to clean up persona fixture ${personas.fixture_id}: ${delP.status()}`);
      }
    }

    expect(pageErrors).toEqual([]);
  });

  test('markbookLifecycleAndCrossSetRollup_pullMatSchemaTeacherSavesOwnSetGradesSwitchesToHodViewsCrossSetRollupAndAdminRedactsSarTeacherComments_test', async ({
    page,
    request,
  }) => {
    if (!personas || !m1Seeded) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // 1. Seed Mock MAT Central Sync Feed and pull MAT grading schema into `__system_config`
    const seedMatRes = await request.post('/api/test/mat/seed-feed', {
      data: {
        mat_id: 'MAT_OPTIMIS_TRUST_01',
        grading_schema_version: 'KS4_9_TO_1_AND_PRIMARY_WT_EXS_GDS_V2026',
        behaviour_taxonomy: ['MERIT_EXCELLENCE', 'RESTORATIVE_CONVERSATION'],
        approved_plugins: [{ id: 'uk.org.optimis.assessment.markbook', version: '0.1.0' }],
      },
    });
    expect(seedMatRes.status()).toBe(201);

    const pullPoliciesRes = await request.post('/api/sync/mat/pull-policies', {
      data: { staff_id: personas.admin_staff_id },
    });
    expect(pullPoliciesRes.status()).toBe(200);
    const pullPoliciesBody = await pullPoliciesRes.json();
    expect(pullPoliciesBody.ok).toBe(true);
    expect(pullPoliciesBody.policy_bundle.grading_schema_version).toBe(
      'KS4_9_TO_1_AND_PRIMARY_WT_EXS_GDS_V2026'
    );

    // 2. Switch to Classroom Teacher (`m1Seeded.teacher_staff_id`, holding `:own_groups` for `own_group_id` Year 10 Maths Set 1)
    const switchTeacherRes = await request.post('/api/auth/persona', {
      data: { staff_id: m1Seeded.teacher_staff_id },
    });
    expect(switchTeacherRes.ok()).toBe(true);
    const teacherSession = await switchTeacherRes.json();
    expect(teacherSession.caller.active_roles).toContain('TEACHER');
    expect(teacherSession.caller.scopes).toContain('dal.groups.read:own_groups');
    expect(teacherSession.caller.scopes).toContain('dal.pupils.read:own_groups');
    expect(teacherSession.caller.scopes).not.toContain('dal.groups.read:all');

    await page.context().addCookies([
      {
        name: 'mis_session',
        value: teacherSession.session_token,
        url: 'http://127.0.0.1:18080',
        sameSite: 'Strict',
      },
    ]);

    const query = new URLSearchParams({
      pupil_id: m1Seeded.active_pupil_id,
      group_id: m1Seeded.own_group_id,
      other_group_id: m1Seeded.other_group_id,
      reference_date: m1Seeded.reference_date,
      academic_year_id: m1Seeded.academic_year_id,
      staff_id: m1Seeded.teacher_staff_id,
    });
    await page.goto(`/?${query.toString()}`);

    await expect(page.getByTestId('host-shell')).toBeVisible();
    const markbookNavBtn = page.getByTestId('nav-item-plugin-uk.org.optimis.assessment.markbook');
    await expect(markbookNavBtn).toBeVisible();
    await markbookNavBtn.click();

    const iframeEl = page.getByTestId('plugin-iframe-uk.org.optimis.assessment.markbook');
    await expect(iframeEl).toBeVisible();
    await expect(iframeEl).toHaveAttribute('sandbox', 'allow-scripts');

    const pluginFrame = page.frameLocator(
      '[data-testid="plugin-iframe-uk.org.optimis.assessment.markbook"]'
    );
    await expect(pluginFrame.getByTestId('markbook-plugin-root')).toBeVisible();

    // Verify MAT grading schema version synced into `__system_config` is surfaced in the markbook workspace
    await expect(pluginFrame.getByTestId('markbook-mat-schema-badge')).toHaveText(
      'KS4_9_TO_1_AND_PRIMARY_WT_EXS_GDS_V2026'
    );

    // Teacher (:own_groups) creates a markbook column for their own set (`own_group_id` Year 10 Maths Set 1)
    const colTitleInput = pluginFrame.getByTestId('markbook-column-title-input');
    const colScaleSelect = pluginFrame.getByTestId('markbook-column-scale-select');
    const createColBtn = pluginFrame.getByTestId('create-markbook-column-btn');

    await expect(createColBtn).toBeVisible();
    await expect(createColBtn).toHaveAttribute('data-mis-rpc', 'create_markbook_column');

    await colTitleInput.fill('Autumn Half-Term Algebra Assessment');
    await colScaleSelect.selectOption('9-1');
    await createColBtn.click();

    await expect(pluginFrame.getByTestId('markbook-active-column-title')).toContainText(
      'Autumn Half-Term Algebra Assessment'
    );
    await expect(pluginFrame.getByTestId('markbook-active-column-scale')).toHaveText('9-1');

    // Teacher enters grade '9' and a teacher comment containing a third-party pupil name ("Neville Longbottom") for Ada Lovelace (`active_pupil_id`)
    const gradeSelect = pluginFrame.getByTestId(`markbook-grade-select-${m1Seeded.active_pupil_id}`);
    const commentInput = pluginFrame.getByTestId(
      `markbook-comment-input-${m1Seeded.active_pupil_id}`
    );
    const saveGradesBtn = pluginFrame.getByTestId('save-pupil-grades-btn');

    await expect(gradeSelect).toBeVisible();
    await expect(saveGradesBtn).toBeVisible();
    await expect(saveGradesBtn).toHaveAttribute('data-mis-rpc', 'save_pupil_grades');

    await gradeSelect.selectOption('9');
    await commentInput.fill(
      'Scored 96% on higher-tier quadratics paper; peer-checked extension task with third-party pupil Neville Longbottom.'
    );
    await saveGradesBtn.click();

    await expect(pluginFrame.getByTestId('markbook-saved-count-badge')).toContainText('1');
    await expect(
      pluginFrame.getByTestId(`markbook-saved-grade-${m1Seeded.active_pupil_id}`)
    ).toHaveText('9');

    // Verify Teacher (:own_groups) is blocked from Set 2 (`other_group_id`) via UI / RPC
    const groupSelect = pluginFrame.getByTestId('markbook-group-select');
    await groupSelect.selectOption(m1Seeded.other_group_id);
    await createColBtn.click();

    const errorBanner = pluginFrame.getByTestId('markbook-error-banner');
    await expect(errorBanner).toBeVisible();
    await expect(errorBanner).toContainText('access denied under own_groups scope');

    // 3. Switch to Head of Department (`personas.hod_staff_id`, holding `:all` scopes) via Topbar Persona Switcher
    const personaSwitcher = page.getByTestId('persona-switcher-select');
    await expect(personaSwitcher).toBeVisible();
    await personaSwitcher.selectOption(personas.hod_staff_id);

    await expect(pluginFrame.getByTestId('markbook-caller-role-badge')).toContainText('HOD');

    // As HoD (`:all`), select Set 2 (`other_group_id`), create a markbook column, and save grade '6' for Alan Turing (`other_group_pupil_id`)
    await groupSelect.selectOption(m1Seeded.other_group_id);
    await colTitleInput.fill('Autumn Half-Term Algebra Assessment');
    await colScaleSelect.selectOption('9-1');
    await createColBtn.click();

    await expect(
      pluginFrame.getByTestId(`markbook-grade-select-${m1Seeded.other_group_pupil_id}`)
    ).toBeVisible();
    await pluginFrame
      .getByTestId(`markbook-grade-select-${m1Seeded.other_group_pupil_id}`)
      .selectOption('6');
    await pluginFrame
      .getByTestId(`markbook-comment-input-${m1Seeded.other_group_pupil_id}`)
      .fill('Solid foundation algebra performance in Set 2.');
    await saveGradesBtn.click();

    await expect(
      pluginFrame.getByTestId(`markbook-saved-grade-${m1Seeded.other_group_pupil_id}`)
    ).toHaveText('6');

    // HoD (`:all`) clicks "Compare Department Attainment Across Sets" (`get_department_attainment_rollup`)
    const loadRollupBtn = pluginFrame.getByTestId('load-department-rollup-btn');
    await expect(loadRollupBtn).toBeVisible();
    await expect(loadRollupBtn).toHaveAttribute('data-mis-rpc', 'get_department_attainment_rollup');
    await loadRollupBtn.click();

    const rollupPanel = pluginFrame.getByTestId('markbook-department-rollup-panel');
    await expect(rollupPanel).toBeVisible();
    await expect(
      pluginFrame.getByTestId(`rollup-set-avg-${m1Seeded.own_group_id}`)
    ).toHaveText('9.0');
    await expect(
      pluginFrame.getByTestId(`rollup-set-avg-${m1Seeded.other_group_id}`)
    ).toHaveText('6.0');
    await expect(pluginFrame.getByTestId('rollup-highest-set-name')).toContainText(
      'Year 10 Maths Set 1'
    );

    // 4. Switch to System Admin / DPO (`personas.admin_staff_id`) and run UK GDPR SAR Redaction on teacher comments
    await personaSwitcher.selectOption(personas.admin_staff_id);

    const sarPanel = page.getByTestId('sar-redaction-panel');
    await expect(sarPanel).toBeVisible();

    await page.getByTestId('sar-load-export-btn').click();
    await expect(page.getByTestId('sar-export-status')).toHaveText('SAR_LOADED');

    const markbookSarSection = page.getByTestId(
      'sar-section-uk.org.optimis.assessment.markbook-Assessment Markbook Grades & Teacher Comments'
    );
    await expect(markbookSarSection).toBeVisible();

    const teacherCommentRedactInput = page.getByTestId(
      'sar-redact-input-uk.org.optimis.assessment.markbook-Assessment Markbook Grades & Teacher Comments-0-teacher_comment'
    );
    await expect(teacherCommentRedactInput).toBeVisible();
    await expect(teacherCommentRedactInput).toHaveValue(
      'Scored 96% on higher-tier quadratics paper; peer-checked extension task with third-party pupil Neville Longbottom.'
    );

    await teacherCommentRedactInput.fill(
      'Scored 96% on higher-tier quadratics paper; peer-checked extension task with third-party pupil [REDACTED THIRD PARTY].'
    );

    await page.getByTestId('sar-finalize-export-btn').click();
    await expect(page.getByTestId('sar-export-status')).toContainText('SAR_EXPORT_FINALIZED');

    const finalizedJsonEl = page.getByTestId('sar-finalized-json');
    await expect(finalizedJsonEl).toBeVisible();
    const rawExportJson = (await finalizedJsonEl.textContent()) || '';
    expect(rawExportJson).not.toContain('Neville Longbottom');
    expect(rawExportJson).toContain('[REDACTED THIRD PARTY]');
    expect(rawExportJson).toContain('Autumn Half-Term Algebra Assessment');
  });

  test('markbookRbacScopeContrastAndGdprPurge_teacherBlockedFromOtherSetAndHodRollupAndUnsyncedMatSchemaFailsFastAndPurgeRemovesAllRows_test', async ({
    request,
  }) => {
    if (!personas || !m1Seeded) {
      throw new Error('Missing mandatory seeded fixtures');
    }

    // 1. Before pulling MAT policies into `__system_config`, markbook operations fail fast with MAT_GRADING_SCHEMA_NOT_SYNCED
    const unsyncedRes = await request.post(
      `/api/plugins/uk.org.optimis.assessment.markbook/rpc/create_markbook_column?staff_id=${encodeURIComponent(
        m1Seeded.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            group_id: m1Seeded.own_group_id,
            title: 'Should Fail Without MAT Sync',
            grading_scale: '9-1',
            academic_year_id: m1Seeded.academic_year_id,
            reference_date: m1Seeded.reference_date,
          },
        },
      }
    );
    expect(unsyncedRes.status()).toBe(400);
    const unsyncedBody = await unsyncedRes.json();
    expect(unsyncedBody.ok).toBe(false);
    expect(unsyncedBody.error_code).toContain('MAT_GRADING_SCHEMA_NOT_SYNCED');

    // 2. Seed and pull MAT grading schema into `__system_config`
    await request.post('/api/test/mat/seed-feed', {
      data: {
        mat_id: 'MAT_OPTIMIS_TRUST_01',
        grading_schema_version: 'PRIMARY_AND_SECONDARY_MAT_V2026',
        behaviour_taxonomy: ['MERIT_EXCELLENCE'],
        approved_plugins: [{ id: 'uk.org.optimis.assessment.markbook', version: '0.1.0' }],
      },
    });
    const pullRes = await request.post('/api/sync/mat/pull-policies', {
      data: { staff_id: personas.admin_staff_id },
    });
    expect(pullRes.ok()).toBe(true);

    // 3. Classroom Teacher (`:own_groups`) is rejected with 403 when calling HoD-only `get_department_attainment_rollup` (`:all`)
    const teacherRollupRes = await request.post(
      `/api/plugins/uk.org.optimis.assessment.markbook/rpc/get_department_attainment_rollup?staff_id=${encodeURIComponent(
        m1Seeded.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            academic_year_id: m1Seeded.academic_year_id,
            reference_date: m1Seeded.reference_date,
          },
        },
      }
    );
    expect(teacherRollupRes.status()).toBe(403);
    const teacherRollupBody = await teacherRollupRes.json();
    expect(teacherRollupBody.ok).toBe(false);
    expect(teacherRollupBody.error_code).toContain('caller lacks required DAL scopes');

    // 4. Classroom Teacher (`:own_groups`) creates a Primary `WT,EXS,GDS` markbook column for `own_group_id`
    const createColRes = await request.post(
      `/api/plugins/uk.org.optimis.assessment.markbook/rpc/create_markbook_column?staff_id=${encodeURIComponent(
        m1Seeded.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            group_id: m1Seeded.own_group_id,
            title: 'Key Stage Reading Moderation',
            grading_scale: 'WT,EXS,GDS',
            academic_year_id: m1Seeded.academic_year_id,
            reference_date: m1Seeded.reference_date,
          },
        },
      }
    );
    expect(createColRes.ok()).toBe(true);
    const createColBody = await createColRes.json();
    expect(createColBody.ok).toBe(true);
    const columnId: string = createColBody.data.column.id;
    expect(columnId).toMatch(/^col_[0-9a-f-]{36}$/);

    // 5. Invalid grade outside `WT,EXS,GDS` scale fails fast with INVALID_MAT_GRADE
    const invalidGradeRes = await request.post(
      `/api/plugins/uk.org.optimis.assessment.markbook/rpc/save_pupil_grades?staff_id=${encodeURIComponent(
        m1Seeded.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            column_id: columnId,
            group_id: m1Seeded.own_group_id,
            academic_year_id: m1Seeded.academic_year_id,
            reference_date: m1Seeded.reference_date,
            entries: [
              {
                pupil_id: m1Seeded.active_pupil_id,
                grade: '9',
                teacher_comment: 'Invalid numeric grade on WT,EXS,GDS column',
              },
            ],
          },
        },
      }
    );
    expect(invalidGradeRes.status()).toBe(400);
    expect((await invalidGradeRes.json()).error_code).toContain('INVALID_MAT_GRADE');

    // 6. Teacher attempting to save a grade for a transferred pupil (`transferred_pupil_id`) or another set's pupil (`other_group_pupil_id`) fails fast
    const transferredGradeRes = await request.post(
      `/api/plugins/uk.org.optimis.assessment.markbook/rpc/save_pupil_grades?staff_id=${encodeURIComponent(
        m1Seeded.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            column_id: columnId,
            group_id: m1Seeded.own_group_id,
            academic_year_id: m1Seeded.academic_year_id,
            reference_date: m1Seeded.reference_date,
            entries: [
              {
                pupil_id: m1Seeded.transferred_pupil_id,
                grade: 'EXS',
                teacher_comment: 'Should fail for mid-year transferred pupil',
              },
            ],
          },
        },
      }
    );
    expect(transferredGradeRes.status()).toBe(400);
    expect((await transferredGradeRes.json()).error_code).toContain(
      'access denied under own_groups scope'
    );

    // 7. Teacher saves valid `GDS` grade for `active_pupil_id`, then Admin verifies `export_pupil_data` and `purge_pupil_data` leave zero orphaned records
    const validSaveRes = await request.post(
      `/api/plugins/uk.org.optimis.assessment.markbook/rpc/save_pupil_grades?staff_id=${encodeURIComponent(
        m1Seeded.teacher_staff_id
      )}`,
      {
        data: {
          args: {
            column_id: columnId,
            group_id: m1Seeded.own_group_id,
            academic_year_id: m1Seeded.academic_year_id,
            reference_date: m1Seeded.reference_date,
            entries: [
              {
                pupil_id: m1Seeded.active_pupil_id,
                grade: 'GDS',
                teacher_comment: 'Demonstrates greater depth comprehension across unseen texts.',
              },
            ],
          },
        },
      }
    );
    expect(validSaveRes.ok()).toBe(true);

    const exportBeforeRes = await request.post(
      `/api/plugins/uk.org.optimis.assessment.markbook/rpc/export_pupil_data?staff_id=${encodeURIComponent(
        personas.admin_staff_id
      )}`,
      {
        data: { args: { pupil_id: m1Seeded.active_pupil_id } },
      }
    );
    expect(exportBeforeRes.ok()).toBe(true);
    const exportBeforeBody = await exportBeforeRes.json();
    expect(exportBeforeBody.ok).toBe(true);
    expect(exportBeforeBody.data[0].title).toBe('Assessment Markbook Grades & Teacher Comments');
    expect(exportBeforeBody.data[0].records.length).toBe(1);
    expect(exportBeforeBody.data[0].redactable_fields).toEqual(['teacher_comment']);

    const purgeRes = await request.post(
      `/api/plugins/uk.org.optimis.assessment.markbook/rpc/purge_pupil_data?staff_id=${encodeURIComponent(
        personas.admin_staff_id
      )}`,
      {
        data: { args: { pupil_id: m1Seeded.active_pupil_id } },
      }
    );
    expect(purgeRes.ok()).toBe(true);
    const purgeBody = await purgeRes.json();
    expect(purgeBody.ok).toBe(true);
    expect(purgeBody.data.purged_rows).toBe(1);

    const exportAfterRes = await request.post(
      `/api/plugins/uk.org.optimis.assessment.markbook/rpc/export_pupil_data?staff_id=${encodeURIComponent(
        personas.admin_staff_id
      )}`,
      {
        data: { args: { pupil_id: m1Seeded.active_pupil_id } },
      }
    );
    expect(exportAfterRes.ok()).toBe(true);
    const exportAfterBody = await exportAfterRes.json();
    expect(exportAfterBody.data[0].records.length).toBe(0);
  });
});
