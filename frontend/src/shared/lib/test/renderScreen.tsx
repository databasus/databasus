import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ReactElement } from 'react';

import { type Locale } from '@/shared/i18n';
import { ScreenTestProviders } from '@/shared/lib/test/ScreenTestProviders';
import { type ThemeMode } from '@/shared/theme';
import '@/shared/lib/test/screenTestLifecycle';

interface ScreenOptions {
  locale?: Locale;
  theme?: ThemeMode;
  route?: string;
  isStrictMode?: boolean;
}

export const renderScreen = (screen: ReactElement, options: ScreenOptions = {}) => {
  if (options.locale) localStorage.setItem('databasus-locale', options.locale);
  if (options.theme) localStorage.setItem('databasus-theme', options.theme);

  return {
    user: userEvent.setup(),
    ...render(screen, {
      wrapper: ({ children }) => (
        <ScreenTestProviders route={options.route}>{children}</ScreenTestProviders>
      ),
      reactStrictMode: options.isStrictMode,
    }),
  };
};
