import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ReactElement, ReactNode } from 'react';
import { MemoryRouter } from 'react-router';

import { LocaleProvider, type Locale } from '../../i18n';
import { ThemeProvider, type ThemeMode } from '../../theme';
import { ScreenConfiguration } from './ScreenConfiguration';
import './screenTestLifecycle';

interface ScreenOptions {
  locale?: Locale;
  theme?: ThemeMode;
  route?: string;
  isStrictMode?: boolean;
}

export const renderScreen = (screen: ReactElement, options: ScreenOptions = {}) => {
  if (options.locale) localStorage.setItem('databasus-locale', options.locale);
  if (options.theme) localStorage.setItem('databasus-theme', options.theme);
  const Wrapper = ({ children }: { children: ReactNode }) => (
    <ThemeProvider>
      <LocaleProvider>
        <ScreenConfiguration>
          <MemoryRouter initialEntries={[options.route ?? '/']}>{children}</MemoryRouter>
        </ScreenConfiguration>
      </LocaleProvider>
    </ThemeProvider>
  );
  return {
    user: userEvent.setup(),
    ...render(screen, { wrapper: Wrapper, reactStrictMode: options.isStrictMode }),
  };
};
