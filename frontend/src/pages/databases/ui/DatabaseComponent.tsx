import { Alert, Button, Spin } from 'antd';
import { type RefObject, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { LogicalBackupsComponent } from '@/features/backups/logical';
import {
  PhysicalBackupsComponent,
  WalStreamMonitoringComponent,
  isMonitoringTabShown,
} from '@/features/backups/physical';
import { VerificationsComponent } from '@/features/verification/runs';

import { type Database, DatabaseType, databaseApi } from '../../../entities/databases';
import { translateApiError } from '../../../shared/i18n';
import { DatabaseConfigComponent } from './DatabaseConfigComponent';
import { HealthcheckAttemptsComponent } from './healthcheck';

interface Props {
  scrollContainerRef: RefObject<HTMLDivElement | null>;
  databaseId: string;
  onDatabaseChanged: (database: Database) => void;
  onDatabaseDeleted: () => void;
  onDatabaseCopied: (copiedDatabase: Database) => void;
  onDatabaseTransferred: () => void;
  isCanManageDBs: boolean;
}

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
  const [currentTab, setCurrentTab] = useState<
    'config' | 'backups' | 'verifications' | 'monitoring'
  >('backups');

  const [database, setDatabase] = useState<Database | undefined>();
  const [loadError, setLoadError] = useState<unknown>();
  const [loadAttempt, setLoadAttempt] = useState(0);
  const [isHealthcheckVisible, setIsHealthcheckVisible] = useState(false);

  const handleHealthcheckVisibilityChange = (isVisible: boolean) => {
    setIsHealthcheckVisible(isVisible);
  };

  const acceptChangedDatabase = (changedDatabase: Database) => {
    setDatabase(changedDatabase);
    onDatabaseChanged(changedDatabase);
  };

  const isLogicalDatabase = database?.settings.type === DatabaseType.POSTGRES_LOGICAL;
  const isPhysicalDatabase = database?.settings.type === DatabaseType.POSTGRES_PHYSICAL;
  const isMonitoringShown = database !== undefined && isMonitoringTabShown(database);

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
      <Alert
        type="error"
        showIcon
        message={translateApiError(loadError, t)}
        action={
          <Button onClick={() => setLoadAttempt((attempt) => attempt + 1)}>
            {t('common.actions.retry')}
          </Button>
        }
      />
    );
  }

  if (!database) {
    return <Spin />;
  }

  return (
    <div className="w-full">
      <div className="flex">
        <div
          className={`mr-2 cursor-pointer rounded-tl-md rounded-tr-md px-6 py-2 ${currentTab === 'config' ? 'bg-surface' : 'bg-hover'}`}
          onClick={() => setCurrentTab('config')}
        >
          {t('databases.tabs.config')}
        </div>

        <div
          className={`mr-2 cursor-pointer rounded-tl-md rounded-tr-md px-6 py-2 ${currentTab === 'backups' ? 'bg-surface' : 'bg-hover'}`}
          onClick={() => setCurrentTab('backups')}
        >
          {t('databases.tabs.backups')}
        </div>

        {isLogicalDatabase && (
          <div
            className={`mr-2 cursor-pointer rounded-tl-md rounded-tr-md px-6 py-2 ${currentTab === 'verifications' ? 'bg-surface' : 'bg-hover'}`}
            onClick={() => setCurrentTab('verifications')}
          >
            {t('databases.tabs.verifications')}
          </div>
        )}

        {isMonitoringShown && (
          <div
            className={`mr-2 cursor-pointer rounded-tl-md rounded-tr-md px-6 py-2 ${currentTab === 'monitoring' ? 'bg-surface' : 'bg-hover'}`}
            onClick={() => setCurrentTab('monitoring')}
          >
            {t('databases.tabs.monitoring')}
          </div>
        )}
      </div>

      {currentTab === 'config' && (
        <DatabaseConfigComponent
          database={database}
          onDatabaseChanged={acceptChangedDatabase}
          onDatabaseDeleted={onDatabaseDeleted}
          onDatabaseCopied={onDatabaseCopied}
          onDatabaseTransferred={onDatabaseTransferred}
          isCanManageDBs={isCanManageDBs}
        />
      )}

      {currentTab === 'backups' && (
        <>
          <HealthcheckAttemptsComponent
            database={database}
            onVisibilityChange={handleHealthcheckVisibilityChange}
          />

          {isPhysicalDatabase ? (
            <PhysicalBackupsComponent
              database={database}
              isCanManageDBs={isCanManageDBs}
              isDirectlyUnderTab={!isHealthcheckVisible}
              scrollContainerRef={scrollContainerRef}
            />
          ) : (
            <LogicalBackupsComponent
              database={database}
              isCanManageDBs={isCanManageDBs}
              isDirectlyUnderTab={!isHealthcheckVisible}
              scrollContainerRef={scrollContainerRef}
              onNavigateToVerifications={() => setCurrentTab('verifications')}
            />
          )}
        </>
      )}

      {currentTab === 'monitoring' && isMonitoringShown && (
        <WalStreamMonitoringComponent
          database={database}
          isCanManageDBs={isCanManageDBs}
          isDirectlyUnderTab={true}
          onOpenConfig={() => setCurrentTab('config')}
        />
      )}

      {currentTab === 'verifications' && isLogicalDatabase && (
        <VerificationsComponent
          database={database}
          isCanManageDBs={isCanManageDBs}
          isDirectlyUnderTab={true}
          scrollContainerRef={scrollContainerRef}
        />
      )}
    </div>
  );
};
