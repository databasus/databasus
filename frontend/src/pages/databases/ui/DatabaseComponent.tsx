import { Loader, Tabs } from '@mantine/core';
import { CancelledError, useQueryClient } from '@tanstack/react-query';
import { type RefObject, useEffect, useEffectEvent, useLayoutEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, DatabaseType, databaseApi } from '@/entities/databases';
import { HealthcheckAttemptsComponent } from '@/pages/databases/healthcheck/ui';
import { getDatabaseTab } from '@/pages/databases/lib/getDatabaseTab';
import { LogicalBackupsComponent } from '@/pages/databases/logical-backups/ui/LogicalBackupsComponent';
import type { DatabaseTab } from '@/pages/databases/model/DatabaseTab';
import { PhysicalBackupsComponent } from '@/pages/databases/physical-backups/ui/PhysicalBackupsComponent';
import { DatabaseConfigComponent } from '@/pages/databases/settings/ui/DatabaseConfigComponent';
import { VerificationsComponent } from '@/pages/databases/verification/ui/VerificationsComponent';
import { isMonitoringTabShown } from '@/pages/databases/wal-monitoring/lib/monitoringTab';
import { WalStreamMonitoringComponent } from '@/pages/databases/wal-monitoring/ui/WalStreamMonitoringComponent';
import { ApiError, useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useLatestRef } from '@/shared/hooks';
import { useNavigationIntent } from '@/shared/navigation';
import { LoadFailure } from '@/shared/ui';

interface Props {
  scrollContainerRef: RefObject<HTMLDivElement | null>;
  databaseId: string;
  workspaceId: string;
  currentTab: DatabaseTab;
  onTabChange: (tab: DatabaseTab, shouldReplace?: boolean) => void;
  onUnavailable: (databaseId: string) => void;
  onDatabaseChanged: (database: Database) => void;
  onDatabaseDeleted: (deletedDatabaseId: string) => void;
  onDatabaseCopied: (copiedDatabase: Database, sourceDatabaseId: string) => void;
  onDatabaseTransferred: (transferredDatabaseId: string) => void;
  isCanManageDBs: boolean;
}

export const DatabaseComponent = ({
  scrollContainerRef,
  databaseId,
  workspaceId,
  currentTab,
  onTabChange,
  onUnavailable,
  onDatabaseChanged,
  onDatabaseDeleted,
  onDatabaseCopied,
  onDatabaseTransferred,
  isCanManageDBs,
}: Props) => {
  const { t } = useTranslation();
  const isActive = useSessionGuard();
  const navigationIntent = useNavigationIntent();
  const latestNavigationIntent = useLatestRef(navigationIntent);

  const client = useQueryClient();
  const query = useSessionQuery({
    queryKey: ['database', databaseId],
    queryFn: async (signal) => {
      const requestNavigationIntent = latestNavigationIntent.current;
      const isCurrentRead = () =>
        !signal.aborted && isActive() && latestNavigationIntent.current === requestNavigationIntent;

      try {
        const database = await databaseApi.getDatabase(databaseId, { signal });
        if (!isCurrentRead()) throw new CancelledError({ revert: true });

        return database;
      } catch (error) {
        if (!isCurrentRead()) throw new CancelledError({ revert: true });

        throw error;
      }
    },
  });
  const detailNavigation = useRef(navigationIntent);
  const refreshNavigatedDetail = useEffectEvent(() => {
    if (query.isFetching) void query.refetch();
  });
  useLayoutEffect(() => {
    if (detailNavigation.current === navigationIntent) return;

    detailNavigation.current = navigationIntent;
    refreshNavigatedDetail();
  }, [navigationIntent]);
  const database = query.data?.workspaceId === workspaceId ? query.data : undefined;
  const loadError = query.error ?? undefined;
  const isUnavailable =
    (query.data && query.data.workspaceId !== workspaceId) ||
    (loadError instanceof ApiError && [403, 404].includes(loadError.status ?? 0));
  useEffect(() => {
    if (isUnavailable) onUnavailable(databaseId);
  }, [isUnavailable, databaseId, onUnavailable]);

  const acceptChangedDatabase = (changedDatabase: Database) => {
    if (isActive()) {
      void client.cancelQueries({ queryKey: query.queryKey, exact: true });
      client.setQueryData<Database>(query.queryKey, (shownDatabase) =>
        shownDatabase?.id === changedDatabase.id ? changedDatabase : shownDatabase,
      );
    }

    onDatabaseChanged(changedDatabase);
  };

  const isLogicalDatabase = database?.settings.type === DatabaseType.POSTGRES_LOGICAL;
  const isPhysicalDatabase = database?.settings.type === DatabaseType.POSTGRES_PHYSICAL;
  const isMonitoringShown = database !== undefined && isMonitoringTabShown(database);
  const shownTab = database ? getDatabaseTab(database, currentTab) : currentTab;
  useEffect(() => {
    if (database && currentTab !== shownTab) onTabChange(shownTab, true);
  }, [database, currentTab, shownTab, onTabChange]);

  if (isUnavailable) return <Loader role="status" />;

  if (loadError !== undefined) {
    return <LoadFailure error={loadError} onRetry={() => void query.refetch()} />;
  }

  if (!database) {
    return <Loader role="status" />;
  }

  return (
    <div className="w-full">
      <Tabs
        value={shownTab}
        onChange={(tab) => {
          if (tab && tab !== shownTab) onTabChange(tab as DatabaseTab);
        }}
        variant="outline"
        keepMounted={false}
      >
        <Tabs.List className="gap-2">
          <Tabs.Tab value="config">{t('databases.tabs.config')}</Tabs.Tab>
          <Tabs.Tab value="backups">{t('databases.tabs.backups')}</Tabs.Tab>
          {isLogicalDatabase && (
            <Tabs.Tab value="verifications">{t('databases.tabs.verifications')}</Tabs.Tab>
          )}
          {isMonitoringShown && (
            <Tabs.Tab value="monitoring">{t('databases.tabs.monitoring')}</Tabs.Tab>
          )}
        </Tabs.List>
        <Tabs.Panel value={shownTab}>
          {/* The card directly under the tabs joins the active tab, so it has a square corner there. */}
          <div className="[&>:first-child]:rounded-tl-none">
            {shownTab === 'config' && (
              <DatabaseConfigComponent
                database={database}
                onDatabaseChanged={acceptChangedDatabase}
                onDatabaseDeleted={onDatabaseDeleted}
                onDatabaseCopied={onDatabaseCopied}
                onDatabaseTransferred={onDatabaseTransferred}
                isCanManageDBs={isCanManageDBs}
              />
            )}

            {shownTab === 'backups' && (
              <>
                <HealthcheckAttemptsComponent database={database} />

                {isPhysicalDatabase ? (
                  <PhysicalBackupsComponent
                    database={database}
                    isCanManageDBs={isCanManageDBs}
                    scrollContainerRef={scrollContainerRef}
                  />
                ) : (
                  <LogicalBackupsComponent
                    database={database}
                    isCanManageDBs={isCanManageDBs}
                    scrollContainerRef={scrollContainerRef}
                    onNavigateToVerifications={() => onTabChange('verifications')}
                  />
                )}
              </>
            )}

            {shownTab === 'monitoring' && (
              <WalStreamMonitoringComponent
                database={database}
                isCanManageDBs={isCanManageDBs}
                onOpenConfig={() => onTabChange('config')}
              />
            )}

            {shownTab === 'verifications' && (
              <VerificationsComponent
                database={database}
                isCanManageDBs={isCanManageDBs}
                scrollContainerRef={scrollContainerRef}
              />
            )}
          </div>
        </Tabs.Panel>
      </Tabs>
    </div>
  );
};
