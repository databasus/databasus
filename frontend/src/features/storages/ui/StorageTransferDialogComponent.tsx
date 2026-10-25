import { Button, Modal, Select, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { logicalBackupConfigApi } from '../../../entity/backups/logical';
import { type Storage, storageApi } from '../../../entity/storages';
import { type Workspace, workspaceApi } from '../../../entity/workspaces';
import { translateApiError } from '../../../shared/i18n';

interface Props {
  storage: Storage;
  onClose: () => void;
  onTransferred: () => void;
}

export const StorageTransferDialogComponent = ({ storage, onClose, onTransferred }: Props) => {
  const { t } = useTranslation();
  const [isLoading, setIsLoading] = useState(true);
  const [isStorageInUse, setIsStorageInUse] = useState(false);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string | undefined>();
  const [isTransferring, setIsTransferring] = useState(false);

  const loadData = async () => {
    setIsLoading(true);

    try {
      const isUsing = await logicalBackupConfigApi.isStorageUsing(storage.id);
      setIsStorageInUse(isUsing);

      if (!isUsing) {
        const listedWorkspaces = await workspaceApi.listAllWorkspaces();
        const filteredWorkspaces = listedWorkspaces.filter((w) => w.id !== storage.workspaceId);
        setWorkspaces(filteredWorkspaces);
      }
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsLoading(false);
  };

  const transferStorage = async () => {
    if (!selectedWorkspaceId) return;

    setIsTransferring(true);

    try {
      await storageApi.transferStorage(storage.id, selectedWorkspaceId);
      onTransferred();
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsTransferring(false);
  };

  useEffect(() => {
    loadData();
  }, [storage.id]);

  return (
    <Modal
      title={t('storages.transfer.title')}
      footer={null}
      open={true}
      onCancel={onClose}
      maskClosable={false}
    >
      {isLoading ? (
        <div className="flex justify-center py-5">
          <Spin />
        </div>
      ) : isStorageInUse ? (
        <div className="py-3">
          <div className="text-gray-700 dark:text-gray-300">
            {t('storages.transfer.blockedByDatabases')}
          </div>

          <div className="mt-5">
            <Button type="primary" onClick={onClose}>
              {t('common.actions.ok')}
            </Button>
          </div>
        </div>
      ) : (
        <div className="py-3">
          <div className="mb-3 text-gray-500 dark:text-gray-400">
            {t('storages.transfer.description')}
          </div>

          <div className="mb-5 flex items-center">
            <div className="min-w-[120px] pr-2">{t('storages.transfer.targetWorkspace')}</div>

            <Select
              value={selectedWorkspaceId}
              onChange={setSelectedWorkspaceId}
              className="min-w-[200px] grow"
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
