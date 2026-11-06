import { Alert, Button, Modal, Select, Spin } from 'antd';
import { useEffect, useEffectEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { backupConfigApi } from '@/entities/backup-configs';
import { type Storage, storageApi } from '@/entities/storages';
import { type Workspace, workspaceApi } from '@/entities/workspaces';
import { useSessionGuard } from '@/shared/auth';
import { useRequestRevision } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';
import { LoadFailure } from '@/shared/ui';

interface Props {
  storage: Storage;
  onClose: () => void;
  onTransferred: () => void;
}

// One config that backs up into the storage is enough to know the storage is in use.
const STORAGE_USE_CHECK_LIMIT = 1;

export const StorageTransferDialogComponent = ({ storage, onClose, onTransferred }: Props) => {
  const { t } = useTranslation();
  const [loadError, setLoadError] = useState<unknown>();
  const [transferError, setTransferError] = useState<unknown>();
  const [isLoading, setIsLoading] = useState(true);
  const [isStorageInUse, setIsStorageInUse] = useState(false);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string | undefined>();
  const [isTransferring, setIsTransferring] = useState(false);

  const loadData = async () => {
    const revision = contextRevision.GetCurrent();
    const isActive = () => isCurrent() && contextRevision.IsCurrent(revision);
    setLoadError(undefined);
    setTransferError(undefined);
    setIsLoading(true);

    try {
      const storageBackupConfigs = await backupConfigApi.listStorageBackupConfigs(
        storage.id,
        STORAGE_USE_CHECK_LIMIT,
      );
      if (!isActive()) return;

      const isUsed = storageBackupConfigs.length > 0;
      setIsStorageInUse(isUsed);

      if (!isUsed) {
        const listedWorkspaces = await workspaceApi.listAllWorkspaces();
        if (!isActive()) return;

        const filteredWorkspaces = listedWorkspaces.filter((w) => w.id !== storage.workspaceId);
        setWorkspaces(filteredWorkspaces);
      }
    } catch (e) {
      if (!isActive()) return;

      setLoadError(e);
    }

    if (isActive()) setIsLoading(false);
  };

  const transferStorage = async () => {
    const revision = contextRevision.GetCurrent();
    const isActive = () => isCurrent() && contextRevision.IsCurrent(revision);
    setTransferError(undefined);

    if (!selectedWorkspaceId) return;

    setIsTransferring(true);

    try {
      await storageApi.transferStorage(storage.id, selectedWorkspaceId);
      if (!isActive()) return;

      onTransferred();
    } catch (e) {
      if (!isActive()) return;

      setTransferError(e);
    }

    if (isActive()) setIsTransferring(false);
  };

  const isCurrent = useSessionGuard();
  const contextRevision = useRequestRevision();
  const startDataLoad = useEffectEvent(() => {
    void loadData();
  });

  useEffect(() => {
    contextRevision.Invalidate();
    startDataLoad();

    return () => {
      contextRevision.Invalidate();
    };
  }, [storage.id, contextRevision]);

  return (
    <Modal
      title={t('storages.transfer.title')}
      footer={null}
      open={true}
      onCancel={onClose}
      maskClosable={false}
    >
      {loadError !== undefined && <LoadFailure error={loadError} onRetry={() => void loadData()} />}
      {transferError !== undefined && (
        <Alert type="error" showIcon message={translateApiError(transferError, t)} />
      )}
      {isLoading ? (
        <div className="flex justify-center py-5">
          <Spin />
        </div>
      ) : isStorageInUse ? (
        <div className="py-3">
          <div className="text-muted">{t('storages.transfer.blockedByDatabases')}</div>

          <div className="mt-5">
            <Button type="primary" onClick={onClose}>
              {t('common.actions.ok')}
            </Button>
          </div>
        </div>
      ) : (
        <div className="py-3">
          <div className="mb-3 text-muted">{t('storages.transfer.description')}</div>

          <div className="mb-5 flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center">
            <div className="min-w-30 pr-2">{t('storages.transfer.targetWorkspace')}</div>

            <Select
              aria-label={t('storages.transfer.targetWorkspace')}
              value={selectedWorkspaceId}
              onChange={setSelectedWorkspaceId}
              className="min-w-0 grow"
              placeholder={t('storages.transfer.selectWorkspace')}
              options={workspaces.map((w) => ({ label: w.name, value: w.id }))}
            />
          </div>

          <div className="flex gap-2">
            <Button type="default" onClick={onClose}>
              {t('common.actions.cancel')}
            </Button>

            <Button
              type="primary"
              onClick={transferStorage}
              loading={isTransferring}
              disabled={!selectedWorkspaceId || isTransferring}
            >
              {t('storages.transfer.submit')}
            </Button>
          </div>
        </div>
      )}
    </Modal>
  );
};
