import { Alert, Button } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, databaseApi } from '@/entities/databases';
import { hasSameNotifierIds } from '@/pages/databases/model/hasSameNotifierIds';
import { DatabaseNotifiersFields } from '@/pages/databases/ui/DatabaseNotifiersFields';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';

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
        <Alert
          color="danger"
          icon={
            <CircleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />
          }
          title={translateApiError(saveError, t)}
        />
      )}

      <div className="mt-5 flex">
        <Button
          variant="outline"
          className="mr-1"
          color="danger"
          disabled={isSaving}
          onClick={onCancel}
        >
          {t('common.actions.cancel')}
        </Button>

        {!hasSameNotifierIds({ acknowledgedIds: database.notifierIds, shownIds: notifierIds }) && (
          <Button
            variant="filled"
            onClick={() => void saveNotifiers()}
            loading={isSaving}
            disabled={isSaving}
            className="mr-5"
            aria-busy={isSaving}
          >
            {t('common.actions.save')}
          </Button>
        )}
      </div>
    </div>
  );
};
