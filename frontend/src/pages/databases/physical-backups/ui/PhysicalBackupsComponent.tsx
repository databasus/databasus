import { ActionIcon, Button, Loader, Menu } from '@mantine/core';
import { ChevronDown, ListFilter } from 'lucide-react';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, DatabaseType, PhysicalDatabaseBackupType } from '@/entities/databases';
import { BackupConfigNotice } from '@/pages/databases/backup-config/ui/BackupConfigNotice';
import { useBackupConfigLoad } from '@/pages/databases/backup-config/ui/useBackupConfigLoad';
import { physicalBackupsApi } from '@/pages/databases/physical-backups/api/physicalBackupsApi';
import { type PhysicalBackupListItem } from '@/pages/databases/physical-backups/model/PhysicalBackupListItem';
import { type PhysicalBackupsFilters } from '@/pages/databases/physical-backups/model/PhysicalBackupsFilters';
import { PhysicalBackupType } from '@/pages/databases/physical-backups/model/PhysicalBackupType';
import type { PhysicalBackupRowActions } from '@/pages/databases/physical-backups/ui/PhysicalBackupActions';
import { PhysicalBackupCards } from '@/pages/databases/physical-backups/ui/PhysicalBackupCards';
import { PhysicalBackupsFiltersPanelComponent } from '@/pages/databases/physical-backups/ui/PhysicalBackupsFiltersPanelComponent';
import { PhysicalBackupsTable } from '@/pages/databases/physical-backups/ui/PhysicalBackupsTable';
import { PhysicalRestoreComponent } from '@/pages/databases/physical-backups/ui/PhysicalRestoreComponent';
import { usePhysicalBackupActions } from '@/pages/databases/physical-backups/ui/usePhysicalBackupActions';
import { usePersistentState, usePolledPagedList } from '@/shared/hooks';
import { translateApiError, useLocale } from '@/shared/i18n';
import { formatCompactSizeMb } from '@/shared/lib';
import { useFeedback } from '@/shared/lib/feedback';
import { Dialog, ConfirmationComponent, LoadFailure } from '@/shared/ui';

const BACKUPS_PAGE_SIZE = 50;
const BACKUPS_REFRESH_MS = 1000;
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
  const notify = useFeedback();
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
      if (failedPage === 'next') notify({ kind: 'error', title: translateApiError(error, t) });
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
        <Loader role="status" />
      </div>
    );
  }

  return (
    <div className="w-full rounded-md bg-surface p-3 shadow md:p-5">
      <div className="flex items-center gap-2">
        <h2 className="text-lg font-bold md:text-xl">{t('backups.list.title')}</h2>
        <div className="relative">
          <ActionIcon
            variant="subtle"
            size="xs"
            aria-label={t('backups.filters.toggle')}
            aria-expanded={isFilterPanelVisible}
            onClick={() => setIsFilterPanelVisible(!isFilterPanelVisible)}
          >
            {isFilterPanelVisible ? (
              <ListFilter className="inline-block shrink-0 align-middle text-accent" size="1em" />
            ) : (
              <ListFilter size="1em" className="inline-block shrink-0 align-middle" />
            )}
          </ActionIcon>
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

      <div className="mt-4 flex flex-wrap items-center gap-y-2">
        {isCanManageDBs && (
          <Button.Group className="mr-2">
            <Button
              variant="filled"
              loading={backupActions.isStartingBackup}
              disabled={
                backupActions.isStartingBackup ||
                firstLoadError !== undefined ||
                isStartRefusedByConfig
              }
              onClick={() => void backupActions.startBackup('AUTO')}
              aria-busy={backupActions.isStartingBackup}
            >
              {t('backups.physical.list.backUpNow')}
            </Button>
            <Menu position="bottom-end">
              <Menu.Target>
                <Button
                  variant="filled"
                  aria-label={t('common.fields.type')}
                  disabled={
                    backupActions.isStartingBackup ||
                    firstLoadError !== undefined ||
                    isStartRefusedByConfig
                  }
                  px={8}
                >
                  <ChevronDown aria-hidden size={14} />
                </Button>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Item onClick={() => void backupActions.startBackup('FULL')}>
                  {t('backups.physical.list.fullBackup')}
                </Menu.Item>
                <Menu.Item
                  disabled={!isIncrementalAllowed}
                  onClick={() => void backupActions.startBackup('INCREMENTAL')}
                >
                  {t('backups.physical.list.incrementalBackup')}
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
          </Button.Group>
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
              {totalUsageMb !== undefined && (
                <div className="mb-3 text-sm text-muted">
                  {t('backups.list.totalUsage', {
                    size: formatCompactSizeMb(totalUsageMb, formatNumber),
                  })}
                </div>
              )}
              <PhysicalBackupCards
                backups={backups}
                isLoading={isBackupsLoading}
                actions={rowActions}
                onShowError={setShowingBackupError}
              />

              {isLoadingMore && (
                <div className="mt-3 flex justify-center">
                  <Loader role="status" />
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
                totalUsageMb={totalUsageMb}
                backups={backups}
                isLoading={isBackupsLoading}
                actions={rowActions}
                onShowError={setShowingBackupError}
              />
              {isLoadingMore && (
                <div className="mt-2 flex justify-center">
                  <Loader role="status" />
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
      <Dialog
        size={640}
        opened={restoringBackup !== undefined}
        onClose={() => setRestoringBackup(undefined)}
        title={t('backups.list.restoreTitle')}
        closeOnClickOutside={false}
      >
        {restoringBackup && (
          <PhysicalRestoreComponent
            database={database}
            backup={restoringBackup}
            onClose={() => setRestoringBackup(undefined)}
          />
        )}
      </Dialog>

      <Dialog
        size={640}
        opened={isPitrModalOpen}
        onClose={() => setIsPitrModalOpen(false)}
        title={t('backups.physical.list.pointInTimeRestoreTitle')}
        closeOnClickOutside={false}
      >
        {isPitrModalOpen && (
          <PhysicalRestoreComponent database={database} onClose={() => setIsPitrModalOpen(false)} />
        )}
      </Dialog>

      <Dialog
        title={t('backups.list.errorDetailsTitle')}
        opened={showingBackupError !== undefined}
        onClose={() => setShowingBackupError(undefined)}
        closeOnClickOutside={false}
      >
        <div className="text-sm break-words whitespace-pre-wrap">
          {showingBackupError?.failMessage}
        </div>
      </Dialog>
    </div>
  );
};
