import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { logicalBackupApi } from '@/pages/databases/logical-backups/api/logicalBackupApi';
import { isLogicalBackupListedByFilter } from '@/pages/databases/logical-backups/model/isLogicalBackupListedByFilter';
import type { LogicalBackup } from '@/pages/databases/logical-backups/model/logicalBackupContract';
import type { LogicalBackupListFilter } from '@/pages/databases/logical-backups/model/LogicalBackupListFilter';
import { verificationRunsApi } from '@/pages/databases/verification/api/verificationRunsApi';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';

interface Params {
  databaseId: string;
  filter: LogicalBackupListFilter;
  replaceOrPrependBackup: (backup: LogicalBackup) => void;
  reloadBackups: () => Promise<void>;
  onNavigateToVerifications?: () => void;
}

// The file is served with Content-Disposition, so following the link downloads it and leaves the
// page in place.
const openFileLink = (url: string, fileName: string) => {
  const link = document.createElement('a');
  link.href = url;
  link.download = fileName;
  link.style.display = 'none';

  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
};

export const useLogicalBackupActions = ({
  databaseId,
  filter,
  replaceOrPrependBackup,
  reloadBackups,
  onNavigateToVerifications,
}: Params) => {
  const startLogicalBackupMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof logicalBackupApi.startLogicalBackup>) =>
      logicalBackupApi.startLogicalBackup(...parameters),
  });
  const cancelLogicalBackupMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof logicalBackupApi.cancelLogicalBackup>) =>
      logicalBackupApi.cancelLogicalBackup(...parameters),
  });
  const deleteLogicalBackupMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof logicalBackupApi.deleteLogicalBackup>) =>
      logicalBackupApi.deleteLogicalBackup(...parameters),
  });
  const createDownloadTokenMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof logicalBackupApi.createDownloadToken>) =>
      logicalBackupApi.createDownloadToken(...parameters),
  });
  const startVerificationMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof verificationRunsApi.startVerification>) =>
      verificationRunsApi.startVerification(...parameters),
  });

  const { t } = useTranslation();
  const notify = useFeedback();
  const isActive = useSessionGuard();

  const isStartingBackup = startLogicalBackupMutation.isPending;
  const [cancellingBackupId, setCancellingBackupId] = useState<string | undefined>();
  const [deletingBackupId, setDeletingBackupId] = useState<string | undefined>();
  const [downloadingBackupId, setDownloadingBackupId] = useState<string | undefined>();
  const [verifyingBackupId, setVerifyingBackupId] = useState<string | undefined>();

  const showFailure = (error: unknown) => {
    if (isActive()) notify({ kind: 'error', title: translateApiError(error, t) });
  };

  const startBackup = async () => {
    try {
      const startedBackup = await startLogicalBackupMutation.mutateAsync([databaseId]);
      if (isActive() && isLogicalBackupListedByFilter(startedBackup, filter)) {
        replaceOrPrependBackup(startedBackup);
      }
    } catch (error) {
      showFailure(error);
    }

    startLogicalBackupMutation.reset();
  };

  const cancelBackup = async (backupId: string) => {
    if (cancellingBackupId) return;

    setCancellingBackupId(backupId);

    try {
      const cancelledBackup = await cancelLogicalBackupMutation.mutateAsync([backupId]);
      if (isActive()) replaceOrPrependBackup(cancelledBackup);
    } catch (error) {
      showFailure(error);
    }

    setCancellingBackupId(undefined);
  };

  const deleteBackup = async (backupId: string) => {
    setDeletingBackupId(backupId);

    try {
      await deleteLogicalBackupMutation.mutateAsync([backupId]);
      if (isActive()) await reloadBackups();
    } catch (error) {
      showFailure(error);
    }

    setDeletingBackupId(undefined);
  };

  const downloadBackup = async (backupId: string) => {
    if (downloadingBackupId) return;

    setDownloadingBackupId(backupId);

    try {
      const downloadToken = await createDownloadTokenMutation.mutateAsync([backupId]);
      if (isActive()) {
        openFileLink(
          logicalBackupApi.getLogicalBackupFileUrl(backupId, downloadToken.token),
          downloadToken.fileName,
        );
      }
    } catch (error) {
      showFailure(error);
    }

    setDownloadingBackupId(undefined);
  };

  const startVerification = async (backupId: string) => {
    setVerifyingBackupId(backupId);

    try {
      await startVerificationMutation.mutateAsync([backupId]);
      if (isActive()) {
        notify({ kind: 'success', title: t('backups.logical.list.verifyRestore.queued') });
        onNavigateToVerifications?.();
      }
    } catch (error) {
      showFailure(error);
    }

    setVerifyingBackupId(undefined);
  };

  return {
    isStartingBackup,
    cancellingBackupId,
    deletingBackupId,
    downloadingBackupId,
    verifyingBackupId,
    startBackup,
    cancelBackup,
    deleteBackup,
    downloadBackup,
    startVerification,
  };
};
