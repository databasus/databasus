import { Alert, Button, Modal, Select, Spin } from 'antd';
import { type Dispatch, type SetStateAction, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { EditNotifierComponent } from '@/features/edit-notifier';
import { useSessionGuard } from '@/shared/auth';
import { RequestRevision } from '@/shared/lib/RequestRevision';

import { type Notifier, notifierApi } from '../../../entities/notifiers';
import { translateApiError } from '../../../shared/i18n';

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
    const revision = listRevision.current.Invalidate();
    const isActive = () => isCurrent() && listRevision.current.IsCurrent(revision);
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
  const listRevision = useRef(new RequestRevision());
  const acknowledgedNotifiers = useRef<Notifier[]>([]);
  const createButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    acknowledgedNotifiers.current = [];
    setShowCreateNotifier(false);
    loadNotifiers();

    return () => {
      listRevision.current.Invalidate();
    };
  }, [workspaceId]);

  useEffect(() => {
    if (!isShowCreateNotifier) return;

    return () => {
      createButtonRef.current?.focus();
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
      {error !== undefined && (
        <Alert
          type="error"
          message={translateApiError(error, t)}
          action={<Button onClick={() => void loadNotifiers()}>{t('common.actions.retry')}</Button>}
        />
      )}
      <div className="mb-5 max-w-68.75 text-muted">
        {t('databases.notifiers.description')}
        <br />
        <br />
        {t('databases.notifiers.multipleHint')}
      </div>

      <div className="mb-5 flex w-full flex-wrap items-center gap-2">
        <div className="min-w-37.5 pr-2">{t('databases.notifiers.title')}</div>

        <Select
          aria-label={t('databases.notifiers.title')}
          mode="multiple"
          value={notifierIds}
          onChange={onNotifierIdsChange}
          size="small"
          className="max-w-50 grow"
          options={[...notifiers.map((n) => ({ label: n.name, value: n.id }))]}
          placeholder={t('databases.notifiers.selectPlaceholder')}
        />
        <Button ref={createButtonRef} onClick={() => setShowCreateNotifier(true)}>
          {t('databases.notifiers.createNew')}
        </Button>
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
