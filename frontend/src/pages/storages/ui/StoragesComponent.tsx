import { Button, Loader } from '@mantine/core';
import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Storage, storageApi } from '@/entities/storages';
import type { Workspace } from '@/entities/workspaces';
import { EditStorageComponent } from '@/features/edit-storage';
import { StorageCardComponent } from '@/pages/storages/ui/StorageCardComponent';
import { StorageComponent } from '@/pages/storages/ui/StorageComponent';
import { ApiError } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useIsMobile, useResourceCollection, useScopedSelection } from '@/shared/hooks';
import { Dialog } from '@/shared/ui';
import { ResourceCollectionList, ResourceListDetail, LoadFailure } from '@/shared/ui';

interface Props {
  workspace: Workspace;
  onUnavailable?: (workspaceId: string) => void;
  isCanManageStorages: boolean;
}

export const StoragesComponent = ({ workspace, isCanManageStorages, onUnavailable }: Props) => {
  const [isShowAddStorage, setIsShowAddStorage] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const collection = useResourceCollection({
    queryKey: ['storages', workspace.id],
    read: (signal) => storageApi.listAllStorages(workspace.id, { signal }),
    onError: (error) => {
      if (error instanceof ApiError && [403, 404].includes(error.status ?? 0)) {
        collection.clear();
        onUnavailable?.(workspace.id);
      }
    },
  });
  const loadStorages = collection.reload;
  const { loadError, isLoading } = collection;

  const acceptStorage = (resource: Storage) => {
    if (!isCurrent() || resource.workspaceId !== workspace.id) return;

    collection.accept(resource);
    loadStorages(true);
  };

  const removeSelected = () => {
    if (!selectedStorageId) return;

    selection.remove(selectedStorageId);
    collection.remove(selectedStorageId);
    loadStorages(true);
  };

  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const isCurrent = useSessionGuard();
  const addButtonRef = useRef<HTMLButtonElement>(null);
  const storages = collection.items;
  const selection = useScopedSelection({
    // eslint-disable-next-line i18next/no-literal-string -- browser preference key
    scopeKey: `selectedStorageId_${workspace.id}`,
    itemIds: storages.map((resource) => resource.id),
    isReady: collection.isReady,
    isMobile,
  });
  const selectedStorageId = selection.selectedItemId;

  // Read when the form closes, because the button can appear after the form opened.
  const returnFocusToButton = useEffectEvent(() => {
    addButtonRef.current?.focus();
  });

  useEffect(() => {
    if (!isShowAddStorage) return;

    return () => {
      returnFocusToButton();
    };
  }, [isShowAddStorage]);

  const addButton = (
    <Button
      ref={addButtonRef}
      variant="filled"
      className="mb-2 !h-auto min-h-8 w-full !whitespace-normal"
      onClick={() => setIsShowAddStorage(true)}
    >
      {t('storages.list.addStorage')}
    </Button>
  );
  const list = (
    <ResourceCollectionList
      items={storages}
      getKey={(resource) => resource.id}
      getName={(resource) => resource.name}
      renderCard={(resource) => (
        <StorageCardComponent
          storage={resource}
          selectedStorageId={selectedStorageId}
          setSelectedStorageId={selection.select}
        />
      )}
      addAction={isCanManageStorages && addButton}
      description={t('storages.list.description')}
      searchLabel={t('storages.list.searchPlaceholder')}
      searchQuery={searchQuery}
      onSearchChange={setSearchQuery}
      noSearchResults={(query) => t('storages.list.noSearchResults', { query })}
    />
  );

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col">
      {loadError !== undefined && <LoadFailure error={loadError} onRetry={() => loadStorages()} />}
      {isLoading && !collection.isReady ? (
        <Loader role="status" />
      ) : collection.isReady ? (
        <ResourceListDetail
          isMobile={isMobile}
          hasSelection={!!selectedStorageId}
          list={list}
          backAction={
            <Button
              className="!h-auto min-h-8 w-full !whitespace-normal"
              onClick={() => selection.select(undefined)}
            >
              {t('storages.list.backToList')}
            </Button>
          }
          detail={
            selectedStorageId && (
              <StorageComponent
                key={selectedStorageId}
                storageId={selectedStorageId}
                onUnavailable={() => loadStorages(true)}
                onStorageChanged={acceptStorage}
                onStorageDeleted={removeSelected}
                onStorageTransferred={removeSelected}
                isCanManageStorages={isCanManageStorages}
              />
            )
          }
        />
      ) : null}
      {isShowAddStorage && (
        <Dialog
          title={t('storages.list.addStorage')}
          opened
          onClose={() => setIsShowAddStorage(false)}
          closeOnClickOutside={false}
        >
          <p className="my-3 text-muted">{t('storages.list.description')}</p>
          <EditStorageComponent
            workspaceId={workspace.id}
            isShowName
            isShowClose={false}
            onClose={() => setIsShowAddStorage(false)}
            onPersisted={acceptStorage}
            onChanged={(resource) => {
              acceptStorage(resource);
              selection.select(resource.id);
              setIsShowAddStorage(false);
            }}
          />
        </Dialog>
      )}
    </div>
  );
};
