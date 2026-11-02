#!/usr/bin/env node
/**
 * Populate the standalone `optimis-plugin-template` repository with:
 * - Pre-compiled SDK CLI binaries (`./bin/` + cross-platform `./bin/<os>-<arch>/`)
 * - Editable Core SQLite schema (`schema/core/*.sql` + `SCHEMA_MAP.md`)
 * - IDE autocomplete & validation (`.vscode/`, `.mis-types/dal.d.lua`, `sdk/`)
 * - Minimal Vanilla HTML/JS + Lua + SQLite starter plugin (`manifest.json`, `migrations/`, `src/`, `ui/dist/`, `tests/`)
 * - Local developer scripts (`package.json`, `scripts/run-cli.mjs`, `README.md`)
 * - Agent rules, skills, and architectural documentation (`.agents/`, `GEMINI.md`)
 */

import { execFileSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { CONFIG_TEMPLATES } from './templates/config_templates.mjs';
import { CODE_TEMPLATES } from './templates/starter_code.mjs';
import { DOC_TEMPLATES } from './templates/documentation.mjs';

// =============================================================================
// Constants & Configuration
// =============================================================================

export const CLI_TOOLS = ['mis-dev-server', 'mis-test', 'mis-verify', 'mis-pack'];

export const BUILD_TARGETS = [
  { os: 'linux', arch: 'amd64', ext: '' },
  { os: 'darwin', arch: 'arm64', ext: '' },
  { os: 'darwin', arch: 'amd64', ext: '' },
  { os: 'windows', arch: 'amd64', ext: '.exe' },
];

export const REQUIRED_ROOT_SOURCES = [
  '.vscode/settings.json',
  '.mis-types/dal.d.lua',
  '.mis-types/records.d.lua',
  'sdk',
  'schema/core',
];

// =============================================================================
// Core Manifest & Composed Templates
// =============================================================================

export const TEMPLATES = {
  ...CONFIG_TEMPLATES,
  ...CODE_TEMPLATES,
  ...DOC_TEMPLATES,

  manifest: () =>
    JSON.stringify(
      {
        id: 'uk.org.optimis.starter.notes',
        title: 'Classroom Praise & Notes Starter',
        version: '1.0.0',
        core_dal_version: '~1.0',
        icon: 'award',
        max_concurrency: 4,
        max_memory_mb: 32,
        database: {
          alias: 'plugin_starter_notes',
          max_db_size_mb: 100,
        },
        views: [
          {
            id: 'home',
            path: '/',
            entry: 'ui/dist/index.html',
            title: 'Classroom Praise & Notes',
            icon: 'award',
            query_params: ['group_id'],
            keywords: ['praise', 'notes', 'recognition', 'starter', 'commendation'],
          },
          {
            id: 'student_notes',
            path: '/student/:pupil_id',
            entry: 'ui/dist/index.html',
            title: 'Pupil Praise Log',
            icon: 'user-check',
            query_params: ['tab'],
            keywords: ['pupil praise', 'student notes'],
          },
        ],
        widgets: [
          {
            id: 'praise_summary_widget',
            title: 'Recent Classroom Praise',
            description: 'Displays recent classroom praise and commendation notes for your assigned teaching groups.',
            icon: 'award',
            entry: 'ui/dist/widget.html',
            width: 1,
            height: 1,
            keywords: ['praise widget', 'commendations', 'recent notes'],
          },
        ],
        endpoints: {
          load_group_praise_roster: {
            scopes: ['dal.pupils.read:own_groups', 'dal.groups.read:own_groups'],
            mutates: false,
            timeout_ms: 2500,
          },
          record_pupil_praise_note: {
            scopes: ['dal.pupils.read:own_groups', 'dal.groups.read:own_groups'],
            mutates: true,
            timeout_ms: 2500,
          },
          get_recent_praise_summary: {
            scopes: ['dal.pupils.read:own_groups', 'dal.groups.read:own_groups'],
            mutates: false,
            timeout_ms: 2500,
          },
        },
        gdpr_hooks: {
          export_pupil_data: 'export_pupil_data',
          purge_pupil_data: 'purge_pupil_data',
        },
      },
      null,
      2
    ) + '\n',
};

// =============================================================================
// File System & Path Helpers
// =============================================================================

export function createFSUtils(rootDir, targetDir) {
  const ensureDir = (relPath) => {
    const fullPath = path.join(targetDir, relPath);
    fs.mkdirSync(fullPath, { recursive: true });
    return fullPath;
  };

  const writeFile = (relPath, content, executable = false) => {
    const fullPath = path.join(targetDir, relPath);
    fs.mkdirSync(path.dirname(fullPath), { recursive: true });
    fs.writeFileSync(fullPath, content, 'utf-8');
    if (executable) {
      fs.chmodSync(fullPath, 0o755);
    }
    return fullPath;
  };

  const copyFile = (srcRelPath, destRelPath) => {
    const srcFullPath = path.join(rootDir, srcRelPath);
    const destFullPath = path.join(targetDir, destRelPath);
    if (!fs.existsSync(srcFullPath)) {
      throw new Error(`FAIL FAST: Source file does not exist: ${srcFullPath}`);
    }
    fs.mkdirSync(path.dirname(destFullPath), { recursive: true });
    fs.copyFileSync(srcFullPath, destFullPath);
    return destFullPath;
  };

  const copyDir = (srcRelPath, destRelPath) => {
    const srcFullPath = path.join(rootDir, srcRelPath);
    const destFullPath = path.join(targetDir, destRelPath);
    if (!fs.existsSync(srcFullPath)) {
      throw new Error(`FAIL FAST: Source directory does not exist: ${srcFullPath}`);
    }
    fs.mkdirSync(destFullPath, { recursive: true });
    fs.cpSync(srcFullPath, destFullPath, { recursive: true });
    return destFullPath;
  };

  const sanitizeMarkdownLinks = (dirRelPath) => {
    const fullDir = path.join(targetDir, dirRelPath);
    if (!fs.existsSync(fullDir)) return;

    const sanitizeDirRecursively = (currentDir) => {
      for (const entry of fs.readdirSync(currentDir, { withFileTypes: true })) {
        const entryPath = path.join(currentDir, entry.name);
        if (entry.isDirectory()) {
          sanitizeDirRecursively(entryPath);
        } else if (entry.name === 'GEMINI.md' || entry.name.endsWith('.md')) {
          const raw = fs.readFileSync(entryPath, 'utf-8');
          const cleaned = raw.replace(/file:\/\/\/home\/kieren\/Documents\/code\/optimis\//g, './');
          if (cleaned !== raw) {
            fs.writeFileSync(entryPath, cleaned, 'utf-8');
          }
        }
      }
    };

    sanitizeDirRecursively(fullDir);
  };

  return {
    ensureDir,
    writeFile,
    copyFile,
    copyDir,
    sanitizeMarkdownLinks,
  };
}

// =============================================================================
// Validation & Pre-flight Checks
// =============================================================================

export function validatePrerequisites(rootDir, targetDir) {
  if (!fs.existsSync(targetDir)) {
    throw new Error(`FAIL FAST: Target template directory does not exist: ${targetDir}`);
  }

  for (const relSrc of REQUIRED_ROOT_SOURCES) {
    const fullPath = path.join(rootDir, relSrc);
    if (!fs.existsSync(fullPath)) {
      throw new Error(`FAIL FAST: Required Optimis core source path missing: ${fullPath}`);
    }
  }

  for (const tool of CLI_TOOLS) {
    const toolSrc = path.join(rootDir, 'cmd', tool);
    if (!fs.existsSync(toolSrc)) {
      throw new Error(`FAIL FAST: Required CLI tool command source directory missing: ${toolSrc}`);
    }
  }
}

// =============================================================================
// Step Functions
// =============================================================================

export function copySharedAssets(fsUtils) {
  fsUtils.ensureDir('.vscode');
  fsUtils.copyFile('.vscode/settings.json', '.vscode/settings.json');
  fsUtils.writeFile('.vscode/extensions.json', TEMPLATES.vscodeExtensions());

  fsUtils.ensureDir('.mis-types');
  fsUtils.copyFile('.mis-types/dal.d.lua', '.mis-types/dal.d.lua');
  fsUtils.copyFile('.mis-types/records.d.lua', '.mis-types/records.d.lua');

  fsUtils.copyDir('sdk', 'sdk');
  fsUtils.copyDir('schema/core', 'schema/core');
}

export function compileCliBinaries(fsUtils, rootDir, targetDir, skipBuild = false) {
  fsUtils.ensureDir('bin');
  if (skipBuild) {
    console.log('  [build] Skipping CLI compilation (--skip-build specified).');
    return;
  }

  for (const t of BUILD_TARGETS) {
    const platformSubdir = path.join(targetDir, 'bin', `${t.os}-${t.arch}`);
    fs.mkdirSync(platformSubdir, { recursive: true });

    for (const tool of CLI_TOOLS) {
      const outPath = path.join(platformSubdir, `${tool}${t.ext}`);
      console.log(`  [build] Compiling ${tool} for ${t.os}/${t.arch}...`);
      execFileSync('go', ['build', '-trimpath', '-ldflags=-s -w', '-o', outPath, `./cmd/${tool}`], {
        cwd: rootDir,
        env: {
          ...process.env,
          CGO_ENABLED: '0',
          GOOS: t.os,
          GOARCH: t.arch,
        },
        stdio: 'inherit',
      });

      if (t.os === 'linux' && t.arch === 'amd64') {
        const topBinPath = path.join(targetDir, 'bin', tool);
        fs.copyFileSync(outPath, topBinPath);
        fs.chmodSync(topBinPath, 0o755);
      }
    }
  }
}

export function writeDeveloperScripts(fsUtils) {
  fsUtils.writeFile('scripts/run-cli.mjs', TEMPLATES.runCli(), true);
  fsUtils.writeFile('package.json', TEMPLATES.packageJson());
  fsUtils.writeFile('.gitignore', TEMPLATES.gitignore());
}

export function writeStarterPlugin(fsUtils) {
  fsUtils.writeFile('manifest.json', TEMPLATES.manifest());
  fsUtils.writeFile('migrations/001_init.sql', TEMPLATES.initSql());
  fsUtils.writeFile('src/init.lua', TEMPLATES.initLua());
  fsUtils.writeFile('ui/dist/index.html', TEMPLATES.uiIndexHtml());
  fsUtils.writeFile('ui/dist/widget.html', TEMPLATES.uiWidgetHtml());
  fsUtils.writeFile('tests/init_test.lua', TEMPLATES.initTestLua());
}

export function writeAgentGuidelines(fsUtils) {
  fsUtils.writeFile('.agents/AGENTS.md', TEMPLATES.agentsMd());
  fsUtils.writeFile('.agents/GEMINI.md', TEMPLATES.agentsGeminiMd());
  fsUtils.writeFile('.agents/skills/test-author/SKILL.md', TEMPLATES.skillTestAuthor());
  fsUtils.writeFile('.agents/skills/code-implementer/SKILL.md', TEMPLATES.skillCodeImplementer());
}

export function writeDocumentation(fsUtils) {
  fsUtils.writeFile('README.md', TEMPLATES.readmeMd());
  fsUtils.writeFile('GEMINI.md', TEMPLATES.rootGeminiMd());
  fsUtils.writeFile('.mis-types/GEMINI.md', TEMPLATES.misTypesGeminiMd());
  fsUtils.writeFile('bin/GEMINI.md', TEMPLATES.binGeminiMd());
  fsUtils.writeFile('migrations/GEMINI.md', TEMPLATES.migrationsGeminiMd());
  fsUtils.writeFile('src/GEMINI.md', TEMPLATES.srcGeminiMd());
  fsUtils.writeFile('ui/GEMINI.md', TEMPLATES.uiGeminiMd());
  fsUtils.writeFile('tests/GEMINI.md', TEMPLATES.testsGeminiMd());
  fsUtils.writeFile('scripts/GEMINI.md', TEMPLATES.scriptsGeminiMd());
}

export function sanitizeDocumentationLinks(fsUtils) {
  fsUtils.sanitizeMarkdownLinks('sdk');
  fsUtils.sanitizeMarkdownLinks('schema/core');
}

// =============================================================================
// Main Orchestration Function
// =============================================================================

export function populatePluginTemplate(options = {}) {
  const rootDir = options.rootDir ? path.resolve(options.rootDir) : process.cwd();
  const defaultTarget = path.resolve(rootDir, '../optimis-plugin-template');
  const targetDir = options.targetDir ? path.resolve(options.targetDir) : defaultTarget;
  const skipBuild = options.skipBuild ?? false;

  console.log('=================================================================');
  console.log(`Populating Optimis Plugin Template`);
  console.log(`Source Root : ${rootDir}`);
  console.log(`Target Dir  : ${targetDir}`);
  console.log(`Skip Build  : ${skipBuild}`);
  console.log('=================================================================');

  const startTime = Date.now();

  console.log('[1/7] Validating prerequisites and target directory...');
  validatePrerequisites(rootDir, targetDir);

  const fsUtils = createFSUtils(rootDir, targetDir);

  console.log('[2/7] Copying shared assets (.vscode/, .mis-types/, sdk/, schema/core/)...');
  copySharedAssets(fsUtils);

  console.log('[3/7] Compiling SDK CLI binaries...');
  compileCliBinaries(fsUtils, rootDir, targetDir, skipBuild);

  console.log('[4/7] Writing developer scripts & configuration (run-cli.mjs, package.json, .gitignore)...');
  writeDeveloperScripts(fsUtils);

  console.log('[5/7] Writing starter plugin files (manifest.json, SQL, Lua, UI, tests)...');
  writeStarterPlugin(fsUtils);

  console.log('[6/7] Writing agent guidelines & skills (.agents/)...');
  writeAgentGuidelines(fsUtils);

  console.log('[7/7] Writing documentation and sanitizing relative markdown links...');
  writeDocumentation(fsUtils);
  sanitizeDocumentationLinks(fsUtils);

  const elapsedSec = ((Date.now() - startTime) / 1000).toFixed(2);
  console.log('-----------------------------------------------------------------');
  console.log(`Successfully populated Optimis Plugin Template in ${elapsedSec}s!`);
  console.log('=================================================================');
}

// =============================================================================
// CLI Argument Parsing & Invocation
// =============================================================================

export function parseArgs(argv = process.argv.slice(2)) {
  let targetDir = null;
  let skipBuild = false;

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === '--help' || arg === '-h') {
      console.log(`
Usage: node scripts/populate_plugin_template.mjs [targetDir] [options]

Arguments:
  targetDir            Target directory (default: ../optimis-plugin-template)

Options:
  --target <dir>       Explicit target directory
  --skip-build         Skip cross-compilation of SDK CLI binaries
  -h, --help           Show this help message
`);
      process.exit(0);
    } else if (arg === '--skip-build') {
      skipBuild = true;
    } else if (arg === '--target') {
      i++;
      if (i >= argv.length || argv[i].startsWith('-')) {
        throw new Error('FAIL FAST: Missing value for --target flag');
      }
      targetDir = argv[i];
    } else if (!arg.startsWith('-') && !targetDir) {
      targetDir = arg;
    }
  }

  return { targetDir, skipBuild };
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  populatePluginTemplate({
    rootDir: process.cwd(),
    targetDir: args.targetDir,
    skipBuild: args.skipBuild,
  });
}

const currentFile = fileURLToPath(import.meta.url);
const invokedFile = process.argv[1] ? path.resolve(process.argv[1]) : '';
if (invokedFile === currentFile) {
  try {
    main();
  } catch (err) {
    console.error(`Error populating plugin template: ${err.message}`);
    process.exit(1);
  }
}
