import { Spin, Tabs } from 'antd';
import { type RefObject, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, DatabaseType, databaseApi } from '../../../entities/databases';
import { useSessionGuard } from '../../../shared/auth';
import { LoadFailure } from '../../../shared/ui';
import { isMonitoringTabShown } from '../lib/wal-stream-monitoring/monitoringTab';
import { DatabaseConfigComponent } from './DatabaseConfigComponent';
import { HealthcheckAttemptsComponent } from './healthcheck';
import { LogicalBackupsComponent } from './logical-backups/LogicalBackupsComponent';
import { PhysicalBackupsComponent } from './physical-backups/PhysicalBackupsComponent';
import { VerificationsComponent } from './verification/VerificationsComponent';
import { WalStreamMonitoringComponent } from './wal-stream-monitoring/WalStreamMonitoringComponent';

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

  const [database, setDatabase] = useState<Database | undefined>();
  const [loadError, setLoadError] = useState<unknown>();
  const [loadAttempt, setLoadAttempt] = useState(0);

  const acceptChangedDatabase = (changedDatabase: Database) => {
    if (isActive()) {
      setDatabase((shownDatabase) =>
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

  useEffect(() => {
    let isActive = true;
    setDatabase(undefined);
    setLoadError(undefined);
    databaseApi
      .getDatabase(databaseId)
      .then((loadedDatabase) => {
        if (isActive) setDatabase(loadedDatabase);
      })
      .catch((error) => {
        if (isActive) setLoadError(error);
      });
    return () => {
      isActive = false;
    };
  }, [databaseId, loadAttempt]);

  if (loadError !== undefined) {
    return (
      <LoadFailure error={loadError} onRetry={() => setLoadAttempt((attempt) => attempt + 1)} />
    );
  }

  if (!database) {
    return <Spin />;
  }

  return (
    <div className="w-full">
      <Tabs
        className="[&_.ant-tabs-tab-btn]:!font-normal [&_.ant-tabs-tab-btn]:!text-content [&_.ant-tabs-tab-btn]:![text-shadow:none]"
        type="card"
        tabBarGutter={8}
        tabBarStyle={{ marginBottom: 0 }}
        activeKey={shownTab}
        onChange={(tab) => setCurrentTab(tab as DatabaseTab)}
        items={[
          { key: 'config', label: t('databases.tabs.config') },
          { key: 'backups', label: t('databases.tabs.backups') },
          ...(isLogicalDatabase
            ? [{ key: 'verifications', label: t('databases.tabs.verifications') }]
            : []),
          ...(isMonitoringShown
            ? [{ key: 'monitoring', label: t('databases.tabs.monitoring') }]
            : []),
        ]}
      />

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
    </div>
  );
};
