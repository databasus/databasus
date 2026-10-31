import { Button, Modal, Spin } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type BackupConfig,
  backupConfigApi,
  StorageSwitchHistoryComponent,
} from '../../../entities/backups/config';
import { databaseApi, sortUnavailableDatabasesFirst } from '../../../entities/databases';
import type { Database } from '../../../entities/databases';
import { type Storage, storageApi } from '../../../entities/storages';
import type { Workspace } from '../../../entities/workspaces';
import { useSessionGuard } from '../../../shared/auth';
import { useIsMobile } from '../../../shared/hooks';
import { translateApiError } from '../../../shared/i18n';
import { RequestRevision } from '../../../shared/lib/RequestRevision';
import { CreateDatabaseComponent } from './CreateDatabaseComponent';
import { DatabaseCardComponent } from './DatabaseCardComponent';
import { DatabaseComponent } from './DatabaseComponent';

interface Props {
  contentHeight: number;
  workspace: Workspace;
  isCanManageDBs: boolean;
}

// eslint-disable-next-line i18next/no-literal-string -- localStorage key
const SELECTED_DATABASE_STORAGE_KEY = 'selectedDatabaseId';

// A healthcheck can change a database's availability every minute, so the cards refresh twice as
// often to show the change within a minute.
const DATABASES_REFRESH_INTERVAL_MS = 30_000;

// eslint-disable-next-line i18next/no-literal-string -- DOM event name
const VISIBILITY_CHANGE_EVENT = 'visibilitychange';

const mapStoragesByDatabaseId = (
  backupConfigs: BackupConfig[],
  storages: Storage[],
): Map<string, Storage> => {
  const storagesById = new Map(storages.map((storage) => [storage.id, storage]));
  const storagesByDatabaseId = new Map<string, Storage>();

  for (const backupConfig of backupConfigs) {
    const storage = backupConfig.storageId ? storagesById.get(backupConfig.storageId) : undefined;
    if (storage) storagesByDatabaseId.set(backupConfig.databaseId, storage);
  }

  return storagesByDatabaseId;
};

export const DatabasesComponent = ({ contentHeight, workspace, isCanManageDBs }: Props) => {
  const [isLoading, setIsLoading] = useState(true);
  const [databases, setDatabases] = useState<Database[]>([]);
  const [storagesByDatabaseId, setStoragesByDatabaseId] = useState<Map<string, Storage>>(new Map());
  const [searchQuery, setSearchQuery] = useState('');

  const [isShowAddDatabase, setIsShowAddDatabase] = useState(false);
  const [hasConnectionError, setHasConnectionError] = useState(false);
  const [isWideStep, setIsWideStep] = useState(false);
  const [selectedDatabaseId, setSelectedDatabaseId] = useState<string | undefined>(undefined);

  const updateSelectedDatabaseId = (databaseId: string | undefined) => {
    if (!isCurrent()) return;
    setSelectedDatabaseId(databaseId);
    if (databaseId) {
      localStorage.setItem(`${SELECTED_DATABASE_STORAGE_KEY}_${workspace.id}`, databaseId);
    } else {
      localStorage.removeItem(`${SELECTED_DATABASE_STORAGE_KEY}_${workspace.id}`);
    }
  };

  const loadDatabases = (isSilent = false, selectDatabaseId?: string) => {
    if (!isCurrent()) return;
    const revision = loadRevision.current.Invalidate();
    if (!isSilent) {
      setIsLoading(true);
    }

    Promise.all([
      databaseApi.listAllDatabases(workspace.id),
      backupConfigApi.listAllWorkspaceBackupConfigs(workspace.id),
      storageApi.listAllStorages(workspace.id),
    ])
      .then(([listedDatabases, listedBackupConfigs, listedStorages]) => {
        if (!isCurrent() || !loadRevision.current.IsCurrent(revision)) return;
        const databases = sortUnavailableDatabasesFirst(listedDatabases);
        setDatabases(databases);
        setStoragesByDatabaseId(mapStoragesByDatabaseId(listedBackupConfigs, listedStorages));
        if (selectDatabaseId) {
          updateSelectedDatabaseId(selectDatabaseId);
        } else if (!selectedDatabaseId && !isSilent && !isMobile) {
          // On desktop, auto-select a database; on mobile, keep it unselected to show the list first
          const savedDatabaseId = localStorage.getItem(
            `${SELECTED_DATABASE_STORAGE_KEY}_${workspace.id}`,
          );
          const databaseToSelect =
            savedDatabaseId && databases.some((database) => database.id === savedDatabaseId)
              ? savedDatabaseId
              : databases[0]?.id;
          updateSelectedDatabaseId(databaseToSelect);
        }
      })
      .catch((e) => {
        // A background refresh retries on its own; an alert every 30 seconds would only get in the way.
        if (!isSilent && isCurrent() && loadRevision.current.IsCurrent(revision)) {
          alert(translateApiError(e, i18n.t));
        }
      })
      .finally(() => {
        if (isCurrent() && loadRevision.current.IsCurrent(revision)) setIsLoading(false);
      });
  };

  const { t, i18n } = useTranslation();
  const isMobile = useIsMobile();
  const isCurrent = useSessionGuard();
  const loadRevision = useRef(new RequestRevision());

  useEffect(() => {
    loadDatabases();

    const refreshIfVisible = () => {
      if (document.visibilityState !== 'hidden') {
        loadDatabases(true);
      }
    };

    const interval = setInterval(refreshIfVisible, DATABASES_REFRESH_INTERVAL_MS);
    document.addEventListener(VISIBILITY_CHANGE_EVENT, refreshIfVisible);

    return () => {
      loadRevision.current.Invalidate();
      clearInterval(interval);
      document.removeEventListener(VISIBILITY_CHANGE_EVENT, refreshIfVisible);
    };
  }, []);

  if (isLoading) {
    return (
      <div className="mx-3 my-3 flex w-[250px] justify-center">
        <Spin />
      </div>
    );
  }

  const addDatabaseButton = (
    <Button
      type="primary"
      className="mb-2 w-full"
      onClick={() => {
        setHasConnectionError(false);
        setIsShowAddDatabase(true);
      }}
    >
      {t('databases.list.addDatabase')}
    </Button>
  );

  const filteredDatabases = databases.filter((database) =>
    database.name.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  // On mobile, show either the list or the database details
  const isDatabaseListShown = !isMobile || !selectedDatabaseId;
  const isDatabaseDetailsShown = selectedDatabaseId && (!isMobile || selectedDatabaseId);

  return (
    <>
      <StorageSwitchHistoryComponent workspaceId={workspace.id} />
      <div className="flex grow">
        {isDatabaseListShown && (
          <div
            className="w-full overflow-y-auto md:mx-3 md:w-[250px] md:min-w-[250px] md:pr-2"
            style={{ height: contentHeight }}
          >
            {databases.length >= 5 && (
              <>
                {isCanManageDBs && addDatabaseButton}

                <div className="mb-2">
                  <input
                    placeholder={t('databases.list.searchPlaceholder')}
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="w-full border-b border-gray-300 p-1 text-gray-500 outline-none dark:text-gray-400"
                  />
                </div>
              </>
            )}

            {filteredDatabases.length > 0
              ? filteredDatabases.map((database) => (
                  <DatabaseCardComponent
                    key={database.id}
                    database={database}
                    storage={storagesByDatabaseId.get(database.id)}
                    selectedDatabaseId={selectedDatabaseId}
                    setSelectedDatabaseId={updateSelectedDatabaseId}
                  />
                ))
              : searchQuery && (
                  <div className="mb-4 text-center text-sm text-gray-500 dark:text-gray-400">
                    {t('databases.list.noSearchResults', { query: searchQuery })}
                  </div>
                )}

            {databases.length < 5 && isCanManageDBs && addDatabaseButton}

            <div className="mx-3 text-center text-xs text-gray-500 dark:text-gray-400">
              {t('databases.list.description')}
            </div>
          </div>
        )}

        {isDatabaseDetailsShown && (
          <div className="flex w-full flex-col md:flex-1">
            {isMobile && (
              <div className="mb-2">
                <Button
                  type="default"
                  onClick={() => updateSelectedDatabaseId(undefined)}
                  className="w-full"
                >
                  {t('databases.list.backToList')}
                </Button>
              </div>
            )}

            <DatabaseComponent
              contentHeight={isMobile ? contentHeight - 50 : contentHeight}
              databaseId={selectedDatabaseId}
              onDatabaseChanged={() => {
                loadDatabases();
              }}
              onDatabaseDeleted={() => {
                const remainingDatabases = databases.filter(
                  (database) => database.id !== selectedDatabaseId,
                );
                updateSelectedDatabaseId(remainingDatabases[0]?.id);
                loadDatabases();
              }}
              isCanManageDBs={isCanManageDBs}
            />
          </div>
        )}
      </div>

      {isShowAddDatabase && (
        <Modal
          title={t('databases.create.title')}
          footer={<div />}
          open={isShowAddDatabase}
          onCancel={() => setIsShowAddDatabase(false)}
          maskClosable={false}
          closable={!hasConnectionError}
          width={hasConnectionError || isWideStep ? 640 : 420}
        >
          <div className="mt-5" />

          <CreateDatabaseComponent
            workspaceId={workspace.id}
            onCreated={(databaseId) => {
              loadDatabases(false, databaseId);
              setIsShowAddDatabase(false);
            }}
            onClose={() => setIsShowAddDatabase(false)}
            onConnectionErrorChange={setHasConnectionError}
            onWideStepChange={setIsWideStep}
          />
        </Modal>
      )}
    </>
  );
};
