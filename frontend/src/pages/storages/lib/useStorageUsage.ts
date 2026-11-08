import { useEffect, useEffectEvent, useState } from 'react';

import { storageUsageApi } from '@/pages/storages/api/storageUsageApi';
import { ApiError, useFixedQueryPolling, useSessionQuery } from '@/shared/api';

interface StorageUsageSource {
  storageId: string;
  workspaceId: string;
}

export const useStorageUsage = ({ storageId, workspaceId }: StorageUsageSource) => {
  const query = useSessionQuery({
    shouldFollowSession: true,
    queryKey: ['storage-usage', workspaceId, storageId],
    queryFn: (signal) => storageUsageApi.getUsage(storageId, { signal }),
  });
  const [isCleared, setIsCleared] = useState(false);
  const [retryRevision, setRetryRevision] = useState(0);
  useFixedQueryPolling({
    query,
    intervalMs: 1000,
    shouldRefresh: query.error === null,
    scheduleRevision: retryRevision,
  });
  const { refetch } = query;
  const refreshIfVisible = useEffectEvent(() => {
    if (document.visibilityState !== 'hidden' && query.error === null && !query.isFetching) {
      void refetch({ cancelRefetch: false });
    }
  });

  useEffect(() => {
    const refresh = () => refreshIfVisible();
    document.addEventListener('visibilitychange', refresh);

    return () => document.removeEventListener('visibilitychange', refresh);
  }, []);

  const isUnauthorized =
    query.error instanceof ApiError && [401, 403, 404].includes(query.error.status ?? 0);

  useEffect(() => {
    if (isUnauthorized) setIsCleared(true);
  }, [isUnauthorized]);
  useEffect(() => {
    if (query.data !== undefined) setIsCleared(false);
  }, [query.data, query.dataRevision]);

  return {
    totalUsageMb: isUnauthorized || isCleared ? undefined : query.data?.totalUsageMb,
    usageError: query.error ?? undefined,
    retry: () => {
      setRetryRevision((revision) => revision + 1);
      void refetch();
    },
  };
};
