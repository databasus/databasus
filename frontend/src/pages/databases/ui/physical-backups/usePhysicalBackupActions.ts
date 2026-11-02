import { App } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useSessionGuard } from '../../../../shared/auth';
import { translateApiError } from '../../../../shared/i18n';
import { type TriggerPhysicalBackupType, physicalBackupsApi } from '../../api/physicalBackupsApi';

interface Params {
  databaseId: string;
  refreshBackups: () => Promise<void>;
  reloadBackups: () => Promise<void>;
}

export const usePhysicalBackupActions = ({ databaseId, refreshBackups, reloadBackups }: Params) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const isActive = useSessionGuard();

  const [isStartingBackup, setIsStartingBackup] = useState(false);
  const [cancellingBackupId, setCancellingBackupId] = useState<string | undefined>();
  const [deletingBackupId, setDeletingBackupId] = useState<string | undefined>();

  const showFailure = (error: unknown) => {
    if (isActive()) message.error(translateApiError(error, t));
  };

  // The server only records the request and answers with no body, so the new backup reaches the
  // list through its poll once the scheduler creates it.
  const startBackup = async (kind: TriggerPhysicalBackupType) => {
    setIsStartingBackup(true);

    try {
      await physicalBackupsApi.triggerPhysicalBackup(databaseId, kind);
    } catch (error) {
      showFailure(error);
    }

    setIsStartingBackup(false);
  };

  const cancelBackup = async (backupId: string) => {
    if (cancellingBackupId) return;

    setCancellingBackupId(backupId);

    try {
      await physicalBackupsApi.cancelPhysicalBackup(backupId);
      if (isActive()) await refreshBackups();
    } catch (error) {
      showFailure(error);
    }

    setCancellingBackupId(undefined);
  };

  const deleteBackup = async (backupId: string) => {
    setDeletingBackupId(backupId);

    try {
      await physicalBackupsApi.deletePhysicalBackup(backupId);
      if (isActive()) await reloadBackups();
    } catch (error) {
      showFailure(error);
    }

    setDeletingBackupId(undefined);
  };

  return {
    isStartingBackup,
    cancellingBackupId,
    deletingBackupId,
    startBackup,
    cancelBackup,
    deleteBackup,
  };
};
