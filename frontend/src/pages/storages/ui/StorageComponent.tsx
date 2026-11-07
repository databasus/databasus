import { ActionIcon, Alert, Button, Loader, TextInput } from '@mantine/core';
import { ArrowRight, CircleAlert, Trash2, X } from 'lucide-react';
import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Storage, storageApi } from '@/entities/storages';
import { EditStorageComponent } from '@/features/edit-storage';
import { ShowStorageComponent } from '@/pages/storages/ui/show/ShowStorageComponent';
import { StorageBackupUsage } from '@/pages/storages/ui/StorageBackupUsage';
import { StorageTransferDialogComponent } from '@/pages/storages/ui/StorageTransferDialogComponent';
import { ApiError } from '@/shared/api';
import { useSession, useSessionGuard } from '@/shared/auth';
import { useLatestRef, useRequestRevision } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { Dialog } from '@/shared/ui';
import { ConfirmationComponent, LoadFailure } from '@/shared/ui';

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
  const [isShowCleanupWarning, setIsShowCleanupWarning] = useState(false);

  const [isShowTransferDialog, setIsShowTransferDialog] = useState(false);

  const testConnection = async () => {
    if (!storage || !isCurrent() || isTestingConnection) return;

    const revision = contextRevision.GetCurrent();
    const testRevision = savedTestRevision.Invalidate();
    const isActive = () =>
      isCurrent() &&
      contextRevision.IsCurrent(revision) &&
      savedTestRevision.IsCurrent(testRevision);

    setError(undefined);
    setConnectionError(undefined);
    setIsDeleteBlocked(false);
    setIsTestingConnection(true);

    try {
      const testResult = await storageApi.testStorage(storage.id);
      if (!isActive()) return;

      if (testResult.isConnected) {
        notify({
          kind: 'success',
          title: t('storages.connectionTestSucceeded.title'),
          description: t('storages.connectionTestSucceeded.description'),
          lifetimeMs: 4500,
          position: 'top-right',
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
  const remove = async (shouldAbandonCleanup = false) => {
    if (!storage || !isCurrent() || isRemoving || !isCanManageStorages) return;

    const revision = contextRevision.GetCurrent();
    const isActive = () => isCurrent() && contextRevision.IsCurrent(revision);
    setError(undefined);
    setConnectionError(undefined);
    setIsDeleteBlocked(false);

    if (!storage) return;

    setIsRemoving(true);

    try {
      await storageApi.deleteStorage(storage.id, shouldAbandonCleanup);
      if (!isActive()) return;

      onStorageDeleted();
    } catch (e) {
      if (!isActive()) return;

      setIsShowRemoveConfirm(false);

      if (e instanceof ApiError && e.code === 'cleanup_pending') {
        setIsShowCleanupWarning(true);
      } else if (e instanceof ApiError && e.code === STORAGE_IN_USE_CODE) {
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
      savedTestRevision.Invalidate();
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
    const revision = contextRevision.GetCurrent();
    const isActive = () => isCurrent() && contextRevision.IsCurrent(revision);
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
    savedTestRevision.Invalidate();
    setStorage(savedStorage);
    onStorageChanged(savedStorage);
  };

  const loadSettings = () => {
    const revision = contextRevision.Invalidate();
    savedTestRevision.Invalidate();
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
        if (isCurrent() && contextRevision.IsCurrent(revision)) setStorage(loadedStorage);
      })
      .catch((failure) => {
        if (isCurrent() && contextRevision.IsCurrent(revision)) {
          setError(failure);

          if (failure instanceof ApiError && [403, 404].includes(failure.status ?? 0))
            onUnavailable?.();
        }
      });
  };
  const notify = useFeedback();
  const waitButtonRef = useRef<HTMLButtonElement>(null);
  const isCurrent = useSessionGuard();
  const session = useSession();
  const contextRevision = useRequestRevision();
  const savedTestRevision = useRequestRevision();
  const latestName = useLatestRef(editStorage?.name);

  const startSettingsLoad = useEffectEvent(() => {
    void loadSettings();
  });

  useEffect(() => {
    setIsEditName(false);
    setIsEditSettings(false);
    setIsShowTransferDialog(false);
    setIsShowRemoveConfirm(false);
    setIsShowCleanupWarning(false);
    startSettingsLoad();

    return () => {
      contextRevision.Invalidate();
    };
  }, [storageId, contextRevision, session]);

  const isNameUnsaved = isEditName && editStorage?.name !== storage?.name;

  return (
    <div className="w-full min-w-0">
      <div className="min-w-0 grow rounded bg-surface p-5 shadow">
        {error !== undefined &&
          (storage ? (
            <Alert
              color="danger"
              icon={
                <CircleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />
              }
              title={translateApiError(error, t)}
            />
          ) : (
            <LoadFailure error={error} onRetry={loadSettings} />
          ))}
        {connectionError !== undefined && (
          <Alert
            color="danger"
            title={t('storages.connectionTestFailed', { error: connectionError })}
          />
        )}
        {isDeleteBlocked && (
          <Alert color="danger" title={t('storages.config.removeBlockedByBackups')} />
        )}
        {!storage && error === undefined ? (
          <div className="mt-10 flex justify-center">
            <Loader role="status" />
          </div>
        ) : storage ? (
          <div>
            {!isEditName ? (
              <div className="flex items-center text-2xl font-bold">
                <span className="min-w-0 wrap-anywhere">{storage.name}</span>
                {isCanManageStorages && (
                  <Button
                    variant="subtle"
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
                  <TextInput
                    aria-label={t('common.fields.name')}
                    className="max-w-62.5"
                    value={editStorage?.name}
                    onChange={(e) => {
                      if (!editStorage) return;
                      setEditStorage({ ...editStorage, name: e.target.value });
                    }}
                    placeholder={t('storages.config.namePlaceholder')}
                    size="md"
                  />

                  <div className="ml-1 flex items-center">
                    <Button
                      variant="subtle"
                      aria-label={t('common.actions.cancel')}
                      className="flex h-6 w-6 items-center justify-center p-0"
                      onClick={() => {
                        setIsEditName(false);
                        setEditStorage(undefined);
                      }}
                    >
                      <X className="inline-block shrink-0 align-middle text-muted" size="1em" />
                    </Button>
                  </div>
                </div>

                {isNameUnsaved && (
                  <Button
                    className="mt-1"
                    variant="filled"
                    onClick={() => saveName()}
                    aria-label={t('common.actions.save')}
                    loading={isSaving}
                    disabled={!editStorage?.name}
                    aria-busy={isSaving}
                  >
                    {t('common.actions.save')}
                  </Button>
                )}
              </div>
            )}

            {storage.id === storageId && (
              <StorageBackupUsage storageId={storage.id} workspaceId={storage.workspaceId} />
            )}

            <div className="mt-5 flex items-center font-bold">
              <div>{t('storages.config.settingsTitle')}</div>

              {!isEditSettings && isCanManageStorages ? (
                <Button
                  variant="subtle"
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
                  variant="filled"
                  className="mr-1"
                  onClick={testConnection}
                  loading={isTestingConnection}
                  disabled={isTestingConnection}
                  aria-busy={isTestingConnection}
                >
                  {t('storages.actions.testConnection')}
                </Button>

                {isCanManageStorages && (
                  <>
                    <ActionIcon
                      variant="outline"
                      color="accent"
                      aria-label={t('storages.transfer.title')}
                      onClick={() => setIsShowTransferDialog(true)}
                      className="mr-1"
                    >
                      <ArrowRight
                        aria-hidden
                        size="1em"
                        className="inline-block shrink-0 align-middle"
                      />
                    </ActionIcon>

                    <ActionIcon
                      variant="outline"
                      color="danger"
                      aria-label={t('common.actions.remove')}
                      onClick={() => setIsShowRemoveConfirm(true)}
                      loading={isRemoving}
                      disabled={isRemoving}
                      aria-busy={isRemoving}
                    >
                      <Trash2
                        aria-hidden
                        size="1em"
                        className="inline-block shrink-0 align-middle"
                      />
                    </ActionIcon>
                  </>
                )}
              </div>
            )}
          </div>
        ) : null}

        {isShowRemoveConfirm && (
          <ConfirmationComponent
            onConfirm={() => void remove()}
            onDecline={() => setIsShowRemoveConfirm(false)}
            description={t('storages.config.removeConfirmation')}
            actionText={t('common.actions.remove')}
            isDestructive
          />
        )}
      </div>

      {isShowCleanupWarning && (
        <Dialog
          title={t('storages.cleanup.title')}
          onEnterTransitionEnd={() => waitButtonRef.current?.focus()}
          opened
          onClose={() => {
            if (!isRemoving) setIsShowCleanupWarning(false);
          }}
          withCloseButton={!isRemoving}
          closeOnEscape={!isRemoving}
          closeOnClickOutside={false}
        >
          <p>{t('storages.cleanup.warning')}</p>
          {error !== undefined && (
            <Alert
              color="danger"
              icon={
                <CircleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />
              }
              title={translateApiError(error, t)}
            />
          )}
          {isDeleteBlocked && (
            <Alert
              color="danger"
              icon={
                <CircleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />
              }
              title={t('storages.config.removeBlockedByBackups')}
            />
          )}
          <div className="mt-5 flex flex-wrap justify-end gap-2">
            <div className="flex flex-wrap justify-end gap-2">
              <Button
                key="wait"
                ref={waitButtonRef}
                data-autofocus
                className="!h-auto min-h-8 max-w-full !whitespace-normal"
                disabled={isRemoving}
                onClick={() => setIsShowCleanupWarning(false)}
              >
                {t('storages.cleanup.wait')}
              </Button>
              <Button
                key="delete"
                className="!h-auto min-h-8 max-w-full !whitespace-normal"
                color="danger"
                variant="filled"
                loading={isRemoving}
                disabled={isRemoving}
                onClick={() => void remove(true)}
                aria-busy={isRemoving}
              >
                {t('storages.cleanup.deleteAnyway')}
              </Button>
            </div>
          </div>
        </Dialog>
      )}

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
