import { useEffect } from 'react';

import { systemApi } from '@/entities/system';
import { useFixedQueryPolling, useSessionQuery } from '@/shared/api';
import { APP_VERSION } from '@/shared/config';

const VERSION_CHECK_INTERVAL_MS = 5 * 60 * 1000;
const RELOAD_COOLDOWN_MS = 10 * 1000;
const LAST_RELOAD_KEY = 'lastVersionReload';

export function useVersionCheck() {
  const query = useSessionQuery({
    shouldFollowSession: true,
    queryKey: ['application-version'],
    enabled: APP_VERSION !== 'dev',
    queryFn: (signal) => systemApi.getVersion({ signal }),
  });
  useFixedQueryPolling({
    query,
    intervalMs: VERSION_CHECK_INTERVAL_MS,
    isEnabled: APP_VERSION !== 'dev',
    shouldRunInBackground: true,
  });
  useEffect(() => {
    const version = query.data?.version;
    if (!version || version === APP_VERSION) return;

    try {
      const lastReload = Number(localStorage.getItem(LAST_RELOAD_KEY) || '0');
      if (Date.now() - lastReload < RELOAD_COOLDOWN_MS) return;

      localStorage.setItem(LAST_RELOAD_KEY, String(Date.now()));
      window.location.reload();
    } catch {}
  }, [query.data, query.dataUpdatedAt]);
}
