import type { Page } from '@playwright/test';

import type { Locale } from '../src/shared/i18n/dictionaries';

export const openAccount = async ({
  page,
  locale = 'en',
  theme = 'light',
  isSignedIn = true,
}: {
  page: Page;
  locale?: Locale;
  theme?: string;
  isSignedIn?: boolean;
}) => {
  await page.addInitScript(
    ({ locale, theme, isSignedIn }) => {
      localStorage.setItem('databasus-locale', locale);
      localStorage.setItem('databasus-theme', theme);
      if (isSignedIn) {
        localStorage.setItem('databasus_user_token', 'synthetic-session');
        localStorage.setItem('databasus_user_id', 'owner-id');
      }
    },
    { locale, theme, isSignedIn },
  );
  await page.goto('/');
  await page.evaluate(() => document.fonts.ready);
};
