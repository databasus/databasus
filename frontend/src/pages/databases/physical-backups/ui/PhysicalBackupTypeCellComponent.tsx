import { Tooltip } from '@mantine/core';
import type { TFunction } from 'i18next';
import { ArrowRight } from 'lucide-react';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import { type PhysicalBackupListItem } from '@/pages/databases/physical-backups/model/PhysicalBackupListItem';
import {
  PhysicalBackupType,
  PHYSICAL_BACKUP_TYPE_LABEL_KEYS,
} from '@/pages/databases/physical-backups/model/PhysicalBackupType';

interface Props {
  backup: PhysicalBackupListItem;
}

const renderTypeBadge = (type: PhysicalBackupType, t: TFunction): JSX.Element => {
  const label = t(PHYSICAL_BACKUP_TYPE_LABEL_KEYS[type]);

  if (type === PhysicalBackupType.FULL) {
    return (
      <span className="inline-flex items-center rounded bg-accent/10 px-2 py-0.5 text-xs font-semibold text-accent ring-1 ring-accent/40 ring-inset">
        {label}
      </span>
    );
  }

  if (type === PhysicalBackupType.WAL) {
    return (
      <span className="inline-flex items-center rounded bg-muted/10 px-2 py-0.5 text-xs font-medium text-muted ring-1 ring-muted/30 ring-inset">
        {label}
      </span>
    );
  }

  return (
    <span className="inline-flex items-center rounded bg-accent/5 px-2 py-0.5 text-xs font-normal text-accent ring-1 ring-accent/30 ring-inset">
      {label}
    </span>
  );
};

// The segment names are only needed when investigating the archive, so they
// stay in the badge tooltip instead of the row.
export const WalRangeSegmentsComponent = ({ backup }: Props): JSX.Element => {
  const isSingleSegment = backup.fromWalFilename === backup.toWalFilename;

  return (
    <div className="flex flex-wrap items-center gap-1 font-mono text-xs">
      {backup.fromWalFilename}
      {!isSingleSegment && (
        <>
          <ArrowRight className="inline-block shrink-0 align-middle text-[10px]" size="1em" />
          {backup.toWalFilename}
        </>
      )}
    </div>
  );
};

export const PhysicalBackupTypeCellComponent = ({ backup }: Props): JSX.Element => {
  const { t } = useTranslation();

  const typeBadge = renderTypeBadge(backup.type, t);

  if (backup.type !== PhysicalBackupType.WAL) {
    return typeBadge;
  }

  return (
    <Tooltip label={<WalRangeSegmentsComponent backup={backup} />}>
      <span className="cursor-help">{typeBadge}</span>
    </Tooltip>
  );
};
