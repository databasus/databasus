import { Button } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { LogicalBackupConfigInput } from '@/entities/backup-configs';
import { createDefaultLogicalBackupConfig } from '@/pages/databases/model/createDefaultLogicalBackupConfig';
import { isLogicalBackupConfigComplete } from '@/pages/databases/model/isLogicalBackupConfigComplete';
import { LogicalBackupConfigFields } from '@/pages/databases/ui/backup-config/LogicalBackupConfigFields';

interface Props {
  workspaceId: string;
  initialConfig?: LogicalBackupConfigInput;
  onBack?: (config: LogicalBackupConfigInput) => void;
  onContinue: (config: LogicalBackupConfigInput) => void;
}

export const CreateLogicalBackupConfigStep = ({
  workspaceId,
  initialConfig,
  onBack,
  onContinue,
}: Props) => {
  const { t } = useTranslation();
  const [draft, setDraft] = useState(() => initialConfig ?? createDefaultLogicalBackupConfig());

  const canContinue = isLogicalBackupConfigComplete(draft) && Boolean(draft.storageId);

  return (
    <div>
      <LogicalBackupConfigFields
        config={draft}
        onChange={setDraft}
        workspaceId={workspaceId}
        canClearStorage={false}
      />

      <div className="mt-5 flex">
        {onBack && (
          <Button className="mr-1" type="primary" ghost onClick={() => onBack(draft)}>
            {t('common.actions.back')}
          </Button>
        )}

        <Button
          type="primary"
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
