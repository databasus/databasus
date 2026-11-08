import { useQueryClient } from '@tanstack/react-query';

import { type BackupConfig, backupConfigApi } from '@/entities/backup-configs';
import { useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';

interface ConfigScope {
  databaseId: string;
  workspaceId: string;
}

export const useBackupConfigLoad = ({ databaseId, workspaceId }: ConfigScope) => {
  const client = useQueryClient();
  const isCurrent = useSessionGuard();
  const query = useSessionQuery({
    queryKey: ['backup-config', workspaceId, databaseId],
    queryFn: (signal) => backupConfigApi.getBackupConfig(databaseId, { signal }),
  });

  const acknowledgeBackupConfig = (config: BackupConfig) => {
    if (!isCurrent()) return;

    void client.cancelQueries({ queryKey: query.queryKey, exact: true });
    client.setQueryData(query.queryKey, config);
  };

  return {
    backupConfig: query.data,
    backupConfigRevision: query.dataRevision,
    isBackupConfigLoading: query.isPending,
    backupConfigError: query.error ?? undefined,
    reloadBackupConfig: () => void query.refetch(),
    acknowledgeBackupConfig,
  };
};
