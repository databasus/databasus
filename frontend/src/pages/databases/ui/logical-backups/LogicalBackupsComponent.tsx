import { FilterFilled, FilterOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { Alert, App, Button, Modal, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { type BackupConfig, backupConfigApi } from '../../../../entities/backup-configs';
import type { Database } from '../../../../entities/databases';
import { usePersistentState, usePolledPagedList } from '../../../../shared/hooks';
import { getWebsitePageUrl, translateApiError, useLocale } from '../../../../shared/i18n';
import { ConfirmationComponent } from '../../../../shared/ui';
import { logicalBackupApi } from '../../api/logicalBackupApi';
import type { LogicalBackup } from '../../model/LogicalBackup';
import type { LogicalBackupListFilter } from '../../model/LogicalBackupListFilter';
import { RestoresComponent } from '../restores/RestoresComponent';
import type { LogicalBackupRowActions } from './LogicalBackupActions';
import { LogicalBackupCards } from './LogicalBackupCards';
import { LogicalBackupsFiltersPanelComponent } from './LogicalBackupsFiltersPanelComponent';
import { LogicalBackupsTable } from './LogicalBackupsTable';
import { useLogicalBackupActions } from './useLogicalBackupActions';

const BACKUPS_PAGE_SIZE = 50;
const BACKUPS_REFRESH_MS = 1_000;

interface Props {
  database: Database;
  isCanManageDBs: boolean;
  isDirectlyUnderTab?: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
  onNavigateToVerifications?: () => void;
}

export const LogicalBackupsComponent = ({
  database,
  isCanManageDBs,
  isDirectlyUnderTab,
  scrollContainerRef,
  onNavigateToVerifications,
}: Props) => {
  const { t } = useTranslation();
  const { locale } = useLocale();
  const { message, modal } = App.useApp();

  const [backupConfig, setBackupConfig] = useState<BackupConfig | undefined>();
  const [isBackupConfigLoading, setIsBackupConfigLoading] = useState(false);
  const [backupConfigError, setBackupConfigError] = useState<unknown>();
  const [backupConfigLoadAttempt, setBackupConfigLoadAttempt] = useState(0);

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

  const backupActions = useLogicalBackupActions({
    databaseId: database.id,
    filter,
    replaceOrPrependBackup,
    reloadBackups,
    onNavigateToVerifications,
  });

  const confirmVerifyRestore = (backupId: string) => {
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
      onOk: () => backupActions.startVerification(backupId),
    });
  };

  useEffect(() => {
    let isActive = true;

    setIsBackupConfigLoading(true);
    setBackupConfig(undefined);
    setBackupConfigError(undefined);

    backupConfigApi
      .getBackupConfig(database.id)
      .then((loadedBackupConfig) => {
        if (isActive) setBackupConfig(loadedBackupConfig);
      })
      .catch((error: unknown) => {
        if (isActive) setBackupConfigError(error);
      })
      .finally(() => {
        if (isActive) setIsBackupConfigLoading(false);
      });

    return () => {
      isActive = false;
    };
  }, [database.id, backupConfigLoadAttempt]);

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
    <div
      className={`w-full bg-surface p-3 shadow md:p-5 ${isDirectlyUnderTab ? 'rounded-tr-md rounded-br-md rounded-bl-md' : 'rounded-md'}`}
    >
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

      {backupConfigError !== undefined && (
        <Alert
          className="mt-3"
          type="error"
          showIcon
          message={translateApiError(backupConfigError, t)}
          action={
            <Button onClick={() => setBackupConfigLoadAttempt((attempt) => attempt + 1)}>
              {t('common.actions.retry')}
            </Button>
          }
        />
      )}

      {backupConfig && !backupConfig.isBackupsEnabled && (
        <div className="text-sm text-danger">{t('backups.list.scheduledBackupsDisabled')}</div>
      )}

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
          <Alert
            type="error"
            showIcon
            message={translateApiError(firstLoadError, t)}
            action={
              <Button onClick={() => void reloadBackups()}>{t('common.actions.retry')}</Button>
            }
          />
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
