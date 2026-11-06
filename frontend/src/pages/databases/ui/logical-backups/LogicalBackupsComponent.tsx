import { FilterFilled, FilterOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { App, Button, Modal, Spin } from 'antd';
import { useRef, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import type { Database } from '@/entities/databases';
import { logicalBackupApi } from '@/pages/databases/api/logicalBackupApi';
import type { LogicalBackup } from '@/pages/databases/model/LogicalBackup';
import type { LogicalBackupListFilter } from '@/pages/databases/model/LogicalBackupListFilter';
import { BackupConfigNotice } from '@/pages/databases/ui/backup-config/BackupConfigNotice';
import { useBackupConfigLoad } from '@/pages/databases/ui/backup-config/useBackupConfigLoad';
import type { LogicalBackupRowActions } from '@/pages/databases/ui/logical-backups/LogicalBackupActions';
import { LogicalBackupCards } from '@/pages/databases/ui/logical-backups/LogicalBackupCards';
import { LogicalBackupsFiltersPanelComponent } from '@/pages/databases/ui/logical-backups/LogicalBackupsFiltersPanelComponent';
import { LogicalBackupsTable } from '@/pages/databases/ui/logical-backups/LogicalBackupsTable';
import { useLogicalBackupActions } from '@/pages/databases/ui/logical-backups/useLogicalBackupActions';
import { RestoresComponent } from '@/pages/databases/ui/restores/RestoresComponent';
import { usePersistentState, usePolledPagedList } from '@/shared/hooks';
import { getWebsitePageUrl, translateApiError, useLocale } from '@/shared/i18n';
import { ConfirmationComponent, LoadFailure } from '@/shared/ui';

const BACKUPS_PAGE_SIZE = 50;
const BACKUPS_REFRESH_MS = 1_000;

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
  const { locale } = useLocale();
  const { message, modal } = App.useApp();

  const [showingBackupError, setShowingBackupError] = useState<LogicalBackup | undefined>();
  const [backupToDeleteId, setBackupToDeleteId] = useState<string | undefined>();
  const [restoringBackup, setRestoringBackup] = useState<LogicalBackup | undefined>();

  const [isFilterPanelVisible, setIsFilterPanelVisible] = useState(false);
  const [filter, setFilter] = usePersistentState<LogicalBackupListFilter>(
    // eslint-disable-next-line i18next/no-literal-string -- localStorage key
    'logicalBackupListFilter',
    {},
  );

  const fetchBackupsPage = (after: string | undefined) =>
    logicalBackupApi.listLogicalBackups({
      databaseId: database.id,
      filter,
      limit: BACKUPS_PAGE_SIZE,
      after,
    });

  const {
    items: backups,
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
      if (failedPage === 'next') message.error(translateApiError(error, t));
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

  const isVerifyRestoreConfirmOpenRef = useRef(false);

  const confirmVerifyRestore = (backupId: string) => {
    if (isVerifyRestoreConfirmOpenRef.current) return;

    isVerifyRestoreConfirmOpenRef.current = true;

    const markVerifyRestoreConfirmClosed = () => {
      isVerifyRestoreConfirmOpenRef.current = false;
    };

    modal.confirm({
      title: t('backups.logical.list.verifyRestore.title'),
      icon: <InfoCircleOutlined className="!text-accent" />,
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
      onOk: async () => {
        await backupActions.startVerification(backupId);
        markVerifyRestoreConfirmClosed();
      },
      onCancel: markVerifyRestoreConfirmClosed,
    });
  };

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
          className="mr-1"
          type="primary"
          disabled={backupActions.isStartingBackup || firstLoadError !== undefined}
          loading={backupActions.isStartingBackup}
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
              <LogicalBackupCards
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
              <LogicalBackupsTable
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
        <Modal
          width={400}
          open
          onCancel={() => setRestoringBackup(undefined)}
          title={t('backups.list.restoreTitle')}
          footer={null}
          maskClosable={false}
        >
          <RestoresComponent database={database} backup={restoringBackup} />
        </Modal>
      )}

      {showingBackupError && (
        <Modal
          title={t('backups.list.errorDetailsTitle')}
          open
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
