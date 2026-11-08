import { Button, Loader, TextInput } from '@mantine/core';
import { CancelledError, useQueryClient } from '@tanstack/react-query';
import { useEffect, useEffectEvent, useLayoutEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type BackupConfig, backupConfigApi } from '@/entities/backup-configs';
import { databaseApi, sortUnavailableDatabasesFirst, type Database } from '@/entities/databases';
import { type Storage, storageApi } from '@/entities/storages';
import type { Workspace } from '@/entities/workspaces';
import { CreateDatabaseComponent } from '@/pages/databases/create-wizard/ui/CreateDatabaseComponent';
import { getDatabaseTab } from '@/pages/databases/lib/getDatabaseTab';
import type { DatabaseDestination } from '@/pages/databases/model/DatabaseDestination';
import { useDatabaseNavigation } from '@/pages/databases/model/useDatabaseNavigation';
import { DatabaseCardComponent } from '@/pages/databases/ui/DatabaseCardComponent';
import { DatabaseComponent } from '@/pages/databases/ui/DatabaseComponent';
import { ApiError, useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useIsMobile, useLatestRef } from '@/shared/hooks';
import { getApplicationPath } from '@/shared/navigation';
import { Dialog, ResourceListDetail, LoadFailure } from '@/shared/ui';

interface Props {
  navigationIntent: string;
  workspace: Workspace;
  onUnavailable?: (workspaceId: string) => void;
  isCanManageDBs: boolean;
  selectedDatabaseId?: string;
  selectedTab?: string;
  onNavigate: (destination: DatabaseDestination) => void;
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

export const DatabasesComponent = ({
  navigationIntent,
  workspace,
  isCanManageDBs,
  onUnavailable,
  selectedDatabaseId: requestedDatabaseId,
  selectedTab,
  onNavigate,
}: Props) => {
  const client = useQueryClient();
  const [isUnavailable, setIsUnavailable] = useState(false);
  const [readError, setReadError] = useState<unknown>();
  const [rejectedDatabaseIds, setRejectedDatabaseIds] = useState<string[]>([]);
  const [reconcilingDatabaseId, setReconcilingDatabaseId] = useState<string>();
  const [reconciliationError, setReconciliationError] = useState<unknown>();
  const [resolutionIntent, setResolutionIntent] = useState(navigationIntent);
  if (resolutionIntent !== navigationIntent) {
    setResolutionIntent(navigationIntent);
    setRejectedDatabaseIds([]);
    setReconcilingDatabaseId(undefined);
    setReconciliationError(undefined);
  }
  const latestNavigationIntent = useLatestRef(navigationIntent);
  const query = useSessionQuery({
    queryKey: ['database-collection', workspace.id],
    queryFn: async (signal): Promise<DatabaseCollection> => {
      const requestNavigationIntent = latestNavigationIntent.current;
      const isCurrentRead = () =>
        !signal.aborted &&
        isCurrent() &&
        latestNavigationIntent.current === requestNavigationIntent;
      setReadError(undefined);
      let hasReportedUnavailable = false;

      const handleWorkspaceReadError = (error: unknown): never => {
        if (isCurrentRead()) setReadError((current: unknown) => current ?? error);

        if (
          !hasReportedUnavailable &&
          isCurrentRead() &&
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
        if (!isCurrentRead()) throw new CancelledError({ revert: true });

        setIsUnavailable(false);

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
  const collectionNavigation = useRef(navigationIntent);
  const refreshNavigatedCollection = useEffectEvent(() => {
    if (query.isFetching) void query.refetch();
  });
  useLayoutEffect(() => {
    if (collectionNavigation.current === navigationIntent) return;

    collectionNavigation.current = navigationIntent;
    refreshNavigatedCollection();
  }, [navigationIntent]);
  const collection = isUnavailable
    ? createEmptyCollection()
    : (query.data ?? createEmptyCollection());
  const loadError =
    !isUnavailable && !query.data ? (readError ?? query.error ?? undefined) : undefined;
  const currentReconciliationError = reconcilingDatabaseId
    ? (readError ?? reconciliationError)
    : undefined;
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

  const acceptAddedDatabase = (database: Database, canSelectDatabase: boolean) => {
    if (!isCurrent() || database.workspaceId !== workspace.id) return;

    addDatabaseToList(database);
    if (canSelectDatabase) onNavigate({ databaseId: database.id, tab: 'backups' });
    void query.refetch();
  };

  const acceptCopiedDatabase = (copiedDatabase: Database, sourceDatabaseId: string) => {
    if (!isCurrent() || copiedDatabase.workspaceId !== workspace.id) return;

    addDatabaseToList(copiedDatabase);

    if (
      latestSelectedDatabaseId.current === sourceDatabaseId &&
      selectionLifetime === latestSelectionLifetime.current
    )
      onNavigate({
        databaseId: copiedDatabase.id,
        tab: getDatabaseTab(copiedDatabase, latestSelectedTab.current),
      });

    void query.refetch();
  };

  const rememberDatabaseCreatedInWizard = (database: Database) => {
    databaseCreatedInWizard.current = database;
  };

  // Opening details before configuration is saved would cache the unfinished backup configuration.
  const closeWizardAndAcceptCreatedDatabase = () => {
    const database = databaseCreatedInWizard.current;
    const canSelectDatabase = wizardSelectionLifetime.current === latestSelectionLifetime.current;
    databaseCreatedInWizard.current = undefined;
    wizardSelectionLifetime.current = undefined;

    setIsShowAddDatabase(false);
    if (database) acceptAddedDatabase(database, canSelectDatabase);
  };

  const removeDatabase = (databaseId: string) => {
    if (!isCurrent()) return;

    if (latestSelectedDatabaseId.current === databaseId) {
      const replacementDatabase = databases.find((database) => database.id !== databaseId);
      onNavigate({
        databaseId: replacementDatabase?.id,
        tab: replacementDatabase
          ? getDatabaseTab(replacementDatabase, latestSelectedTab.current)
          : undefined,
        shouldReplace: true,
      });
    }

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
  const wizardSelectionLifetime = useRef<number | undefined>(undefined);
  const databases = collection.items.filter(
    (database) =>
      database.workspaceId === workspace.id && !rejectedDatabaseIds.includes(database.id),
  );
  const selection = useDatabaseNavigation({
    workspaceId: workspace.id,
    databases,
    requestedDatabaseId,
    requestedTab: selectedTab,
    isReady: collection.isReady && !reconcilingDatabaseId,
    isMobile,
    canSelectDefaultDatabase: !isShowAddDatabase,
    onNavigate,
  });
  const selectedDatabaseId = selection.database?.id;
  const [selectionScope, setSelectionScope] = useState(requestedDatabaseId);
  const [selectionLifetime, setSelectionLifetime] = useState(0);
  if (selectionScope !== requestedDatabaseId) {
    setSelectionScope(requestedDatabaseId);
    setSelectionLifetime(selectionLifetime + 1);
  }
  const latestSelectionLifetime = useLatestRef(selectionLifetime);
  const latestSelectedTab = useLatestRef(selection.tab);
  // A late answer runs a handler from the render that started the action, so its selection can be
  // stale.
  const latestSelectedDatabaseId = useLatestRef(selectedDatabaseId);
  const latestRequestedDatabaseId = useLatestRef(requestedDatabaseId);

  const reconcileUnavailableDatabase = async (databaseId: string) => {
    if (
      !isCurrent() ||
      latestRequestedDatabaseId.current !== databaseId ||
      latestNavigationIntent.current !== navigationIntent
    )
      return;

    setRejectedDatabaseIds((current) =>
      current.includes(databaseId) ? current : [...current, databaseId],
    );
    setReconcilingDatabaseId(databaseId);
    setReconciliationError(undefined);

    const reconciliation = await query.refetch();
    if (
      !isCurrent() ||
      latestRequestedDatabaseId.current !== databaseId ||
      latestNavigationIntent.current !== navigationIntent
    )
      return;

    if (reconciliation.error) setReconciliationError(reconciliation.error);
    else setReconcilingDatabaseId(undefined);
  };

  useEffect(() => {
    setReconcilingDatabaseId(undefined);
    setReconciliationError(undefined);
  }, [requestedDatabaseId]);

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
        wizardSelectionLifetime.current = latestSelectionLifetime.current;
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
              href={getApplicationPath({
                section: 'databases',
                workspaceId: workspace.id,
                databaseId: database.id,
                databaseTab: getDatabaseTab(database, selection.tab),
              })}
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
      {currentReconciliationError !== undefined && reconcilingDatabaseId && (
        <LoadFailure
          error={currentReconciliationError}
          onRetry={() => void reconcileUnavailableDatabase(reconcilingDatabaseId)}
        />
      )}

      {(isLoading && !collection.isReady) ||
      (collection.isReady && !selection.isResolved && currentReconciliationError === undefined) ? (
        <div className="my-3 mr-3 flex w-62.5 justify-center">
          <Loader role="status" />
        </div>
      ) : collection.isReady && selection.isResolved ? (
        <ResourceListDetail
          isMobile={isMobile}
          hasSelection={!!selectedDatabaseId}
          list={list}
          detailScrollRef={detailScrollRef}
          backAction={
            <Button
              className="!h-auto min-h-8 w-full !whitespace-normal"
              onClick={() => onNavigate({})}
            >
              {t('databases.list.backToList')}
            </Button>
          }
          detail={
            selectedDatabaseId && (
              <DatabaseComponent
                workspaceId={workspace.id}
                currentTab={selection.tab}
                onTabChange={(tab, shouldReplace) =>
                  onNavigate({ databaseId: selectedDatabaseId, tab, shouldReplace })
                }
                onUnavailable={reconcileUnavailableDatabase}
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
