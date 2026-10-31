import {
  CloseCircleOutlined,
  CloudUploadOutlined,
  DeleteOutlined,
  DownOutlined,
  FilterFilled,
  FilterOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { App, Button, Dropdown, type MenuProps, Modal, Spin, Table, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import type { TFunction } from 'i18next';
import { type JSX, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type BackupConfig, backupConfigApi } from '../../../../entities/backups/config';
import {
  PHYSICAL_BACKUP_STATUS_LABEL_KEYS,
  type PhysicalBackupListItem,
  PhysicalBackupStatus,
  PhysicalBackupType,
  type PhysicalBackupsFilters,
  type TriggerPhysicalBackupType,
  physicalBackupsApi,
} from '../../../../entities/backups/physical';
import {
  type Database,
  DatabaseType,
  PhysicalDatabaseBackupType,
} from '../../../../entities/databases';
import { usePersistentState, usePolledPagedList } from '../../../../shared/hooks';
import { type LocaleContextValue, translateApiError, useLocale } from '../../../../shared/i18n';
import { formatDuration } from '../../../../shared/lib';
import { getUserTimeFormat } from '../../../../shared/time';
import { ConfirmationComponent } from '../../../../shared/ui';
import { PHYSICAL_BACKUP_STATUS_BADGE_STYLES } from '../model/physicalBackupStatus';
import { PhysicalBackupsFiltersPanelComponent } from './PhysicalBackupsFiltersPanelComponent';
import { PhysicalBackupTypeCellComponent } from './PhysicalBackupTypeCellComponent';
import { PhysicalRestoreComponent } from './PhysicalRestoreComponent';

const BACKUPS_PAGE_SIZE = 50;
const BACKUPS_REFRESH_MS = 1_000;

interface Props {
  database: Database;
  isCanManageDBs: boolean;
  isDirectlyUnderTab?: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
}

const formatSize = (sizeMb: number, formatNumber: LocaleContextValue['formatNumber']): string => {
  if (sizeMb >= 1024) {
    const sizeGb = sizeMb / 1024;
    return `${formatNumber(Number(sizeGb.toFixed(2)))} GB`;
  }

  return `${formatNumber(Number((sizeMb ?? 0).toFixed(2)))} MB`;
};

// A missing segment before a WAL range is its health state, so it takes the
// place of the status badge, styled like the other broken-chain warning.
const renderGapBadge = (t: TFunction): JSX.Element => {
  const badgeStyle = PHYSICAL_BACKUP_STATUS_BADGE_STYLES[PhysicalBackupStatus.CHAIN_BROKEN];

  return (
    <Tooltip title={t('backups.physical.list.walRange.gapBeforeTooltip')}>
      <span
        className={`inline-flex cursor-help items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ring-1 ring-inset ${badgeStyle.pillClasses}`}
      >
        <span className={`h-1.5 w-1.5 rounded-full ${badgeStyle.dotClasses}`} />
        {t('backups.physical.list.walRange.gap')}
      </span>
    </Tooltip>
  );
};

const renderStatusBadge = (
  backup: PhysicalBackupListItem,
  onShowError: (backup: PhysicalBackupListItem) => void,
  t: TFunction,
): JSX.Element => {
  if (backup.hasGapBefore) {
    return renderGapBadge(t);
  }

  const badgeStyle = PHYSICAL_BACKUP_STATUS_BADGE_STYLES[backup.status];
  const label = t(PHYSICAL_BACKUP_STATUS_LABEL_KEYS[backup.status]);

  const badge = (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ring-1 ring-inset ${badgeStyle.pillClasses}`}
    >
      {backup.status === PhysicalBackupStatus.IN_PROGRESS ? (
        <SyncOutlined spin style={{ fontSize: 10 }} />
      ) : (
        <span className={`h-1.5 w-1.5 rounded-full ${badgeStyle.dotClasses}`} />
      )}
      {label}
    </span>
  );

  if (!backup.failMessage) {
    return badge;
  }

  return (
    <Tooltip title={t('backups.list.clickToSeeErrorDetails')}>
      <span className="cursor-pointer" onClick={() => onShowError(backup)}>
        {badge}
      </span>
    </Tooltip>
  );
};

export const PhysicalBackupsComponent = ({
  database,
  isCanManageDBs,
  isDirectlyUnderTab,
  scrollContainerRef,
}: Props): JSX.Element => {
  const { t } = useTranslation();
  const { formatNumber, formatRelativeTime } = useLocale();
  const { message } = App.useApp();

  const [backupConfig, setBackupConfig] = useState<BackupConfig | undefined>();
  const [isBackupConfigLoading, setIsBackupConfigLoading] = useState(false);

  const [isTriggeringBackup, setIsTriggeringBackup] = useState(false);
  const [cancellingBackupId, setCancellingBackupId] = useState<string | undefined>();
  const [deletingBackupId, setDeletingBackupId] = useState<string | undefined>();
  const [deleteConfirmationBackup, setDeleteConfirmationBackup] = useState<
    PhysicalBackupListItem | undefined
  >();

  const [restoringBackup, setRestoringBackup] = useState<PhysicalBackupListItem | undefined>();
  const [isPitrModalOpen, setIsPitrModalOpen] = useState(false);

  const [showingBackupError, setShowingBackupError] = useState<
    PhysicalBackupListItem | undefined
  >();

  const [isFilterPanelVisible, setIsFilterPanelVisible] = useState(false);
  const [filters, setFilters] = usePersistentState<PhysicalBackupsFilters>(
    // eslint-disable-next-line i18next/no-literal-string -- localStorage key
    'physicalBackupsFilters',
    {},
  );

  const isIncrementalAllowed =
    database.settings.type === DatabaseType.POSTGRES_PHYSICAL &&
    database.settings.backupType !== PhysicalDatabaseBackupType.FULL;

  const fetchBackupsPage = async (after: string | undefined) => {
    const response = await physicalBackupsApi.getPhysicalBackups(
      database.id,
      BACKUPS_PAGE_SIZE,
      after,
      filters,
    );

    return {
      items: response.items,
      nextCursor: response.nextCursor,
      metadata: response.totalUsageMb,
    };
  };

  const triggerBackup = async (type: TriggerPhysicalBackupType) => {
    setIsTriggeringBackup(true);

    try {
      await physicalBackupsApi.triggerPhysicalBackup(database.id, type);
      await new Promise((resolve) => setTimeout(resolve, 1000));
      await reloadBackups();
    } catch (e) {
      message.error(translateApiError(e, t));
    }

    setIsTriggeringBackup(false);
  };

  const cancelBackup = async (backupId: string) => {
    setCancellingBackupId(backupId);

    try {
      await physicalBackupsApi.cancelPhysicalBackup(backupId);
      await refreshBackups();
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setCancellingBackupId(undefined);
  };

  const deleteBackup = async () => {
    if (!deleteConfirmationBackup) return;

    const backupId = deleteConfirmationBackup.id;
    setDeleteConfirmationBackup(undefined);
    setDeletingBackupId(backupId);

    try {
      await physicalBackupsApi.deletePhysicalBackup(backupId);
      await reloadBackups();
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setDeletingBackupId(undefined);
  };

  const {
    items: backups,
    metadata: totalUsageMb = 0,
    isLoading: isBackupsLoading,
    isLoadingMore,
    hasMore,
    reload: reloadBackups,
    refreshFirstPage: refreshBackups,
  } = usePolledPagedList({
    fetchPage: fetchBackupsPage,
    getItemKey: (backup: PhysicalBackupListItem) => backup.id,
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

  const renderActions = (record: PhysicalBackupListItem): JSX.Element => (
    <div className="flex gap-2 text-lg">
      {/* WAL ranges have no standalone backup identity - they are not restored or deleted on their own (use point-in-time restore instead). */}
      {record.type === PhysicalBackupType.WAL ? null : (
        <>
          {record.status === PhysicalBackupStatus.IN_PROGRESS && isCanManageDBs && (
            <>
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
            </>
          )}

          {record.status === PhysicalBackupStatus.COMPLETED && (
            <>
              <Tooltip title={t('backups.physical.list.restoreFromThisBackup')}>
                <CloudUploadOutlined
                  className="cursor-pointer"
                  onClick={() => setRestoringBackup(record)}
                  style={{ color: '#155dfc' }}
                />
              </Tooltip>

              {isCanManageDBs &&
                (deletingBackupId === record.id ? (
                  <SyncOutlined spin />
                ) : (
                  <Tooltip title={t('backups.list.actions.deleteBackup')}>
                    <DeleteOutlined
                      className="cursor-pointer"
                      onClick={() => {
                        if (deletingBackupId) return;
                        setDeleteConfirmationBackup(record);
                      }}
                      style={{ color: '#ff0000', opacity: deletingBackupId ? 0.2 : 1 }}
                    />
                  </Tooltip>
                ))}
            </>
          )}
        </>
      )}
    </div>
  );

  const renderDuration = (record: PhysicalBackupListItem): string => {
    // A WAL range's end is the receive time of its last segment, not the end
    // of a backup run, so it has no duration to show.
    if (record.type === PhysicalBackupType.WAL) {
      return '-';
    }

    if (record.completedAt) {
      return formatDuration(dayjs(record.completedAt).diff(dayjs(record.createdAt)));
    }

    if (record.status === PhysicalBackupStatus.IN_PROGRESS) {
      // Live elapsed time; the 1s list refresh re-renders this and makes it tick.
      return formatDuration(dayjs().diff(dayjs(record.createdAt)));
    }

    return '-';
  };

  const renderCreatedAt = (backup: PhysicalBackupListItem): JSX.Element => {
    const timeFormat = getUserTimeFormat().format;
    const createdAt = dayjs.utc(backup.createdAt).local();

    if (backup.type === PhysicalBackupType.WAL && backup.completedAt) {
      const lastReceivedAt = dayjs.utc(backup.completedAt).local();

      return (
        <div className="grid grid-cols-[auto_1fr] gap-x-2">
          <span className="text-gray-500 dark:text-gray-400">
            {t('backups.physical.list.walRange.from')}
          </span>
          <span>
            {createdAt.format(timeFormat)}{' '}
            <span className="text-gray-500 dark:text-gray-400">
              ({formatRelativeTime(createdAt)})
            </span>
          </span>

          <span className="text-gray-500 dark:text-gray-400">
            {t('backups.physical.list.walRange.to')}
          </span>
          <span>
            {lastReceivedAt.format(timeFormat)}{' '}
            <span className="text-gray-500 dark:text-gray-400">
              ({formatRelativeTime(lastReceivedAt)})
            </span>
          </span>

          <span className="text-gray-500 dark:text-gray-400">
            {t('backups.physical.list.walRange.segments')}
          </span>
          <span>{backup.walSegmentsCount}</span>
        </div>
      );
    }

    return (
      <div>
        {createdAt.format(timeFormat)}
        <div className="text-gray-500 dark:text-gray-400">({formatRelativeTime(createdAt)})</div>
      </div>
    );
  };

  const columns: ColumnsType<PhysicalBackupListItem> = [
    {
      title: t('common.fields.type'),
      dataIndex: 'type',
      key: 'type',
      render: (_type: PhysicalBackupType, record: PhysicalBackupListItem) => (
        <PhysicalBackupTypeCellComponent backup={record} />
      ),
    },
    {
      title: t('common.fields.status'),
      dataIndex: 'status',
      key: 'status',
      render: (_status: PhysicalBackupStatus, record: PhysicalBackupListItem) =>
        renderStatusBadge(record, setShowingBackupError, t),
    },
    {
      title: t('backups.list.columns.size'),
      dataIndex: 'sizeMb',
      key: 'sizeMb',
      width: 110,
      render: (sizeMb: number) => formatSize(sizeMb, formatNumber),
    },
    {
      title: t('backups.list.columns.duration'),
      key: 'duration',
      width: 150,
      render: (_, record: PhysicalBackupListItem) => renderDuration(record),
    },
    {
      title: t('backups.list.columns.created'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (_createdAt: Date, record: PhysicalBackupListItem) => renderCreatedAt(record),
    },
    {
      title: t('backups.list.columns.actions'),
      key: 'actions',
      render: (_, record: PhysicalBackupListItem) => renderActions(record),
    },
  ];

  const backupNowMenu: MenuProps = {
    items: [
      { key: 'full', label: t('backups.physical.list.fullBackup') },
      {
        key: 'incremental',
        label: t('backups.physical.list.incrementalBackup'),
        disabled: !isIncrementalAllowed,
      },
    ],
    onClick: ({ key }) => triggerBackup(key as TriggerPhysicalBackupType),
  };

  const isAnyFilterApplied =
    (filters.types !== undefined && filters.types.length > 0) ||
    (filters.statuses !== undefined && filters.statuses.length > 0) ||
    filters.createdBefore !== undefined;

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
          <PhysicalBackupsFiltersPanelComponent filters={filters} onFiltersChange={setFilters} />
        </div>
      )}

      {!isBackupConfigLoading && !backupConfig?.isBackupsEnabled && (
        <div className="text-sm text-red-600">{t('backups.list.scheduledBackupsDisabled')}</div>
      )}

      <div className="mt-3 text-sm text-gray-600 dark:text-gray-400">
        {t('backups.physical.list.totalUsage', { size: formatSize(totalUsageMb, formatNumber) })}
      </div>

      <div className="mt-4 flex items-center">
        <Dropdown.Button
          className="!w-auto"
          type="primary"
          loading={isTriggeringBackup}
          disabled={isTriggeringBackup}
          icon={<DownOutlined />}
          menu={backupNowMenu}
          onClick={() => triggerBackup('AUTO')}
        >
          {t('backups.physical.list.backUpNow')}
        </Dropdown.Button>

        <Button className="ml-2" onClick={() => setIsPitrModalOpen(true)}>
          {t('backups.physical.list.restore')}
        </Button>
      </div>

      <div className="mt-5 w-full md:max-w-[890px]">
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
                          {t('backups.list.columns.created')}
                        </div>
                        <div className="text-sm font-medium">{renderCreatedAt(backup)}</div>
                      </div>
                      <div>{renderStatusBadge(backup, setShowingBackupError, t)}</div>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {t('common.fields.type')}
                        </div>
                        <div className="text-sm font-medium">
                          <PhysicalBackupTypeCellComponent backup={backup} />
                        </div>
                      </div>
                      <div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {t('backups.list.columns.size')}
                        </div>
                        <div className="text-sm font-medium">
                          {formatSize(backup.sizeMb, formatNumber)}
                        </div>
                      </div>
                      <div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {t('backups.list.columns.duration')}
                        </div>
                        <div className="text-sm font-medium">{renderDuration(backup)}</div>
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

      {deleteConfirmationBackup && (
        <ConfirmationComponent
          onConfirm={deleteBackup}
          onDecline={() => setDeleteConfirmationBackup(undefined)}
          description={
            deleteConfirmationBackup.type === PhysicalBackupType.FULL
              ? t('backups.physical.list.deleteFullConfirmation')
              : t('backups.physical.list.deleteIncrementalConfirmation')
          }
          isDestructive
          actionText={t('common.actions.delete')}
        />
      )}

      {restoringBackup && (
        <Modal
          width={640}
          open={!!restoringBackup}
          onCancel={() => setRestoringBackup(undefined)}
          title={t('backups.list.restoreTitle')}
          footer={null}
          maskClosable={false}
        >
          <PhysicalRestoreComponent
            database={database}
            backup={restoringBackup}
            onClose={() => setRestoringBackup(undefined)}
          />
        </Modal>
      )}

      {isPitrModalOpen && (
        <Modal
          width={640}
          open={isPitrModalOpen}
          onCancel={() => setIsPitrModalOpen(false)}
          title={t('backups.physical.list.pointInTimeRestoreTitle')}
          footer={null}
          maskClosable={false}
        >
          <PhysicalRestoreComponent database={database} onClose={() => setIsPitrModalOpen(false)} />
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
          <div className="text-sm break-words whitespace-pre-wrap">
            {showingBackupError.failMessage}
          </div>
        </Modal>
      )}
    </div>
  );
};
