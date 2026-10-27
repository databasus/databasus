import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  CloudUploadOutlined,
  DeleteOutlined,
  DownloadOutlined,
  ExclamationCircleOutlined,
  FilterFilled,
  FilterOutlined,
  InfoCircleOutlined,
  LockOutlined,
  SafetyOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { App, Button, Modal, Spin, Table, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import type { TFunction } from 'i18next';
import { type JSX, useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { type BackupConfig, backupConfigApi } from '../../../../entity/backups/config';
import {
  type BackupsFilters,
  LOGICAL_BACKUP_STATUS_LABEL_KEYS,
  type LogicalBackup,
  LogicalBackupStatus,
  RESTORE_VERIFICATION_STATUS_LABEL_KEYS,
  RestoreVerificationStatus,
  logicalBackupsApi,
} from '../../../../entity/backups/logical';
import { BackupEncryption } from '../../../../entity/backups/shared';
import { type Database, DatabaseType } from '../../../../entity/databases';
import { verificationRunsApi } from '../../../../entity/verification/runs';
import { usePersistentState, usePolledPagedList } from '../../../../shared/hooks';
import {
  type TranslationKey,
  getWebsitePageUrl,
  translateApiError,
  useLocale,
} from '../../../../shared/i18n';
import { formatDuration } from '../../../../shared/lib';
import { getUserTimeFormat } from '../../../../shared/time';
import { ConfirmationComponent } from '../../../../shared/ui';
import { RestoresComponent } from '../../../restores';
import { RESTORE_VERIFICATION_STATUS_BADGE_STYLES } from '../model/restoreVerificationStatus';
import { LogicalBackupsFiltersPanelComponent } from './LogicalBackupsFiltersPanelComponent';

const BACKUPS_PAGE_SIZE = 50;
const BACKUPS_REFRESH_MS = 1_000;

// Physical databases have their own backups list; the generic text keeps the map exhaustive.
const DOWNLOAD_TOOLTIP_KEYS: Record<DatabaseType, TranslationKey> = {
  [DatabaseType.POSTGRES_LOGICAL]: 'backups.logical.list.actions.download.postgresql',
  [DatabaseType.POSTGRES_PHYSICAL]: 'backups.logical.list.actions.download.other',
  [DatabaseType.MYSQL]: 'backups.logical.list.actions.download.mysql',
  [DatabaseType.MARIADB]: 'backups.logical.list.actions.download.mariadb',
  [DatabaseType.MONGODB]: 'backups.logical.list.actions.download.mongodb',
};

interface Props {
  database: Database;
  isCanManageDBs: boolean;
  isDirectlyUnderTab?: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
  onNavigateToVerifications?: () => void;
}

const renderRestoreVerificationTag = (
  status: RestoreVerificationStatus | undefined,
  t: TFunction,
): JSX.Element | null => {
  if (!status || status === RestoreVerificationStatus.NOT_VERIFIED) {
    return null;
  }

  const badgeStyle = RESTORE_VERIFICATION_STATUS_BADGE_STYLES[status];
  const labelKey = RESTORE_VERIFICATION_STATUS_LABEL_KEYS[status];
  if (!badgeStyle || !labelKey) {
    return null;
  }

  return (
    <span
      className={`ml-3 inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ring-1 ring-inset ${badgeStyle.pillClasses}`}
    >
      <span className={`h-1.5 w-1.5 rounded-full ${badgeStyle.dotClasses}`} />
      {t(labelKey)}
    </span>
  );
};

export const LogicalBackupsComponent = ({
  database,
  isCanManageDBs,
  isDirectlyUnderTab,
  scrollContainerRef,
  onNavigateToVerifications,
}: Props) => {
  const { t } = useTranslation();
  const { locale, formatNumber, formatRelativeTime } = useLocale();
  const { message, modal } = App.useApp();
  const [backupConfig, setBackupConfig] = useState<BackupConfig | undefined>();
  const [isBackupConfigLoading, setIsBackupConfigLoading] = useState(false);

  const [isMakeBackupRequestLoading, setIsMakeBackupRequestLoading] = useState(false);

  const [showingBackupError, setShowingBackupError] = useState<LogicalBackup | undefined>();

  const [deleteConfimationId, setDeleteConfimationId] = useState<string | undefined>();
  const [deletingBackupId, setDeletingBackupId] = useState<string | undefined>();

  const [showingRestoresBackupId, setShowingRestoresBackupId] = useState<string | undefined>();

  const [downloadingBackupId, setDownloadingBackupId] = useState<string | undefined>();
  const [cancellingBackupId, setCancellingBackupId] = useState<string | undefined>();
  const [verifyingBackupId, setVerifyingBackupId] = useState<string | undefined>();

  const [isFilterPanelVisible, setIsFilterPanelVisible] = useState(false);
  // eslint-disable-next-line i18next/no-literal-string -- localStorage key
  const [filters, setFilters] = usePersistentState<BackupsFilters>('logicalBackupsFilters', {});

  const downloadBackup = async (backupId: string) => {
    try {
      await logicalBackupsApi.downloadBackup(backupId);
    } catch (e) {
      alert(translateApiError(e, t));
    } finally {
      setDownloadingBackupId(undefined);
    }
  };

  const fetchBackupsPage = async (offset: number | undefined) => {
    const pageOffset = offset ?? 0;
    const response = await logicalBackupsApi.getBackups(
      database.id,
      BACKUPS_PAGE_SIZE,
      pageOffset,
      filters,
    );

    const loadedCount = pageOffset + response.backups.length;

    return {
      items: response.backups,
      nextCursor: loadedCount < response.total ? loadedCount : undefined,
    };
  };

  const makeBackup = async () => {
    setIsMakeBackupRequestLoading(true);

    try {
      await logicalBackupsApi.makeBackup(database.id);
      await new Promise((resolve) => setTimeout(resolve, 1000));
      await reloadBackups();
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsMakeBackupRequestLoading(false);
  };

  const deleteBackup = async () => {
    if (!deleteConfimationId) {
      return;
    }

    setDeleteConfimationId(undefined);
    setDeletingBackupId(deleteConfimationId);

    try {
      await logicalBackupsApi.deleteBackup(deleteConfimationId);
      await reloadBackups();
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setDeletingBackupId(undefined);
    setDeleteConfimationId(undefined);
  };

  const cancelBackup = async (backupId: string) => {
    setCancellingBackupId(backupId);

    try {
      await logicalBackupsApi.cancelBackup(backupId);
      await refreshBackups();
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setCancellingBackupId(undefined);
  };

  const enqueueVerifyRestore = async (backupId: string) => {
    setVerifyingBackupId(backupId);

    try {
      await verificationRunsApi.enqueue(backupId);
      message.success(t('backups.logical.list.verifyRestore.queued'));
      onNavigateToVerifications?.();
    } catch (e) {
      message.error(translateApiError(e, t));
    } finally {
      setVerifyingBackupId(undefined);
    }
  };

  const confirmVerifyRestore = (backupId: string) => {
    modal.confirm({
      title: t('backups.logical.list.verifyRestore.title'),
      icon: <InfoCircleOutlined className="!text-blue-600" />,
      content: (
        <span>
          <Trans
            i18nKey="backups.logical.list.verifyRestore.description"
            components={{
              docsLink: (
                <a
                  href={getWebsitePageUrl('restoreVerification', locale)}
                  target="_blank"
                  rel="noopener noreferrer"
                />
              ),
            }}
          />
        </span>
      ),
      okText: t('backups.logical.list.verifyRestore.queue'),
      okType: 'primary',
      okButtonProps: { type: 'primary', danger: false },
      cancelText: t('common.actions.cancel'),
      onOk: () => enqueueVerifyRestore(backupId),
    });
  };

  const {
    items: backups,
    isLoading: isBackupsLoading,
    isLoadingMore,
    hasMore,
    reload: reloadBackups,
    refreshFirstPage: refreshBackups,
  } = usePolledPagedList({
    fetchPage: fetchBackupsPage,
    getItemKey: (backup: LogicalBackup) => backup.id,
    refreshIntervalMs: BACKUPS_REFRESH_MS,
    scrollContainerRef,
    resetKey: JSON.stringify([database.id, filters]),
    onError: (e) => alert(translateApiError(e, t)),
  });

  useEffect(() => {
    setIsBackupConfigLoading(true);

    backupConfigApi.getBackupConfig(database.id).then((loadedBackupConfig) => {
      setBackupConfig(loadedBackupConfig);
      setIsBackupConfigLoading(false);
    });
  }, [database]);

  useEffect(() => {
    if (downloadingBackupId) {
      downloadBackup(downloadingBackupId);
    }
  }, [downloadingBackupId]);

  const renderBackupStatusLabel = (status: LogicalBackupStatus, record: LogicalBackup) => {
    if (status === LogicalBackupStatus.FAILED) {
      return (
        <Tooltip title={t('backups.list.clickToSeeErrorDetails')}>
          <div
            className="flex cursor-pointer items-center text-red-600 underline"
            onClick={() => setShowingBackupError(record)}
          >
            <ExclamationCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
            <div>{t(LOGICAL_BACKUP_STATUS_LABEL_KEYS[status])}</div>
          </div>
        </Tooltip>
      );
    }

    if (status === LogicalBackupStatus.COMPLETED) {
      return (
        <div className="flex items-center text-green-600">
          <CheckCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
          <div>{t(LOGICAL_BACKUP_STATUS_LABEL_KEYS[status])}</div>
          {record.encryption === BackupEncryption.ENCRYPTED && (
            <Tooltip title={t('backups.logical.list.encrypted')}>
              <LockOutlined className="ml-1" style={{ fontSize: 14 }} />
            </Tooltip>
          )}
        </div>
      );
    }

    if (status === LogicalBackupStatus.DELETED) {
      return (
        <div className="flex items-center text-gray-600">
          <DeleteOutlined className="mr-2" style={{ fontSize: 16 }} />
          <div>{t(LOGICAL_BACKUP_STATUS_LABEL_KEYS[status])}</div>
        </div>
      );
    }

    if (status === LogicalBackupStatus.IN_PROGRESS) {
      return (
        <div className="flex items-center font-bold text-blue-600">
          <SyncOutlined spin />
          <span className="ml-2">{t(LOGICAL_BACKUP_STATUS_LABEL_KEYS[status])}</span>
        </div>
      );
    }

    if (status === LogicalBackupStatus.CANCELED) {
      return (
        <div className="flex items-center text-gray-600">
          <CloseCircleOutlined className="mr-2" style={{ fontSize: 16 }} />
          <div>{t(LOGICAL_BACKUP_STATUS_LABEL_KEYS[status])}</div>
        </div>
      );
    }

    return <span className="font-bold">{status}</span>;
  };

  const renderStatus = (status: LogicalBackupStatus, record: LogicalBackup) => {
    const verificationTag = renderRestoreVerificationTag(record.restoreVerificationStatus, t);
    if (!verificationTag) {
      return renderBackupStatusLabel(status, record);
    }

    return (
      <div className="flex items-center">
        {renderBackupStatusLabel(status, record)}
        {verificationTag}
      </div>
    );
  };

  const renderActions = (record: LogicalBackup) => {
    return (
      <div className="flex gap-2 text-lg">
        {record.status === LogicalBackupStatus.IN_PROGRESS && isCanManageDBs && (
          <div className="flex gap-2">
            {cancellingBackupId === record.id ? (
              <SyncOutlined spin />
            ) : (
              <Tooltip title={t('backups.list.actions.cancelBackup')}>
                <CloseCircleOutlined
                  className="cursor-pointer"
                  onClick={() => {
                    if (cancellingBackupId) return;
                    cancelBackup(record.id);
                  }}
                  style={{ color: '#ff0000', opacity: cancellingBackupId ? 0.2 : 1 }}
                />
              </Tooltip>
            )}
          </div>
        )}

        {record.status === LogicalBackupStatus.COMPLETED && (
          <div className="flex gap-2">
            {deletingBackupId === record.id ? (
              <SyncOutlined spin />
            ) : (
              <>
                {isCanManageDBs && (
                  <Tooltip title={t('backups.list.actions.deleteBackup')}>
                    <DeleteOutlined
                      className="cursor-pointer"
                      onClick={() => {
                        if (deletingBackupId) return;
                        setDeleteConfimationId(record.id);
                      }}
                      style={{ color: '#ff0000', opacity: deletingBackupId ? 0.2 : 1 }}
                    />
                  </Tooltip>
                )}

                {isCanManageDBs &&
                  database.settings.type === DatabaseType.POSTGRES_LOGICAL &&
                  (verifyingBackupId === record.id ? (
                    <SyncOutlined spin style={{ color: '#155dfc' }} />
                  ) : (
                    <Tooltip title={t('backups.logical.list.actions.verifyRestore')}>
                      <SafetyOutlined
                        className="cursor-pointer"
                        onClick={() => {
                          if (verifyingBackupId) return;
                          confirmVerifyRestore(record.id);
                        }}
                        style={{
                          color: '#155dfc',
                          opacity: verifyingBackupId ? 0.2 : 1,
                        }}
                      />
                    </Tooltip>
                  ))}

                <Tooltip title={t('backups.logical.list.actions.restore')}>
                  <CloudUploadOutlined
                    className="cursor-pointer"
                    onClick={() => {
                      setShowingRestoresBackupId(record.id);
                    }}
                    style={{
                      color: '#155dfc',
                    }}
                  />
                </Tooltip>

                <Tooltip title={t(DOWNLOAD_TOOLTIP_KEYS[database.settings.type])}>
                  {downloadingBackupId === record.id ? (
                    <SyncOutlined spin style={{ color: '#155dfc' }} />
                  ) : (
                    <DownloadOutlined
                      className="cursor-pointer"
                      onClick={() => {
                        if (downloadingBackupId) return;
                        setDownloadingBackupId(record.id);
                      }}
                      style={{
                        opacity: downloadingBackupId ? 0.2 : 1,
                        color: '#155dfc',
                      }}
                    />
                  )}
                </Tooltip>
              </>
            )}
          </div>
        )}
      </div>
    );
  };

  const formatSize = (sizeMb: number) => {
    if (sizeMb >= 1024) {
      const sizeGb = sizeMb / 1024;
      return `${formatNumber(Number(sizeGb.toFixed(2)))} GB`;
    }
    return `${formatNumber(Number(sizeMb?.toFixed(2)))} MB`;
  };

  const columns: ColumnsType<LogicalBackup> = [
    {
      title: t('backups.list.columns.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (createdAt: string) => (
        <div>
          {dayjs.utc(createdAt).local().format(getUserTimeFormat().format)} <br />
          <span className="text-gray-500 dark:text-gray-400">
            ({formatRelativeTime(dayjs.utc(createdAt).local())})
          </span>
        </div>
      ),
      sorter: (a, b) => dayjs(a.createdAt).unix() - dayjs(b.createdAt).unix(),
      defaultSortOrder: 'descend',
    },
    {
      title: t('common.fields.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status: LogicalBackupStatus, record: LogicalBackup) => renderStatus(status, record),
    },
    {
      title: (
        <div className="flex items-center">
          {t('backups.list.columns.size')}
          <Tooltip className="ml-1" title={t('backups.logical.list.sizeTooltip')}>
            <InfoCircleOutlined />
          </Tooltip>
        </div>
      ),
      dataIndex: 'backupSizeMb',
      key: 'backupSizeMb',
      width: 150,
      render: (sizeMb: number, record: LogicalBackup) => (
        <div>
          <div>{formatSize(sizeMb)}</div>
          {record.backupRawDbSizeMb > 0 && (
            <div className="text-xs text-gray-500 dark:text-gray-500">
              {t('backups.logical.list.databaseSize', {
                size: formatSize(record.backupRawDbSizeMb),
              })}
            </div>
          )}
        </div>
      ),
    },
    {
      title: t('backups.list.columns.duration'),
      dataIndex: 'backupDurationMs',
      key: 'backupDurationMs',
      width: 150,
      render: (durationMs: number) => formatDuration(durationMs),
    },
    {
      title: t('backups.list.columns.actions'),
      dataIndex: '',
      key: '',
      render: (_, record: LogicalBackup) => renderActions(record),
    },
  ];

  const isAnyFilterApplied =
    (filters.statuses && filters.statuses.length > 0) || filters.beforeDate !== undefined;

  if (isBackupConfigLoading) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );
  }

  return (
    <div
      className={`w-full bg-white p-3 shadow md:p-5 dark:bg-gray-800 ${isDirectlyUnderTab ? 'rounded-tr-md rounded-br-md rounded-bl-md' : 'rounded-md'}`}
    >
      <div className="flex items-center gap-2">
        <h2 className="text-lg font-bold md:text-xl dark:text-white">{t('backups.list.title')}</h2>
        <div className="relative">
          {isFilterPanelVisible ? (
            <FilterFilled
              className="cursor-pointer text-blue-600"
              onClick={() => setIsFilterPanelVisible(false)}
            />
          ) : (
            <FilterOutlined
              className="cursor-pointer"
              onClick={() => setIsFilterPanelVisible(true)}
            />
          )}
          {!isFilterPanelVisible && isAnyFilterApplied && (
            <span className="absolute -top-1 -right-1 h-2 w-2 rounded-full bg-blue-600" />
          )}
        </div>
      </div>

      {isFilterPanelVisible && (
        <div className="mt-3">
          <LogicalBackupsFiltersPanelComponent filters={filters} onFiltersChange={setFilters} />
        </div>
      )}

      {!isBackupConfigLoading && !backupConfig?.isBackupsEnabled && (
        <div className="text-sm text-red-600">{t('backups.list.scheduledBackupsDisabled')}</div>
      )}

      <div className="mt-5" />

      <div className="flex items-center">
        <Button
          onClick={makeBackup}
          className="mr-1"
          type="primary"
          disabled={isMakeBackupRequestLoading}
          loading={isMakeBackupRequestLoading}
        >
          <span className="md:hidden">{t('backups.logical.list.backupNow')}</span>
          <span className="hidden md:inline">{t('backups.logical.list.makeBackupNow')}</span>
        </Button>
      </div>

      <div className="mt-5 w-full md:max-w-[850px]">
        {/* Mobile card view */}
        <div className="md:hidden">
          {isBackupsLoading ? (
            <div className="flex justify-center py-8">
              <Spin />
            </div>
          ) : (
            <div>
              {backups.map((backup) => (
                <div
                  key={backup.id}
                  className="mb-2 rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800"
                >
                  <div className="space-y-3">
                    <div className="flex items-start justify-between">
                      <div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {t('backups.list.columns.createdAt')}
                        </div>
                        <div className="text-sm font-medium">
                          {dayjs.utc(backup.createdAt).local().format(getUserTimeFormat().format)}
                        </div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          ({formatRelativeTime(dayjs.utc(backup.createdAt).local())})
                        </div>
                      </div>
                      <div>{renderStatus(backup.status, backup)}</div>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {t('backups.list.columns.size')}
                        </div>
                        <div className="text-sm font-medium">{formatSize(backup.backupSizeMb)}</div>
                        {backup.backupRawDbSizeMb > 0 && (
                          <div className="text-xs text-gray-500 dark:text-gray-500">
                            {t('backups.logical.list.databaseSize', {
                              size: formatSize(backup.backupRawDbSizeMb),
                            })}
                          </div>
                        )}
                      </div>
                      <div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {t('backups.list.columns.duration')}
                        </div>
                        <div className="text-sm font-medium">
                          {formatDuration(backup.backupDurationMs)}
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center justify-end border-t border-gray-200 pt-3">
                      {renderActions(backup)}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {isLoadingMore && (
            <div className="mt-3 flex justify-center">
              <Spin />
            </div>
          )}
          {!hasMore && backups.length > 0 && (
            <div className="mt-3 text-center text-sm text-gray-500 dark:text-gray-400">
              {t('backups.list.allLoaded', { count: backups.length })}
            </div>
          )}
          {!isBackupsLoading && backups.length === 0 && (
            <div className="py-8 text-center text-gray-500 dark:text-gray-400">
              {t('backups.list.empty')}
            </div>
          )}
        </div>

        {/* Desktop table view */}
        <div className="hidden md:block">
          <Table
            bordered
            columns={columns}
            dataSource={backups}
            rowKey="id"
            loading={isBackupsLoading}
            size="small"
            pagination={false}
          />
          {isLoadingMore && (
            <div className="mt-2 flex justify-center">
              <Spin />
            </div>
          )}
          {!hasMore && backups.length > 0 && (
            <div className="mt-2 text-center text-gray-500 dark:text-gray-400">
              {t('backups.list.allLoaded', { count: backups.length })}
            </div>
          )}
        </div>
      </div>

      {deleteConfimationId && (
        <ConfirmationComponent
          onConfirm={deleteBackup}
          onDecline={() => setDeleteConfimationId(undefined)}
          description={t('backups.logical.list.deleteConfirmation')}
          actionButtonColor="red"
          actionText={t('common.actions.delete')}
        />
      )}

      {showingRestoresBackupId && (
        <Modal
          width={400}
          open={!!showingRestoresBackupId}
          onCancel={() => setShowingRestoresBackupId(undefined)}
          title={t('backups.list.restoreTitle')}
          footer={null}
          maskClosable={false}
        >
          <RestoresComponent
            database={database}
            backup={backups.find((b) => b.id === showingRestoresBackupId) as LogicalBackup}
          />
        </Modal>
      )}

      {showingBackupError && (
        <Modal
          title={t('backups.list.errorDetailsTitle')}
          open={!!showingBackupError}
          onCancel={() => setShowingBackupError(undefined)}
          maskClosable={false}
          footer={null}
        >
          <div className="text-sm">{showingBackupError.failMessage}</div>
        </Modal>
      )}
    </div>
  );
};
