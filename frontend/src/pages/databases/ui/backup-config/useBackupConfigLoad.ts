import { useEffect, useState } from 'react';

import { type BackupConfig, backupConfigApi } from '../../../../entities/backup-configs';
import { useSession, useSessionGuard } from '../../../../shared/auth';
import { useRequestRevision } from '../../../../shared/hooks';

interface ConfigScope {
  databaseId: string;
  workspaceId: string;
}

export const useBackupConfigLoad = ({ databaseId, workspaceId }: ConfigScope) => {
  const [backupConfig, setBackupConfig] = useState<BackupConfig | undefined>();
  const [isBackupConfigLoading, setIsBackupConfigLoading] = useState(false);
  const [backupConfigError, setBackupConfigError] = useState<unknown>();
  const [loadAttempt, setLoadAttempt] = useState(0);

  const reloadBackupConfig = () => setLoadAttempt((attempt) => attempt + 1);

  const acknowledgeBackupConfig = (config: BackupConfig) => {
    if (!isCurrentSession()) return;

    loadRevision.Invalidate();
    setBackupConfig(config);
    setBackupConfigError(undefined);
    setIsBackupConfigLoading(false);
  };

  const loadRevision = useRequestRevision();
  const session = useSession();
  const isCurrentSession = useSessionGuard();

  useEffect(() => {
    const revision = loadRevision.Invalidate();
    let isActive = true;
    const isCurrentLoad = () => isActive && isCurrentSession() && loadRevision.IsCurrent(revision);

    setIsBackupConfigLoading(true);
    setBackupConfig(undefined);
    setBackupConfigError(undefined);

    if (!isCurrentSession()) {
      setIsBackupConfigLoading(false);
      return;
    }

    backupConfigApi
      .getBackupConfig(databaseId)
      .then((loadedBackupConfig) => {
        if (isCurrentLoad()) setBackupConfig(loadedBackupConfig);
      })
      .catch((error: unknown) => {
        if (isCurrentLoad()) setBackupConfigError(error);
      })
      .finally(() => {
        if (isCurrentLoad()) setIsBackupConfigLoading(false);
      });

    return () => {
      isActive = false;
    };
  }, [databaseId, workspaceId, loadAttempt, session, isCurrentSession, loadRevision]);

  return {
    backupConfig,
    isBackupConfigLoading,
    backupConfigError,
    reloadBackupConfig,
    acknowledgeBackupConfig,
  };
};
