import { render } from '@testing-library/react';

import '../../shared/lib/test/screenTestLifecycle';
import App from './App';

export const renderApp = () => render(<App />);
