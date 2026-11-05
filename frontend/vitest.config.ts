import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  server: {
    fs: {
      // websitePages.test.ts reads ../website/app/i18n.ts to check the application's links to
      // website pages against the pages the website translates.
      allow: ['..'],
    },
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.{ts,tsx}', 'tooling/**/*.test.ts'],
    setupFiles: ['src/shared/lib/test/conditionWait.ts'],
    // The only time limit of a test. Its waits follow conditions, so this bounds a hung test and
    // does not pace a healthy one, however loaded the machine is.
    testTimeout: 60_000,
    hookTimeout: 60_000,
  },
});
