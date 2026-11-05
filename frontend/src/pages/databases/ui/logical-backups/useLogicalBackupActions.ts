import { App } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useSessionGuard } from '../../../../shared/auth';
import { translateApiError } from '../../../../shared/i18n';
import { logicalBackupApi } from '../../api/logicalBackupApi';
import { verificationRunsApi } from '../../api/verificationRunsApi';
import { isLogicalBackupListedByFilter } from '../../model/isLogicalBackupListedByFilter';
import type { LogicalBackup } from '../../model/LogicalBackup';
import type { LogicalBackupListFilter } from '../../model/LogicalBackupListFilter';

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
  const { t } = useTranslation();
  const { message } = App.useApp();
  const isActive = useSessionGuard();

  const [isStartingBackup, setIsStartingBackup] = useState(false);
  const [cancellingBackupId, setCancellingBackupId] = useState<string | undefined>();
  const [deletingBackupId, setDeletingBackupId] = useState<string | undefined>();
  const [downloadingBackupId, setDownloadingBackupId] = useState<string | undefined>();
  const [verifyingBackupId, setVerifyingBackupId] = useState<string | undefined>();

  const showFailure = (error: unknown) => {
    if (isActive()) message.error(translateApiError(error, t));
  };

  const startBackup = async () => {
    setIsStartingBackup(true);

    try {
      const startedBackup = await logicalBackupApi.startLogicalBackup(databaseId);
      if (isActive() && isLogicalBackupListedByFilter(startedBackup, filter)) {
        replaceOrPrependBackup(startedBackup);
      }
    } catch (error) {
      showFailure(error);
    }

    setIsStartingBackup(false);
  };

  const cancelBackup = async (backupId: string) => {
    if (cancellingBackupId) return;

    setCancellingBackupId(backupId);

    try {
      const cancelledBackup = await logicalBackupApi.cancelLogicalBackup(backupId);
      if (isActive()) replaceOrPrependBackup(cancelledBackup);
    } catch (error) {
      showFailure(error);
    }

    setCancellingBackupId(undefined);
  };

  const deleteBackup = async (backupId: string) => {
    setDeletingBackupId(backupId);

    try {
      await logicalBackupApi.deleteLogicalBackup(backupId);
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
      const downloadToken = await logicalBackupApi.createDownloadToken(backupId);
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
      await verificationRunsApi.startVerification(backupId);
      if (isActive()) {
        message.success(t('backups.logical.list.verifyRestore.queued'));
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
