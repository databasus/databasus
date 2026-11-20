import { useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';

import { instanceBackupApi } from '@/entities/instance-backups/api/instanceBackupApi';
import { useFixedQueryPolling, useSessionQuery } from '@/shared/api';

const STATUS_QUERY_NAME = 'instance-backup-status';
const STATUS_POLL_INTERVAL_MS = 30_000;

// The settings section and the floating notices read one cached status, so a change acknowledged by
// one of them reaches the other without waiting for the next poll.
export const useInstanceBackupStatus = ({ isEnabled }: { isEnabled: boolean }) => {
  const query = useSessionQuery({
    queryKey: [STATUS_QUERY_NAME],
    queryFn: (signal) => instanceBackupApi.getStatus({ signal }),
    enabled: isEnabled,
    shouldFollowSession: true,
  });

  useFixedQueryPolling({ query, intervalMs: STATUS_POLL_INTERVAL_MS, isEnabled });

  return query;
};

export const useRefreshInstanceBackupStatus = () => {
  const client = useQueryClient();

  return useCallback(
    () =>
      client.invalidateQueries({
        predicate: (query) => query.queryKey.includes(STATUS_QUERY_NAME),
      }),
    [client],
  );
};
