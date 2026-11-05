import { render } from '@testing-library/react';
import { StrictMode } from 'react';

import '../../shared/lib/test/screenTestLifecycle';
import App from './App';

export const renderApp = ({ isStrictMode = false }: { isStrictMode?: boolean } = {}) =>
  render(
    isStrictMode ? (
      <StrictMode>
        <App />
      </StrictMode>
    ) : (
      <App />
    ),
  );
