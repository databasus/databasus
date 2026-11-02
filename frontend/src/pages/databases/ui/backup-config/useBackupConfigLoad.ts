import { useEffect, useState } from 'react';

import { type BackupConfig, backupConfigApi } from '../../../../entities/backup-configs';

export const useBackupConfigLoad = (databaseId: string) => {
  const [backupConfig, setBackupConfig] = useState<BackupConfig | undefined>();
  const [isBackupConfigLoading, setIsBackupConfigLoading] = useState(false);
  const [backupConfigError, setBackupConfigError] = useState<unknown>();
  const [loadAttempt, setLoadAttempt] = useState(0);

  const retryBackupConfigLoad = () => setLoadAttempt((attempt) => attempt + 1);

  useEffect(() => {
    let isActive = true;

    setIsBackupConfigLoading(true);
    setBackupConfig(undefined);
    setBackupConfigError(undefined);

    backupConfigApi
      .getBackupConfig(databaseId)
      .then((loadedBackupConfig) => {
        if (isActive) setBackupConfig(loadedBackupConfig);
      })
      .catch((error: unknown) => {
        if (isActive) setBackupConfigError(error);
      })
      .finally(() => {
        if (isActive) setIsBackupConfigLoading(false);
      });

    return () => {
      isActive = false;
    };
  }, [databaseId, loadAttempt]);

  return { backupConfig, isBackupConfigLoading, backupConfigError, retryBackupConfigLoad };
};
