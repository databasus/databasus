import { defineConfig } from '@playwright/test';

export default defineConfig({
  tsconfig: './tsconfig.node.json',
  testDir: './browser/scenarios',
  projects: [{ name: 'ui', testDir: './browser/scenarios', use: { browserName: 'chromium' } }],
  forbidOnly: true,
  fullyParallel: true,
  workers: 2,
  retries: 0,
  timeout: 30_000,
  expect: { timeout: 5_000, toHaveScreenshot: { animations: 'disabled', maxDiffPixels: 0 } },
  updateSnapshots: process.env.UPDATE_REFERENCES === '1' ? 'all' : 'none',
  snapshotPathTemplate: '{testDir}/../references/{arg}{ext}',
  use: {
    baseURL: process.env.BROWSER_BASE_URL,
    locale: 'en-US',
    timezoneId: 'UTC',
    reducedMotion: 'reduce',
    viewport: { width: 1280, height: 900 },
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  reporter: [['list']],
});
