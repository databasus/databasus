import { Alert, Button, Empty, Modal, Spin } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Storage, storageApi } from '@/entities/storages';
import type { Workspace } from '@/entities/workspaces';
import { EditStorageComponent } from '@/features/edit-storage';
import { ApiError } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useIsMobile, useScopedSelection } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';
import { RequestRevision } from '@/shared/lib/RequestRevision';
import { ResourceListDetail } from '@/shared/ui';

import { StorageCardComponent } from './StorageCardComponent';
import { StorageComponent } from './StorageComponent';

interface Props {
  workspace: Workspace;
  onUnavailable?: (workspaceId: string) => void;
  isCanManageStorages: boolean;
}

export const StoragesComponent = ({ workspace, isCanManageStorages, onUnavailable }: Props) => {
  const [collection, setCollection] = useState<{
    workspaceId: string;
    items: Storage[];
    isReady: boolean;
  }>({ workspaceId: workspace.id, items: [], isReady: false });
  const [loadError, setLoadError] = useState<unknown>();
  const [isLoading, setIsLoading] = useState(true);
  const [isShowAddStorage, setIsShowAddStorage] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  const loadStorages = (isSilent = false) => {
    if (!isCurrent()) return;

    const revision = loadRevision.current.Invalidate();
    const workspaceId = workspace.id;

    if (!isSilent) setIsLoading(true);
    setLoadError(undefined);
    storageApi
      .listAllStorages(workspaceId)
      .then((items) => {
        if (!isCurrent() || !loadRevision.current.IsCurrent(revision)) return;
        setCollection({ workspaceId, items, isReady: true });
      })
      .catch((error) => {
        if (isCurrent() && loadRevision.current.IsCurrent(revision)) {
          setLoadError(error);

          if (error instanceof ApiError && [403, 404].includes(error.status ?? 0)) {
            setCollection({ workspaceId: workspace.id, items: [], isReady: false });
            onUnavailable?.(workspace.id);
          }
        }
      })
      .finally(() => {
        if (isCurrent() && loadRevision.current.IsCurrent(revision)) setIsLoading(false);
      });
  };

  const acceptStorage = (resource: Storage) => {
    if (!isCurrent() || resource.workspaceId !== workspace.id) return;

    setCollection((current) => ({
      workspaceId: workspace.id,
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
  const loadRevision = useRef(new RequestRevision());
  const addButtonRef = useRef<HTMLButtonElement>(null);
  const storages = collection.workspaceId === workspace.id ? collection.items : [];
  const selection = useScopedSelection({
    // eslint-disable-next-line i18next/no-literal-string -- browser preference key
    scopeKey: `selectedStorageId_${workspace.id}`,
    itemIds: storages.map((resource) => resource.id),
    isReady: collection.workspaceId === workspace.id && collection.isReady,
    isMobile,
  });
  const selectedStorageId = selection.selectedItemId;

  useEffect(() => {
    setCollection({ workspaceId: workspace.id, items: [], isReady: false });
    setIsShowAddStorage(false);
    setSearchQuery('');
    loadStorages();

    return () => {
      loadRevision.current.Invalidate();
    };
  }, [workspace.id]);

  useEffect(() => {
    if (!isShowAddStorage) return;

    return () => {
      addButtonRef.current?.focus();
    };
  }, [isShowAddStorage]);

  const filteredStorages = storages.filter((resource) =>
    resource.name.toLowerCase().includes(searchQuery.toLowerCase()),
  );
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
      {collection.isReady && storages.length === 0 && (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('storages.list.empty')} />
      )}
      {storages.length >= 5 && isCanManageStorages && addButton}
      {filteredStorages.map((resource) => (
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
      {loadError !== undefined && (
        <Alert
          type="error"
          showIcon
          message={translateApiError(loadError, t)}
          action={<Button onClick={() => loadStorages()}>{t('common.actions.retry')}</Button>}
        />
      )}
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
                key={`${workspace.id}/${selectedStorageId}`}
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
      {isShowAddStorage && collection.workspaceId === workspace.id && (
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
