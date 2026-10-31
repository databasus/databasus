import { Button, Modal, Spin } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Storage, storageApi } from '../../../entities/storages';
import type { Workspace } from '../../../entities/workspaces';
import { useSessionGuard } from '../../../shared/auth';
import { useIsMobile } from '../../../shared/hooks';
import { translateApiError } from '../../../shared/i18n';
import { RequestRevision } from '../../../shared/lib/RequestRevision';
import { EditStorageComponent } from './edit/EditStorageComponent';
import { StorageCardComponent } from './StorageCardComponent';
import { StorageComponent } from './StorageComponent';

interface Props {
  contentHeight: number;
  workspace: Workspace;
  isCanManageStorages: boolean;
}

// eslint-disable-next-line i18next/no-literal-string -- localStorage key
const SELECTED_STORAGE_STORAGE_KEY = 'selectedStorageId';

export const StoragesComponent = ({ contentHeight, workspace, isCanManageStorages }: Props) => {
  const [isLoading, setIsLoading] = useState(true);
  const [storages, setStorages] = useState<Storage[]>([]);

  const [isShowAddStorage, setIsShowAddStorage] = useState(false);
  const [selectedStorageId, setSelectedStorageId] = useState<string | undefined>(undefined);

  const updateSelectedStorageId = (storageId: string | undefined) => {
    if (!isCurrent()) return;
    setSelectedStorageId(storageId);
    if (storageId) {
      localStorage.setItem(`${SELECTED_STORAGE_STORAGE_KEY}_${workspace.id}`, storageId);
    } else {
      localStorage.removeItem(`${SELECTED_STORAGE_STORAGE_KEY}_${workspace.id}`);
    }
  };

  const loadStorages = (isSilent = false, selectStorageId?: string) => {
    if (!isCurrent()) return;
    const revision = loadRevision.current.Invalidate();
    if (!isSilent) {
      setIsLoading(true);
    }

    storageApi
      .listAllStorages(workspace.id)
      .then((storages: Storage[]) => {
        if (!isCurrent() || !loadRevision.current.IsCurrent(revision)) return;
        setStorages(storages);
        if (selectStorageId) {
          updateSelectedStorageId(selectStorageId);
        } else if (!selectedStorageId && !isSilent && !isMobile) {
          // On desktop, auto-select a storage; on mobile, keep it unselected to show the list first
          const savedStorageId = localStorage.getItem(
            `${SELECTED_STORAGE_STORAGE_KEY}_${workspace.id}`,
          );
          const storageToSelect =
            savedStorageId && storages.some((storage) => storage.id === savedStorageId)
              ? savedStorageId
              : storages[0]?.id;
          updateSelectedStorageId(storageToSelect);
        }
      })
      .catch((e: unknown) => {
        if (isCurrent() && loadRevision.current.IsCurrent(revision))
          alert(translateApiError(e, i18n.t));
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
    loadStorages();
    return () => {
      loadRevision.current.Invalidate();
    };
  }, []);

  if (isLoading) {
    return (
      <div className="mx-3 my-3 flex w-[250px] justify-center">
        <Spin />
      </div>
    );
  }

  const addStorageButton = (
    <Button type="primary" className="mb-2 w-full" onClick={() => setIsShowAddStorage(true)}>
      {t('storages.list.addStorage')}
    </Button>
  );

  // On mobile, show either the list or the storage details
  const isStorageListShown = !isMobile || !selectedStorageId;
  const isStorageDetailsShown = selectedStorageId && (!isMobile || selectedStorageId);

  return (
    <>
      <div className="flex grow">
        {isStorageListShown && (
          <div
            className="w-full overflow-y-auto md:mx-3 md:w-[250px] md:min-w-[250px] md:pr-2"
            style={{ height: contentHeight }}
          >
            {storages.length >= 5 && isCanManageStorages && addStorageButton}

            {storages.map((storage) => (
              <StorageCardComponent
                key={storage.id}
                storage={storage}
                selectedStorageId={selectedStorageId}
                setSelectedStorageId={updateSelectedStorageId}
              />
            ))}

            {storages.length < 5 && isCanManageStorages && addStorageButton}

            <div className="mx-3 text-center text-xs text-gray-500 dark:text-gray-400">
              {t('storages.list.description')}
            </div>
          </div>
        )}

        {isStorageDetailsShown && (
          <div className="flex w-full flex-col md:flex-1">
            {isMobile && (
              <div className="mb-2">
                <Button
                  type="default"
                  onClick={() => updateSelectedStorageId(undefined)}
                  className="w-full"
                >
                  {t('storages.list.backToList')}
                </Button>
              </div>
            )}

            <StorageComponent
              storageId={selectedStorageId}
              onStorageChanged={() => {
                loadStorages(true);
              }}
              onStorageDeleted={() => {
                const remainingStorages = storages.filter(
                  (storage) => storage.id !== selectedStorageId,
                );
                updateSelectedStorageId(remainingStorages[0]?.id);
                loadStorages();
              }}
              onStorageTransferred={() => {
                const remainingStorages = storages.filter(
                  (storage) => storage.id !== selectedStorageId,
                );
                updateSelectedStorageId(remainingStorages[0]?.id);
                loadStorages();
              }}
              isCanManageStorages={isCanManageStorages}
            />
          </div>
        )}
      </div>

      {isShowAddStorage && (
        <Modal
          title={t('storages.list.addStorage')}
          footer={<div />}
          open={isShowAddStorage}
          onCancel={() => setIsShowAddStorage(false)}
          maskClosable={false}
        >
          <div className="my-3 max-w-[250px] text-gray-500 dark:text-gray-400">
            {t('storages.list.description')}
          </div>

          <EditStorageComponent
            workspaceId={workspace.id}
            isShowName
            isShowClose={false}
            onClose={() => setIsShowAddStorage(false)}
            onChanged={(storage) => {
              loadStorages(false, storage.id);
              setIsShowAddStorage(false);
            }}
          />
        </Modal>
      )}
    </>
  );
};
