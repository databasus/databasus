import { ArrowRightOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import type { TFunction } from 'i18next';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import {
  PHYSICAL_BACKUP_TYPE_LABEL_KEYS,
  type PhysicalBackupListItem,
  PhysicalBackupType,
} from '../../../../entities/backups/physical';

interface Props {
  backup: PhysicalBackupListItem;
}

const renderTypeBadge = (type: PhysicalBackupType, t: TFunction): JSX.Element => {
  const label = t(PHYSICAL_BACKUP_TYPE_LABEL_KEYS[type]);

  if (type === PhysicalBackupType.FULL) {
    return (
      <span className="inline-flex items-center rounded bg-indigo-500/15 px-2 py-0.5 text-xs font-semibold text-indigo-700 ring-1 ring-indigo-500/40 ring-inset dark:text-indigo-300">
        {label}
      </span>
    );
  }

  if (type === PhysicalBackupType.WAL) {
    return (
      <span className="inline-flex items-center rounded bg-slate-400/10 px-2 py-0.5 text-xs font-medium text-slate-600 ring-1 ring-slate-400/30 ring-inset dark:text-slate-300">
        {label}
      </span>
    );
  }

  return (
    <span className="inline-flex items-center rounded bg-indigo-400/10 px-2 py-0.5 text-xs font-medium text-indigo-600 ring-1 ring-indigo-400/30 ring-inset dark:text-indigo-300">
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
          <ArrowRightOutlined className="text-[10px]" />
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
    <Tooltip title={<WalRangeSegmentsComponent backup={backup} />}>
      <span className="cursor-help">{typeBadge}</span>
    </Tooltip>
  );
};
