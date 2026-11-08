import { ActionIcon, Button, Loader } from '@mantine/core';
import { Info, ListFilter } from 'lucide-react';
import { useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import type { Database } from '@/entities/databases';
import { BackupConfigNotice } from '@/pages/databases/backup-config/ui/BackupConfigNotice';
import { useBackupConfigLoad } from '@/pages/databases/backup-config/ui/useBackupConfigLoad';
import { logicalBackupApi } from '@/pages/databases/logical-backups/api/logicalBackupApi';
import type { LogicalBackup } from '@/pages/databases/logical-backups/model/logicalBackupContract';
import type { LogicalBackupListFilter } from '@/pages/databases/logical-backups/model/LogicalBackupListFilter';
import type { LogicalBackupRowActions } from '@/pages/databases/logical-backups/ui/LogicalBackupActions';
import { LogicalBackupCards } from '@/pages/databases/logical-backups/ui/LogicalBackupCards';
import { LogicalBackupsFiltersPanelComponent } from '@/pages/databases/logical-backups/ui/LogicalBackupsFiltersPanelComponent';
import { LogicalBackupsTable } from '@/pages/databases/logical-backups/ui/LogicalBackupsTable';
import { useLogicalBackupActions } from '@/pages/databases/logical-backups/ui/useLogicalBackupActions';
import { RestoresComponent } from '@/pages/databases/restores/ui/RestoresComponent';
import { usePersistentState, usePolledPagedList } from '@/shared/hooks';
import { getWebsitePageUrl, translateApiError, useLocale } from '@/shared/i18n';
import { formatCompactSizeMb } from '@/shared/lib';
import { useFeedback } from '@/shared/lib/feedback';
import { Dialog, ConfirmationComponent, LoadFailure } from '@/shared/ui';

const BACKUPS_PAGE_SIZE = 50;
const BACKUPS_REFRESH_MS = 1000;
interface Props {
  database: Database;
  isCanManageDBs: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
  onNavigateToVerifications?: () => void;
}

export const LogicalBackupsComponent = ({
  database,
  isCanManageDBs,
  scrollContainerRef,
  onNavigateToVerifications,
}: Props) => {
  const { t } = useTranslation();
  const { locale, formatNumber } = useLocale();
  const notify = useFeedback();
  const [showingBackupError, setShowingBackupError] = useState<LogicalBackup | undefined>();
  const [backupToDeleteId, setBackupToDeleteId] = useState<string | undefined>();
  const [restoringBackup, setRestoringBackup] = useState<LogicalBackup | undefined>();

  const [isFilterPanelVisible, setIsFilterPanelVisible] = useState(false);
  const [filter, setFilter] = usePersistentState<LogicalBackupListFilter>(
    // eslint-disable-next-line i18next/no-literal-string -- localStorage key
    'logicalBackupListFilter',
    {},
  );

  const fetchBackupsPage = async (after: string | undefined) => {
    const response = await logicalBackupApi.listLogicalBackups({
      databaseId: database.id,
      filter,
      limit: BACKUPS_PAGE_SIZE,
      after,
    });

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
    replaceOrPrependItem: replaceOrPrependBackup,
  } = usePolledPagedList({
    fetchPage: fetchBackupsPage,
    getItemKey: (backup: LogicalBackup) => backup.id,
    refreshIntervalMs: BACKUPS_REFRESH_MS,
    scrollContainerRef,
    resetKey: JSON.stringify([database.id, filter]),
    onError: (error, failedPage) => {
      if (failedPage === 'next') notify({ kind: 'error', title: translateApiError(error, t) });
    },
  });

  const { backupConfig, isBackupConfigLoading, backupConfigError, reloadBackupConfig } =
    useBackupConfigLoad({ databaseId: database.id, workspaceId: database.workspaceId });

  const backupActions = useLogicalBackupActions({
    databaseId: database.id,
    filter,
    replaceOrPrependBackup,
    reloadBackups,
    onNavigateToVerifications,
  });
  const [verifyingBackupId, setVerifyingBackupId] = useState<string>();
  const confirmVerifyRestore = (backupId: string) =>
    setVerifyingBackupId((current) => current ?? backupId);
  const isAnyFilterApplied =
    (filter.statuses && filter.statuses.length > 0) || filter.createdBefore !== undefined;

  const rowActions: LogicalBackupRowActions = {
    databaseType: database.settings.type,
    isCanManageDBs,
    cancellingBackupId: backupActions.cancellingBackupId,
    deletingBackupId: backupActions.deletingBackupId,
    downloadingBackupId: backupActions.downloadingBackupId,
    verifyingBackupId: backupActions.verifyingBackupId,
    onCancel: (backupId) => void backupActions.cancelBackup(backupId),
    onDelete: setBackupToDeleteId,
    onVerify: confirmVerifyRestore,
    onRestore: setRestoringBackup,
    onDownload: (backupId) => void backupActions.downloadBackup(backupId),
  };

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
          <LogicalBackupsFiltersPanelComponent filter={filter} onFilterChange={setFilter} />
        </div>
      )}

      <BackupConfigNotice
        backupConfig={backupConfig}
        backupConfigError={backupConfigError}
        onRetry={reloadBackupConfig}
      />

      <div className="mt-5" />

      <div className="flex items-center">
        <Button
          onClick={() => void backupActions.startBackup()}
          className="mr-1 !h-auto min-h-8 max-w-full !whitespace-normal"
          variant="filled"
          disabled={backupActions.isStartingBackup || firstLoadError !== undefined}
          loading={backupActions.isStartingBackup}
          aria-busy={backupActions.isStartingBackup}
        >
          <span className="md:hidden">{t('backups.logical.list.backupNow')}</span>
          <span className="hidden md:inline">{t('backups.logical.list.makeBackupNow')}</span>
        </Button>
      </div>

      <div className="mt-5 w-full md:max-w-[850px]">
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
              <LogicalBackupCards
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
              <LogicalBackupsTable
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

      {backupToDeleteId && (
        <ConfirmationComponent
          onConfirm={() => {
            setBackupToDeleteId(undefined);
            void backupActions.deleteBackup(backupToDeleteId);
          }}
          onDecline={() => setBackupToDeleteId(undefined)}
          description={t('backups.logical.list.deleteConfirmation')}
          isDestructive
          actionText={t('common.actions.delete')}
        />
      )}

      {restoringBackup && (
        <Dialog
          size={400}
          opened
          onClose={() => setRestoringBackup(undefined)}
          title={t('backups.list.restoreTitle')}
          closeOnClickOutside={false}
        >
          <RestoresComponent database={database} backup={restoringBackup} />
        </Dialog>
      )}

      {showingBackupError && (
        <Dialog
          title={t('backups.list.errorDetailsTitle')}
          opened
          onClose={() => setShowingBackupError(undefined)}
          closeOnClickOutside={false}
        >
          <div className="text-sm">{showingBackupError.failMessage}</div>
        </Dialog>
      )}

      {verifyingBackupId && (
        <Dialog
          opened
          title={t('backups.logical.list.verifyRestore.title')}
          onClose={() => setVerifyingBackupId(undefined)}
        >
          <div className="flex items-start gap-3">
            <Info aria-hidden size={20} className="shrink-0 text-accent" />
            <div>
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
            </div>
          </div>
          <div className="mt-5 flex flex-wrap justify-end gap-2">
            <Button onClick={() => setVerifyingBackupId(undefined)}>
              {t('common.actions.cancel')}
            </Button>
            <Button
              variant="filled"
              loading={backupActions.verifyingBackupId === verifyingBackupId}
              onClick={async () => {
                await backupActions.startVerification(verifyingBackupId);
                setVerifyingBackupId(undefined);
              }}
              aria-busy={backupActions.verifyingBackupId === verifyingBackupId}
            >
              {t('backups.logical.list.verifyRestore.queue')}
            </Button>
          </div>
        </Dialog>
      )}
    </div>
  );
};
