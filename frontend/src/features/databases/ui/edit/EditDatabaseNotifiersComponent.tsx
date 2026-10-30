import { Button, Modal, Select, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, databaseApi } from '../../../../entities/databases';
import { type Notifier, notifierApi } from '../../../../entities/notifiers';
import { translateApiError } from '../../../../shared/i18n';
import { EditNotifierComponent } from '../../../notifiers/ui/edit/EditNotifierComponent';

interface Props {
  database: Database;
  workspaceId: string;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  isShowBackButton: boolean;
  onBack: () => void;

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

  const [notifiers, setNotifiers] = useState<Notifier[]>([]);
  const [isNotifiersLoading, setIsNotifiersLoading] = useState(false);
  const [isShowCreateNotifier, setShowCreateNotifier] = useState(false);
  const [notifierSelectKey, setNotifierSelectKey] = useState(0);

  const saveDatabase = async () => {
    if (!editingDatabase) return;

    if (!isSaveToApi) {
      onSaved(editingDatabase);
      return;
    }

    setIsSaving(true);

    try {
      onSaved(await databaseApi.updateDatabase(editingDatabase.id, editingDatabase));
      setIsUnsaved(false);
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsSaving(false);
  };

  const loadNotifiers = async () => {
    setIsNotifiersLoading(true);

    try {
      setNotifiers(await notifierApi.listAllNotifiers(workspaceId));
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsNotifiersLoading(false);
  };

  useEffect(() => {
    setIsSaving(false);
    setEditingDatabase({ ...database });
    loadNotifiers();
  }, [database]);

  if (!editingDatabase) return null;

  if (isNotifiersLoading)
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );

  return (
    <div>
      <div className="mb-5 max-w-[275px] text-gray-500 dark:text-gray-400">
        {t('databases.notifiers.description')}
        <br />
        <br />
        {t('databases.notifiers.multipleHint')}
      </div>

      <div className="mb-5 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.notifiers.title')}</div>

        <Select
          key={notifierSelectKey}
          mode="multiple"
          value={editingDatabase.notifierIds}
          onChange={(notifiersIds) => {
            if (notifiersIds.includes('create-new-notifier')) {
              setShowCreateNotifier(true);
              return;
            }

            setEditingDatabase({ ...editingDatabase, notifierIds: notifiersIds });

            setIsUnsaved(true);
          }}
          size="small"
          className="max-w-[200px] grow"
          options={[
            ...notifiers.map((n) => ({ label: n.name, value: n.id })),
            {
              label: t('databases.notifiers.createNew'),
              value: 'create-new-notifier',
            },
          ]}
          placeholder={t('databases.notifiers.selectPlaceholder')}
        />
      </div>

      <div className="mt-5 flex">
        {isShowCancelButton && (
          <Button className="mr-1" danger ghost onClick={() => onCancel()}>
            {t('common.actions.cancel')}
          </Button>
        )}

        {isShowBackButton && (
          <Button className="mr-auto" type="primary" ghost onClick={() => onBack()}>
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
            setNotifierSelectKey((prev) => prev + 1);
          }}
          maskClosable={false}
        >
          <div className="my-3 max-w-[275px] text-gray-500 dark:text-gray-400">
            {t('databases.notifiers.description')}
          </div>

          <EditNotifierComponent
            workspaceId={workspaceId}
            isShowName
            isShowClose={false}
            onClose={() => setShowCreateNotifier(false)}
            onChanged={() => {
              loadNotifiers();
              setShowCreateNotifier(false);
            }}
          />
        </Modal>
      )}
    </div>
  );
};
