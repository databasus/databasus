import { defineConfig } from 'vitest/config';

import configuration from './vitest.config';

export default defineConfig({
  ...configuration,
  test: { ...configuration.test, include: ['tooling/runBrowserE2E.test.ts'], exclude: [] },
});
