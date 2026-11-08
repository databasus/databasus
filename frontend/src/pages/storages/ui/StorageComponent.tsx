import { ActionIcon, Alert, Button, Loader, TextInput } from '@mantine/core';
import { useQueryClient } from '@tanstack/react-query';
import { ArrowRight, CircleAlert, Trash2, X } from 'lucide-react';
import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Storage, storageApi } from '@/entities/storages';
import { EditStorageComponent } from '@/features/edit-storage';
import { ShowStorageComponent } from '@/pages/storages/ui/show/ShowStorageComponent';
import { StorageBackupUsage } from '@/pages/storages/ui/StorageBackupUsage';
import { StorageTransferDialogComponent } from '@/pages/storages/ui/StorageTransferDialogComponent';
import { useSessionMutation } from '@/shared/api';
import { ApiError, useSessionQuery } from '@/shared/api';
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
  const testStorageMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof storageApi.testStorage>) =>
      storageApi.testStorage(...parameters),
  });
  const deleteStorageMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof storageApi.deleteStorage>) =>
      storageApi.deleteStorage(...parameters),
  });
  const updateStorageMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof storageApi.updateStorage>) =>
      storageApi.updateStorage(...parameters),
  });

  const { t } = useTranslation();
  const [actionError, setError] = useState<unknown>();
  const [connectionError, setConnectionError] = useState<string>();
  const [isDeleteBlocked, setIsDeleteBlocked] = useState(false);
  const client = useQueryClient();
  const query = useSessionQuery({
    queryKey: ['storage', storageId],
    queryFn: (signal) => storageApi.getStorage(storageId, { signal }),
  });
  const [isUnavailable, setIsUnavailable] = useState(false);
  const isAccessRefused =
    query.error instanceof ApiError && [403, 404].includes(query.error.status ?? 0);
  const storage = isAccessRefused || isUnavailable ? undefined : query.data;
  const error = actionError ?? query.error ?? undefined;
  const setStorage = (resource: Storage) => {
    void client.cancelQueries({ queryKey: query.queryKey, exact: true });
    client.setQueryData(query.queryKey, resource);
  };

  const [isEditName, setIsEditName] = useState(false);
  const [isEditSettings, setIsEditSettings] = useState(false);

  const [editStorage, setEditStorage] = useState<Storage | undefined>();
  const isSaving = updateStorageMutation.isPending;

  const isTestingConnection = testStorageMutation.isPending;

  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const isRemoving = deleteStorageMutation.isPending;
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

    try {
      const testResult = await testStorageMutation.mutateAsync([storage.id]);
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

    testStorageMutation.reset();
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

    try {
      await deleteStorageMutation.mutateAsync([storage.id, shouldAbandonCleanup]);
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

    deleteStorageMutation.reset();
  };

  const startEdit = (type: 'name' | 'settings') => {
    if (isEditName || isEditSettings || isSaving) return;

    if (type === 'settings') {
      savedTestRevision.Invalidate();
      testStorageMutation.reset();
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

    try {
      const renamedStorage = await updateStorageMutation.mutateAsync([
        editStorage.id,
        editStorage.name,
        editStorage.settings,
      ]);
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

    updateStorageMutation.reset();
  };

  const acknowledgeStorage = (savedStorage: Storage) => {
    savedTestRevision.Invalidate();
    setStorage(savedStorage);
    onStorageChanged(savedStorage);
  };

  const loadSettings = () => {
    contextRevision.Invalidate();
    savedTestRevision.Invalidate();
    testStorageMutation.reset();
    updateStorageMutation.reset();
    deleteStorageMutation.reset();
    setError(undefined);
    setConnectionError(undefined);
    setEditStorage(undefined);

    void query.refetch();
  };
  const reportUnavailable = useEffectEvent(() => {
    if (isAccessRefused) {
      setIsUnavailable(true);
      onUnavailable?.();
    }
  });
  useEffect(() => {
    if (query.error !== null) reportUnavailable();
  }, [query.error, query.errorUpdatedAt]);
  useEffect(() => {
    if (query.data !== undefined) setIsUnavailable(false);
  }, [query.data, query.dataRevision]);
  const notify = useFeedback();
  const waitButtonRef = useRef<HTMLButtonElement>(null);
  const isCurrent = useSessionGuard();
  const session = useSession();
  const contextRevision = useRequestRevision();
  const savedTestRevision = useRequestRevision();
  const latestName = useLatestRef(editStorage?.name);

  useEffect(() => {
    setIsEditName(false);
    setIsEditSettings(false);
    setIsShowTransferDialog(false);
    setIsShowRemoveConfirm(false);
    setIsShowCleanupWarning(false);
    contextRevision.Invalidate();
    setError(undefined);

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
              <div className="mt-5 flex flex-wrap items-center gap-1">
                <Button
                  variant="filled"
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
