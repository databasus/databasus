import { Alert, Button, Modal, Select, Spin } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { EditNotifierComponent } from '@/features/edit-notifier';
import { useSessionGuard } from '@/shared/auth';
import { RequestRevision } from '@/shared/lib/RequestRevision';

import { type Database, databaseApi } from '../../../entities/databases';
import { type Notifier, notifierApi } from '../../../entities/notifiers';
import { translateApiError } from '../../../shared/i18n';

interface Props {
  database: Database;
  workspaceId: string;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  isShowBackButton: boolean;
  onBack: (notifierIds: string[]) => void;

  isShowSaveOnlyForUnsaved: boolean;
  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (database: Database) => void;
}

export const EditDatabaseNotifiersComponent = ({
  database,
  workspaceId,

  isShowCancelButton,
  onCancel,

  isShowBackButton,
  onBack,

  isShowSaveOnlyForUnsaved,
  saveButtonText,
  isSaveToApi,
  onSaved,
}: Props) => {
  const { t } = useTranslation();
  const [editingDatabase, setEditingDatabase] = useState<Database>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [error, setError] = useState<unknown>();
  const [notifiers, setNotifiers] = useState<Notifier[]>([]);
  const [isNotifiersLoading, setIsNotifiersLoading] = useState(false);
  const [isShowCreateNotifier, setShowCreateNotifier] = useState(false);

  const saveDatabase = async () => {
    if (!editingDatabase) return;

    if (!isSaveToApi) {
      onSaved(editingDatabase);
      return;
    }

    setIsSaving(true);

    const revision = contextRevision.current.GetCurrent();

    try {
      const savedDatabase = await databaseApi.updateDatabase(editingDatabase.id, editingDatabase);
      if (!isCurrent() || !contextRevision.current.IsCurrent(revision)) return;

      onSaved(savedDatabase);
      setIsUnsaved(false);
    } catch (e) {
      if (!isCurrent() || !contextRevision.current.IsCurrent(revision)) return;

      setError(e);
    }

    if (isCurrent() && contextRevision.current.IsCurrent(revision)) setIsSaving(false);
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
  const contextRevision = useRef(new RequestRevision());
  const listRevision = useRef(new RequestRevision());
  const acknowledgedNotifiers = useRef<Notifier[]>([]);
  const createButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    contextRevision.current.Invalidate();
    acknowledgedNotifiers.current = [];
    setIsSaving(false);
    setEditingDatabase({ ...database });
    setShowCreateNotifier(false);
    loadNotifiers();

    return () => {
      contextRevision.current.Invalidate();
      listRevision.current.Invalidate();
    };
  }, [database.id, workspaceId]);

  useEffect(() => {
    if (!isShowCreateNotifier) return;

    return () => {
      createButtonRef.current?.focus();
    };
  }, [isShowCreateNotifier]);

  if (!editingDatabase) return null;

  if (isNotifiersLoading)
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );

  return (
    <div>
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
          value={editingDatabase.notifierIds}
          onChange={(notifiersIds) => {
            setEditingDatabase({ ...editingDatabase, notifierIds: notifiersIds });
            setIsUnsaved(true);
          }}
          size="small"
          className="max-w-50 grow"
          options={[...notifiers.map((n) => ({ label: n.name, value: n.id }))]}
          placeholder={t('databases.notifiers.selectPlaceholder')}
        />
        <Button ref={createButtonRef} onClick={() => setShowCreateNotifier(true)}>
          {t('databases.notifiers.createNew')}
        </Button>
      </div>

      <div className="mt-5 flex">
        {isShowCancelButton && (
          <Button className="mr-1" danger ghost onClick={() => onCancel()}>
            {t('common.actions.cancel')}
          </Button>
        )}

        {isShowBackButton && (
          <Button
            className="mr-auto"
            type="primary"
            ghost
            onClick={() => onBack(editingDatabase?.notifierIds ?? database.notifierIds)}
          >
            {t('common.actions.back')}
          </Button>
        )}

        {(!isShowSaveOnlyForUnsaved || isUnsaved) && (
          <Button
            type="primary"
            onClick={() => saveDatabase()}
            loading={isSaving}
            disabled={isSaving}
            className="mr-5"
          >
            {saveButtonText || t('common.actions.save')}
          </Button>
        )}
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
              setEditingDatabase((current) =>
                current
                  ? { ...current, notifierIds: [...new Set([...current.notifierIds, notifier.id])] }
                  : current,
              );
              setIsUnsaved(true);
              setShowCreateNotifier(false);
            }}
          />
        </Modal>
      )}
    </div>
  );
};
