import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type TriggerPhysicalBackupType,
  physicalBackupsApi,
} from '@/pages/databases/physical-backups/api/physicalBackupsApi';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';

interface Params {
  databaseId: string;
  refreshBackups: () => Promise<void>;
  reloadBackups: () => Promise<void>;
}

export const usePhysicalBackupActions = ({ databaseId, refreshBackups, reloadBackups }: Params) => {
  const triggerPhysicalBackupMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof physicalBackupsApi.triggerPhysicalBackup>) =>
      physicalBackupsApi.triggerPhysicalBackup(...parameters),
  });
  const cancelPhysicalBackupMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof physicalBackupsApi.cancelPhysicalBackup>) =>
      physicalBackupsApi.cancelPhysicalBackup(...parameters),
  });
  const deletePhysicalBackupMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof physicalBackupsApi.deletePhysicalBackup>) =>
      physicalBackupsApi.deletePhysicalBackup(...parameters),
  });

  const { t } = useTranslation();
  const notify = useFeedback();
  const isActive = useSessionGuard();

  const isStartingBackup = triggerPhysicalBackupMutation.isPending;
  const [cancellingBackupId, setCancellingBackupId] = useState<string | undefined>();
  const [deletingBackupId, setDeletingBackupId] = useState<string | undefined>();

  const showFailure = (error: unknown) => {
    if (isActive()) notify({ kind: 'error', title: translateApiError(error, t) });
  };

  // The server only records the request and answers with no body, so the new backup reaches the
  // list through its poll once the scheduler creates it.
  const startBackup = async (kind: TriggerPhysicalBackupType) => {
    try {
      await triggerPhysicalBackupMutation.mutateAsync([databaseId, kind]);
    } catch (error) {
      showFailure(error);
    }

    triggerPhysicalBackupMutation.reset();
  };

  const cancelBackup = async (backupId: string) => {
    if (cancellingBackupId) return;

    setCancellingBackupId(backupId);

    try {
      await cancelPhysicalBackupMutation.mutateAsync([backupId]);
      if (isActive()) await refreshBackups();
    } catch (error) {
      showFailure(error);
    }

    setCancellingBackupId(undefined);
  };

  const deleteBackup = async (backupId: string) => {
    setDeletingBackupId(backupId);

    try {
      await deletePhysicalBackupMutation.mutateAsync([backupId]);
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
