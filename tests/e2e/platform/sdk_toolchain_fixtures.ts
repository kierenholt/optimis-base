import { execFileSync } from 'child_process';
import type { APIRequestContext } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

export interface CommandResult {
  stdout: string;
  stderr: string;
  exitCode: number;
}

export interface PersonaSeedResponse {
  fixture_id: string;
  pupil_id: string;
  admin_staff_id: string;
  dsl_staff_id: string;
  hod_staff_id: string;
  teacher_staff_id: string;
  terminated_staff_id: string;
}

export const REQUIRED_TEMPLATE_FILES: string[] = [
  'README.md',
  'GEMINI.md',
  '.agents/AGENTS.md',
  '.agents/GEMINI.md',
  '.agents/skills/test-author/SKILL.md',
  '.agents/skills/code-implementer/SKILL.md',
  'package.json',
  'manifest.json',
  '.vscode/settings.json',
  '.vscode/extensions.json',
  '.mis-types/dal.d.lua',
  '.mis-types/records.d.lua',
  '.mis-types/GEMINI.md',
  'bin/GEMINI.md',
  'sdk/GEMINI.md',
  'sdk/manifest.schema.json',
  'sdk/ui/mis-bridge.js',
  'sdk/ui/mis-theme.css',
  'schema/core/GEMINI.md',
  'schema/core/00_system_and_academic_cycle.sql',
  'schema/core/SCHEMA_MAP.md',
  'migrations/GEMINI.md',
  'migrations/001_init.sql',
  'src/GEMINI.md',
  'src/init.lua',
  'ui/GEMINI.md',
  'ui/dist/index.html',
  'ui/dist/widget.html',
  'tests/GEMINI.md',
  'tests/init_test.lua',
  'scripts/GEMINI.md',
  'scripts/run-cli.mjs',
];

export const EXPECTED_DAL_TYPE_SIGNATURES: string[] = [
  'OptimisAcademicDAL',
  'list_years',
  'upsert_year',
  'list_terms',
  'upsert_term',
  'list_days_of_week',
  'upsert_day_of_week',
  'list_special_days',
  'upsert_special_day',
  'list_alert_levels',
  'OptimisDashboardDAL',
  'list_available_widgets',
  'list_role_defaults',
  'save_role_default',
  'get_staff_layout',
  'save_staff_layout',
  'reset_staff_layout',
  'list_fsm_periods_batch',
  'upsert_fsm_period',
  'upsert_pupil_admission',
  'upsert_year_enrolment',
  'list_exclusions',
  'upsert_exclusion',
  'list_alternative_provisions',
  'upsert_alternative_provision',
  'list_post16_learning_aims',
  'upsert_post16_learning_aim',
  'list_pastoral_notes',
  'upsert_pastoral_note',
  'list_need_types',
  'upsert_need_type',
  'upsert_sen_record',
  'upsert_sen_need',
  'upsert_sen_assistant',
  'list_conditions',
  'upsert_condition',
  'upsert_guardian_contact',
  'list_siblings_for_pupil',
  'list_households',
  'list_scr_register',
  'list_contracts',
  'list_absences',
  'upsert_scr_record',
  'upsert_contract',
  'upsert_absence',
  'upsert_staff_member',
  'list_roles',
  'list_cover_assignments',
  'upsert_cover_assignment',
  'list_houses',
  'upsert_house',
  'list_rooms',
  'upsert_room',
  'list_courses',
  'upsert_course',
  'list_departments',
  'upsert_department',
  'list_year_groups',
  'upsert_year_group',
  'list_timetable_periods',
  'list_timetable_slots',
  'get_pupil_timetable_slots',
  'list_pupil_memberships',
  'upsert_group',
  'upsert_timetable_period',
  'upsert_timetable_slot',
  'get_staff_timetable_slots_batch',
  'list_free_staff_for_period',
  'revoke_staff_assignment',
  'list_codes',
  'upsert_code',
  'get_group_lesson',
  'upsert_lesson_batch',
  'list_session_marks',
  'list_register_submissions',
  'create_postbox',
  'get_group_session',
  'get_pupil_summary',
];

export const BEHAVIOUR_LUA_TEST = `
function test_log_merit_and_export_purge_pupil_data(t)
  local res = t.rpc("log_behaviour_event", {
    pupil_id = "pup_yr7_alice",
    group_id = "grp_reg_7a",
    event_type = "MERIT",
    points = 5,
    category = "Academic Excellence",
    incident_notes = "Outstanding algebra homework"
  })
  assert(res.ok == true, "expected log_behaviour_event to succeed")

  local exp = t.rpc("export_pupil_data", { pupil_id = "pup_yr7_alice" }, "stf_admin_01")
  assert(exp.ok == true, "expected export_pupil_data to succeed")
  assert(#exp.data >= 1, "expected at least 1 exported section")

  local prg = t.rpc("purge_pupil_data", { pupil_id = "pup_yr7_alice" }, "stf_admin_01")
  assert(prg.ok == true, "expected purge_pupil_data to succeed")
end
`;

export const FAILING_LUA_TEST = `
function test_deliberate_failure(t)
  error("DELIBERATE_TEST_ASSERTION_FAILURE")
end
`;

export const PROD_DAL_PROBE_MANIFEST = {
  id: 'uk.org.optimis.test.prod_dal_probe',
  title: 'Production DAL Readiness Probe',
  version: '1.0.0',
  core_dal_version: '~1.4',
  icon: 'database',
  views: [
    {
      id: 'home',
      path: '/',
      entry: 'ui/dist/index.html',
      title: 'Production DAL Probe',
      icon: 'database',
      keywords: ['dal', 'probe'],
    },
  ],
  endpoints: {
    exercise_all_new_dal_domains: {
      scopes: [
        'dal.academic.read:all',
        'dal.academic.write:all',
        'dal.dashboard.read:all',
        'dal.dashboard.write:all',
        'dal.pupils.read:all',
        'dal.pupils.write:all',
        'dal.sen.read:all',
        'dal.sen.write:all',
        'dal.medical.read:all',
        'dal.medical.write:all',
        'dal.guardians.read:all',
        'dal.guardians.write:all',
        'dal.staff.read:admin',
        'dal.staff.write:admin',
        'dal.groups.read:all',
        'dal.groups.write:all',
        'dal.attendance.read:all',
        'dal.attendance.write:all',
        'dal.census.read:all',
        'plugin.config.write',
      ],
      mutates: true,
      timeout_ms: 5000,
    },
  },
};

export const PROD_DAL_PROBE_HTML =
  '<!doctype html><html><body><button data-mis-rpc="exercise_all_new_dal_domains">Run</button></body></html>';

export const PROD_DAL_PROBE_INIT_LUA: string = fs.readFileSync(
  path.resolve(__dirname, 'fixtures/prod_dal_probe_init.lua'),
  'utf-8'
);

export const PROD_DAL_PROBE_TEST_LUA: string = fs.readFileSync(
  path.resolve(__dirname, 'fixtures/prod_dal_probe_test.lua'),
  'utf-8'
);

export function runBinary(
  binPath: string,
  args: string[],
  cwd: string = process.cwd()
): CommandResult {
  try {
    const stdout = execFileSync(binPath, args, {
      cwd,
      encoding: 'utf-8',
      env: { ...process.env },
    });
    return { stdout: stdout.trim(), stderr: '', exitCode: 0 };
  } catch (err: unknown) {
    const execErr = err as {
      stdout?: string | Buffer;
      stderr?: string | Buffer;
      status?: number;
      message?: string;
    };
    const stdout = typeof execErr.stdout === 'string'
      ? execErr.stdout.trim()
      : (execErr.stdout?.toString('utf-8') || '').trim();
    let stderr = typeof execErr.stderr === 'string'
      ? execErr.stderr.trim()
      : (execErr.stderr?.toString('utf-8') || '').trim();
    if (!stderr && execErr.message) {
      stderr = execErr.message.trim();
    }
    return {
      stdout,
      stderr,
      exitCode: typeof execErr.status === 'number' ? execErr.status : 1,
    };
  }
}

export function copyPlugin(srcFolder: string, destPath: string, rootDir: string = process.cwd()): string {
  fs.cpSync(path.resolve(rootDir, srcFolder), destPath, { recursive: true });
  return destPath;
}

export function createProdDalProbePlugin(destDir: string): void {
  fs.mkdirSync(path.join(destDir, 'src'), { recursive: true });
  fs.mkdirSync(path.join(destDir, 'ui/dist'), { recursive: true });
  fs.mkdirSync(path.join(destDir, 'tests'), { recursive: true });

  fs.writeFileSync(path.join(destDir, 'manifest.json'), JSON.stringify(PROD_DAL_PROBE_MANIFEST, null, 2));
  fs.writeFileSync(path.join(destDir, 'ui/dist/index.html'), PROD_DAL_PROBE_HTML);
  fs.writeFileSync(path.join(destDir, 'src/init.lua'), PROD_DAL_PROBE_INIT_LUA);
  fs.writeFileSync(path.join(destDir, 'tests/probe_test.lua'), PROD_DAL_PROBE_TEST_LUA);
}

export async function seedPersonaFixtures(
  request: APIRequestContext,
  data: Record<string, unknown> = {
    academic_year_id: 'ay_2026_2027',
    effective_from: '2026-09-01',
  }
): Promise<PersonaSeedResponse> {
  const seedRes = await request.post('/api/test/seed-personas', { data });
  if (!seedRes.ok()) {
    throw new Error(`Failed to seed persona fixtures: ${seedRes.status()}`);
  }
  return (await seedRes.json()) as PersonaSeedResponse;
}

export async function installAndActivatePlugin(
  request: APIRequestContext,
  folder: string,
  pluginId: string,
  staffId: string
): Promise<string> {
  const installRes = await request.post('/api/admin/plugins/install', {
    data: { folder, staff_id: staffId },
  });
  if (!installRes.ok()) {
    throw new Error(`Failed to install plugin ${folder}: ${installRes.status()}`);
  }
  const plugId = (await installRes.json()).plug_id;

  const activateRes = await request.post(`/api/admin/plugins/${encodeURIComponent(pluginId)}/activate`, {
    data: { staff_id: staffId },
  });
  if (!activateRes.ok()) {
    throw new Error(`Failed to activate plugin ${pluginId}: ${activateRes.status()}`);
  }
  return plugId;
}

export function verifyVsCodeAndManifestSchema(
  rootDir: string,
  assertExpect: (val: unknown) => {
    toBe: (expected: unknown) => void;
    toContain: (expected: unknown) => void;
    toBeDefined: () => void;
    toHaveProperty: (prop: string) => void;
    toEqual: (expected: unknown) => void;
    not: { toHaveProperty: (prop: string) => void };
  }
): void {
  const vscodeSettingsPath = path.resolve(rootDir, '.vscode/settings.json');
  assertExpect(fs.existsSync(vscodeSettingsPath)).toBe(true);
  const vscodeSettings = JSON.parse(fs.readFileSync(vscodeSettingsPath, 'utf-8'));
  assertExpect(vscodeSettings['Lua.runtime.version']).toBe('Lua 5.1');
  assertExpect(Array.isArray(vscodeSettings['Lua.workspace.library'])).toBe(true);
  assertExpect(vscodeSettings['Lua.workspace.library']).toContain('.mis-types/dal.d.lua');
  assertExpect(vscodeSettings['Lua.workspace.library']).toContain('.mis-types/records.d.lua');
  assertExpect(vscodeSettings['Lua.diagnostics.disable'] || vscodeSettings['Lua.runtime.builtin']).toBeDefined();
  assertExpect(Array.isArray(vscodeSettings['json.schemas'])).toBe(true);

  const dalLuaTypesPath = path.resolve(rootDir, '.mis-types/dal.d.lua');
  assertExpect(fs.existsSync(dalLuaTypesPath)).toBe(true);
  const dalLuaContent = fs.readFileSync(dalLuaTypesPath, 'utf-8');
  assertExpect(dalLuaContent).toContain('OptimisPluginContext');
  assertExpect(dalLuaContent).toContain('OptimisDAL');

  const recordsLuaTypesPath = path.resolve(rootDir, '.mis-types/records.d.lua');
  assertExpect(fs.existsSync(recordsLuaTypesPath)).toBe(true);

  const manifestSchemaPath = path.resolve(rootDir, 'sdk/manifest.schema.json');
  assertExpect(fs.existsSync(manifestSchemaPath)).toBe(true);
  const manifestSchema = JSON.parse(fs.readFileSync(manifestSchemaPath, 'utf-8'));
  assertExpect(manifestSchema.properties).toHaveProperty('id');
  assertExpect(manifestSchema.properties).toHaveProperty('version');
  assertExpect(manifestSchema.properties).toHaveProperty('core_dal_version');
  assertExpect(manifestSchema.properties).toHaveProperty('endpoints');
  assertExpect(manifestSchema.properties.widgets.items.required).toEqual([
    'id',
    'title',
    'description',
    'entry',
    'width',
    'height',
  ]);
  assertExpect(manifestSchema.properties.widgets.items.properties.width).toEqual({
    type: 'integer',
    enum: [1, 2],
  });
  assertExpect(manifestSchema.properties.widgets.items.properties.height).toEqual({
    type: 'integer',
    enum: [1, 2],
  });
  assertExpect(manifestSchema.properties.widgets.items.properties).not.toHaveProperty('default_col_span');
  assertExpect(manifestSchema.properties.widgets.items.properties).not.toHaveProperty('default_row_span');
}

export function verifySdkPackages(
  rootDir: string,
  assertExpect: (val: unknown) => { toBe: (expected: unknown) => void }
): void {
  const bridgePkgPath = path.resolve(rootDir, 'sdk/packages/bridge/package.json');
  const tailwindPkgPath = path.resolve(rootDir, 'sdk/packages/tailwind-preset/package.json');
  assertExpect(fs.existsSync(bridgePkgPath)).toBe(true);
  assertExpect(fs.existsSync(tailwindPkgPath)).toBe(true);

  const bridgePkg = JSON.parse(fs.readFileSync(bridgePkgPath, 'utf-8'));
  const tailwindPkg = JSON.parse(fs.readFileSync(tailwindPkgPath, 'utf-8'));
  assertExpect(bridgePkg.name).toBe('@optimis/bridge');
  assertExpect(tailwindPkg.name).toBe('@optimis/tailwind-preset');
}

export function verifyDalAndRecordTypeStubs(
  rootDir: string,
  assertExpect: (val: unknown, msg?: string) => {
    toBe: (expected: unknown) => void;
    toContain: (expected: string) => void;
    toBeLessThan: (expected: number) => void;
  }
): void {
  const dalTypesPath = path.resolve(rootDir, '.mis-types/dal.d.lua');
  const sdkCtxTypesPath = path.resolve(rootDir, 'sdk/lua/ctx_types.lua');
  const dalTypesContent = fs.readFileSync(dalTypesPath, 'utf-8');
  const sdkCtxTypesContent = fs.readFileSync(sdkCtxTypesPath, 'utf-8');
  assertExpect(dalTypesContent).toBe(sdkCtxTypesContent);
  assertExpect(dalTypesContent.split('\n').length).toBeLessThan(500);
  assertExpect(sdkCtxTypesContent.split('\n').length).toBeLessThan(500);

  for (const sig of EXPECTED_DAL_TYPE_SIGNATURES) {
    assertExpect(dalTypesContent, `Expected ${sig} in .mis-types/dal.d.lua`).toContain(sig);
  }

  const recordsTypesPath = path.resolve(rootDir, '.mis-types/records.d.lua');
  const sdkRecordsTypesPath = path.resolve(rootDir, 'sdk/lua/records.d.lua');
  const recordsContent = fs.readFileSync(recordsTypesPath, 'utf-8');
  const sdkRecordsContent = fs.readFileSync(sdkRecordsTypesPath, 'utf-8');
  assertExpect(recordsContent).toBe(sdkRecordsContent);
  assertExpect(recordsContent.split('\n').length).toBeLessThan(500);
  assertExpect(sdkRecordsContent.split('\n').length).toBeLessThan(500);
  assertExpect(recordsContent).toContain('OptimisPupilRecord');
  assertExpect(recordsContent).toContain('OptimisStaffRecord');
  assertExpect(recordsContent).toContain('OptimisGuardianRecord');
  assertExpect(recordsContent).toContain('OptimisSenAndMedicalSummary');
}
