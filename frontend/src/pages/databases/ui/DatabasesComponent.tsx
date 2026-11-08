import { Button, Loader, TextInput } from '@mantine/core';
import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type BackupConfig, backupConfigApi } from '@/entities/backup-configs';
import { databaseApi, sortUnavailableDatabasesFirst } from '@/entities/databases';
import type { Database } from '@/entities/databases';
import { type Storage, storageApi } from '@/entities/storages';
import type { Workspace } from '@/entities/workspaces';
import { CreateDatabaseComponent } from '@/pages/databases/ui/CreateDatabaseComponent';
import { DatabaseCardComponent } from '@/pages/databases/ui/DatabaseCardComponent';
import { DatabaseComponent } from '@/pages/databases/ui/DatabaseComponent';
import { ApiError, useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useIsMobile, useLatestRef, useScopedSelection } from '@/shared/hooks';
import { Dialog } from '@/shared/ui';
import { ResourceListDetail, LoadFailure } from '@/shared/ui';

interface Props {
  workspace: Workspace;
  onUnavailable?: (workspaceId: string) => void;
  isCanManageDBs: boolean;
}

interface DatabaseCollection {
  items: Database[];
  storagesByDatabaseId: Map<string, Storage>;
  isReady: boolean;
}

// A healthcheck can change a database's availability every minute, so the cards refresh twice as
// often to show the change within a minute.
const DATABASES_REFRESH_INTERVAL_MS = 30000;
// eslint-disable-next-line i18next/no-literal-string -- DOM event name
const VISIBILITY_CHANGE_EVENT = 'visibilitychange';

const createEmptyCollection = (): DatabaseCollection => ({
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

export const DatabasesComponent = ({ workspace, isCanManageDBs, onUnavailable }: Props) => {
  const client = useQueryClient();
  const [isUnavailable, setIsUnavailable] = useState(false);
  const [readError, setReadError] = useState<unknown>();
  const query = useSessionQuery({
    queryKey: ['database-collection', workspace.id],
    queryFn: async (signal): Promise<DatabaseCollection> => {
      setReadError(undefined);
      let hasReportedUnavailable = false;

      const handleWorkspaceReadError = (error: unknown): never => {
        if (!signal.aborted && isCurrent()) setReadError((current: unknown) => current ?? error);

        if (
          !hasReportedUnavailable &&
          !signal.aborted &&
          isCurrent() &&
          error instanceof ApiError &&
          [403, 404].includes(error.status ?? 0)
        ) {
          hasReportedUnavailable = true;
          setIsUnavailable(true);
          onUnavailable?.(workspace.id);
        }

        throw error;
      };
      const reads = [
        databaseApi.listAllDatabases(workspace.id, { signal }).catch(handleWorkspaceReadError),
        backupConfigApi
          .listAllWorkspaceBackupConfigs(workspace.id, { signal })
          .catch(handleWorkspaceReadError),
        storageApi.listAllStorages(workspace.id, { signal }).catch(handleWorkspaceReadError),
      ] as const;

      try {
        const [databases, configs, storages] = await Promise.all(reads);
        if (!signal.aborted && isCurrent()) setIsUnavailable(false);

        return {
          items: sortUnavailableDatabasesFirst(databases),
          storagesByDatabaseId: mapStoragesByDatabaseId(configs, storages),
          isReady: true,
        };
      } finally {
        // Keep sibling HTTP work cancellable after publishing the first failure.
        await Promise.allSettled(reads);
      }
    },
  });
  const collection = isUnavailable
    ? createEmptyCollection()
    : (query.data ?? createEmptyCollection());
  const loadError =
    !isUnavailable && !query.data ? (readError ?? query.error ?? undefined) : undefined;
  const isLoading = query.isPending && loadError === undefined;
  const setCollection = (update: (current: DatabaseCollection) => DatabaseCollection) => {
    void client.cancelQueries({ queryKey: query.queryKey, exact: true });
    client.setQueryData<DatabaseCollection>(query.queryKey, (current) =>
      update(current ?? createEmptyCollection()),
    );
  };
  const [searchQuery, setSearchQuery] = useState('');

  const [isShowAddDatabase, setIsShowAddDatabase] = useState(false);
  const [hasConnectionError, setHasConnectionError] = useState(false);
  const [isWideStep, setIsWideStep] = useState(false);

  const loadDatabases = (isSilent = false) => {
    if (!isCurrent() || (isSilent && query.isFetching)) return;
    void query.refetch({ cancelRefetch: !isSilent });
  };

  const acceptDatabase = (database: Database) => {
    if (!isCurrent() || database.workspaceId !== workspace.id) return;

    setCollection((current) => ({
      ...current,
      items: current.items.map((item) => (item.id === database.id ? database : item)),
    }));
    void query.refetch();
  };

  const addDatabaseToList = (database: Database) => {
    setCollection((current) => ({
      ...current,
      items: [...current.items.filter((item) => item.id !== database.id), database],
    }));
  };

  const acceptAddedDatabase = (database: Database) => {
    if (!isCurrent() || database.workspaceId !== workspace.id) return;

    addDatabaseToList(database);
    selection.select(database.id);
    void query.refetch();
  };

  const acceptCopiedDatabase = (copiedDatabase: Database, sourceDatabaseId: string) => {
    if (!isCurrent() || copiedDatabase.workspaceId !== workspace.id) return;

    addDatabaseToList(copiedDatabase);

    if (latestSelectedDatabaseId.current === sourceDatabaseId) selection.select(copiedDatabase.id);

    void query.refetch();
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
    if (!isCurrent()) return;

    if (latestSelectedDatabaseId.current === databaseId) selection.remove(databaseId);

    setCollection((current) => ({
      ...current,
      items: current.items.filter((item) => item.id !== databaseId),
    }));
    void query.refetch();
  };

  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const isCurrent = useSessionGuard();
  const detailScrollRef = useRef<HTMLDivElement>(null);
  const databaseCreatedInWizard = useRef<Database | undefined>(undefined);
  const databases = collection.items;
  const selection = useScopedSelection({
    // eslint-disable-next-line i18next/no-literal-string -- browser preference key
    scopeKey: `selectedDatabaseId_${workspace.id}`,
    itemIds: databases.map((database) => database.id),
    isReady: collection.isReady,
    isMobile,
  });
  const selectedDatabaseId = selection.selectedItemId;
  // A late answer runs a handler from the render that started the action, so its selection can be
  // stale.
  const latestSelectedDatabaseId = useLatestRef(selectedDatabaseId);

  const refreshIfVisible = useEffectEvent(() => {
    if (document.visibilityState !== 'hidden' && collection.isReady) loadDatabases(true);
  });

  useEffect(() => {
    if (!query.isSessionActive) return;

    const refresh = () => refreshIfVisible();
    const interval = setInterval(refresh, DATABASES_REFRESH_INTERVAL_MS);
    document.addEventListener(VISIBILITY_CHANGE_EVENT, refresh);

    return () => {
      clearInterval(interval);
      document.removeEventListener(VISIBILITY_CHANGE_EVENT, refresh);
    };
  }, [query.isSessionActive]);

  const filteredDatabases = databases.filter((database) =>
    database.name.toLowerCase().includes(searchQuery.toLowerCase()),
  );
  const addDatabaseButton = (
    <Button
      variant="filled"
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
            <TextInput
              aria-label={t('databases.list.searchPlaceholder')}
              placeholder={t('databases.list.searchPlaceholder')}
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
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
      {loadError !== undefined && <LoadFailure error={loadError} onRetry={() => loadDatabases()} />}

      {isLoading && !collection.isReady ? (
        <div className="my-3 mr-3 flex w-62.5 justify-center">
          <Loader role="status" />
        </div>
      ) : collection.isReady ? (
        <ResourceListDetail
          isMobile={isMobile}
          hasSelection={!!selectedDatabaseId}
          list={list}
          detailScrollRef={detailScrollRef}
          backAction={
            <Button
              className="!h-auto min-h-8 w-full !whitespace-normal"
              onClick={() => selection.select(undefined)}
            >
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

      {isShowAddDatabase && (
        <Dialog
          title={t('databases.create.title')}
          opened
          onClose={closeWizardAndAcceptCreatedDatabase}
          closeOnClickOutside={false}
          withCloseButton={!hasConnectionError}
          size={hasConnectionError || isWideStep ? 640 : 420}
        >
          <div className="mt-5" />

          <CreateDatabaseComponent
            workspaceId={workspace.id}
            onCreated={rememberDatabaseCreatedInWizard}
            onClose={closeWizardAndAcceptCreatedDatabase}
            onConnectionErrorChange={setHasConnectionError}
            onWideStepChange={setIsWideStep}
          />
          <div className="mt-5 flex flex-wrap justify-end gap-2">
            <div />
          </div>
        </Dialog>
      )}
    </div>
  );
};
