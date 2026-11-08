import { Loader, Tabs } from '@mantine/core';
import { useQueryClient } from '@tanstack/react-query';
import { type RefObject, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, DatabaseType, databaseApi } from '@/entities/databases';
import { HealthcheckAttemptsComponent } from '@/pages/databases/healthcheck/ui';
import { LogicalBackupsComponent } from '@/pages/databases/logical-backups/ui/LogicalBackupsComponent';
import { PhysicalBackupsComponent } from '@/pages/databases/physical-backups/ui/PhysicalBackupsComponent';
import { DatabaseConfigComponent } from '@/pages/databases/settings/ui/DatabaseConfigComponent';
import { VerificationsComponent } from '@/pages/databases/verification/ui/VerificationsComponent';
import { isMonitoringTabShown } from '@/pages/databases/wal-monitoring/lib/monitoringTab';
import { WalStreamMonitoringComponent } from '@/pages/databases/wal-monitoring/ui/WalStreamMonitoringComponent';
import { useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { LoadFailure } from '@/shared/ui';

interface Props {
  scrollContainerRef: RefObject<HTMLDivElement | null>;
  databaseId: string;
  onDatabaseChanged: (database: Database) => void;
  onDatabaseDeleted: (deletedDatabaseId: string) => void;
  onDatabaseCopied: (copiedDatabase: Database, sourceDatabaseId: string) => void;
  onDatabaseTransferred: (transferredDatabaseId: string) => void;
  isCanManageDBs: boolean;
}

type DatabaseTab = 'config' | 'backups' | 'verifications' | 'monitoring';

export const DatabaseComponent = ({
  scrollContainerRef,
  databaseId,
  onDatabaseChanged,
  onDatabaseDeleted,
  onDatabaseCopied,
  onDatabaseTransferred,
  isCanManageDBs,
}: Props) => {
  const { t } = useTranslation();
  const isActive = useSessionGuard();
  const [currentTab, setCurrentTab] = useState<DatabaseTab>('backups');

  const client = useQueryClient();
  const query = useSessionQuery({
    queryKey: ['database', databaseId],
    queryFn: (signal) => databaseApi.getDatabase(databaseId, { signal }),
  });
  const database = query.data;
  const loadError = query.error ?? undefined;

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
  const isCurrentTabOffered =
    (currentTab !== 'verifications' || isLogicalDatabase) &&
    (currentTab !== 'monitoring' || isMonitoringShown);
  const shownTab = isCurrentTabOffered ? currentTab : 'backups';

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
          if (tab) setCurrentTab(tab as DatabaseTab);
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
                    onNavigateToVerifications={() => setCurrentTab('verifications')}
                  />
                )}
              </>
            )}

            {shownTab === 'monitoring' && (
              <WalStreamMonitoringComponent
                database={database}
                isCanManageDBs={isCanManageDBs}
                onOpenConfig={() => setCurrentTab('config')}
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
