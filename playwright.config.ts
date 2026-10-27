import { defineConfig } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

const envPath = path.resolve(__dirname, '.env');
if (!fs.existsSync(envPath)) {
  throw new Error(`Mandatory .env file is missing at ${envPath}`);
}
process.loadEnvFile(envPath);

const baseURL = process.env.BASE_URL;
if (!baseURL) {
  throw new Error('Missing mandatory environment variable: BASE_URL');
}

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: 'list',
  use: {
    baseURL,
    trace: 'retain-on-failure',
    headless: true,
  },
});
