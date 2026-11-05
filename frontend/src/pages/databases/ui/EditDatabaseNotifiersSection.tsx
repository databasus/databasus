import { Alert, Button } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useSessionGuard } from '@/shared/auth';

import { type Database, databaseApi } from '../../../entities/databases';
import { translateApiError } from '../../../shared/i18n';
import { hasSameNotifierIds } from '../model/hasSameNotifierIds';
import { DatabaseNotifiersFields } from './DatabaseNotifiersFields';

interface Props {
  database: Database;
  onCancel: () => void;
  onSaved: (database: Database) => void;
}

export const EditDatabaseNotifiersSection = ({ database, onCancel, onSaved }: Props) => {
  const { t } = useTranslation();
  const [notifierIds, setNotifierIds] = useState(database.notifierIds);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<unknown>();

  const saveNotifiers = async () => {
    setIsSaving(true);
    setSaveError(undefined);

    try {
      const savedDatabase = await databaseApi.updateDatabase(database.id, {
        ...database,
        notifierIds,
      });
      if (!isCurrent()) return;

      onSaved(savedDatabase);
    } catch (e) {
      if (!isCurrent()) return;

      setSaveError(e);
    }

    setIsSaving(false);
  };

  const isCurrent = useSessionGuard();

  return (
    <div>
      <DatabaseNotifiersFields
        workspaceId={database.workspaceId}
        notifierIds={notifierIds}
        onNotifierIdsChange={setNotifierIds}
      />

      {saveError !== undefined && (
        <Alert type="error" showIcon message={translateApiError(saveError, t)} />
      )}

      <div className="mt-5 flex">
        <Button className="mr-1" danger ghost disabled={isSaving} onClick={onCancel}>
          {t('common.actions.cancel')}
        </Button>

        {!hasSameNotifierIds({ acknowledgedIds: database.notifierIds, shownIds: notifierIds }) && (
          <Button
            type="primary"
            onClick={() => void saveNotifiers()}
            loading={isSaving}
            disabled={isSaving}
            className="mr-5"
          >
            {t('common.actions.save')}
          </Button>
        )}
      </div>
    </div>
  );
};
