import { CheckCircleOutlined, ExclamationCircleOutlined, SwapOutlined } from '@ant-design/icons';
import { Alert, App, Button, Modal, Radio, Select, Spin } from 'antd';
import { useEffect, useEffectEvent, useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type DatabaseTransfer, backupConfigApi } from '@/entities/backup-configs';
import { type Database, databaseApi } from '@/entities/databases';
import type { Notifier } from '@/entities/notifiers';
import { notifierApi } from '@/entities/notifiers';
import { type Storage, getStorageLogoFromType, storageApi } from '@/entities/storages';
import { type Workspace, workspaceApi } from '@/entities/workspaces';
import { EditNotifierComponent } from '@/features/edit-notifier';
import { EditStorageComponent } from '@/features/edit-storage';
import { useSessionGuard } from '@/shared/auth';
import { useLatestRef, useRequestRevision } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';

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
  const { t } = useTranslation();
  const storageOptionLabelId = useId();
  const notifierOptionLabelId = useId();
  const [error, setError] = useState<unknown>();
  const [isLoading, setIsLoading] = useState(true);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string | undefined>();

  const [storageOption, setStorageOption] = useState<'transfer' | 'select'>('select');
  const [isStorageSharedWithOtherDatabases, setIsStorageSharedWithOtherDatabases] = useState(false);
  const [isLoadingStorageSharing, setIsLoadingStorageSharing] = useState(false);
  const [targetStorages, setTargetStorages] = useState<Storage[]>([]);
  const [selectedStorageId, setSelectedStorageId] = useState<string | undefined>();
  const [isShowCreateStorage, setIsShowCreateStorage] = useState(false);

  const [notifierOption, setNotifierOption] = useState<'transfer' | 'select'>('select');
  const [notifierUsageInfo, setNotifierUsageInfo] = useState<NotifierUsageInfo[]>([]);
  const [isLoadingNotifierCounts, setIsLoadingNotifierCounts] = useState(false);
  const [targetNotifiers, setTargetNotifiers] = useState<Notifier[]>([]);
  const [selectedNotifierIds, setSelectedNotifierIds] = useState<string[]>([]);
  const [isShowCreateNotifier, setIsShowCreateNotifier] = useState(false);

  const [isTransferring, setIsTransferring] = useState(false);

  const { notification } = App.useApp();
  const isCurrent = useSessionGuard();
  const contextRevision = useRequestRevision();
  const targetRevision = useRequestRevision();
  const acknowledgedStorages = useRef<Storage[]>([]);
  const acknowledgedNotifiers = useRef<Notifier[]>([]);
  const targetWorkspace = useLatestRef(selectedWorkspaceId);

  const hasCurrentStorage = !!currentStorageId;
  const hasCurrentNotifiers = database.notifierIds.length > 0;

  const loadWorkspaces = async () => {
    const revision = contextRevision.GetCurrent();
    const isActive = () => isCurrent() && contextRevision.IsCurrent(revision);
    setIsLoading(true);

    try {
      const listedWorkspaces = await workspaceApi.listAllWorkspaces();
      if (!isActive()) return;

      const filteredWorkspaces = listedWorkspaces.filter(
        (workspace) => workspace.id !== database.workspaceId,
      );
      setWorkspaces(filteredWorkspaces);
    } catch (e) {
      if (!isActive()) return;

      setError(e);
    }

    setIsLoading(false);
  };

  const loadStorageSharing = async () => {
    const revision = contextRevision.GetCurrent();
    const isActive = () => isCurrent() && contextRevision.IsCurrent(revision);

    if (!currentStorageId) return;

    setIsLoadingStorageSharing(true);

    try {
      const storageBackupConfigs = await backupConfigApi.listStorageBackupConfigs(
        currentStorageId,
        STORAGE_SHARING_CHECK_LIMIT,
      );
      if (!isActive()) return;

      setIsStorageSharedWithOtherDatabases(
        storageBackupConfigs.some((backupConfig) => backupConfig.databaseId !== database.id),
      );
    } catch (e) {
      if (!isActive()) return;

      setError(e);
    }

    setIsLoadingStorageSharing(false);
  };

  const loadNotifierUsageCounts = async () => {
    const revision = contextRevision.GetCurrent();
    const isActive = () => isCurrent() && contextRevision.IsCurrent(revision);

    if (database.notifierIds.length === 0) return;

    setIsLoadingNotifierCounts(true);

    try {
      const workspaceNotifiers = await notifierApi.listAllNotifiers(database.workspaceId);
      if (!isActive()) return;

      const databaseNotifiers = workspaceNotifiers.filter((notifier) =>
        database.notifierIds.includes(notifier.id),
      );

      const usageInfo: NotifierUsageInfo[] = await Promise.all(
        databaseNotifiers.map(async (notifier) => {
          const notifierDatabases = await databaseApi.listAllDatabases(database.workspaceId, {
            notifierId: notifier.id,
          });
          const count = notifierDatabases.length;

          return {
            notifier,
            databaseCount: count,
            canTransfer: count === 1,
          };
        }),
      );

      if (!isActive()) return;

      setNotifierUsageInfo(usageInfo);
    } catch (e) {
      if (!isActive()) return;

      setError(e);
    }

    setIsLoadingNotifierCounts(false);
  };

  const acceptTargetStorage = (storage: Storage) => {
    if (!isCurrent() || storage.workspaceId !== targetWorkspace.current) return;

    acknowledgedStorages.current = [
      ...acknowledgedStorages.current.filter((item) => item.id !== storage.id),
      storage,
    ];
    setTargetStorages((current) => [...current.filter((item) => item.id !== storage.id), storage]);
    void loadTargetWorkspaceData(storage.workspaceId);
  };

  const acceptTargetNotifier = (notifier: Notifier) => {
    if (!isCurrent() || notifier.workspaceId !== targetWorkspace.current) return;

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

  const loadTargetWorkspaceData = async (workspaceId: string) => {
    const revision = targetRevision.Invalidate();
    const isActive = () =>
      isCurrent() && targetRevision.IsCurrent(revision) && targetWorkspace.current === workspaceId;

    try {
      const [storages, notifiers] = await Promise.all([
        storageApi.listAllStorages(workspaceId),
        notifierApi.listAllNotifiers(workspaceId),
      ]);
      if (!isActive()) return;

      setTargetStorages([
        ...storages.filter(
          (storage) => !acknowledgedStorages.current.some((item) => item.id === storage.id),
        ),
        ...acknowledgedStorages.current,
      ]);
      setTargetNotifiers([
        ...notifiers.filter(
          (notifier) => !acknowledgedNotifiers.current.some((item) => item.id === notifier.id),
        ),
        ...acknowledgedNotifiers.current,
      ]);
    } catch (e) {
      if (!isActive()) return;

      setError(e);
    }
  };

  const handleTransfer = async () => {
    if (!isCurrent() || isTransferring) return;

    const revision = contextRevision.GetCurrent();
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

    setIsTransferring(true);

    try {
      await backupConfigApi.transferDatabase(database.id, transfer);
      if (!isActive()) return;

      notification.success({
        message: t('databases.transfer.transferred.title'),
        description: t('databases.transfer.transferred.description', {
          name: database.name,
        }),
      });

      onTransferred();
    } catch (e) {
      if (!isActive()) return;

      setError(e);
    }

    setIsTransferring(false);
  };

  const startContextLoad = useEffectEvent(() => {
    loadWorkspaces();
    loadStorageSharing();
    loadNotifierUsageCounts();
  });

  useEffect(() => {
    contextRevision.Invalidate();
    startContextLoad();

    return () => {
      contextRevision.Invalidate();
      targetRevision.Invalidate();
    };
  }, [database.id, contextRevision, targetRevision]);

  const startTargetWorkspaceLoad = useEffectEvent((workspaceId: string) => {
    loadTargetWorkspaceData(workspaceId);
  });

  useEffect(() => {
    targetRevision.Invalidate();
    acknowledgedStorages.current = [];
    acknowledgedNotifiers.current = [];
    setIsShowCreateStorage(false);
    setIsShowCreateNotifier(false);
    setError(undefined);
    setTargetStorages([]);
    setTargetNotifiers([]);

    if (selectedWorkspaceId) {
      startTargetWorkspaceLoad(selectedWorkspaceId);
      setSelectedStorageId(undefined);
      setSelectedNotifierIds([]);
    }
  }, [selectedWorkspaceId, targetRevision]);

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
    <Modal
      title={
        <div className="flex items-center gap-2">
          <SwapOutlined />
          {t('databases.transfer.title')}
        </div>
      }
      footer={null}
      open={true}
      onCancel={onClose}
      maskClosable={false}
      closable={!isTransferring}
      keyboard={!isTransferring}
      width={550}
      className="[&_.ant-btn]:!h-auto [&_.ant-btn]:min-h-6 [&_.ant-btn]:max-w-full [&_.ant-btn]:leading-normal [&_.ant-btn]:!whitespace-normal [&_.ant-select-single]:!h-auto [&_.ant-select-single_.ant-select-selection-item]:!leading-normal [&_.ant-select-single_.ant-select-selection-item]:!whitespace-normal [&_.ant-select-single_.ant-select-selector]:!h-auto"
    >
      {error !== undefined && (
        <Alert
          type="error"
          showIcon
          message={translateApiError(error, t)}
          action={
            <Button
              disabled={isTransferring}
              onClick={() => {
                setError(undefined);
                void loadWorkspaces();
                void loadStorageSharing();
                void loadNotifierUsageCounts();

                if (selectedWorkspaceId) void loadTargetWorkspaceData(selectedWorkspaceId);
              }}
            >
              {t('common.actions.retry')}
            </Button>
          }
        />
      )}
      {isLoading ? (
        <div className="flex justify-center py-5">
          <Spin />
        </div>
      ) : (
        <div className="py-3">
          <div className="mb-5">
            <div className="mb-2 font-medium">{t('databases.transfer.targetWorkspace')}</div>
            <Select
              aria-label={t('databases.transfer.targetWorkspace')}
              disabled={isTransferring}
              value={selectedWorkspaceId}
              onChange={setSelectedWorkspaceId}
              className="w-full"
              placeholder={t('databases.transfer.selectWorkspace')}
              options={workspaces.map((workspace) => ({
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
                    onChange={(e) => setStorageOption(e.target.value)}
                    className="flex flex-col gap-3"
                  >
                    {hasCurrentStorage && (
                      <div>
                        <Radio value="transfer">
                          <span className="flex items-center gap-2">
                            {t('databases.transfer.storage.transferExisting')}
                            {isLoadingStorageSharing && <Spin size="small" />}
                          </span>
                        </Radio>
                      </div>
                    )}
                    <div>
                      <Radio value="select">
                        {t('databases.transfer.storage.selectFromTarget')}
                      </Radio>
                    </div>
                  </Radio.Group>
                </div>

                {storageOption === 'transfer' && hasCurrentStorage && (
                  <div className="mt-2 ml-6">
                    {isLoadingStorageSharing ? (
                      <Spin size="small" />
                    ) : !canTransferWithStorage ? (
                      <div className="flex items-center gap-2 rounded border border-border bg-surface p-2 text-sm text-danger">
                        <ExclamationCircleOutlined />
                        <span>{t('databases.transfer.storage.blocked')}</span>
                      </div>
                    ) : (
                      <div className="flex items-center gap-2 text-sm text-success">
                        <CheckCircleOutlined />
                        {t('databases.transfer.storage.canTransfer')}
                      </div>
                    )}
                  </div>
                )}

                {storageOption === 'select' && (
                  <div className="mt-2 ml-6">
                    <div className="flex min-w-0 flex-col items-start gap-2 sm:flex-row sm:items-center">
                      <Select
                        aria-label={t('databases.transfer.storage.title')}
                        disabled={isTransferring}
                        value={selectedStorageId}
                        onChange={(storageId) => {
                          setSelectedStorageId(storageId);
                        }}
                        className="w-full max-w-75 min-w-0 sm:flex-1"
                        placeholder={t('databases.transfer.storage.selectPlaceholder')}
                        options={[
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
                      onChange={(e) => setNotifierOption(e.target.value)}
                      className="flex flex-col gap-3"
                    >
                      <div>
                        <Radio value="transfer">
                          {t('databases.transfer.notifiers.transferWithDatabase')}
                        </Radio>
                      </div>
                      <div>
                        <Radio value="select">
                          {t('databases.transfer.notifiers.selectFromTarget')}
                        </Radio>
                      </div>
                    </Radio.Group>
                  </div>

                  {notifierOption === 'transfer' && (
                    <div className="mt-2 ml-6">
                      {isLoadingNotifierCounts ? (
                        <Spin size="small" />
                      ) : (
                        <div className="space-y-1">
                          {notifiersCanTransfer.length > 0 && (
                            <div className="text-sm text-success">
                              <div className="mb-1 flex items-center gap-1">
                                <CheckCircleOutlined />
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
                                <ExclamationCircleOutlined />
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
                      <Select
                        mode="multiple"
                        aria-label={t('databases.transfer.notifiers.title')}
                        disabled={isTransferring}
                        value={selectedNotifierIds}
                        onChange={(notifierIds) => {
                          setSelectedNotifierIds(notifierIds);
                        }}
                        className="w-full max-w-75 min-w-0 sm:flex-1 [&_.ant-select-selection-placeholder]:!relative [&_.ant-select-selection-placeholder]:!inset-x-0 [&_.ant-select-selection-placeholder]:!top-auto [&_.ant-select-selection-placeholder]:min-w-0 [&_.ant-select-selection-placeholder]:basis-full [&_.ant-select-selection-placeholder]:!transform-none [&_.ant-select-selection-placeholder]:!leading-normal [&_.ant-select-selection-placeholder]:!whitespace-normal [&_.ant-select-selection-wrap]:flex-wrap"
                        placeholder={t('databases.transfer.notifiers.selectPlaceholder')}
                        options={[
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
            <Alert type="warning" showIcon message={t('backups.storageSwitch.warning')} />
          )}
          <div className="mt-5 flex flex-wrap gap-2">
            <Button type="default" onClick={onClose} disabled={isTransferring}>
              {t('common.actions.cancel')}
            </Button>

            <Button
              type="primary"
              onClick={handleTransfer}
              aria-label={t('databases.transfer.submit')}
              loading={isTransferring}
              disabled={!isFormValid || isTransferring}
            >
              {t('databases.transfer.submit')}
            </Button>
          </div>
        </div>
      )}

      {isShowCreateStorage && selectedWorkspaceId && (
        <Modal
          title={t('databases.transfer.storage.addTitle')}
          footer={null}
          open={isShowCreateStorage}
          onCancel={() => {
            setIsShowCreateStorage(false);
          }}
          maskClosable={false}
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
        </Modal>
      )}

      {isShowCreateNotifier && selectedWorkspaceId && (
        <Modal
          title={t('databases.notifiers.addTitle')}
          footer={null}
          open={isShowCreateNotifier}
          onCancel={() => {
            setIsShowCreateNotifier(false);
          }}
          maskClosable={false}
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
        </Modal>
      )}
    </Modal>
  );
};
