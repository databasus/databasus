import type { ReactNode } from 'react';
import { MemoryRouter } from 'react-router';

import { LocaleProvider } from '@/shared/i18n';
import { ScreenConfiguration } from '@/shared/lib/test/ScreenConfiguration';
import { ThemeProvider } from '@/shared/theme';

export const ScreenTestProviders = ({
  children,
  route = '/',
}: {
  children: ReactNode;
  route?: string;
}) => (
  <ThemeProvider>
    <LocaleProvider>
      <ScreenConfiguration>
        <MemoryRouter initialEntries={[route]}>{children}</MemoryRouter>
      </ScreenConfiguration>
    </LocaleProvider>
  </ThemeProvider>
);
