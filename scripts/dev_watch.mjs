import { spawn, spawnSync } from 'node:child_process';
import * as fs from 'node:fs';
import * as path from 'node:path';

const rootDir = process.cwd();

console.log('[dev] Running initial build (UI + Go Host)...');
const initBuild = spawnSync('npm', ['run', 'build'], { cwd: rootDir, stdio: 'inherit' });
if (initBuild.status !== 0) {
  process.exit(initBuild.status ?? 1);
}

// 1. Start Vite watch for live React Host UI Shell rebuilds
const viteWatcher = spawn('npx', ['vite', 'build', '--watch', '--config', 'ui/vite.config.ts'], {
  cwd: rootDir,
  stdio: 'inherit',
});

// 2. Manage the Go Host Server process
let goServer = null;
let rebuildTimer = null;

function startGoServer() {
  if (goServer) {
    goServer.kill('SIGTERM');
    goServer = null;
  }
  goServer = spawn('./bin/mis-server', [], {
    cwd: rootDir,
    stdio: 'inherit',
  });
}

function rebuildAndRestartGo(changedFile) {
  console.log(`\n[dev] Go source changed (${changedFile}) — recompiling bin/mis-server...`);
  const res = spawnSync('npm', ['run', 'build:go'], { cwd: rootDir, stdio: 'inherit' });
  if (res.status === 0) {
    console.log('[dev] Restarting bin/mis-server...');
    startGoServer();
  }
}

startGoServer();

// 3. Watch Go source directories (cmd/ and internal/)
for (const dir of ['cmd', 'internal']) {
  const targetDir = path.join(rootDir, dir);
  if (!fs.existsSync(targetDir)) continue;

  fs.watch(targetDir, { recursive: true }, (_eventType, filename) => {
    if (!filename || !filename.endsWith('.go') || filename.endsWith('_test.go')) {
      return;
    }
    if (rebuildTimer) {
      clearTimeout(rebuildTimer);
    }
    rebuildTimer = setTimeout(() => rebuildAndRestartGo(path.join(dir, filename)), 150);
  });
}

function shutdown() {
  if (goServer) goServer.kill('SIGTERM');
  if (viteWatcher) viteWatcher.kill('SIGTERM');
  process.exit(0);
}

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);
