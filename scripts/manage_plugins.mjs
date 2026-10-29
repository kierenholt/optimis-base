#!/usr/bin/env node
import * as fs from 'node:fs';
import * as path from 'node:path';

function loadEnv() {
  const envPath = path.resolve(process.cwd(), '.env');
  if (!fs.existsSync(envPath)) {
    throw new Error(`Mandatory .env file missing at ${envPath}`);
  }
  const content = fs.readFileSync(envPath, 'utf8');
  for (const rawLine of content.split('\n')) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#')) continue;
    const eqIdx = line.indexOf('=');
    if (eqIdx === -1) continue;
    const key = line.slice(0, eqIdx).trim();
    const val = line.slice(eqIdx + 1).trim();
    if (!process.env[key]) {
      process.env[key] = val;
    }
  }
}

loadEnv();

const baseUrl = (process.env.BASE_URL || `http://127.0.0.1:${process.env.PORT || '18080'}`).replace(/\/+$/, '');
const pluginsDir = process.env.PLUGINS_DIR || path.resolve(process.cwd(), 'plugins');
const adminStaffId = 'stf_admin_01';

function discoverPluginFolders(includeTestPlugin = false) {
  if (!fs.existsSync(pluginsDir)) {
    throw new Error(`PLUGINS_DIR not found at ${pluginsDir}`);
  }
  const entries = fs.readdirSync(pluginsDir, { withFileTypes: true });
  const folders = [];
  for (const entry of entries) {
    if (!entry.isDirectory()) continue;
    if (!includeTestPlugin && entry.name === 'test_plugin') continue;
    const manifestPath = path.join(pluginsDir, entry.name, 'manifest.json');
    if (fs.existsSync(manifestPath)) {
      folders.push(entry.name);
    }
  }
  return folders.sort();
}

async function apiRequest(method, route, body = undefined) {
  const url = `${baseUrl}${route}`;
  const res = await fetch(url, {
    method,
    headers: {
      'Content-Type': 'application/json',
      'X-MIS-Staff-ID': adminStaffId,
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok || data.ok === false) {
    throw new Error(`HTTP ${res.status} on ${method} ${route}: ${data.error_code || data.error || JSON.stringify(data)}`);
  }
  return data;
}

async function listInstalledPlugins() {
  const data = await apiRequest('GET', `/api/admin/plugins?staff_id=${encodeURIComponent(adminStaffId)}`);
  return Array.isArray(data.plugins) ? data.plugins : [];
}

async function installAndActivateFolder(folder) {
  const installRes = await apiRequest('POST', '/api/admin/plugins/install', {
    folder,
    staff_id: adminStaffId,
  });
  const pluginId = installRes.plugin_id;
  await apiRequest('POST', `/api/admin/plugins/${encodeURIComponent(pluginId)}/activate`, {
    staff_id: adminStaffId,
  });
  console.log(`  [+] Installed & Activated: ${pluginId.padEnd(36)} (folder: ${folder})`);
}

async function removeAllPlugins() {
  const current = await listInstalledPlugins();
  if (current.length === 0) {
    console.log('No plugins are currently installed.');
    return;
  }
  for (const p of current) {
    await apiRequest('DELETE', `/api/admin/plugins/${encodeURIComponent(p.id)}`, {
      staff_id: adminStaffId,
    });
    console.log(`  [-] Removed plugin:        ${p.id.padEnd(36)} (folder: ${p.folder})`);
  }
}

async function printStatus() {
  const current = await listInstalledPlugins();
  console.log(`\nOptimis Host Server: ${baseUrl}`);
  console.log(`Installed Plugins  : ${current.length}\n`);
  if (current.length === 0) {
    console.log('  (Zero plugins currently installed — clean Host UI Shell mode)');
    return;
  }
  console.log('  PLUGIN ID                            STATE     VERSION  FOLDER');
  console.log('  -----------------------------------  --------  -------  ------------------------------');
  for (const p of current) {
    console.log(
      `  ${(p.id || '').padEnd(35)}  ${(p.state || '').padEnd(8)}  ${(p.version || '').padEnd(7)}  ${p.folder || ''}`
    );
  }
  console.log('');
}

async function main() {
  const args = process.argv.slice(2);
  const action = (args[0] || 'status').toLowerCase();
  const includeTest = args.includes('--include-test-plugin');

  try {
    if (action === 'all' || action === 'install-all') {
      const folders = discoverPluginFolders(includeTest);
      console.log(`Installing and activating ${folders.length} plugins from ${pluginsDir}...`);
      for (const folder of folders) {
        await installAndActivateFolder(folder);
      }
      await printStatus();
    } else if (action === 'none' || action === 'remove-all' || action === 'clear') {
      console.log('Removing all active plugins from the running Host server...');
      await removeAllPlugins();
      await printStatus();
    } else if (action === 'load' || action === 'install') {
      const targetFolders = args
        .slice(1)
        .filter((a) => !a.startsWith('--'))
        .flatMap((a) => a.split(','))
        .map((s) => s.trim())
        .filter(Boolean);
      if (targetFolders.length === 0) {
        throw new Error('Usage: npm run plugins:load -- <plugin_folder_1> [plugin_folder_2 ...]');
      }
      for (const folder of targetFolders) {
        await installAndActivateFolder(folder);
      }
      await printStatus();
    } else if (action === 'status' || action === 'list') {
      await printStatus();
    } else {
      throw new Error(`Unknown command "${action}". Expected: all | none | load <folder> | status`);
    }
  } catch (err) {
    console.error(`\n[plugins] ERROR: ${err.message}`);
    console.error(`Make sure the Optimis Host Server is running at ${baseUrl} (e.g., via "npm run dev" or "npm start").`);
    process.exit(1);
  }
}

main();
