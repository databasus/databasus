import { Button } from '@mantine/core';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { BackupConfig } from '@/entities/backup-configs';

interface Props {
  backupMethod: BackupConfig['settings']['type'];
}

export const GfsRetentionHint = ({ backupMethod }: Props) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const hintId = useId();
  const { t } = useTranslation();

  return (
    <div>
      <Button
        variant="transparent"
        size="xs"
        className="!h-auto !justify-start !px-0 !text-left !text-xs !whitespace-normal"
        aria-expanded={isExpanded}
        aria-controls={hintId}
        onClick={() => setIsExpanded(!isExpanded)}
      >
        {isExpanded
          ? t('backups.logical.config.retention.hideGfsHint')
          : t('backups.logical.config.retention.showGfsHint')}
      </Button>

      {isExpanded && (
        <div id={hintId} className="mt-1 max-w-[280px] text-xs text-muted">
          {backupMethod === 'LOGICAL'
            ? t('backups.logical.config.retention.gfsHint')
            : t('backups.physical.config.gfsHint')}
        </div>
      )}
    </div>
  );
};
