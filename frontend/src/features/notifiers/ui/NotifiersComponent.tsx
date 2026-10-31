import { Button, Modal, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { notifierApi } from '../../../entities/notifiers';
import type { Notifier } from '../../../entities/notifiers';
import type { Workspace } from '../../../entities/workspaces';
import { useIsMobile } from '../../../shared/hooks';
import { translateApiError } from '../../../shared/i18n';
import { EditNotifierComponent } from './edit/EditNotifierComponent';
import { NotifierCardComponent } from './NotifierCardComponent';
import { NotifierComponent } from './NotifierComponent';

interface Props {
  contentHeight: number;
  workspace: Workspace;
  isCanManageNotifiers: boolean;
}

// eslint-disable-next-line i18next/no-literal-string -- localStorage key
const SELECTED_NOTIFIER_STORAGE_KEY = 'selectedNotifierId';

export const NotifiersComponent = ({ contentHeight, workspace, isCanManageNotifiers }: Props) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [isLoading, setIsLoading] = useState(true);
  const [notifiers, setNotifiers] = useState<Notifier[]>([]);
  const [searchQuery, setSearchQuery] = useState('');

  const [isShowAddNotifier, setIsShowAddNotifier] = useState(false);
  const [selectedNotifierId, setSelectedNotifierId] = useState<string | undefined>(undefined);

  const updateSelectedNotifierId = (notifierId: string | undefined) => {
    setSelectedNotifierId(notifierId);
    if (notifierId) {
      localStorage.setItem(`${SELECTED_NOTIFIER_STORAGE_KEY}_${workspace.id}`, notifierId);
    } else {
      localStorage.removeItem(`${SELECTED_NOTIFIER_STORAGE_KEY}_${workspace.id}`);
    }
  };

  const loadNotifiers = (isSilent = false) => {
    if (!isSilent) {
      setIsLoading(true);
    }

    notifierApi
      .listAllNotifiers(workspace.id)
      .then((notifiers) => {
        setNotifiers(notifiers);
        if (!selectedNotifierId && !isSilent && !isMobile) {
          // On desktop, auto-select a notifier; on mobile, keep it unselected to show the list first
          const savedNotifierId = localStorage.getItem(
            `${SELECTED_NOTIFIER_STORAGE_KEY}_${workspace.id}`,
          );
          const notifierToSelect =
            savedNotifierId && notifiers.some((n) => n.id === savedNotifierId)
              ? savedNotifierId
              : notifiers[0]?.id;
          updateSelectedNotifierId(notifierToSelect);
        }
      })
      .catch((e: unknown) => alert(translateApiError(e, t)))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadNotifiers();

    const interval = setInterval(() => {
      loadNotifiers(true);
    }, 5 * 60_000);

    return () => clearInterval(interval);
  }, []);

  if (isLoading) {
    return (
      <div className="mx-3 my-3 flex w-[250px] justify-center">
        <Spin />
      </div>
    );
  }

  const addNotifierButton = (
    <Button type="primary" className="mb-2 w-full" onClick={() => setIsShowAddNotifier(true)}>
      {t('notifiers.list.addNotifier')}
    </Button>
  );

  const filteredNotifiers = notifiers.filter((notifier) =>
    notifier.name.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  // On mobile, show either the list or the notifier details
  const isNotifierListShown = !isMobile || !selectedNotifierId;
  const isNotifierDetailsShown = selectedNotifierId && (!isMobile || selectedNotifierId);

  return (
    <>
      <div className="flex min-h-0 min-w-0 grow" style={{ height: contentHeight }}>
        {isNotifierListShown && (
          <div className="w-full overflow-y-auto md:mx-3 md:w-[250px] md:min-w-[250px] md:pr-2">
            {notifiers.length >= 5 && (
              <>
                {isCanManageNotifiers && addNotifierButton}

                <div className="mb-2">
                  <input
                    placeholder={t('notifiers.list.searchPlaceholder')}
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="w-full border-b border-border p-1 text-muted focus-visible:outline-2 focus-visible:outline-accent"
                  />
                </div>
              </>
            )}

            {filteredNotifiers.length > 0
              ? filteredNotifiers.map((notifier) => (
                  <NotifierCardComponent
                    key={notifier.id}
                    notifier={notifier}
                    selectedNotifierId={selectedNotifierId}
                    setSelectedNotifierId={updateSelectedNotifierId}
                  />
                ))
              : searchQuery && (
                  <div className="mb-4 text-center text-sm text-muted">
                    {t('notifiers.list.noSearchResults', {
                      query: searchQuery,
                    })}
                  </div>
                )}

            {notifiers.length < 5 && isCanManageNotifiers && addNotifierButton}

            <div className="mx-3 text-center text-xs text-muted">
              {t('notifiers.list.description')}
            </div>
          </div>
        )}

        {isNotifierDetailsShown && (
          <div className="flex min-h-0 w-full min-w-0 flex-col md:flex-1">
            {isMobile && (
              <div className="mb-2">
                <Button
                  type="default"
                  onClick={() => updateSelectedNotifierId(undefined)}
                  className="w-full"
                >
                  {t('notifiers.list.backToList')}
                </Button>
              </div>
            )}

            <NotifierComponent
              notifierId={selectedNotifierId}
              onNotifierChanged={() => {
                loadNotifiers(true);
              }}
              onNotifierDeleted={() => {
                const remainingNotifiers = notifiers.filter(
                  (notifier) => notifier.id !== selectedNotifierId,
                );
                updateSelectedNotifierId(remainingNotifiers[0]?.id);
                loadNotifiers();
              }}
              onNotifierTransferred={() => {
                const remainingNotifiers = notifiers.filter(
                  (notifier) => notifier.id !== selectedNotifierId,
                );
                updateSelectedNotifierId(remainingNotifiers[0]?.id);
                loadNotifiers();
              }}
              isCanManageNotifiers={isCanManageNotifiers}
            />
          </div>
        )}
      </div>

      {isShowAddNotifier && (
        <Modal
          title={t('notifiers.create.title')}
          footer={<div />}
          open={isShowAddNotifier}
          onCancel={() => setIsShowAddNotifier(false)}
          maskClosable={false}
        >
          <div className="my-3 max-w-[250px] text-muted">{t('notifiers.list.description')}</div>

          <EditNotifierComponent
            workspaceId={workspace.id}
            isShowName
            isShowClose={false}
            onClose={() => setIsShowAddNotifier(false)}
            onPersisted={() => loadNotifiers(true)}
            onChanged={(notifier) => {
              updateSelectedNotifierId(notifier.id);
              setIsShowAddNotifier(false);
            }}
          />
        </Modal>
      )}
    </>
  );
};
