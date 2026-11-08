import { useTranslation } from 'react-i18next';

import { backupConfigApi } from '@/entities/backup-configs';
import { type Storage, storageApi } from '@/entities/storages';
import { workspaceApi } from '@/entities/workspaces';
import { ResourceTransferDialog } from '@/shared/ui';

interface Props {
  storage: Storage;
  onClose: () => void;
  onTransferred: () => void;
}

// One config that backs up into the storage is enough to know the storage is in use.
const STORAGE_USE_CHECK_LIMIT = 1;

export const StorageTransferDialogComponent = ({ storage, onClose, onTransferred }: Props) => {
  const { t } = useTranslation();

  return (
    <ResourceTransferDialog
      contextKey={`${storage.workspaceId}/${storage.id}`}
      loadAvailability={async (signal) => {
        const storageBackupConfigs = await backupConfigApi.listStorageBackupConfigs(
          storage.id,
          STORAGE_USE_CHECK_LIMIT,
          { signal },
        );
        signal.throwIfAborted();

        if (storageBackupConfigs.length > 0) return { isBlocked: true };

        const workspaces = await workspaceApi.listAllWorkspaces({ signal });

        return {
          isBlocked: false,
          targets: workspaces
            .filter((workspace) => workspace.id !== storage.workspaceId)
            .map((workspace) => ({ label: workspace.name, value: workspace.id })),
        };
      }}
      transfer={(targetWorkspaceId) => storageApi.transferStorage(storage.id, targetWorkspaceId)}
      onClose={onClose}
      onTransferred={onTransferred}
      labels={{
        title: t('storages.transfer.title'),
        blocked: t('storages.transfer.blockedByDatabases'),
        description: t('storages.transfer.description'),
        target: t('storages.transfer.targetWorkspace'),
        placeholder: t('storages.transfer.selectWorkspace'),
        submit: t('storages.transfer.submit'),
      }}
    />
  );
};
