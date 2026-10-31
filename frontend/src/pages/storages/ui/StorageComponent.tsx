import { ArrowRightOutlined, CloseOutlined, DeleteOutlined } from '@ant-design/icons';
import { Alert, App, Button, Input, Spin } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { EditStorageComponent } from '@/features/edit-storage';
import { useSessionGuard } from '@/shared/auth';
import { RequestRevision } from '@/shared/lib/RequestRevision';

import { type Storage, storageApi } from '../../../entities/storages';
import { ApiError } from '../../../shared/api';
import { translateApiError } from '../../../shared/i18n';
import { ConfirmationComponent } from '../../../shared/ui';
import { ShowStorageComponent } from './show/ShowStorageComponent';
import { StorageTransferDialogComponent } from './StorageTransferDialogComponent';

// eslint-disable-next-line i18next/no-literal-string -- API error code, not copy
const STORAGE_IN_USE_CODE = 'in_use';

interface Props {
  onUnavailable?: () => void;
  storageId: string;
  onStorageChanged: (storage: Storage) => void;
  onStorageDeleted: () => void;
  onStorageTransferred: () => void;
  isCanManageStorages: boolean;
}

export const StorageComponent = ({
  onUnavailable,
  storageId,
  onStorageChanged,
  onStorageDeleted,
  onStorageTransferred,
  isCanManageStorages,
}: Props) => {
  const { t } = useTranslation();
  const [error, setError] = useState<unknown>();
  const [connectionError, setConnectionError] = useState<string>();
  const [isDeleteBlocked, setIsDeleteBlocked] = useState(false);
  const [storage, setStorage] = useState<Storage | undefined>();

  const [isEditName, setIsEditName] = useState(false);
  const [isEditSettings, setIsEditSettings] = useState(false);

  const [editStorage, setEditStorage] = useState<Storage | undefined>();
  const [isSaving, setIsSaving] = useState(false);

  const [isTestingConnection, setIsTestingConnection] = useState(false);

  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);

  const [isShowTransferDialog, setIsShowTransferDialog] = useState(false);

  const testConnection = async () => {
    if (!storage || !isCurrent() || isTestingConnection) return;

    const revision = contextRevision.current.GetCurrent();
    const testRevision = savedTestRevision.current.Invalidate();
    const isActive = () =>
      isCurrent() &&
      contextRevision.current.IsCurrent(revision) &&
      savedTestRevision.current.IsCurrent(testRevision);

    setError(undefined);
    setConnectionError(undefined);
    setIsDeleteBlocked(false);
    setIsTestingConnection(true);

    try {
      const testResult = await storageApi.testStorage(storage.id);
      if (!isActive()) return;

      if (testResult.isConnected) {
        notification.success({
          message: t('storages.connectionTestSucceeded.title'),
          description: t('storages.connectionTestSucceeded.description'),
        });
      } else {
        setConnectionError(testResult.error ?? '');
      }
    } catch (e) {
      if (!isActive()) return;

      setError(e);
    }

    setIsTestingConnection(false);
  };

  // A storage that a backup config uses or that still holds backups is refused with in_use; the
  // generic in_use text names only databases, so the storage has its own.
  const remove = async () => {
    const revision = contextRevision.current.GetCurrent();
    const isActive = () => isCurrent() && contextRevision.current.IsCurrent(revision);
    setError(undefined);
    setConnectionError(undefined);
    setIsDeleteBlocked(false);

    if (!storage) return;

    setIsRemoving(true);

    try {
      await storageApi.deleteStorage(storage.id);
      if (!isActive()) return;

      onStorageDeleted();
    } catch (e) {
      if (!isActive()) return;

      setIsShowRemoveConfirm(false);

      if (e instanceof ApiError && e.code === STORAGE_IN_USE_CODE) {
        setIsDeleteBlocked(true);
      } else {
        setError(e);
      }
    }

    setIsRemoving(false);
  };

  const startEdit = (type: 'name' | 'settings') => {
    if (isEditName || isEditSettings || isSaving) return;

    if (type === 'settings') {
      savedTestRevision.current.Invalidate();
      setIsTestingConnection(false);
      setError(undefined);
      setConnectionError(undefined);
    }

    setEditStorage(structuredClone(storage));
    setIsEditName(type === 'name');
    setIsEditSettings(type === 'settings');
  };

  // The settings of a read storage carry no secrets, so the rename keeps the stored ones.
  const saveName = async () => {
    const revision = contextRevision.current.GetCurrent();
    const isActive = () => isCurrent() && contextRevision.current.IsCurrent(revision);
    setError(undefined);
    setConnectionError(undefined);
    setIsDeleteBlocked(false);

    if (!editStorage || isSaving || !isCurrent()) return;

    const submittedName = editStorage.name;

    setIsSaving(true);

    try {
      const renamedStorage = await storageApi.updateStorage(
        editStorage.id,
        editStorage.name,
        editStorage.settings,
      );
      if (!isActive()) return;

      setStorage(renamedStorage);
      const currentName = latestName.current;
      const isNewerName = currentName !== submittedName && currentName !== renamedStorage.name;

      if (!isNewerName) {
        setEditStorage(renamedStorage);
        setIsEditName(false);
      }

      onStorageChanged(renamedStorage);
    } catch (e) {
      if (!isActive()) return;

      setError(e);
    }

    setIsSaving(false);
  };

  const acknowledgeStorage = (savedStorage: Storage) => {
    savedTestRevision.current.Invalidate();
    setStorage(savedStorage);
    onStorageChanged(savedStorage);
  };

  const loadSettings = () => {
    const revision = contextRevision.current.Invalidate();
    savedTestRevision.current.Invalidate();
    setIsTestingConnection(false);
    setIsSaving(false);
    setIsRemoving(false);
    setError(undefined);
    setConnectionError(undefined);
    setStorage(undefined);
    setEditStorage(undefined);
    storageApi
      .getStorage(storageId)
      .then((loadedStorage) => {
        if (isCurrent() && contextRevision.current.IsCurrent(revision)) setStorage(loadedStorage);
      })
      .catch((failure) => {
        if (isCurrent() && contextRevision.current.IsCurrent(revision)) {
          setError(failure);

          if (failure instanceof ApiError && [403, 404].includes(failure.status ?? 0))
            onUnavailable?.();
        }
      });
  };

  const { notification } = App.useApp();
  const isCurrent = useSessionGuard();
  const contextRevision = useRef(new RequestRevision());
  const savedTestRevision = useRef(new RequestRevision());
  const latestName = useRef(editStorage?.name);
  latestName.current = editStorage?.name;

  useEffect(() => {
    setIsEditName(false);
    setIsEditSettings(false);
    setIsShowTransferDialog(false);
    setIsShowRemoveConfirm(false);
    loadSettings();

    return () => {
      contextRevision.current.Invalidate();
    };
  }, [storageId]);

  const isNameUnsaved = isEditName && editStorage?.name !== storage?.name;

  return (
    <div className="w-full min-w-0">
      <div className="min-w-0 grow rounded bg-surface p-5 shadow">
        {error !== undefined && (
          <Alert
            type="error"
            showIcon
            message={translateApiError(error, t)}
            action={
              !storage ? (
                <Button onClick={loadSettings}>{t('common.actions.retry')}</Button>
              ) : undefined
            }
          />
        )}
        {connectionError !== undefined && (
          <Alert
            type="error"
            message={t('storages.connectionTestFailed', { error: connectionError })}
          />
        )}
        {isDeleteBlocked && (
          <Alert type="error" message={t('storages.config.removeBlockedByBackups')} />
        )}
        {!storage && error === undefined ? (
          <div className="mt-10 flex justify-center">
            <Spin />
          </div>
        ) : storage ? (
          <div>
            {!isEditName ? (
              <div className="mb-5 flex items-center text-2xl font-bold">
                {storage.name}
                {isCanManageStorages && (
                  <Button
                    type="text"
                    aria-label={t('common.actions.edit')}
                    className="ml-2"
                    onClick={() => startEdit('name')}
                    disabled={isEditSettings || isSaving}
                  >
                    <img src="/icons/pen-gray.svg" alt="" className="h-4 w-4" />
                  </Button>
                )}
              </div>
            ) : (
              <div>
                <div className="flex items-center">
                  <Input
                    aria-label={t('common.fields.name')}
                    className="max-w-62.5"
                    value={editStorage?.name}
                    onChange={(e) => {
                      if (!editStorage) return;
                      setEditStorage({ ...editStorage, name: e.target.value });
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
                        setEditStorage(undefined);
                      }}
                    >
                      <CloseOutlined className="text-muted" />
                    </Button>
                  </div>
                </div>

                {isNameUnsaved && (
                  <Button
                    className="mt-1"
                    type="primary"
                    onClick={() => saveName()}
                    aria-label={t('common.actions.save')}
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
                <Button
                  type="text"
                  aria-label={t('common.actions.edit')}
                  className="ml-2"
                  onClick={() => startEdit('settings')}
                  disabled={isEditName || isSaving}
                >
                  <img src="/icons/pen-gray.svg" alt="" className="h-4 w-4" />
                </Button>
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
                  onPersisted={acknowledgeStorage}
                  onChanged={() => {}}
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
                      aria-label={t('storages.transfer.title')}
                      icon={<ArrowRightOutlined aria-hidden />}
                      onClick={() => setIsShowTransferDialog(true)}
                      className="mr-1"
                    />

                    <Button
                      type="primary"
                      ghost
                      danger
                      aria-label={t('common.actions.remove')}
                      icon={<DeleteOutlined aria-hidden />}
                      onClick={() => setIsShowRemoveConfirm(true)}
                      loading={isRemoving}
                      disabled={isRemoving}
                    />
                  </>
                )}
              </div>
            )}
          </div>
        ) : null}

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
