import { Button } from '@mantine/core';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { PhysicalDatabaseBackupType } from '@/entities/databases';
import { createDefaultPhysicalBackupConfigDraft } from '@/pages/databases/backup-config/model/createDefaultPhysicalBackupConfigDraft';
import { isPhysicalBackupConfigComplete } from '@/pages/databases/backup-config/model/isPhysicalBackupConfigComplete';
import type { PhysicalBackupConfigDraft } from '@/pages/databases/backup-config/model/PhysicalBackupConfigDraft';
import { PhysicalBackupConfigFields } from '@/pages/databases/backup-config/ui/PhysicalBackupConfigFields';

interface Props {
  workspaceId: string;
  backupType: PhysicalDatabaseBackupType;
  initialDraft?: PhysicalBackupConfigDraft;
  onBack?: (draft: PhysicalBackupConfigDraft) => void;
  onContinue: (draft: PhysicalBackupConfigDraft) => void;
}

export const CreatePhysicalBackupConfigStep = ({
  workspaceId,
  backupType,
  initialDraft,
  onBack,
  onContinue,
}: Props) => {
  const { t } = useTranslation();
  const [draft, setDraft] = useState(
    () => initialDraft ?? createDefaultPhysicalBackupConfigDraft(backupType),
  );

  const canContinue = isPhysicalBackupConfigComplete(draft, backupType) && Boolean(draft.storageId);

  return (
    <div>
      <PhysicalBackupConfigFields
        draft={draft}
        onChange={setDraft}
        workspaceId={workspaceId}
        backupType={backupType}
        canClearStorage={false}
      />

      <div className="mt-5 flex">
        {onBack && (
          <Button className="mr-1" variant="outline" onClick={() => onBack(draft)}>
            {t('common.actions.back')}
          </Button>
        )}

        <Button
          variant="filled"
          className="mr-5 ml-auto"
          disabled={!canContinue}
          onClick={() => onContinue(draft)}
        >
          {t('common.actions.continue')}
        </Button>
      </div>
    </div>
  );
};
