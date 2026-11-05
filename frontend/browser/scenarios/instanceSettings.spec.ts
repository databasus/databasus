import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { accountFixtures } from '../accountFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { openAccount } from '../openAccount';
import { openAccountTab } from '../openAccountTab';

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

    await assertAccountReflow(region);
  });
}
