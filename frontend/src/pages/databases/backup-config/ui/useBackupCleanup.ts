import { useEffect } from 'react';

import { type BackupCleanup, backupConfigApi } from '@/entities/backup-configs';
import { useSessionQuery } from '@/shared/api';

interface CleanupSource {
  databaseId: string;
  workspaceId: string;
  initial: BackupCleanup | undefined;
  revision: number;
}

export const useBackupCleanup = ({ databaseId, workspaceId, initial, revision }: CleanupSource) => {
  const query = useSessionQuery({
    queryKey: ['backup-cleanup', workspaceId, databaseId, revision],
    enabled: false,
    queryFn: async (signal) =>
      (await backupConfigApi.getBackupConfig(databaseId, { signal })).cleanup,
  });
  const cleanup = query.data ?? initial;
  const { refetch, isFetching, error, isSessionActive } = query;

  useEffect(() => {
    if (!isSessionActive || !cleanup?.pendingFileCount || error !== null || isFetching) return;

    const timer = setTimeout(() => void refetch(), 2000);

    return () => clearTimeout(timer);
  }, [cleanup, error, isFetching, refetch, isSessionActive]);

  return { cleanup, cleanupError: error ?? undefined, retryCleanup: () => void refetch() };
};
