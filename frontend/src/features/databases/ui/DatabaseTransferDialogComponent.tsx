import { CheckCircleOutlined, ExclamationCircleOutlined, SwapOutlined } from '@ant-design/icons';
import { Button, Modal, Radio, Select, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type DatabaseTransfer, backupConfigApi } from '../../../entity/backups/config';
import { type Database, databaseApi } from '../../../entity/databases';
import type { Notifier } from '../../../entity/notifiers';
import { notifierApi } from '../../../entity/notifiers';
import { type Storage, getStorageLogoFromType, storageApi } from '../../../entity/storages';
import { type Workspace, workspaceApi } from '../../../entity/workspaces';
import { translateApiError } from '../../../shared/i18n';
import { ToastHelper } from '../../../shared/toast';
import { EditNotifierComponent } from '../../notifiers/ui/edit/EditNotifierComponent';
import { EditStorageComponent } from '../../storages/ui/edit/EditStorageComponent';

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
  const [isLoading, setIsLoading] = useState(true);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string | undefined>();

  const [storageOption, setStorageOption] = useState<'transfer' | 'select'>('select');
  const [isStorageSharedWithOtherDatabases, setIsStorageSharedWithOtherDatabases] = useState(false);
  const [isLoadingStorageSharing, setIsLoadingStorageSharing] = useState(false);
  const [targetStorages, setTargetStorages] = useState<Storage[]>([]);
  const [selectedStorageId, setSelectedStorageId] = useState<string | undefined>();
  const [isShowCreateStorage, setIsShowCreateStorage] = useState(false);
  const [storageSelectKey, setStorageSelectKey] = useState(0);

  const [notifierOption, setNotifierOption] = useState<'transfer' | 'select'>('select');
  const [notifierUsageInfo, setNotifierUsageInfo] = useState<NotifierUsageInfo[]>([]);
  const [isLoadingNotifierCounts, setIsLoadingNotifierCounts] = useState(false);
  const [targetNotifiers, setTargetNotifiers] = useState<Notifier[]>([]);
  const [selectedNotifierIds, setSelectedNotifierIds] = useState<string[]>([]);
  const [isShowCreateNotifier, setIsShowCreateNotifier] = useState(false);
  const [notifierSelectKey, setNotifierSelectKey] = useState(0);

  const [isTransferring, setIsTransferring] = useState(false);

  const hasCurrentStorage = !!currentStorageId;
  const hasCurrentNotifiers = database.notifierIds.length > 0;

  const loadWorkspaces = async () => {
    setIsLoading(true);

    try {
      const listedWorkspaces = await workspaceApi.listAllWorkspaces();
      const filteredWorkspaces = listedWorkspaces.filter((w) => w.id !== database.workspaceId);
      setWorkspaces(filteredWorkspaces);
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsLoading(false);
  };

  const loadStorageSharing = async () => {
    if (!currentStorageId) return;

    setIsLoadingStorageSharing(true);

    try {
      const storageBackupConfigs = await backupConfigApi.listStorageBackupConfigs(
        currentStorageId,
        STORAGE_SHARING_CHECK_LIMIT,
      );
      setIsStorageSharedWithOtherDatabases(
        storageBackupConfigs.some((backupConfig) => backupConfig.databaseId !== database.id),
      );
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsLoadingStorageSharing(false);
  };

  const loadNotifierUsageCounts = async () => {
    if (database.notifierIds.length === 0) return;

    setIsLoadingNotifierCounts(true);

    try {
      const workspaceNotifiers = await notifierApi.listAllNotifiers(database.workspaceId);
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

      setNotifierUsageInfo(usageInfo);
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsLoadingNotifierCounts(false);
  };

  const loadTargetWorkspaceData = async (workspaceId: string) => {
    try {
      const [storages, notifiers] = await Promise.all([
        storageApi.listAllStorages(workspaceId),
        notifierApi.listAllNotifiers(workspaceId),
      ]);
      setTargetStorages(storages);
      setTargetNotifiers(notifiers);
    } catch (e) {
      alert(translateApiError(e, t));
    }
  };

  const handleTransfer = async () => {
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
      ToastHelper.showToast({
        title: t('databases.transfer.transferred.title'),
        description: t('databases.transfer.transferred.description', {
          name: database.name,
        }),
      });
      onTransferred();
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsTransferring(false);
  };

  useEffect(() => {
    loadWorkspaces();
    loadStorageSharing();
    loadNotifierUsageCounts();
  }, [database.id]);

  useEffect(() => {
    if (selectedWorkspaceId) {
      loadTargetWorkspaceData(selectedWorkspaceId);
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
      width={550}
    >
      {isLoading ? (
        <div className="flex justify-center py-5">
          <Spin />
        </div>
      ) : (
        <div className="py-3">
          {/* Workspace Selection */}
          <div className="mb-5">
            <div className="mb-2 font-medium">{t('databases.transfer.targetWorkspace')}</div>
            <Select
              value={selectedWorkspaceId}
              onChange={setSelectedWorkspaceId}
              className="w-full"
              placeholder={t('databases.transfer.selectWorkspace')}
              options={workspaces.map((w) => ({ label: w.name, value: w.id }))}
            />
          </div>

          {selectedWorkspaceId && (
            <>
              {/* Storage Transfer Options */}
              <div className="mb-5">
                <div className="mb-2 font-medium">{t('databases.transfer.storage.title')}</div>
                <Radio.Group
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
                    <Radio value="select">{t('databases.transfer.storage.selectFromTarget')}</Radio>
                  </div>
                </Radio.Group>

                {storageOption === 'transfer' && hasCurrentStorage && (
                  <div className="mt-2 ml-6">
                    {isLoadingStorageSharing ? (
                      <Spin size="small" />
                    ) : !canTransferWithStorage ? (
                      <div className="flex items-center gap-2 rounded border border-red-300 bg-red-50 p-2 text-sm text-red-600 dark:border-red-600 dark:bg-red-900/20">
                        <ExclamationCircleOutlined />
                        <span>{t('databases.transfer.storage.blocked')}</span>
                      </div>
                    ) : (
                      <div className="flex items-center gap-2 text-sm text-green-600 dark:text-green-400">
                        <CheckCircleOutlined />
                        {t('databases.transfer.storage.canTransfer')}
                      </div>
                    )}
                  </div>
                )}

                {storageOption === 'select' && (
                  <div className="mt-2 ml-6">
                    <div className="flex items-center gap-2">
                      <Select
                        key={storageSelectKey}
                        value={selectedStorageId}
                        onChange={(storageId) => {
                          if (storageId === 'create-new-storage') {
                            setIsShowCreateStorage(true);
                            return;
                          }
                          setSelectedStorageId(storageId);
                        }}
                        className="w-full max-w-[300px]"
                        placeholder={t('databases.transfer.storage.selectPlaceholder')}
                        options={[
                          ...targetStorages.map((s) => ({
                            label: s.name,
                            value: s.id,
                          })),
                          {
                            label: t('databases.transfer.storage.createNew'),
                            value: 'create-new-storage',
                          },
                        ]}
                      />

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

              {/* Notifier transfer options */}
              {hasCurrentNotifiers && (
                <div className="mb-5">
                  <div className="mb-2 font-medium">{t('databases.transfer.notifiers.title')}</div>

                  <Radio.Group
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

                  {notifierOption === 'transfer' && (
                    <div className="mt-2 ml-6">
                      {isLoadingNotifierCounts ? (
                        <Spin size="small" />
                      ) : (
                        <div className="space-y-1">
                          {notifiersCanTransfer.length > 0 && (
                            <div className="text-sm text-green-600 dark:text-green-400">
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
                            <div className="mt-2 rounded border border-red-300 bg-red-50 p-2 text-sm text-red-600 dark:border-red-600 dark:bg-red-900/20 dark:text-red-600">
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
                              <div className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                                {t('databases.transfer.notifiers.noneTransferred')}
                              </div>
                            )}
                        </div>
                      )}
                    </div>
                  )}

                  {notifierOption === 'select' && (
                    <div className="mt-2 ml-6">
                      <Select
                        key={notifierSelectKey}
                        mode="multiple"
                        value={selectedNotifierIds}
                        onChange={(notifierIds) => {
                          if (notifierIds.includes('create-new-notifier')) {
                            setIsShowCreateNotifier(true);
                            return;
                          }
                          setSelectedNotifierIds(notifierIds);
                        }}
                        className="w-full max-w-[300px]"
                        placeholder={t('databases.transfer.notifiers.selectPlaceholder')}
                        options={[
                          ...targetNotifiers.map((n) => ({
                            label: n.name,
                            value: n.id,
                          })),
                          {
                            label: t('databases.notifiers.createNew'),
                            value: 'create-new-notifier',
                          },
                        ]}
                      />
                    </div>
                  )}
                </div>
              )}
            </>
          )}

          {/* Action Buttons */}
          <div className="mt-5 flex gap-2">
            <Button type="default" onClick={onClose}>
              {t('common.actions.cancel')}
            </Button>

            <Button
              type="primary"
              onClick={handleTransfer}
              loading={isTransferring}
              disabled={!isFormValid || isTransferring}
            >
              {t('databases.transfer.submit')}
            </Button>
          </div>
        </div>
      )}

      {/* Create Storage Modal */}
      {isShowCreateStorage && selectedWorkspaceId && (
        <Modal
          title={t('databases.transfer.storage.addTitle')}
          footer={null}
          open={isShowCreateStorage}
          onCancel={() => {
            setIsShowCreateStorage(false);
            setStorageSelectKey((prev) => prev + 1);
          }}
          maskClosable={false}
        >
          <div className="my-3 max-w-[275px] text-gray-500 dark:text-gray-400">
            {t('databases.transfer.storage.description')}
          </div>

          <EditStorageComponent
            workspaceId={selectedWorkspaceId}
            isShowName
            isShowClose={false}
            onClose={() => setIsShowCreateStorage(false)}
            onChanged={(storage) => {
              loadTargetWorkspaceData(selectedWorkspaceId);
              setSelectedStorageId(storage.id);
              setIsShowCreateStorage(false);
            }}
          />
        </Modal>
      )}

      {/* Create Notifier Modal */}
      {isShowCreateNotifier && selectedWorkspaceId && (
        <Modal
          title={t('databases.notifiers.addTitle')}
          footer={null}
          open={isShowCreateNotifier}
          onCancel={() => {
            setIsShowCreateNotifier(false);
            setNotifierSelectKey((prev) => prev + 1);
          }}
          maskClosable={false}
        >
          <div className="my-3 max-w-[275px] text-gray-500 dark:text-gray-400">
            {t('databases.notifiers.description')}
          </div>

          <EditNotifierComponent
            workspaceId={selectedWorkspaceId}
            isShowName
            isShowClose={false}
            onClose={() => setIsShowCreateNotifier(false)}
            onChanged={(notifier) => {
              loadTargetWorkspaceData(selectedWorkspaceId);
              setSelectedNotifierIds([...selectedNotifierIds, notifier.id]);
              setIsShowCreateNotifier(false);
            }}
          />
        </Modal>
      )}
    </Modal>
  );
};
