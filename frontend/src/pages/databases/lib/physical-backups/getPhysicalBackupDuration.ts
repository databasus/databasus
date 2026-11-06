import dayjs from 'dayjs';

import { type PhysicalBackupListItem } from '@/pages/databases/model/physical-backups/PhysicalBackupListItem';
import { PhysicalBackupStatus } from '@/pages/databases/model/physical-backups/PhysicalBackupStatus';
import { PhysicalBackupType } from '@/pages/databases/model/physical-backups/PhysicalBackupType';
import { formatDuration } from '@/shared/lib';

export const getPhysicalBackupDuration = (backup: PhysicalBackupListItem): string => {
  // A WAL range's end is the receive time of its last segment, not the end
  // of a backup run, so it has no duration to show.
  if (backup.type === PhysicalBackupType.WAL) {
    return '-';
  }

  if (backup.completedAt) {
    return formatDuration(dayjs(backup.completedAt).diff(dayjs(backup.createdAt)));
  }

  if (backup.status === PhysicalBackupStatus.IN_PROGRESS) {
    // Live elapsed time; the 1s list refresh re-renders this and makes it tick.
    return formatDuration(dayjs().diff(dayjs(backup.createdAt)));
  }

  return '-';
};
