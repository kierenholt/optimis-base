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
const dataDirPath = process.env.DATA_DIR_PATH ||
  (process.env.DATABASE_PATH ? path.dirname(path.resolve(process.env.DATABASE_PATH)) : path.resolve(process.cwd(), '.data'));
const logFilePath = path.join(dataDirPath, 'mis-server.log');

const ANSI = {
  reset: '\x1b[0m',
  dim: '\x1b[2m',
  bold: '\x1b[1m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  red: '\x1b[31m',
  cyan: '\x1b[36m',
  magenta: '\x1b[35m',
};

const rawArgs = process.argv.slice(2);
const showAllHistory = !rawArgs.includes('--new-only');
const fileOnlyMode = rawArgs.includes('--file');
let filterPattern = '';
for (let i = 0; i < rawArgs.length; i++) {
  if (rawArgs[i] === '--filter' && rawArgs[i + 1]) {
    filterPattern = rawArgs[i + 1].toLowerCase();
    i++;
  } else if (rawArgs[i].startsWith('--filter=')) {
    filterPattern = rawArgs[i].slice('--filter='.length).toLowerCase();
  }
}

function formatLogLine(line) {
  if (!line || !line.trim()) return null;
  if (filterPattern && !line.toLowerCase().includes(filterPattern)) {
    return null;
  }

  const match = line.match(/^\[([^\]]+)\]\s+(INFO|ERROR|DIAGNOSTIC)\s+(.*)$/);
  if (!match) {
    return `${ANSI.dim}${line}${ANSI.reset}`;
  }
  const [, ts, level, rest] = match;
  let badge = `${ANSI.green}INFO ${ANSI.reset}`;
  let msgColor = ANSI.reset;
  if (level === 'ERROR') {
    badge = `${ANSI.red}${ANSI.bold}ERROR${ANSI.reset}`;
    msgColor = ANSI.red;
  } else if (level === 'DIAGNOSTIC') {
    badge = `${ANSI.magenta}${ANSI.bold}DIAG ${ANSI.reset}`;
    msgColor = ANSI.yellow;
  }

  return `${ANSI.dim}[${ts}]${ANSI.reset} ${badge} ${msgColor}${rest}${ANSI.reset}`;
}

function printFormatted(line) {
  const formatted = formatLogLine(line);
  if (formatted !== null) {
    console.log(formatted);
  }
}

async function tailViaFile() {
  console.log(`${ANSI.cyan}[logs:tail] Tailing local log file: ${logFilePath}${ANSI.reset}`);
  if (filterPattern) {
    console.log(`${ANSI.cyan}[logs:tail] Active filter: "${filterPattern}"${ANSI.reset}`);
  }
  console.log(`${ANSI.dim}Press Ctrl+C to stop tailing.${ANSI.reset}\n`);

  let offset = 0;
  if (fs.existsSync(logFilePath)) {
    const stat = fs.statSync(logFilePath);
    if (showAllHistory) {
      const initial = fs.readFileSync(logFilePath, 'utf8');
      for (const line of initial.split('\n')) {
        printFormatted(line);
      }
    }
    offset = stat.size;
  } else {
    console.log(`${ANSI.yellow}[logs:tail] Waiting for ${logFilePath} to be created...${ANSI.reset}`);
  }

  setInterval(() => {
    if (!fs.existsSync(logFilePath)) return;
    const stat = fs.statSync(logFilePath);
    if (stat.size < offset) {
      // File was truncated or recreated
      offset = 0;
    }
    if (stat.size > offset) {
      const fd = fs.openSync(logFilePath, 'r');
      const length = stat.size - offset;
      const buffer = Buffer.alloc(length);
      fs.readSync(fd, buffer, 0, length, offset);
      fs.closeSync(fd);
      offset = stat.size;
      const chunk = buffer.toString('utf8');
      for (const line of chunk.split('\n')) {
        printFormatted(line);
      }
    }
  }, 250);
}

async function tailViaHttpAndFileFallback() {
  if (fileOnlyMode) {
    return tailViaFile();
  }

  console.log(`${ANSI.cyan}[logs:tail] Connecting to Optimis Host log stream at ${baseUrl} (fallback file: ${logFilePath})${ANSI.reset}`);
  if (filterPattern) {
    console.log(`${ANSI.cyan}[logs:tail] Active filter: "${filterPattern}"${ANSI.reset}`);
  }
  console.log(`${ANSI.dim}Press Ctrl+C to stop tailing.${ANSI.reset}\n`);

  let cursor = '0';
  let seenDiagCount = 0;
  let connectedOnce = false;
  let waitingNoticePrinted = false;

  if (!showAllHistory) {
    try {
      const curRes = await fetch(`${baseUrl}/api/test/logs/cursor`);
      if (curRes.ok) {
        const curData = await curRes.json();
        cursor = String(curData.cursor || '0');
      }
    } catch {
      // Server not reachable yet
    }
  }

  const poll = async () => {
    try {
      const res = await fetch(`${baseUrl}/api/test/logs?since=${encodeURIComponent(cursor)}`);
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
      }
      const data = await res.json();
      if (!connectedOnce) {
        connectedOnce = true;
        waitingNoticePrinted = false;
        console.log(`${ANSI.green}[logs:tail] Connected to live Host Server at ${baseUrl}${ANSI.reset}`);
      }
      const entries = Array.isArray(data.entries) ? data.entries : [];
      if (entries.length > 0) {
        for (const entry of entries) {
          printFormatted(entry);
        }
        cursor = String(Number(cursor) + entries.length);
      } else {
        // Check if server restarted (cursor reset to a smaller number than our current cursor)
        const curRes = await fetch(`${baseUrl}/api/test/logs/cursor`);
        if (curRes.ok) {
          const curData = await curRes.json();
          const serverCursor = Number(curData.cursor || 0);
          if (serverCursor < Number(cursor)) {
            console.log(`${ANSI.yellow}[logs:tail] Server restart detected — resetting log cursor to 0...${ANSI.reset}`);
            cursor = '0';
            seenDiagCount = 0;
          }
        }
      }

      // Also poll sanitized Lua diagnostic ring buffer as Admin
      const diagRes = await fetch(`${baseUrl}/api/admin/diagnostics/bundle?staff_id=stf_admin_01`, {
        headers: { 'X-MIS-Staff-ID': 'stf_admin_01' },
      });
      if (diagRes.ok) {
        const diagData = await diagRes.json();
        const diagLines = typeof diagData.log_archive === 'string' && diagData.log_archive.trim()
          ? diagData.log_archive.trim().split('\n')
          : [];
        if (diagLines.length < seenDiagCount) {
          seenDiagCount = 0;
        }
        if (diagLines.length > seenDiagCount) {
          for (let i = seenDiagCount; i < diagLines.length; i++) {
            printFormatted(diagLines[i]);
          }
          seenDiagCount = diagLines.length;
        }
      }
    } catch {
      if (!waitingNoticePrinted) {
        waitingNoticePrinted = true;
        connectedOnce = false;
        console.log(
          `${ANSI.yellow}[logs:tail] Host server at ${baseUrl} is not reachable right now — waiting for server...${ANSI.reset}`
        );
      }
    }
  };

  await poll();
  setInterval(poll, 400);
}

tailViaHttpAndFileFallback();
