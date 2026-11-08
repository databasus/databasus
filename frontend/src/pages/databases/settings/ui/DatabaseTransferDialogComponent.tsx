import { Alert, Button, Loader, MultiSelect, Radio } from '@mantine/core';
import { useQueryClient } from '@tanstack/react-query';
import { ArrowRightLeft, CircleAlert, CircleCheck, TriangleAlert } from 'lucide-react';
import { useEffect, useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type DatabaseTransfer, backupConfigApi } from '@/entities/backup-configs';
import { type Database, databaseApi } from '@/entities/databases';
import { type Notifier, notifierApi } from '@/entities/notifiers';
import { type Storage, getStorageLogoFromType, storageApi } from '@/entities/storages';
import { workspaceApi } from '@/entities/workspaces';
import { EditNotifierComponent } from '@/features/edit-notifier';
import { EditStorageComponent } from '@/features/edit-storage';
import { readConcurrentRequests, useSessionMutation, useSessionQuery } from '@/shared/api';
import { getSessionSnapshot, isCurrentSession, useSessionGuard } from '@/shared/auth';
import { useLatestRef, useRequestRevision } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { WrappingSelect, Dialog } from '@/shared/ui';

interface Props {
  database: Database;
  currentStorageId?: string;
  onClose: () => void;
  onTransferred: () => void;
}

interface NotifierUsageInfo {
  notifier: Notifier;
  databaseCount: number;
  canTransfer: boolean;
}

// Two configs are enough to learn whether a database other than the transferred one backs up into
// its storage.
const STORAGE_SHARING_CHECK_LIMIT = 2;

const toTransferStorage = (
  storageOption: 'transfer' | 'select',
  selectedStorageId: string | undefined,
): DatabaseTransfer['storage'] | undefined => {
  if (storageOption === 'transfer') return { type: 'MOVE_WITH_DATABASE' };

  if (!selectedStorageId) return undefined;

  return { type: 'SWITCH_TO_STORAGE', storageId: selectedStorageId };
};

export const DatabaseTransferDialogComponent = ({
  database,
  currentStorageId,
  onClose,
  onTransferred,
}: Props) => {
  const transferDatabaseMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof backupConfigApi.transferDatabase>) =>
      backupConfigApi.transferDatabase(...parameters),
  });

  const client = useQueryClient();
  const { t } = useTranslation();
  const storageOptionLabelId = useId();
  const notifierOptionLabelId = useId();
  const [actionError, setError] = useState<unknown>();
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string | undefined>();

  const [storageOption, setStorageOption] = useState<'transfer' | 'select'>('select');

  const [targetStorages, setTargetStorages] = useState<Storage[]>([]);
  const [selectedStorageId, setSelectedStorageId] = useState<string | undefined>();
  const [isShowCreateStorage, setIsShowCreateStorage] = useState(false);

  const [notifierOption, setNotifierOption] = useState<'transfer' | 'select'>('select');

  const [targetNotifiers, setTargetNotifiers] = useState<Notifier[]>([]);
  const [selectedNotifierIds, setSelectedNotifierIds] = useState<string[]>([]);
  const [isShowCreateNotifier, setIsShowCreateNotifier] = useState(false);

  const isTransferring = transferDatabaseMutation.isPending;
  const notify = useFeedback();
  const isCurrent = useSessionGuard();
  const contextRevision = useRequestRevision();
  const acknowledgedStorages = useRef<Storage[]>([]);
  const acknowledgedNotifiers = useRef<Notifier[]>([]);
  const targetWorkspace = useLatestRef(selectedWorkspaceId);

  const hasCurrentStorage = !!currentStorageId;
  const hasCurrentNotifiers = database.notifierIds.length > 0;

  const workspacesQuery = useSessionQuery({
    queryKey: ['database-transfer-workspaces', database.workspaceId, database.id],
    queryFn: (signal) => workspaceApi.listAllWorkspaces({ signal }),
  });
  const sharingQuery = useSessionQuery({
    queryKey: ['database-transfer-sharing', database.id, currentStorageId],
    enabled: hasCurrentStorage,
    queryFn: async (signal) => {
      const configs = await backupConfigApi.listStorageBackupConfigs(
        currentStorageId!,
        STORAGE_SHARING_CHECK_LIMIT,
        { signal },
      );

      return configs.some((config) => config.databaseId !== database.id);
    },
  });
  const usageQuery = useSessionQuery({
    queryKey: [
      'database-transfer-notifiers',
      database.workspaceId,
      database.id,
      database.notifierIds,
    ],
    enabled: hasCurrentNotifiers,
    queryFn: (signal) =>
      readConcurrentRequests(signal, async (signal): Promise<NotifierUsageInfo[]> => {
        const notifiers = await notifierApi.listAllNotifiers(database.workspaceId, { signal });
        signal.throwIfAborted();

        return Promise.all(
          notifiers
            .filter((notifier) => database.notifierIds.includes(notifier.id))
            .map(async (notifier) => {
              const databases = await databaseApi.listAllDatabases(database.workspaceId, {
                notifierId: notifier.id,
                signal,
              });

              return {
                notifier,
                databaseCount: databases.length,
                canTransfer: databases.length === 1,
              };
            }),
        );
      }),
  });
  const targetQuery = useSessionQuery({
    queryKey: ['database-transfer-target', database.workspaceId, database.id, selectedWorkspaceId],
    enabled: !!selectedWorkspaceId,
    queryFn: (signal) =>
      readConcurrentRequests(signal, async (signal) => {
        const [storages, notifiers] = await Promise.all([
          storageApi.listAllStorages(selectedWorkspaceId!, { signal }),
          notifierApi.listAllNotifiers(selectedWorkspaceId!, { signal }),
        ]);

        return {
          storages: [
            ...storages.filter(
              (storage) => !acknowledgedStorages.current.some((item) => item.id === storage.id),
            ),
            ...acknowledgedStorages.current,
          ],
          notifiers: [
            ...notifiers.filter(
              (notifier) => !acknowledgedNotifiers.current.some((item) => item.id === notifier.id),
            ),
            ...acknowledgedNotifiers.current,
          ],
        };
      }),
  });
  const isLoading = workspacesQuery.isPending;
  const workspaces = (workspacesQuery.data ?? []).filter(
    (workspace) => workspace.id !== database.workspaceId,
  );
  const isLoadingStorageSharing = hasCurrentStorage && sharingQuery.isPending;
  const isStorageSharedWithOtherDatabases = sharingQuery.data ?? false;
  const isLoadingNotifierCounts = hasCurrentNotifiers && usageQuery.isPending;
  const notifierUsageInfo = usageQuery.data ?? [];
  const error =
    actionError ??
    workspacesQuery.error ??
    sharingQuery.error ??
    usageQuery.error ??
    targetQuery.error ??
    undefined;

  const acceptTargetStorage = (storage: Storage) => {
    if (!isCurrent() || storage.workspaceId !== targetWorkspace.current) return;

    void client.cancelQueries({ queryKey: targetQuery.queryKey, exact: true });
    acknowledgedStorages.current = [
      ...acknowledgedStorages.current.filter((item) => item.id !== storage.id),
      storage,
    ];
    setTargetStorages((current) => [...current.filter((item) => item.id !== storage.id), storage]);
    void loadTargetWorkspaceData(storage.workspaceId);
  };

  const acceptTargetNotifier = (notifier: Notifier) => {
    if (!isCurrent() || notifier.workspaceId !== targetWorkspace.current) return;

    void client.cancelQueries({ queryKey: targetQuery.queryKey, exact: true });
    acknowledgedNotifiers.current = [
      ...acknowledgedNotifiers.current.filter((item) => item.id !== notifier.id),
      notifier,
    ];
    setTargetNotifiers((current) => [
      ...current.filter((item) => item.id !== notifier.id),
      notifier,
    ]);
    void loadTargetWorkspaceData(notifier.workspaceId);
  };

  const loadTargetWorkspaceData = (workspaceId: string) => {
    if (isCurrent() && targetWorkspace.current === workspaceId) return targetQuery.refetch();
  };
  useEffect(() => {
    if (!targetQuery.data) return;
    setTargetStorages(targetQuery.data.storages);
    setTargetNotifiers(targetQuery.data.notifiers);
  }, [targetQuery.data]);

  const handleTransfer = async () => {
    if (!isCurrent() || isTransferring) return;

    const revision = contextRevision.GetCurrent();
    const transferSession = getSessionSnapshot();
    const targetWorkspaceId = selectedWorkspaceId;
    const isActive = () =>
      isCurrent() &&
      contextRevision.IsCurrent(revision) &&
      targetWorkspace.current === targetWorkspaceId;
    const transferStorage = toTransferStorage(storageOption, selectedStorageId);

    if (!selectedWorkspaceId || !transferStorage) return;

    const transfer: DatabaseTransfer = {
      targetWorkspaceId: selectedWorkspaceId,
      storage: transferStorage,
      isMovingNotifiers: notifierOption === 'transfer',
      targetNotifierIds: notifierOption === 'select' ? selectedNotifierIds : [],
    };

    try {
      await transferDatabaseMutation.mutateAsync([database.id, transfer]);
      if (isActive())
        notify({
          kind: 'success',
          title: t('databases.transfer.transferred.title'),
          description: t('databases.transfer.transferred.description', {
            name: database.name,
          }),
          lifetimeMs: 4500,
          position: 'top-right',
        });

      if (isCurrentSession(transferSession)) onTransferred();
    } catch (e) {
      if (!isActive()) return;

      setError(e);
    }

    if (isActive()) transferDatabaseMutation.reset();
  };

  useEffect(() => {
    contextRevision.Invalidate();
    return () => {
      contextRevision.Invalidate();
    };
  }, [database.id, contextRevision]);

  useEffect(() => {
    acknowledgedStorages.current = [];
    acknowledgedNotifiers.current = [];
    setIsShowCreateStorage(false);
    setIsShowCreateNotifier(false);
    setError(undefined);
    setTargetStorages([]);
    setTargetNotifiers([]);

    if (selectedWorkspaceId) {
      setSelectedStorageId(undefined);
      setSelectedNotifierIds([]);
    }
  }, [selectedWorkspaceId]);

  const canTransferWithStorage = !isStorageSharedWithOtherDatabases;
  const notifiersBlockingTransfer = notifierUsageInfo.filter((info) => !info.canTransfer);
  const notifiersCanTransfer = notifierUsageInfo.filter((info) => info.canTransfer);

  const isStorageValid =
    storageOption === 'transfer'
      ? canTransferWithStorage && hasCurrentStorage
      : !!selectedStorageId;

  const isFormValid = !!selectedWorkspaceId && isStorageValid;

  const selectedStorage = targetStorages.find((storage) => storage.id === selectedStorageId);

  return (
    <Dialog
      title={
        <div className="flex items-center gap-2">
          <ArrowRightLeft size="1em" className="inline-block shrink-0 align-middle" />
          {t('databases.transfer.title')}
        </div>
      }
      opened={true}
      onClose={onClose}
      closeOnClickOutside={false}
      withCloseButton={!isTransferring}
      closeOnEscape={!isTransferring}
      size={550}
    >
      {error !== undefined && (
        <Alert
          color="danger"
          icon={
            <CircleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />
          }
          title={translateApiError(error, t)}
        >
          <Button
            disabled={isTransferring}
            onClick={() => {
              setError(undefined);
              void workspacesQuery.refetch();
              if (hasCurrentStorage) void sharingQuery.refetch();
              if (hasCurrentNotifiers) void usageQuery.refetch();
              if (selectedWorkspaceId) void loadTargetWorkspaceData(selectedWorkspaceId);
            }}
          >
            {t('common.actions.retry')}
          </Button>
        </Alert>
      )}
      {isLoading ? (
        <div className="flex justify-center py-5">
          <Loader role="status" />
        </div>
      ) : (
        <div className="py-3">
          <div className="mb-5">
            <div className="mb-2 font-medium">{t('databases.transfer.targetWorkspace')}</div>
            <WrappingSelect
              aria-label={t('databases.transfer.targetWorkspace')}
              disabled={isTransferring}
              value={selectedWorkspaceId}
              onChange={setSelectedWorkspaceId}
              className="w-full"
              placeholder={t('databases.transfer.selectWorkspace')}
              data={workspaces.map((workspace) => ({
                label: workspace.name,
                value: workspace.id,
              }))}
            />
          </div>

          {selectedWorkspaceId && (
            <>
              <div className="mb-5">
                <div id={storageOptionLabelId} className="mb-2 font-medium">
                  {t('databases.transfer.storage.title')}
                </div>
                <div role="radiogroup" aria-labelledby={storageOptionLabelId}>
                  <Radio.Group
                    disabled={isTransferring}
                    value={storageOption}
                    onChange={(value) => setStorageOption(value)}
                    className="flex flex-col gap-3"
                  >
                    {hasCurrentStorage && (
                      <div>
                        <Radio
                          value="transfer"
                          label={
                            <>
                              <span className="flex items-center gap-2">
                                {t('databases.transfer.storage.transferExisting')}
                                {isLoadingStorageSharing && <Loader size="xs" role="status" />}
                              </span>
                            </>
                          }
                        />
                      </div>
                    )}
                    <div>
                      <Radio
                        value="select"
                        label={<>{t('databases.transfer.storage.selectFromTarget')}</>}
                      />
                    </div>
                  </Radio.Group>
                </div>

                {storageOption === 'transfer' && hasCurrentStorage && (
                  <div className="mt-2 ml-6">
                    {isLoadingStorageSharing ? (
                      <Loader size="xs" role="status" />
                    ) : !canTransferWithStorage ? (
                      <div className="flex items-center gap-2 rounded border border-border bg-surface p-2 text-sm text-danger">
                        <CircleAlert size="1em" className="inline-block shrink-0 align-middle" />
                        <span>{t('databases.transfer.storage.blocked')}</span>
                      </div>
                    ) : (
                      <div className="flex items-center gap-2 text-sm text-success">
                        <CircleCheck size="1em" className="inline-block shrink-0 align-middle" />
                        {t('databases.transfer.storage.canTransfer')}
                      </div>
                    )}
                  </div>
                )}

                {storageOption === 'select' && (
                  <div className="mt-2 ml-6">
                    <div className="flex min-w-0 flex-col items-start gap-2 sm:flex-row sm:items-center">
                      <WrappingSelect
                        aria-label={t('databases.transfer.storage.title')}
                        disabled={isTransferring}
                        value={selectedStorageId}
                        onChange={(storageId) => {
                          setSelectedStorageId(storageId);
                        }}
                        className="w-full max-w-75 min-w-0 sm:flex-1"
                        placeholder={t('databases.transfer.storage.selectPlaceholder')}
                        data={[
                          ...targetStorages.map((storage) => ({
                            label: storage.name,
                            value: storage.id,
                          })),
                        ]}
                      />
                      <Button
                        disabled={isTransferring}
                        onClick={() => setIsShowCreateStorage(true)}
                      >
                        {t('databases.transfer.storage.createNew')}
                      </Button>

                      {selectedStorage && (
                        <img
                          src={getStorageLogoFromType(selectedStorage.settings.type)}
                          alt=""
                          className="h-4 w-4"
                        />
                      )}
                    </div>
                  </div>
                )}
              </div>

              {hasCurrentNotifiers && (
                <div className="mb-5">
                  <div id={notifierOptionLabelId} className="mb-2 font-medium">
                    {t('databases.transfer.notifiers.title')}
                  </div>

                  <div role="radiogroup" aria-labelledby={notifierOptionLabelId}>
                    <Radio.Group
                      disabled={isTransferring}
                      value={notifierOption}
                      onChange={(value) => setNotifierOption(value)}
                      className="flex flex-col gap-3"
                    >
                      <div>
                        <Radio
                          value="transfer"
                          label={<>{t('databases.transfer.notifiers.transferWithDatabase')}</>}
                        />
                      </div>
                      <div>
                        <Radio
                          value="select"
                          label={<>{t('databases.transfer.notifiers.selectFromTarget')}</>}
                        />
                      </div>
                    </Radio.Group>
                  </div>

                  {notifierOption === 'transfer' && (
                    <div className="mt-2 ml-6">
                      {isLoadingNotifierCounts ? (
                        <Loader size="xs" role="status" />
                      ) : (
                        <div className="space-y-1">
                          {notifiersCanTransfer.length > 0 && (
                            <div className="text-sm text-success">
                              <div className="mb-1 flex items-center gap-1">
                                <CircleCheck
                                  size="1em"
                                  className="inline-block shrink-0 align-middle"
                                />
                                <span>{t('databases.transfer.notifiers.willTransfer')}</span>
                              </div>
                              <ul className="ml-5 list-disc">
                                {notifiersCanTransfer.map((info) => (
                                  <li key={info.notifier.id}>{info.notifier.name}</li>
                                ))}
                              </ul>
                            </div>
                          )}

                          {notifiersBlockingTransfer.length > 0 && (
                            <div className="mt-2 rounded border border-border bg-surface p-2 text-sm text-danger">
                              <div className="mb-1 flex items-center gap-1">
                                <CircleAlert
                                  size="1em"
                                  className="inline-block shrink-0 align-middle"
                                />
                                <span>{t('databases.transfer.notifiers.willNotTransfer')}</span>
                              </div>
                              <ul className="ml-5 list-disc">
                                {notifiersBlockingTransfer.map((info) => (
                                  <li key={info.notifier.id}>
                                    {t('databases.transfer.notifiers.usedByDatabases', {
                                      name: info.notifier.name,
                                      count: info.databaseCount,
                                    })}
                                  </li>
                                ))}
                              </ul>
                            </div>
                          )}

                          {notifiersCanTransfer.length === 0 &&
                            notifiersBlockingTransfer.length > 0 && (
                              <div className="mt-1 text-sm text-muted">
                                {t('databases.transfer.notifiers.noneTransferred')}
                              </div>
                            )}
                        </div>
                      )}
                    </div>
                  )}

                  {notifierOption === 'select' && (
                    <div className="mt-2 ml-6 flex min-w-0 flex-col items-start gap-2 sm:flex-row sm:items-center">
                      <MultiSelect
                        aria-label={t('databases.transfer.notifiers.title')}
                        disabled={isTransferring}
                        value={selectedNotifierIds}
                        onChange={(notifierIds) => {
                          setSelectedNotifierIds(notifierIds);
                        }}
                        className="w-full max-w-75 min-w-0 sm:flex-1"
                        placeholder={t('databases.transfer.notifiers.selectPlaceholder')}
                        data={[
                          ...targetNotifiers.map((notifier) => ({
                            label: notifier.name,
                            value: notifier.id,
                          })),
                        ]}
                      />
                      <Button
                        disabled={isTransferring}
                        onClick={() => setIsShowCreateNotifier(true)}
                      >
                        {t('databases.notifiers.createNew')}
                      </Button>
                    </div>
                  )}
                </div>
              )}
            </>
          )}

          {storageOption === 'select' && selectedStorageId && (
            <Alert
              color="warning"
              icon={
                <TriangleAlert
                  aria-hidden
                  size={16}
                  className="inline-block shrink-0 align-middle"
                />
              }
              title={t('backups.storageSwitch.warning')}
            />
          )}
          <div className="mt-5 flex flex-wrap gap-2">
            <Button variant="default" onClick={onClose} disabled={isTransferring}>
              {t('common.actions.cancel')}
            </Button>

            <Button
              variant="filled"
              onClick={handleTransfer}
              aria-label={t('databases.transfer.submit')}
              loading={isTransferring}
              disabled={!isFormValid || isTransferring}
              aria-busy={isTransferring}
            >
              {t('databases.transfer.submit')}
            </Button>
          </div>
        </div>
      )}

      {isShowCreateStorage && selectedWorkspaceId && (
        <Dialog
          title={t('databases.transfer.storage.addTitle')}
          opened={isShowCreateStorage}
          onClose={() => {
            setIsShowCreateStorage(false);
          }}
          closeOnClickOutside={false}
        >
          <div className="my-3 max-w-68.75 text-muted">
            {t('databases.transfer.storage.description')}
          </div>

          <EditStorageComponent
            workspaceId={selectedWorkspaceId}
            isShowName
            isShowClose={false}
            onClose={() => setIsShowCreateStorage(false)}
            onPersisted={acceptTargetStorage}
            onChanged={(storage) => {
              if (!isCurrent() || storage.workspaceId !== targetWorkspace.current) return;
              setSelectedStorageId(storage.id);
              setIsShowCreateStorage(false);
            }}
          />
        </Dialog>
      )}

      {isShowCreateNotifier && selectedWorkspaceId && (
        <Dialog
          title={t('databases.notifiers.addTitle')}
          opened={isShowCreateNotifier}
          onClose={() => {
            setIsShowCreateNotifier(false);
          }}
          closeOnClickOutside={false}
        >
          <div className="my-3 max-w-68.75 text-muted">{t('databases.notifiers.description')}</div>

          <EditNotifierComponent
            workspaceId={selectedWorkspaceId}
            isShowName
            isShowClose={false}
            onClose={() => setIsShowCreateNotifier(false)}
            onPersisted={acceptTargetNotifier}
            onChanged={(notifier) => {
              if (!isCurrent() || notifier.workspaceId !== targetWorkspace.current) return;
              setSelectedNotifierIds([...selectedNotifierIds, notifier.id]);
              setIsShowCreateNotifier(false);
            }}
          />
        </Dialog>
      )}
    </Dialog>
  );
};
