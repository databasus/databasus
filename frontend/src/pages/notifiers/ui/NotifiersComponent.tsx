import { Button, Loader } from '@mantine/core';
import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Notifier, notifierApi } from '@/entities/notifiers';
import type { Workspace } from '@/entities/workspaces';
import { EditNotifierComponent } from '@/features/edit-notifier';
import { NotifierCardComponent } from '@/pages/notifiers/ui/NotifierCardComponent';
import { NotifierComponent } from '@/pages/notifiers/ui/NotifierComponent';
import { ApiError } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useIsMobile, useResourceCollection, useScopedSelection } from '@/shared/hooks';
import { Dialog } from '@/shared/ui';
import { ResourceCollectionList, ResourceListDetail, LoadFailure } from '@/shared/ui';

interface Props {
  workspace: Workspace;
  onUnavailable?: (workspaceId: string) => void;
  isCanManageNotifiers: boolean;
}

export const NotifiersComponent = ({ workspace, isCanManageNotifiers, onUnavailable }: Props) => {
  const [isShowAddNotifier, setIsShowAddNotifier] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const collection = useResourceCollection({
    queryKey: ['notifiers', workspace.id],
    read: (signal) => notifierApi.listAllNotifiers(workspace.id, { signal }),
    onError: (error) => {
      if (error instanceof ApiError && [403, 404].includes(error.status ?? 0)) {
        collection.clear();
        onUnavailable?.(workspace.id);
      }
    },
  });
  const loadNotifiers = collection.reload;
  const { loadError, isLoading } = collection;

  const acceptNotifier = (resource: Notifier) => {
    if (!isCurrent() || resource.workspaceId !== workspace.id) return;

    collection.accept(resource);
    loadNotifiers(true);
  };

  const removeSelected = () => {
    if (!selectedNotifierId) return;

    selection.remove(selectedNotifierId);
    collection.remove(selectedNotifierId);
    loadNotifiers(true);
  };

  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const isCurrent = useSessionGuard();
  const addButtonRef = useRef<HTMLButtonElement>(null);
  const notifiers = collection.items;
  const selection = useScopedSelection({
    // eslint-disable-next-line i18next/no-literal-string -- browser preference key
    scopeKey: `selectedNotifierId_${workspace.id}`,
    itemIds: notifiers.map((resource) => resource.id),
    isReady: collection.isReady,
    isMobile,
  });
  const selectedNotifierId = selection.selectedItemId;

  const startNotifiersLoad = useEffectEvent((isSilent?: boolean) => {
    loadNotifiers(isSilent);
  });

  useEffect(() => {
    if (!collection.isSessionActive) return;

    const interval = setInterval(() => startNotifiersLoad(true), 5 * 60000);

    return () => {
      clearInterval(interval);
    };
  }, [workspace.id, collection.isSessionActive]);

  // Read when the form closes, because the button can appear after the form opened.
  const returnFocusToButton = useEffectEvent(() => {
    addButtonRef.current?.focus();
  });

  useEffect(() => {
    if (!isShowAddNotifier) return;

    return () => {
      returnFocusToButton();
    };
  }, [isShowAddNotifier]);

  const addButton = (
    <Button
      ref={addButtonRef}
      variant="filled"
      className="mb-2 !h-auto min-h-8 w-full !whitespace-normal"
      onClick={() => setIsShowAddNotifier(true)}
    >
      {t('notifiers.list.addNotifier')}
    </Button>
  );
  const list = (
    <ResourceCollectionList
      items={notifiers}
      getKey={(resource) => resource.id}
      getName={(resource) => resource.name}
      renderCard={(resource) => (
        <NotifierCardComponent
          notifier={resource}
          selectedNotifierId={selectedNotifierId}
          setSelectedNotifierId={selection.select}
        />
      )}
      addAction={isCanManageNotifiers && addButton}
      description={t('notifiers.list.description')}
      searchLabel={t('notifiers.list.searchPlaceholder')}
      searchQuery={searchQuery}
      onSearchChange={setSearchQuery}
      noSearchResults={(query) => t('notifiers.list.noSearchResults', { query })}
    />
  );

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col">
      {loadError !== undefined && <LoadFailure error={loadError} onRetry={() => loadNotifiers()} />}
      {isLoading && !collection.isReady ? (
        <Loader role="status" />
      ) : collection.isReady ? (
        <ResourceListDetail
          isMobile={isMobile}
          hasSelection={!!selectedNotifierId}
          list={list}
          backAction={
            <Button
              className="!h-auto min-h-8 w-full !whitespace-normal"
              onClick={() => selection.select(undefined)}
            >
              {t('notifiers.list.backToList')}
            </Button>
          }
          detail={
            selectedNotifierId && (
              <NotifierComponent
                key={selectedNotifierId}
                notifierId={selectedNotifierId}
                onUnavailable={() => loadNotifiers(true)}
                onNotifierChanged={acceptNotifier}
                onNotifierDeleted={removeSelected}
                onNotifierTransferred={removeSelected}
                isCanManageNotifiers={isCanManageNotifiers}
              />
            )
          }
        />
      ) : null}
      {isShowAddNotifier && (
        <Dialog
          title={t('notifiers.list.addNotifier')}
          opened
          onClose={() => setIsShowAddNotifier(false)}
          closeOnClickOutside={false}
        >
          <p className="my-3 text-muted">{t('notifiers.list.description')}</p>
          <EditNotifierComponent
            workspaceId={workspace.id}
            isShowName
            isShowClose={false}
            onClose={() => setIsShowAddNotifier(false)}
            onPersisted={acceptNotifier}
            onChanged={(resource) => {
              acceptNotifier(resource);
              selection.select(resource.id);
              setIsShowAddNotifier(false);
            }}
          />
        </Dialog>
      )}
    </div>
  );
};
