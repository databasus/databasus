import { Button, Modal, Spin } from 'antd';
import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Storage, storageApi } from '@/entities/storages';
import type { Workspace } from '@/entities/workspaces';
import { EditStorageComponent } from '@/features/edit-storage';
import { StorageCardComponent } from '@/pages/storages/ui/StorageCardComponent';
import { StorageComponent } from '@/pages/storages/ui/StorageComponent';
import { ApiError } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useIsMobile, useRequestRevision, useScopedSelection } from '@/shared/hooks';
import { ResourceListDetail, LoadFailure } from '@/shared/ui';

interface Props {
  workspace: Workspace;
  onUnavailable?: (workspaceId: string) => void;
  isCanManageStorages: boolean;
}

export const StoragesComponent = ({ workspace, isCanManageStorages, onUnavailable }: Props) => {
  const [collection, setCollection] = useState<{
    items: Storage[];
    isReady: boolean;
  }>({ items: [], isReady: false });
  const [loadError, setLoadError] = useState<unknown>();
  const [isLoading, setIsLoading] = useState(true);
  const [isShowAddStorage, setIsShowAddStorage] = useState(false);

  const loadStorages = (isSilent = false) => {
    if (!isCurrent()) return;

    const revision = loadRevision.Invalidate();

    if (!isSilent) setIsLoading(true);
    setLoadError(undefined);
    storageApi
      .listAllStorages(workspace.id)
      .then((items) => {
        if (!isCurrent() || !loadRevision.IsCurrent(revision)) return;
        setCollection({ items, isReady: true });
      })
      .catch((error) => {
        if (isCurrent() && loadRevision.IsCurrent(revision)) {
          setLoadError(error);

          if (error instanceof ApiError && [403, 404].includes(error.status ?? 0)) {
            setCollection({ items: [], isReady: false });
            onUnavailable?.(workspace.id);
          }
        }
      })
      .finally(() => {
        if (isCurrent() && loadRevision.IsCurrent(revision)) setIsLoading(false);
      });
  };

  const acceptStorage = (resource: Storage) => {
    if (!isCurrent() || resource.workspaceId !== workspace.id) return;

    setCollection((current) => ({
      isReady: true,
      items: [...current.items.filter((item) => item.id !== resource.id), resource],
    }));
    loadStorages(true);
  };

  const removeSelected = () => {
    if (!selectedStorageId) return;

    selection.remove(selectedStorageId);
    setCollection((current) => ({
      ...current,
      items: current.items.filter((item) => item.id !== selectedStorageId),
    }));
    loadStorages(true);
  };

  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const isCurrent = useSessionGuard();
  const loadRevision = useRequestRevision();
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

  const startStoragesLoad = useEffectEvent(() => {
    void loadStorages();
  });

  useEffect(() => {
    startStoragesLoad();
  }, []);

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
      type="primary"
      className="mb-2 w-full"
      onClick={() => setIsShowAddStorage(true)}
    >
      {t('storages.list.addStorage')}
    </Button>
  );
  const list = (
    <>
      {storages.length >= 5 && isCanManageStorages && addButton}
      {storages.map((resource) => (
        <StorageCardComponent
          key={resource.id}
          storage={resource}
          selectedStorageId={selectedStorageId}
          setSelectedStorageId={selection.select}
        />
      ))}
      {storages.length < 5 && isCanManageStorages && addButton}
      <div className="mx-3 text-center text-xs text-muted">{t('storages.list.description')}</div>
    </>
  );

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col">
      {loadError !== undefined && <LoadFailure error={loadError} onRetry={() => loadStorages()} />}
      {isLoading && !collection.isReady ? (
        <Spin />
      ) : collection.isReady ? (
        <ResourceListDetail
          isMobile={isMobile}
          hasSelection={!!selectedStorageId}
          list={list}
          backAction={
            <Button className="w-full" onClick={() => selection.select(undefined)}>
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
        <Modal
          title={t('storages.list.addStorage')}
          footer={null}
          open
          onCancel={() => setIsShowAddStorage(false)}
          maskClosable={false}
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
        </Modal>
      )}
    </div>
  );
};
