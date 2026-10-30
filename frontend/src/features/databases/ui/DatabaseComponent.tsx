import { Spin } from 'antd';
import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, DatabaseType, databaseApi } from '../../../entities/databases';
import { translateApiError } from '../../../shared/i18n';
import { LogicalBackupsComponent } from '../../backups/logical';
import {
  PhysicalBackupsComponent,
  WalStreamMonitoringComponent,
  isMonitoringTabShown,
} from '../../backups/physical';
import { VerificationsComponent } from '../../verification/runs';
import { DatabaseConfigComponent } from './DatabaseConfigComponent';
import { HealthcheckAttemptsComponent } from './healthcheck';

interface Props {
  contentHeight: number;
  databaseId: string;
  onDatabaseChanged: (database: Database) => void;
  onDatabaseDeleted: () => void;
  isCanManageDBs: boolean;
}

export const DatabaseComponent = ({
  contentHeight,
  databaseId,
  onDatabaseChanged,
  onDatabaseDeleted,
  isCanManageDBs,
}: Props) => {
  const { t } = useTranslation();
  const [currentTab, setCurrentTab] = useState<
    'config' | 'backups' | 'verifications' | 'monitoring'
  >('backups');

  const [database, setDatabase] = useState<Database | undefined>();
  const [editDatabase, setEditDatabase] = useState<Database | undefined>();

  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const [isHealthcheckVisible, setIsHealthcheckVisible] = useState(false);

  const handleHealthcheckVisibilityChange = (isVisible: boolean) => {
    setIsHealthcheckVisible(isVisible);
  };

  const isLogicalDatabase = database?.settings.type === DatabaseType.POSTGRES_LOGICAL;
  const isPhysicalDatabase = database?.settings.type === DatabaseType.POSTGRES_PHYSICAL;
  const isMonitoringShown = database !== undefined && isMonitoringTabShown(database);

  const showLoadError = useEffectEvent((error: unknown) => alert(translateApiError(error, t)));

  useEffect(() => {
    let isActive = true;
    setDatabase(undefined);
    setEditDatabase(undefined);
    databaseApi
      .getDatabase(databaseId)
      .then((loadedDatabase) => {
        if (isActive) setDatabase(loadedDatabase);
      })
      .catch((error) => {
        if (isActive) showLoadError(error);
      });
    return () => {
      isActive = false;
    };
  }, [databaseId]);

  if (!database) {
    return <Spin />;
  }

  return (
    <div
      className="w-full overflow-y-auto"
      style={{ maxHeight: contentHeight }}
      ref={scrollContainerRef}
    >
      <div className="flex">
        <div
          className={`mr-2 cursor-pointer rounded-tl-md rounded-tr-md px-6 py-2 ${currentTab === 'config' ? 'bg-white dark:bg-gray-800' : 'bg-gray-200 dark:bg-gray-700'}`}
          onClick={() => setCurrentTab('config')}
        >
          {t('databases.tabs.config')}
        </div>

        <div
          className={`mr-2 cursor-pointer rounded-tl-md rounded-tr-md px-6 py-2 ${currentTab === 'backups' ? 'bg-white dark:bg-gray-800' : 'bg-gray-200 dark:bg-gray-700'}`}
          onClick={() => setCurrentTab('backups')}
        >
          {t('databases.tabs.backups')}
        </div>

        {isLogicalDatabase && (
          <div
            className={`mr-2 cursor-pointer rounded-tl-md rounded-tr-md px-6 py-2 ${currentTab === 'verifications' ? 'bg-white dark:bg-gray-800' : 'bg-gray-200 dark:bg-gray-700'}`}
            onClick={() => setCurrentTab('verifications')}
          >
            {t('databases.tabs.verifications')}
          </div>
        )}

        {isMonitoringShown && (
          <div
            className={`mr-2 cursor-pointer rounded-tl-md rounded-tr-md px-6 py-2 ${currentTab === 'monitoring' ? 'bg-white dark:bg-gray-800' : 'bg-gray-200 dark:bg-gray-700'}`}
            onClick={() => setCurrentTab('monitoring')}
          >
            {t('databases.tabs.monitoring')}
          </div>
        )}
      </div>

      {currentTab === 'config' && (
        <DatabaseConfigComponent
          database={database}
          setDatabase={setDatabase}
          onDatabaseChanged={onDatabaseChanged}
          onDatabaseDeleted={onDatabaseDeleted}
          editDatabase={editDatabase}
          setEditDatabase={setEditDatabase}
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
