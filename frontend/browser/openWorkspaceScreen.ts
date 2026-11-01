import type { Page } from '@playwright/test';

import { DICTIONARIES, type Locale } from '../src/shared/i18n/dictionaries';

interface Options {
  page: Page;
  locale: Locale;
  tab: 'databases' | 'storages' | 'notifiers' | 'workspaceSettings';
}

export const openWorkspaceScreen = async ({ page, locale, tab }: Options) => {
  const dictionary = DICTIONARIES[locale];
  const menu = page.getByRole('button', { name: dictionary.app.navigation.menu, exact: true });

  if (await menu.isVisible()) await menu.click();

  await page.getByRole('button', { name: dictionary.app.navigation[tab], exact: true }).click();
};
