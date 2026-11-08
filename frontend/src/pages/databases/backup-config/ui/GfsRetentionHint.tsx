import { Button } from '@mantine/core';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { BackupConfig } from '@/entities/backup-configs';
import { TransByKey, type TranslationKey } from '@/shared/i18n';

interface Props {
  backupMethod: BackupConfig['settings']['type'];
}

const HINT_KEYS = {
  LOGICAL: {
    summary: 'backups.logical.config.retention.gfsHint.summary',
    rules: [
      'backups.logical.config.retention.gfsHint.selection',
      'backups.logical.config.retention.gfsHint.fileCount',
    ],
  },
  PHYSICAL: {
    summary: 'backups.physical.config.gfsHint.summary',
    rules: [
      'backups.physical.config.gfsHint.selection',
      'backups.physical.config.gfsHint.fileCount',
      'backups.physical.config.gfsHint.scope',
    ],
  },
} satisfies Record<Props['backupMethod'], { summary: TranslationKey; rules: TranslationKey[] }>;

export const GfsRetentionHint = ({ backupMethod }: Props) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const hintId = useId();
  const { t } = useTranslation();
  const hintKeys = HINT_KEYS[backupMethod];

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
        <div id={hintId} className="mt-1 max-w-[280px] space-y-2 text-xs text-muted">
          <p>{t(hintKeys.summary)}</p>
          {hintKeys.rules.map((ruleKey) => (
            <p key={ruleKey}>
              <TransByKey
                i18nKey={ruleKey}
                components={{ strong: <strong className="font-semibold" /> }}
              />
            </p>
          ))}
        </div>
      )}
    </div>
  );
};
