import { Button, Input, Modal, Spin } from 'antd';
import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Notifier, notifierApi } from '@/entities/notifiers';
import type { Workspace } from '@/entities/workspaces';
import { EditNotifierComponent } from '@/features/edit-notifier';
import { NotifierCardComponent } from '@/pages/notifiers/ui/NotifierCardComponent';
import { NotifierComponent } from '@/pages/notifiers/ui/NotifierComponent';
import { ApiError } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useIsMobile, useRequestRevision, useScopedSelection } from '@/shared/hooks';
import { ResourceListDetail, LoadFailure } from '@/shared/ui';

interface Props {
  workspace: Workspace;
  onUnavailable?: (workspaceId: string) => void;
  isCanManageNotifiers: boolean;
}

export const NotifiersComponent = ({ workspace, isCanManageNotifiers, onUnavailable }: Props) => {
  const [collection, setCollection] = useState<{
    items: Notifier[];
    isReady: boolean;
  }>({ items: [], isReady: false });
  const [loadError, setLoadError] = useState<unknown>();
  const [isLoading, setIsLoading] = useState(true);
  const [isShowAddNotifier, setIsShowAddNotifier] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  const loadNotifiers = (isSilent = false) => {
    if (!isCurrent()) return;

    const revision = loadRevision.Invalidate();

    if (!isSilent) setIsLoading(true);
    setLoadError(undefined);
    notifierApi
      .listAllNotifiers(workspace.id)
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

  const acceptNotifier = (resource: Notifier) => {
    if (!isCurrent() || resource.workspaceId !== workspace.id) return;

    setCollection((current) => ({
      isReady: true,
      items: [...current.items.filter((item) => item.id !== resource.id), resource],
    }));
    loadNotifiers(true);
  };

  const removeSelected = () => {
    if (!selectedNotifierId) return;

    selection.remove(selectedNotifierId);
    setCollection((current) => ({
      ...current,
      items: current.items.filter((item) => item.id !== selectedNotifierId),
    }));
    loadNotifiers(true);
  };

  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const isCurrent = useSessionGuard();
  const loadRevision = useRequestRevision();
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
    startNotifiersLoad();
    const interval = setInterval(() => startNotifiersLoad(true), 5 * 60_000);

    return () => {
      clearInterval(interval);
    };
  }, []);

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

  const filteredNotifiers = notifiers.filter((resource) =>
    resource.name.toLowerCase().includes(searchQuery.toLowerCase()),
  );
  const addButton = (
    <Button
      ref={addButtonRef}
      type="primary"
      className="mb-2 w-full"
      onClick={() => setIsShowAddNotifier(true)}
    >
      {t('notifiers.list.addNotifier')}
    </Button>
  );
  const list = (
    <>
      {notifiers.length >= 5 && isCanManageNotifiers && addButton}
      {notifiers.length >= 5 && (
        <Input
          aria-label={t('notifiers.list.searchPlaceholder')}
          placeholder={t('notifiers.list.searchPlaceholder')}
          value={searchQuery}
          onChange={(event) => setSearchQuery(event.target.value)}
        />
      )}
      {filteredNotifiers.map((resource) => (
        <NotifierCardComponent
          key={resource.id}
          notifier={resource}
          selectedNotifierId={selectedNotifierId}
          setSelectedNotifierId={selection.select}
        />
      ))}
      {searchQuery && filteredNotifiers.length === 0 && notifiers.length > 0 && (
        <p className="text-sm text-muted">
          {t('notifiers.list.noSearchResults', { query: searchQuery })}
        </p>
      )}
      {notifiers.length < 5 && isCanManageNotifiers && addButton}
      <div className="mx-3 text-center text-xs text-muted">{t('notifiers.list.description')}</div>
    </>
  );

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col">
      {loadError !== undefined && <LoadFailure error={loadError} onRetry={() => loadNotifiers()} />}
      {isLoading && !collection.isReady ? (
        <Spin />
      ) : collection.isReady ? (
        <ResourceListDetail
          isMobile={isMobile}
          hasSelection={!!selectedNotifierId}
          list={list}
          backAction={
            <Button className="w-full" onClick={() => selection.select(undefined)}>
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
        <Modal
          title={t('notifiers.list.addNotifier')}
          footer={null}
          open
          onCancel={() => setIsShowAddNotifier(false)}
          maskClosable={false}
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
        </Modal>
      )}
    </div>
  );
};
