import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { database, databaseFixtures } from '../databaseFixtures';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';
import { visualContexts } from '../visualContexts';

test.use({
  // eslint-disable-next-line no-empty-pattern -- Playwright requires an object pattern for fixture dependencies
  httpFixtures: async ({}, provideFixtures) => {
    await provideFixtures(
      databaseFixtures().map((fixture) =>
        fixture.path === '/healthcheck-configs/{databaseId}'
          ? apiResponse({
              method: 'get',
              path: '/healthcheck-configs/{databaseId}',
              status: 200,
              body: {
                databaseId: database.id,
                isHealthcheckEnabled: true,
                isNotificationEnabled: false,
                intervalMinutes: 1,
                failedAttemptsBeforeUnavailable: 3,
                attemptRetentionDays: 7,
              },
            })
          : fixture,
      ),
    );
  },
});

for (const context of visualContexts) {
  test(`retains healthcheck recovery, empty history and timestamp tooltips in ${context.theme} ${context.locale} ${context.width}`, async ({
    page,
    httpFixtures,
  }) => {
    const dictionary = DICTIONARIES[context.locale];
    const attempts = httpFixtures.find((fixture) => fixture.path === '/healthcheck-attempts')!;
    attempts.status = 500;
    attempts.body = {
      type: 'urn:databasus:problem:internal',
      title: 'Internal',
      detail: 'Internal',
      status: 500,
      code: 'internal',
    };
    await page.setViewportSize({ width: context.width, height: 1000 });
    await openAccount({ page, ...context });
    await openWorkspaceScreen({ page, locale: context.locale, tab: 'databases' });
    if (context.width === 320) await page.getByRole('link', { name: /^Orders/ }).click();
    const retry = page.getByRole('button', { name: dictionary.common.actions.retry, exact: true });
    await expect(retry).toBeVisible();
    await assertEnlargedTextReflow(page);

    attempts.status = 200;
    attempts.body = { items: [] };
    await retry.click();
    await expect(
      page.getByText(dictionary.healthcheck.attempts.empty, { exact: true }),
    ).toBeVisible();
    await assertEnlargedTextReflow(page);

    attempts.body = {
      items: [
        { id: 'available-attempt', status: 'AVAILABLE', createdAt: '2026-10-18T10:00:00Z' },
        { id: 'unavailable-attempt', status: 'UNAVAILABLE', createdAt: '2026-10-18T09:59:00Z' },
      ],
    };
    const period = page.getByRole('combobox', {
      name: dictionary.healthcheck.attempts.period,
      exact: true,
    });
    await period.focus();
    await period.press('ArrowDown');
    await page
      .getByText(dictionary.healthcheck.attempts.periods.sevenDays, { exact: true })
      .last()
      .click();
    const marks = page.locator('[data-status]');
    await expect(marks).toHaveCount(2);
    await marks.first().hover();
    await expect(page.getByRole('tooltip')).toBeVisible();
    await assertEnlargedTextReflow(page);
  });
}
