import { expect, type Page } from '@playwright/test';

import { DICTIONARIES, type Locale } from '../src/shared/i18n/dictionaries';

export const openAccountTab = async ({
  page,
  locale,
  tab,
}: {
  page: Page;
  locale: Locale;
  tab: 'profile' | 'users' | 'databasusSettings';
}) => {
  const dictionary = DICTIONARIES[locale];
  const menu = page.getByRole('button', { name: dictionary.app.navigation.menu, exact: true });
  if (await menu.isVisible()) await menu.click();
  await page.getByRole('button', { name: dictionary.app.navigation[tab], exact: true }).click();
  await expect(
    page.getByRole('dialog', { name: dictionary.app.navigation.menu, exact: true }),
  ).not.toBeVisible();
};
