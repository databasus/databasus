import { useEffect } from 'react';

import { systemApi } from '@/entities/system';
import { APP_VERSION } from '@/shared/config';

const VERSION_CHECK_INTERVAL_MS = 5 * 60 * 1000;
const RELOAD_COOLDOWN_MS = 10 * 1000;
const LAST_RELOAD_KEY = 'lastVersionReload';

export function useVersionCheck() {
  useEffect(() => {
    if (APP_VERSION === 'dev') return;

    let isActive = true;
    let isChecking = false;
    const checkVersion = async () => {
      if (isChecking) return;
      isChecking = true;
      try {
        const { version } = await systemApi.getVersion();
        if (!isActive || !version || version === APP_VERSION) return;

        const lastReload = Number(localStorage.getItem(LAST_RELOAD_KEY) || '0');
        if (Date.now() - lastReload < RELOAD_COOLDOWN_MS) return;

        localStorage.setItem(LAST_RELOAD_KEY, String(Date.now()));
        window.location.reload();
      } catch {
      } finally {
        isChecking = false;
      }
    };

    void checkVersion();
    const interval = setInterval(() => void checkVersion(), VERSION_CHECK_INTERVAL_MS);

    return () => {
      isActive = false;
      clearInterval(interval);
    };
  }, []);
}
