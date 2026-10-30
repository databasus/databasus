import { useEffect } from 'react';

import { APP_VERSION } from '../../constants';
import { systemApi } from '../../entities/system';

const VERSION_CHECK_INTERVAL_MS = 5 * 60 * 1000;
const RELOAD_COOLDOWN_MS = 10 * 1000;
// eslint-disable-next-line i18next/no-literal-string -- localStorage key
const LAST_RELOAD_KEY = 'lastVersionReload';

export function useVersionCheck() {
  useEffect(() => {
    if (APP_VERSION === 'dev') {
      return;
    }

    const checkVersion = async () => {
      try {
        const { version } = await systemApi.getVersion();

        if (version && version !== APP_VERSION) {
          const lastReload = Number(localStorage.getItem(LAST_RELOAD_KEY) || '0');

          if (Date.now() - lastReload < RELOAD_COOLDOWN_MS) {
            return;
          }

          localStorage.setItem(LAST_RELOAD_KEY, String(Date.now()));
          window.location.reload();
        }
      } catch {
        // Silently ignore errors — network issues shouldn't break the app
      }
    };

    checkVersion();
    const interval = setInterval(checkVersion, VERSION_CHECK_INTERVAL_MS);

    return () => clearInterval(interval);
  }, []);
}
