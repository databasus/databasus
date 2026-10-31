import './app/fonts/jost.css';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { createRoot } from 'react-dom/client';

import './index.css';
import App from './app/ui/App.tsx';

dayjs.extend(utc);

createRoot(document.getElementById('root')!).render(<App />);
