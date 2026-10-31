import { test as base } from '@playwright/test';

import { applicationFixtures } from './applicationFixtures';
import { installHttpFixtures, type HttpFixture } from './HttpFixture';

export const test = base.extend<{
  httpFixtures: HttpFixture[];
  boundary: Awaited<ReturnType<typeof installHttpFixtures>>;
}>({
  httpFixtures: [applicationFixtures(), { option: true }],
  boundary: [
    async ({ page, httpFixtures }, use) => {
      const boundary = await installHttpFixtures({ page, fixtures: httpFixtures });
      await use(boundary);
      boundary.assertNoFailures();
    },
    { auto: true },
  ],
});
