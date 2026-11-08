import { Button } from '@mantine/core';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { DatabaseNotifiersFields } from '@/pages/databases/settings/ui/DatabaseNotifiersFields';

interface Props {
  workspaceId: string;
  initialNotifierIds: string[];
  onBack?: (notifierIds: string[]) => void;
  onContinue: (notifierIds: string[]) => void;
}

export const CreateDatabaseNotifiersStep = ({
  workspaceId,
  initialNotifierIds,
  onBack,
  onContinue,
}: Props) => {
  const { t } = useTranslation();
  const [notifierIds, setNotifierIds] = useState(initialNotifierIds);

  return (
    <div>
      <DatabaseNotifiersFields
        workspaceId={workspaceId}
        notifierIds={notifierIds}
        onNotifierIdsChange={setNotifierIds}
      />

      <div className="mt-5 flex">
        {onBack && (
          <Button className="mr-auto" variant="outline" onClick={() => onBack(notifierIds)}>
            {t('common.actions.back')}
          </Button>
        )}

        <Button variant="filled" className="mr-5" onClick={() => onContinue(notifierIds)}>
          {t('databases.create.complete')}
        </Button>
      </div>
    </div>
  );
};
