import { useCallback, useState } from 'react';

import { type InstanceBackup, instanceBackupApi } from '@/entities/instance-backups';
import { usePolledPagedList } from '@/shared/hooks';

const REFRESH_INTERVAL_MS = 15_000;

export const useInstanceBackupList = ({ pageSize }: { pageSize: number }) => {
  const [loadError, setLoadError] = useState<unknown>();

  const fetchPage = useCallback(
    async (after: string | undefined) => {
      setLoadError(undefined);
      return instanceBackupApi.listBackups({ limit: pageSize, after });
    },
    [pageSize],
  );
  const getItemKey = useCallback((backup: InstanceBackup) => backup.id, []);
  const list = usePolledPagedList<InstanceBackup, string>({
    fetchPage,
    getItemKey,
    onError: setLoadError,
    resetKey: '',
    refreshIntervalMs: REFRESH_INTERVAL_MS,
  });

  return { ...list, loadError };
};
