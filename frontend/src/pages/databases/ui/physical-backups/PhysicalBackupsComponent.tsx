import { DownOutlined, FilterFilled, FilterOutlined } from '@ant-design/icons';
import { App, Button, Dropdown, type MenuProps, Modal, Spin } from 'antd';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, DatabaseType, PhysicalDatabaseBackupType } from '@/entities/databases';
import { physicalBackupsApi } from '@/pages/databases/api/physicalBackupsApi';
import { formatBackupSize } from '@/pages/databases/lib/formatBackupSize';
import { type PhysicalBackupListItem } from '@/pages/databases/model/physical-backups/PhysicalBackupListItem';
import { type PhysicalBackupsFilters } from '@/pages/databases/model/physical-backups/PhysicalBackupsFilters';
import { PhysicalBackupType } from '@/pages/databases/model/physical-backups/PhysicalBackupType';
import { BackupConfigNotice } from '@/pages/databases/ui/backup-config/BackupConfigNotice';
import { useBackupConfigLoad } from '@/pages/databases/ui/backup-config/useBackupConfigLoad';
import type { PhysicalBackupRowActions } from '@/pages/databases/ui/physical-backups/PhysicalBackupActions';
import { PhysicalBackupCards } from '@/pages/databases/ui/physical-backups/PhysicalBackupCards';
import { PhysicalBackupsFiltersPanelComponent } from '@/pages/databases/ui/physical-backups/PhysicalBackupsFiltersPanelComponent';
import { PhysicalBackupsTable } from '@/pages/databases/ui/physical-backups/PhysicalBackupsTable';
import { PhysicalRestoreComponent } from '@/pages/databases/ui/physical-backups/PhysicalRestoreComponent';
import { usePhysicalBackupActions } from '@/pages/databases/ui/physical-backups/usePhysicalBackupActions';
import { usePersistentState, usePolledPagedList } from '@/shared/hooks';
import { translateApiError, useLocale } from '@/shared/i18n';
import { ConfirmationComponent, LoadFailure } from '@/shared/ui';

const BACKUPS_PAGE_SIZE = 50;
const BACKUPS_REFRESH_MS = 1_000;

interface Props {
  database: Database;
  isCanManageDBs: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
}

export const PhysicalBackupsComponent = ({
  database,
  isCanManageDBs,
  scrollContainerRef,
}: Props): JSX.Element => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();
  const { message } = App.useApp();

  const [backupToDelete, setBackupToDelete] = useState<PhysicalBackupListItem | undefined>();
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

  const {
    items: backups,
    metadata: totalUsageMb,
    isLoading: isBackupsLoading,
    isLoadingMore,
    hasMore,
    firstLoadError,
    reload: reloadBackups,
    refreshFirstPage: refreshBackups,
  } = usePolledPagedList({
    fetchPage: fetchBackupsPage,
    getItemKey: (backup: PhysicalBackupListItem) => backup.id,
    refreshIntervalMs: BACKUPS_REFRESH_MS,
    scrollContainerRef,
    resetKey: JSON.stringify([database.id, filters]),
    onError: (error, failedPage) => {
      if (failedPage === 'next') message.error(translateApiError(error, t));
    },
  });

  const { backupConfig, isBackupConfigLoading, backupConfigError, reloadBackupConfig } =
    useBackupConfigLoad({ databaseId: database.id, workspaceId: database.workspaceId });

  const backupActions = usePhysicalBackupActions({
    databaseId: database.id,
    refreshBackups,
    reloadBackups,
  });

  const isIncrementalAllowed =
    database.settings.type === DatabaseType.POSTGRES_PHYSICAL &&
    database.settings.backupType !== PhysicalDatabaseBackupType.FULL;

  const isAnyFilterApplied =
    (filters.types !== undefined && filters.types.length > 0) ||
    (filters.statuses !== undefined && filters.statuses.length > 0) ||
    filters.createdBefore !== undefined;

  const backupNowMenu: MenuProps = {
    items: [
      { key: 'FULL', label: t('backups.physical.list.fullBackup') },
      {
        key: 'INCREMENTAL',
        label: t('backups.physical.list.incrementalBackup'),
        disabled: !isIncrementalAllowed,
      },
    ],
    onClick: ({ key }) => {
      if (key === 'FULL' || key === 'INCREMENTAL') void backupActions.startBackup(key);
    },
  };

  const rowActions: PhysicalBackupRowActions = {
    isCanManageDBs,
    cancellingBackupId: backupActions.cancellingBackupId,
    deletingBackupId: backupActions.deletingBackupId,
    onCancel: (backupId) => void backupActions.cancelBackup(backupId),
    onDelete: setBackupToDelete,
    onRestore: setRestoringBackup,
  };

  const isStartRefusedByConfig =
    backupConfig !== undefined && (!backupConfig.isBackupsEnabled || !backupConfig.storageId);

  if (isBackupConfigLoading) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );
  }

  return (
    <div className="w-full rounded-md bg-surface p-3 shadow md:p-5">
      <div className="flex items-center gap-2">
        <h2 className="text-lg font-bold md:text-xl">{t('backups.list.title')}</h2>
        <div className="relative">
          <Button
            type="text"
            size="small"
            aria-label={t('backups.filters.toggle')}
            aria-expanded={isFilterPanelVisible}
            icon={
              isFilterPanelVisible ? <FilterFilled className="text-accent" /> : <FilterOutlined />
            }
            onClick={() => setIsFilterPanelVisible(!isFilterPanelVisible)}
          />
          {!isFilterPanelVisible && isAnyFilterApplied && (
            <span className="absolute -top-1 -right-1 h-2 w-2 rounded-full bg-accent" />
          )}
        </div>
      </div>

      {isFilterPanelVisible && (
        <div className="mt-3">
          <PhysicalBackupsFiltersPanelComponent filters={filters} onFiltersChange={setFilters} />
        </div>
      )}

      <BackupConfigNotice
        backupConfig={backupConfig}
        backupConfigError={backupConfigError}
        onRetry={reloadBackupConfig}
      />

      {totalUsageMb !== undefined && firstLoadError === undefined && (
        <div className="mt-3 text-sm text-muted">
          {t('backups.physical.list.totalUsage', {
            size: formatBackupSize(totalUsageMb, formatNumber),
          })}
        </div>
      )}

      <div className="mt-4 flex items-center">
        {isCanManageDBs && (
          <Dropdown.Button
            className="mr-2 !w-auto"
            type="primary"
            loading={backupActions.isStartingBackup}
            disabled={
              backupActions.isStartingBackup ||
              firstLoadError !== undefined ||
              isStartRefusedByConfig
            }
            icon={<DownOutlined />}
            menu={backupNowMenu}
            onClick={() => void backupActions.startBackup('AUTO')}
          >
            {t('backups.physical.list.backUpNow')}
          </Dropdown.Button>
        )}

        <Button onClick={() => setIsPitrModalOpen(true)}>
          {t('backups.physical.list.restore')}
        </Button>
      </div>

      {isCanManageDBs && isStartRefusedByConfig && (
        <div className="mt-2 text-sm text-muted">{t('backups.physical.list.startRequirement')}</div>
      )}

      <div className="mt-5 w-full md:max-w-[890px]">
        {firstLoadError !== undefined ? (
          <LoadFailure error={firstLoadError} onRetry={() => void reloadBackups()} />
        ) : (
          <>
            <div className="md:hidden">
              <PhysicalBackupCards
                backups={backups}
                isLoading={isBackupsLoading}
                actions={rowActions}
                onShowError={setShowingBackupError}
              />

              {isLoadingMore && (
                <div className="mt-3 flex justify-center">
                  <Spin />
                </div>
              )}
              {!hasMore && backups.length > 0 && (
                <div className="mt-3 text-center text-sm text-muted">
                  {t('backups.list.allLoaded', { count: backups.length })}
                </div>
              )}
              {!isBackupsLoading && backups.length === 0 && (
                <div className="py-8 text-center text-muted">{t('backups.list.empty')}</div>
              )}
            </div>

            <div className="hidden md:block">
              <PhysicalBackupsTable
                backups={backups}
                isLoading={isBackupsLoading}
                actions={rowActions}
                onShowError={setShowingBackupError}
              />
              {isLoadingMore && (
                <div className="mt-2 flex justify-center">
                  <Spin />
                </div>
              )}
              {!hasMore && backups.length > 0 && (
                <div className="mt-2 text-center text-muted">
                  {t('backups.list.allLoaded', { count: backups.length })}
                </div>
              )}
            </div>
          </>
        )}
      </div>

      {backupToDelete && (
        <ConfirmationComponent
          onConfirm={() => {
            setBackupToDelete(undefined);
            void backupActions.deleteBackup(backupToDelete.id);
          }}
          onDecline={() => setBackupToDelete(undefined)}
          description={
            backupToDelete.type === PhysicalBackupType.FULL
              ? t('backups.physical.list.deleteFullConfirmation')
              : t('backups.physical.list.deleteIncrementalConfirmation')
          }
          isDestructive
          actionText={t('common.actions.delete')}
        />
      )}

      {/* These three dialogs stay mounted while closed, so their close animation can hand the focus
          back to the control that opened them. */}
      <Modal
        width={640}
        open={restoringBackup !== undefined}
        onCancel={() => setRestoringBackup(undefined)}
        title={t('backups.list.restoreTitle')}
        footer={null}
        maskClosable={false}
        destroyOnHidden
      >
        {restoringBackup && (
          <PhysicalRestoreComponent
            database={database}
            backup={restoringBackup}
            onClose={() => setRestoringBackup(undefined)}
          />
        )}
      </Modal>

      <Modal
        width={640}
        open={isPitrModalOpen}
        onCancel={() => setIsPitrModalOpen(false)}
        title={t('backups.physical.list.pointInTimeRestoreTitle')}
        footer={null}
        maskClosable={false}
        destroyOnHidden
      >
        {isPitrModalOpen && (
          <PhysicalRestoreComponent database={database} onClose={() => setIsPitrModalOpen(false)} />
        )}
      </Modal>

      <Modal
        title={t('backups.list.errorDetailsTitle')}
        open={showingBackupError !== undefined}
        onCancel={() => setShowingBackupError(undefined)}
        maskClosable={false}
        footer={null}
        destroyOnHidden
      >
        <div className="text-sm break-words whitespace-pre-wrap">
          {showingBackupError?.failMessage}
        </div>
      </Modal>
    </div>
  );
};
