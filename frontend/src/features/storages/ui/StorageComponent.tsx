import { ArrowRightOutlined, CloseOutlined, DeleteOutlined } from '@ant-design/icons';
import { Button, Input, Spin } from 'antd';
import { useState } from 'react';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { type Storage, storageApi } from '../../../entities/storages';
import { ApiError } from '../../../shared/api';
import { translateApiError } from '../../../shared/i18n';
import { ToastHelper } from '../../../shared/toast';
import { ConfirmationComponent } from '../../../shared/ui';
import { EditStorageComponent } from './edit/EditStorageComponent';
import { ShowStorageComponent } from './show/ShowStorageComponent';
import { StorageTransferDialogComponent } from './StorageTransferDialogComponent';

// eslint-disable-next-line i18next/no-literal-string -- API error code, not copy
const STORAGE_IN_USE_CODE = 'in_use';

interface Props {
  storageId: string;
  onStorageChanged: (storage: Storage) => void;
  onStorageDeleted: () => void;
  onStorageTransferred: () => void;
  isCanManageStorages: boolean;
}

export const StorageComponent = ({
  storageId,
  onStorageChanged,
  onStorageDeleted,
  onStorageTransferred,
  isCanManageStorages,
}: Props) => {
  const { t } = useTranslation();
  const [storage, setStorage] = useState<Storage | undefined>();

  const [isEditName, setIsEditName] = useState(false);
  const [isEditSettings, setIsEditSettings] = useState(false);

  const [editStorage, setEditStorage] = useState<Storage | undefined>();
  const [isNameUnsaved, setIsNameUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [isTestingConnection, setIsTestingConnection] = useState(false);

  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);

  const [isShowTransferDialog, setIsShowTransferDialog] = useState(false);

  const testConnection = async () => {
    if (!storage) return;

    setIsTestingConnection(true);

    try {
      const testResult = await storageApi.testStorage(storage.id);

      if (testResult.isConnected) {
        ToastHelper.showToast({
          title: t('storages.connectionTestSucceeded.title'),
          description: t('storages.connectionTestSucceeded.description'),
        });
      } else {
        alert(t('storages.connectionTestFailed', { error: testResult.error }));
      }
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsTestingConnection(false);
  };

  // A storage that a backup config uses or that still holds backups is refused with in_use; the
  // generic in_use text names only databases, so the storage has its own.
  const remove = async () => {
    if (!storage) return;

    setIsRemoving(true);

    try {
      await storageApi.deleteStorage(storage.id);
      onStorageDeleted();
    } catch (e) {
      if (e instanceof ApiError && e.code === STORAGE_IN_USE_CODE) {
        alert(t('storages.config.removeBlockedByBackups'));
        setIsShowRemoveConfirm(false);
      } else {
        alert(translateApiError(e, t));
      }
    }

    setIsRemoving(false);
  };

  const startEdit = (type: 'name' | 'settings') => {
    setEditStorage(structuredClone(storage));
    setIsEditName(type === 'name');
    setIsEditSettings(type === 'settings');
    setIsNameUnsaved(false);
  };

  // The settings of a read storage carry no secrets, so the rename keeps the stored ones.
  const saveName = async () => {
    if (!editStorage) return;

    setIsSaving(true);

    try {
      const renamedStorage = await storageApi.updateStorage(
        editStorage.id,
        editStorage.name,
        editStorage.settings,
      );
      setStorage(renamedStorage);
      setIsNameUnsaved(false);
      setIsEditName(false);
      onStorageChanged(renamedStorage);
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsSaving(false);
  };

  const acknowledgeStorage = (savedStorage: Storage) => {
    setStorage(savedStorage);
    onStorageChanged(savedStorage);
  };

  const loadSettings = () => {
    setStorage(undefined);
    setEditStorage(undefined);
    storageApi.getStorage(storageId).then(setStorage);
  };

  useEffect(() => {
    setIsEditName(false);
    setIsEditSettings(false);
    loadSettings();
  }, [storageId]);

  return (
    <div className="w-full">
      <div className="grow overflow-y-auto rounded bg-white p-5 shadow dark:bg-gray-800">
        {!storage ? (
          <div className="mt-10 flex justify-center">
            <Spin />
          </div>
        ) : (
          <div>
            {!isEditName ? (
              <div className="mb-5 flex items-center text-2xl font-bold">
                {storage.name}
                {isCanManageStorages && (
                  <div className="ml-2 cursor-pointer" onClick={() => startEdit('name')}>
                    <img src="/icons/pen-gray.svg" />
                  </div>
                )}
              </div>
            ) : (
              <div>
                <div className="flex items-center">
                  <Input
                    className="max-w-[250px]"
                    value={editStorage?.name}
                    onChange={(e) => {
                      if (!editStorage) return;

                      setEditStorage({ ...editStorage, name: e.target.value });
                      setIsNameUnsaved(true);
                    }}
                    placeholder={t('storages.config.namePlaceholder')}
                    size="large"
                  />

                  <div className="ml-1 flex items-center">
                    <Button
                      type="text"
                      className="flex h-6 w-6 items-center justify-center p-0"
                      onClick={() => {
                        setIsEditName(false);
                        setIsNameUnsaved(false);
                        setEditStorage(undefined);
                      }}
                    >
                      <CloseOutlined className="text-gray-500 dark:text-gray-400" />
                    </Button>
                  </div>
                </div>

                {isNameUnsaved && (
                  <Button
                    className="mt-1"
                    type="primary"
                    onClick={() => saveName()}
                    loading={isSaving}
                    disabled={!editStorage?.name}
                  >
                    {t('common.actions.save')}
                  </Button>
                )}
              </div>
            )}

            <div className="mt-5 flex items-center font-bold">
              <div>{t('storages.config.settingsTitle')}</div>

              {!isEditSettings && isCanManageStorages ? (
                <div className="ml-2 h-4 w-4 cursor-pointer" onClick={() => startEdit('settings')}>
                  <img src="/icons/pen-gray.svg" />
                </div>
              ) : (
                <div />
              )}
            </div>

            <div className="mt-1 text-sm">
              {isEditSettings && isCanManageStorages ? (
                <EditStorageComponent
                  workspaceId={storage.workspaceId}
                  isShowClose
                  onClose={() => {
                    setIsEditSettings(false);
                    setEditStorage(undefined);
                    loadSettings();
                  }}
                  isShowName={false}
                  editingStorage={editStorage}
                  onChanged={acknowledgeStorage}
                />
              ) : (
                <ShowStorageComponent storage={storage} />
              )}
            </div>

            {!isEditSettings && (
              <div className="mt-5">
                <Button
                  type="primary"
                  className="mr-1"
                  onClick={testConnection}
                  loading={isTestingConnection}
                  disabled={isTestingConnection}
                >
                  {t('storages.actions.testConnection')}
                </Button>

                {isCanManageStorages && (
                  <>
                    <Button
                      type="primary"
                      ghost
                      icon={<ArrowRightOutlined />}
                      onClick={() => setIsShowTransferDialog(true)}
                      className="mr-1"
                    />

                    <Button
                      type="primary"
                      ghost
                      danger
                      icon={<DeleteOutlined />}
                      onClick={() => setIsShowRemoveConfirm(true)}
                      loading={isRemoving}
                      disabled={isRemoving}
                    />
                  </>
                )}
              </div>
            )}
          </div>
        )}

        {isShowRemoveConfirm && (
          <ConfirmationComponent
            onConfirm={remove}
            onDecline={() => setIsShowRemoveConfirm(false)}
            description={t('storages.config.removeConfirmation')}
            actionText={t('common.actions.remove')}
            isDestructive
          />
        )}
      </div>

      {isShowTransferDialog && storage && (
        <StorageTransferDialogComponent
          storage={storage}
          onClose={() => setIsShowTransferDialog(false)}
          onTransferred={() => {
            setIsShowTransferDialog(false);
            onStorageTransferred();
          }}
        />
      )}
    </div>
  );
};
