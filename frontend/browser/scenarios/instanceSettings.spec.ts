import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { accountFixtures } from '../accountFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openAccountTab } from '../openAccountTab';
import { visualContexts } from '../visualContexts';

test.use({ httpFixtures: [accountFixtures(), { scope: 'test' }] });

for (const context of [
  { locale: 'en' as const, theme: 'light' },
  { locale: 'ru' as const, theme: 'dark' },
]) {
  test(`keeps instance settings draft and errors reachable at 320px with 200% text in ${context.locale} ${context.theme}`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 320, height: 900 });
    await openAccount({ page, ...context });
    await openAccountTab({ page, locale: context.locale, tab: 'databasusSettings' });

    const dictionary = DICTIONARIES[context.locale],
      region = page.getByRole('main');
    await region
      .getByRole('switch', { name: dictionary.settings.externalRegistrations.title, exact: true })
      .click();
    await region
      .getByRole('button', { name: dictionary.settings.saveChanges, exact: true })
      .click();

    await expect(
      region.getByText(dictionary.errors.humanVerificationFailed, { exact: true }),
    ).toBeVisible();
    await enlargeText(region);

    for (const name of [dictionary.settings.saveChanges, dictionary.settings.reset]) {
      const action = region.getByRole('button', { name, exact: true });
      await action.scrollIntoViewIfNeeded();
      await expect(action).toBeInViewport();
    }
    await assertEnlargedTextReflow(page);

    await assertAccountReflow(region);
  });
}

test.describe('Mail delivery diagnostics', () => {
  test.use({
    httpFixtures: [
      accountFixtures().concat([
        apiResponse({
          method: 'post',
          path: '/test-emails',
          status: 201,
          body: {
            recipientEmail: 'owner@example.com',
            isDelivered: false,
            failureDetail: '550 mailbox unavailable <literal>',
          },
        }),
      ]),
      { scope: 'test' },
    ],
  });

  for (const context of visualContexts) {
    test(`preserves literal mail delivery diagnostics in ${context.locale} ${context.theme} ${context.width}`, async ({
      page,
    }) => {
      const dictionary = DICTIONARIES[context.locale];

      await page.setViewportSize({ width: context.width, height: 1000 });
      await openAccount({ page, ...context });
      await openAccountTab({ page, locale: context.locale, tab: 'databasusSettings' });
      await page
        .getByRole('button', { name: dictionary.settings.mailServer.sendTestEmail, exact: true })
        .click();

      await expect(
        page.getByText('550 mailbox unavailable <literal>', { exact: true }),
      ).toBeVisible();
      await assertEnlargedTextReflow(page);
    });
  }
});
