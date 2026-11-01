import { defineConfig } from 'vitest/config';

import configuration from './vitest.config';

export default defineConfig({
  ...configuration,
  test: {
    ...configuration.test,
    include: ['tooling/runBrowserE2E.test.ts'],
    exclude: [],
    // Each test drives whole real browser and container runs, which take longer than a screen test.
    testTimeout: 180_000,
  },
});
