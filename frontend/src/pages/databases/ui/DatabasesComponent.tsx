import { Alert, Button, Modal, Spin } from 'antd';
import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type BackupConfig,
  backupConfigApi,
  StorageSwitchHistoryComponent,
} from '../../../entities/backup-configs';
import { databaseApi, sortUnavailableDatabasesFirst } from '../../../entities/databases';
import type { Database } from '../../../entities/databases';
import { type Storage, storageApi } from '../../../entities/storages';
import type { Workspace } from '../../../entities/workspaces';
import { useSessionGuard } from '../../../shared/auth';
import {
  useIsMobile,
  useLatestRef,
  useRequestRevision,
  useScopedSelection,
} from '../../../shared/hooks';
import { translateApiError } from '../../../shared/i18n';
import { ResourceListDetail } from '../../../shared/ui';
import { CreateDatabaseComponent } from './CreateDatabaseComponent';
import { DatabaseCardComponent } from './DatabaseCardComponent';
import { DatabaseComponent } from './DatabaseComponent';

interface Props {
  workspace: Workspace;
  isCanManageDBs: boolean;
}

interface DatabaseCollection {
  workspaceId: string;
  items: Database[];
  storagesByDatabaseId: Map<string, Storage>;
  isReady: boolean;
}

// A healthcheck can change a database's availability every minute, so the cards refresh twice as
// often to show the change within a minute.
const DATABASES_REFRESH_INTERVAL_MS = 30_000;

// eslint-disable-next-line i18next/no-literal-string -- DOM event name
const VISIBILITY_CHANGE_EVENT = 'visibilitychange';

const createEmptyCollection = (workspaceId: string): DatabaseCollection => ({
  workspaceId,
  items: [],
  storagesByDatabaseId: new Map(),
  isReady: false,
});

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

export const DatabasesComponent = ({ workspace, isCanManageDBs }: Props) => {
  const [collection, setCollection] = useState(() => createEmptyCollection(workspace.id));
  const [loadError, setLoadError] = useState<unknown>();
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');

  const [isShowAddDatabase, setIsShowAddDatabase] = useState(false);
  const [hasConnectionError, setHasConnectionError] = useState(false);
  const [isWideStep, setIsWideStep] = useState(false);

  const loadDatabases = (isSilent = false) => {
    if (!isCurrent()) return;

    const revision = loadRevision.Invalidate();
    const workspaceId = workspace.id;

    if (!isSilent) {
      setIsLoading(true);
      setLoadError(undefined);
    }

    Promise.all([
      databaseApi.listAllDatabases(workspaceId),
      backupConfigApi.listAllWorkspaceBackupConfigs(workspaceId),
      storageApi.listAllStorages(workspaceId),
    ])
      .then(([listedDatabases, listedBackupConfigs, listedStorages]) => {
        if (!isCurrent() || !loadRevision.IsCurrent(revision)) return;

        setCollection({
          workspaceId,
          items: sortUnavailableDatabasesFirst(listedDatabases),
          storagesByDatabaseId: mapStoragesByDatabaseId(listedBackupConfigs, listedStorages),
          isReady: true,
        });
        setLoadError(undefined);
      })
      .catch((error) => {
        // A background refresh retries on its own, so it keeps the shown list without feedback.
        if (!isSilent && isCurrent() && loadRevision.IsCurrent(revision)) {
          setLoadError(error);
        }
      })
      .finally(() => {
        if (isCurrent() && loadRevision.IsCurrent(revision)) setIsLoading(false);
      });
  };

  const isCurrentWorkspaceAnswer = () => isCurrent() && latestWorkspaceId.current === workspace.id;

  const acceptDatabase = (database: Database) => {
    if (!isCurrentWorkspaceAnswer() || database.workspaceId !== workspace.id) return;

    setCollection((current) => ({
      ...current,
      items: current.items.map((item) => (item.id === database.id ? database : item)),
    }));
    loadDatabases(true);
  };

  const addDatabaseToList = (database: Database) => {
    setCollection((current) => ({
      ...current,
      items: [...current.items.filter((item) => item.id !== database.id), database],
    }));
  };

  const acceptAddedDatabase = (database: Database) => {
    if (!isCurrentWorkspaceAnswer() || database.workspaceId !== workspace.id) return;

    addDatabaseToList(database);
    selection.select(database.id);
    loadDatabases(true);
  };

  const acceptCopiedDatabase = (copiedDatabase: Database, sourceDatabaseId: string) => {
    if (!isCurrentWorkspaceAnswer() || copiedDatabase.workspaceId !== workspace.id) return;

    addDatabaseToList(copiedDatabase);

    if (latestSelectedDatabaseId.current === sourceDatabaseId) selection.select(copiedDatabase.id);

    loadDatabases(true);
  };

  const rememberDatabaseCreatedInWizard = (database: Database) => {
    databaseCreatedInWizard.current = database;
  };

  // Selecting at creation would mount the screen behind the wizard, and its single configuration read
  // would then show the backups as disabled, so the database joins the list when the wizard closes.
  const closeWizardAndAcceptCreatedDatabase = () => {
    const database = databaseCreatedInWizard.current;
    databaseCreatedInWizard.current = undefined;

    setIsShowAddDatabase(false);
    if (database) acceptAddedDatabase(database);
  };

  const removeDatabase = (databaseId: string) => {
    if (!isCurrentWorkspaceAnswer()) return;

    if (latestSelectedDatabaseId.current === databaseId) selection.remove(databaseId);

    setCollection((current) => ({
      ...current,
      items: current.items.filter((item) => item.id !== databaseId),
    }));
    loadDatabases(true);
  };

  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const isCurrent = useSessionGuard();
  const loadRevision = useRequestRevision();
  const detailScrollRef = useRef<HTMLDivElement>(null);
  const databaseCreatedInWizard = useRef<Database | undefined>(undefined);
  const databases = collection.workspaceId === workspace.id ? collection.items : [];
  const selection = useScopedSelection({
    // eslint-disable-next-line i18next/no-literal-string -- browser preference key
    scopeKey: `selectedDatabaseId_${workspace.id}`,
    itemIds: databases.map((database) => database.id),
    isReady: collection.workspaceId === workspace.id && collection.isReady,
    isMobile,
  });
  const selectedDatabaseId = selection.selectedItemId;
  // A late answer runs a handler from the render that started the action, so its selection and
  // workspace can be stale.
  const latestSelectedDatabaseId = useLatestRef(selectedDatabaseId);
  const latestWorkspaceId = useLatestRef(workspace.id);

  const refreshIfVisible = useEffectEvent(() => {
    if (document.visibilityState !== 'hidden' && collection.isReady) loadDatabases(true);
  });

  const startDatabasesLoad = useEffectEvent(() => {
    loadDatabases();
  });

  useEffect(() => {
    setCollection(createEmptyCollection(workspace.id));
    setIsShowAddDatabase(false);
    databaseCreatedInWizard.current = undefined;
    setSearchQuery('');
    startDatabasesLoad();

    const refresh = () => refreshIfVisible();
    const interval = setInterval(refresh, DATABASES_REFRESH_INTERVAL_MS);
    document.addEventListener(VISIBILITY_CHANGE_EVENT, refresh);

    return () => {
      loadRevision.Invalidate();
      clearInterval(interval);
      document.removeEventListener(VISIBILITY_CHANGE_EVENT, refresh);
    };
  }, [workspace.id, loadRevision]);

  const filteredDatabases = databases.filter((database) =>
    database.name.toLowerCase().includes(searchQuery.toLowerCase()),
  );
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
  const list = (
    <>
      {databases.length >= 5 && (
        <>
          {isCanManageDBs && addDatabaseButton}

          <div className="mb-2">
            <input
              placeholder={t('databases.list.searchPlaceholder')}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full border-b border-border p-1 text-muted"
            />
          </div>
        </>
      )}

      {filteredDatabases.length > 0
        ? filteredDatabases.map((database) => (
            <DatabaseCardComponent
              key={database.id}
              database={database}
              storage={collection.storagesByDatabaseId.get(database.id)}
              selectedDatabaseId={selectedDatabaseId}
              setSelectedDatabaseId={selection.select}
            />
          ))
        : searchQuery && (
            <div className="mb-4 text-center text-sm text-muted">
              {t('databases.list.noSearchResults', { query: searchQuery })}
            </div>
          )}

      {databases.length < 5 && isCanManageDBs && addDatabaseButton}

      <div className="mx-3 text-center text-xs text-muted">{t('databases.list.description')}</div>
    </>
  );

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col">
      <StorageSwitchHistoryComponent workspaceId={workspace.id} />

      {loadError !== undefined && (
        <Alert
          type="error"
          showIcon
          message={translateApiError(loadError, t)}
          action={<Button onClick={() => loadDatabases()}>{t('common.actions.retry')}</Button>}
        />
      )}

      {isLoading && !collection.isReady ? (
        <div className="mx-3 my-3 flex w-62.5 justify-center">
          <Spin />
        </div>
      ) : collection.isReady ? (
        <ResourceListDetail
          isMobile={isMobile}
          hasSelection={!!selectedDatabaseId}
          list={list}
          detailScrollRef={detailScrollRef}
          backAction={
            <Button className="w-full" onClick={() => selection.select(undefined)}>
              {t('databases.list.backToList')}
            </Button>
          }
          detail={
            selectedDatabaseId && (
              <DatabaseComponent
                scrollContainerRef={detailScrollRef}
                databaseId={selectedDatabaseId}
                onDatabaseChanged={acceptDatabase}
                onDatabaseDeleted={removeDatabase}
                onDatabaseCopied={acceptCopiedDatabase}
                onDatabaseTransferred={removeDatabase}
                isCanManageDBs={isCanManageDBs}
              />
            )
          }
        />
      ) : null}

      {isShowAddDatabase && collection.workspaceId === workspace.id && (
        <Modal
          title={t('databases.create.title')}
          footer={<div />}
          open
          onCancel={closeWizardAndAcceptCreatedDatabase}
          maskClosable={false}
          closable={!hasConnectionError}
          width={hasConnectionError || isWideStep ? 640 : 420}
        >
          <div className="mt-5" />

          <CreateDatabaseComponent
            workspaceId={workspace.id}
            onCreated={rememberDatabaseCreatedInWizard}
            onClose={closeWizardAndAcceptCreatedDatabase}
            onConnectionErrorChange={setHasConnectionError}
            onWideStepChange={setIsWideStep}
          />
        </Modal>
      )}
    </div>
  );
};
