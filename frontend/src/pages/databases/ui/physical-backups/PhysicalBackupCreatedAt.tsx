import dayjs from 'dayjs';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import { type PhysicalBackupListItem } from '@/pages/databases/model/physical-backups/PhysicalBackupListItem';
import { PhysicalBackupType } from '@/pages/databases/model/physical-backups/PhysicalBackupType';
import { useLocale } from '@/shared/i18n';
import { getUserTimeFormat } from '@/shared/time';

interface Props {
  backup: PhysicalBackupListItem;
}

export const PhysicalBackupCreatedAt = ({ backup }: Props): JSX.Element => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();

  const timeFormat = getUserTimeFormat().format;
  const createdAt = dayjs.utc(backup.createdAt).local();

  if (backup.type === PhysicalBackupType.WAL && backup.completedAt) {
    const lastReceivedAt = dayjs.utc(backup.completedAt).local();

    return (
      <div className="grid grid-cols-[auto_1fr] gap-x-2">
        <span className="text-muted">{t('backups.physical.list.walRange.from')}</span>
        <span>
          {createdAt.format(timeFormat)}{' '}
          <span className="text-muted">({formatRelativeTime(createdAt)})</span>
        </span>

        <span className="text-muted">{t('backups.physical.list.walRange.to')}</span>
        <span>
          {lastReceivedAt.format(timeFormat)}{' '}
          <span className="text-muted">({formatRelativeTime(lastReceivedAt)})</span>
        </span>

        <span className="text-muted">{t('backups.physical.list.walRange.segments')}</span>
        <span>{backup.walSegmentsCount}</span>
      </div>
    );
  }

  return (
    <div>
      {createdAt.format(timeFormat)}
      <div className="text-muted">({formatRelativeTime(createdAt)})</div>
    </div>
  );
};
