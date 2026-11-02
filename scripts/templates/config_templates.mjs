/**
 * Configuration & Runner Script Templates for Optimis Plugin Starter
 */

export const CONFIG_TEMPLATES = {
  vscodeExtensions: () =>
    JSON.stringify(
      {
        recommendations: ['sumneko.lua'],
      },
      null,
      2
    ) + '\n',

  runCli: () => `#!/usr/bin/env node
import { spawnSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const rootDir = path.resolve(__dirname, '..');

const toolName = process.argv[2];
const toolArgs = process.argv.slice(3);

if (!toolName) {
  console.error('Usage: node scripts/run-cli.mjs <mis-verify|mis-test|mis-dev-server|mis-pack> [args...]');
  process.exit(1);
}

const platformMap = {
  linux: 'linux',
  darwin: 'darwin',
  win32: 'windows',
};
const archMap = {
  x64: 'amd64',
  arm64: 'arm64',
};

const osName = platformMap[process.platform] || process.platform;
const archName = archMap[process.arch] || process.arch;
const ext = process.platform === 'win32' ? '.exe' : '';

const platformBin = path.join(rootDir, 'bin', \`\${osName}-\${archName}\`, \`\${toolName}\${ext}\`);
const fallbackBin = path.join(rootDir, 'bin', \`\${toolName}\${ext}\`);
const binPath = fs.existsSync(platformBin) ? platformBin : fallbackBin;

if (!fs.existsSync(binPath)) {
  console.error(\`Toolchain binary not found at \${platformBin} or \${fallbackBin}\`);
  process.exit(1);
}

const res = spawnSync(binPath, toolArgs, {
  cwd: rootDir,
  stdio: 'inherit',
  env: process.env,
});

process.exit(res.status ?? 1);
`,

  packageJson: () =>
    JSON.stringify(
      {
        name: 'optimis-plugin-starter',
        version: '1.0.0',
        private: true,
        description: 'Starter template for building sandboxed Optimis School MIS (.mispkg) plugins',
        scripts: {
          dev: 'node scripts/run-cli.mjs mis-dev-server --plugin . --persona stf_tch_7a --port 18080',
          verify: 'node scripts/run-cli.mjs mis-verify .',
          test: 'node scripts/run-cli.mjs mis-test . --persona stf_tch_7a',
          pack: 'node scripts/run-cli.mjs mis-pack .',
        },
      },
      null,
      2
    ) + '\n',

  gitignore: () => `node_modules/
*.db
*.db-wal
*.db-shm
*.mispkg
.DS_Store
`,
};
