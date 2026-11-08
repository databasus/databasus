import { Button, Tooltip } from '@mantine/core';
import { RefreshCw } from 'lucide-react';
import type { JSX, ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { type PhysicalBackupListItem } from '@/pages/databases/physical-backups/model/PhysicalBackupListItem';
import {
  PhysicalBackupStatus,
  PHYSICAL_BACKUP_STATUS_LABEL_KEYS,
} from '@/pages/databases/physical-backups/model/PhysicalBackupStatus';
import { PHYSICAL_BACKUP_STATUS_BADGE_STYLES } from '@/pages/databases/physical-backups/model/physicalBackupStatusBadgeStyles';

interface Props {
  backup: PhysicalBackupListItem;
  onShowError: (backup: PhysicalBackupListItem) => void;
}

const StatusPill = ({ className, children }: { className: string; children: ReactNode }) => (
  <span
    className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ring-1 ring-inset ${className}`}
  >
    {children}
  </span>
);

export const PhysicalBackupStatusBadge = ({ backup, onShowError }: Props): JSX.Element => {
  const { t } = useTranslation();

  // A missing segment before a WAL range is its health state, so it takes the
  // place of the status badge, styled like the other broken-chain warning.
  if (backup.hasGapBefore) {
    const gapStyle = PHYSICAL_BACKUP_STATUS_BADGE_STYLES[PhysicalBackupStatus.CHAIN_BROKEN];

    return (
      <Tooltip label={t('backups.physical.list.walRange.gapBeforeTooltip')}>
        <span className="cursor-help">
          <StatusPill className={gapStyle.pillClasses}>
            <span className={`h-1.5 w-1.5 rounded-full ${gapStyle.dotClasses}`} />
            {t('backups.physical.list.walRange.gap')}
          </StatusPill>
        </span>
      </Tooltip>
    );
  }

  const badgeStyle = PHYSICAL_BACKUP_STATUS_BADGE_STYLES[backup.status];

  const badge = (
    <StatusPill className={badgeStyle.pillClasses}>
      {backup.status === PhysicalBackupStatus.IN_PROGRESS ? (
        <RefreshCw
          style={{ fontSize: 10 }}
          className="inline-block shrink-0 animate-spin align-middle"
          size="1em"
        />
      ) : (
        <span className={`h-1.5 w-1.5 rounded-full ${badgeStyle.dotClasses}`} />
      )}
      {t(PHYSICAL_BACKUP_STATUS_LABEL_KEYS[backup.status])}
    </StatusPill>
  );

  if (!backup.failMessage) return badge;

  return (
    <Tooltip label={t('backups.list.clickToSeeErrorDetails')}>
      <Button
        variant="subtle"
        size="xs"
        className="!h-auto !p-0 !leading-none"
        onClick={() => onShowError(backup)}
      >
        {badge}
      </Button>
    </Tooltip>
  );
};
