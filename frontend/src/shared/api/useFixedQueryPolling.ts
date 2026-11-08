import { hashKey, type QueryKey, type RefetchOptions } from '@tanstack/react-query';
import { useEffect, useEffectEvent } from 'react';

interface Options {
  query: {
    queryKey: QueryKey;
    isSessionActive: boolean;
    isFetching: boolean;
    refetch: (options: RefetchOptions) => Promise<unknown>;
  };
  intervalMs: number;
  isEnabled?: boolean;
  shouldRefresh?: boolean;
  shouldRunInBackground?: boolean;
  scheduleRevision?: number;
}

export const useFixedQueryPolling = ({
  query,
  intervalMs,
  isEnabled = true,
  shouldRefresh = true,
  shouldRunInBackground = false,
  scheduleRevision = 0,
}: Options) => {
  const queryIdentity = hashKey(query.queryKey);
  const refresh = useEffectEvent(() => {
    if (
      !isEnabled ||
      !query.isSessionActive ||
      query.isFetching ||
      !shouldRefresh ||
      (!shouldRunInBackground && document.visibilityState === 'hidden')
    )
      return;

    void query.refetch({ cancelRefetch: false });
  });

  useEffect(() => {
    if (!isEnabled || !query.isSessionActive) return;

    const interval = setInterval(() => refresh(), intervalMs);

    return () => clearInterval(interval);
  }, [queryIdentity, query.isSessionActive, intervalMs, isEnabled, scheduleRevision]);
};
