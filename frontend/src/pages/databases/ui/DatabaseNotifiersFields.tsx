import { Button, Modal, Select, Spin, type GetRef } from 'antd';
import {
  type Dispatch,
  type SetStateAction,
  useEffect,
  useEffectEvent,
  useRef,
  useState,
} from 'react';
import { useTranslation } from 'react-i18next';

import { EditNotifierComponent } from '@/features/edit-notifier';
import { useSessionGuard } from '@/shared/auth';
import { useRequestRevision } from '@/shared/hooks';

import { type Notifier, notifierApi } from '../../../entities/notifiers';
import { LoadFailure } from '../../../shared/ui';

interface Props {
  workspaceId: string;
  notifierIds: string[];
  onNotifierIdsChange: Dispatch<SetStateAction<string[]>>;
}

export const DatabaseNotifiersFields = ({
  workspaceId,
  notifierIds,
  onNotifierIdsChange,
}: Props) => {
  const { t } = useTranslation();

  const [error, setError] = useState<unknown>();
  const [notifiers, setNotifiers] = useState<Notifier[]>([]);
  const [isNotifiersLoading, setIsNotifiersLoading] = useState(false);
  const [isShowCreateNotifier, setShowCreateNotifier] = useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);

  const createNotifier = () => {
    setIsDropdownOpen(false);
    notifierSelectRef.current?.focus();
    setShowCreateNotifier(true);
  };

  const acceptNotifier = (notifier: Notifier) => {
    if (!isCurrent() || notifier.workspaceId !== workspaceId) return;

    acknowledgedNotifiers.current = [
      ...acknowledgedNotifiers.current.filter((item) => item.id !== notifier.id),
      notifier,
    ];
    setNotifiers((current) => [...current.filter((item) => item.id !== notifier.id), notifier]);
    void loadNotifiers(true);
  };

  const loadNotifiers = async (isSilent = false) => {
    const revision = listRevision.Invalidate();
    const isActive = () => isCurrent() && listRevision.IsCurrent(revision);
    setError(undefined);

    if (!isSilent) setIsNotifiersLoading(true);

    try {
      const listedNotifiers = await notifierApi.listAllNotifiers(workspaceId);
      if (!isActive()) return;

      setNotifiers([
        ...listedNotifiers.filter(
          (notifier) => !acknowledgedNotifiers.current.some((item) => item.id === notifier.id),
        ),
        ...acknowledgedNotifiers.current,
      ]);
    } catch (e) {
      if (!isActive()) return;

      setError(e);
    }

    if (isActive()) setIsNotifiersLoading(false);
  };

  const isCurrent = useSessionGuard();
  const listRevision = useRequestRevision();
  const acknowledgedNotifiers = useRef<Notifier[]>([]);
  const notifierSelectRef = useRef<GetRef<typeof Select>>(null);
  const createButtonRef = useRef<HTMLButtonElement>(null);

  const startNotifiersLoad = useEffectEvent(() => {
    void loadNotifiers();
  });

  useEffect(() => {
    acknowledgedNotifiers.current = [];
    setShowCreateNotifier(false);
    setIsDropdownOpen(false);
    startNotifiersLoad();

    return () => {
      listRevision.Invalidate();
    };
  }, [workspaceId, listRevision]);

  const returnFocusToSelect = useEffectEvent(() => {
    notifierSelectRef.current?.focus();
  });

  useEffect(() => {
    if (!isShowCreateNotifier) return;

    return () => {
      returnFocusToSelect();
    };
  }, [isShowCreateNotifier]);

  if (isNotifiersLoading)
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );

  return (
    <>
      {error !== undefined && <LoadFailure error={error} onRetry={() => void loadNotifiers()} />}
      <div className="mb-5 max-w-68.75 text-muted">
        {t('databases.notifiers.description')}
        <br />
        <br />
        {t('databases.notifiers.multipleHint')}
      </div>

      <div className="mb-5 flex w-full flex-wrap items-center gap-2">
        <div className="min-w-37.5 pr-2">{t('databases.notifiers.title')}</div>

        <Select
          ref={notifierSelectRef}
          aria-label={t('databases.notifiers.title')}
          mode="multiple"
          value={notifierIds}
          onChange={onNotifierIdsChange}
          size="small"
          className="max-w-50 min-w-0 flex-1"
          open={isDropdownOpen}
          onOpenChange={setIsDropdownOpen}
          onInputKeyDown={(event) => {
            if (event.key === 'Tab' && !event.shiftKey && isDropdownOpen) {
              event.preventDefault();
              event.stopPropagation();
              createButtonRef.current?.focus();
            }
          }}
          popupRender={(menu) => (
            <>
              {notifiers.length > 0 && menu}
              <div className="mt-1 border-t border-border pt-1">
                <Button
                  ref={createButtonRef}
                  type="text"
                  block
                  className="!h-auto min-h-8 !justify-start !whitespace-normal !text-accent"
                  onClick={createNotifier}
                  onKeyDown={(event) => {
                    event.stopPropagation();

                    if (event.key === 'Escape' || (event.key === 'Tab' && event.shiftKey)) {
                      event.preventDefault();
                      if (event.key === 'Escape') setIsDropdownOpen(false);
                      notifierSelectRef.current?.focus();
                    }
                  }}
                >
                  {t('databases.notifiers.createNew')}
                </Button>
              </div>
            </>
          )}
          options={[...notifiers.map((n) => ({ label: n.name, value: n.id }))]}
          placeholder={t('databases.notifiers.selectPlaceholder')}
        />
      </div>

      {isShowCreateNotifier && (
        <Modal
          title={t('databases.notifiers.addTitle')}
          footer={<div />}
          open={isShowCreateNotifier}
          onCancel={() => {
            setShowCreateNotifier(false);
          }}
          maskClosable={false}
        >
          <div className="my-3 max-w-68.75 text-muted">{t('databases.notifiers.description')}</div>

          <EditNotifierComponent
            workspaceId={workspaceId}
            isShowName
            isShowClose={false}
            onClose={() => setShowCreateNotifier(false)}
            onPersisted={acceptNotifier}
            onChanged={(notifier) => {
              onNotifierIdsChange((current) => [...new Set([...current, notifier.id])]);
              setShowCreateNotifier(false);
            }}
          />
        </Modal>
      )}
    </>
  );
};
