import { cleanup, render } from '@testing-library/react';
import { App } from 'antd';
import type { ReactElement } from 'react';
import { afterEach } from 'vitest';

import { LocaleProvider } from '../../i18n';

// Vitest runs without globals, so Testing Library cannot unmount screens by itself. Each test also
// starts signed out.
afterEach(() => {
  cleanup();
  localStorage.clear();
});

// Renders a screen the way the application wraps it: the selected language and the antd App
// that serves message toasts.
export const renderScreen = (screen: ReactElement) =>
  render(
    <LocaleProvider>
      <App>{screen}</App>
    </LocaleProvider>,
  );
