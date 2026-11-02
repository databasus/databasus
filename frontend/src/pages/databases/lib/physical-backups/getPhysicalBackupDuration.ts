import dayjs from 'dayjs';

import { formatDuration } from '../../../../shared/lib';
import { type PhysicalBackupListItem } from '../../model/physical-backups/PhysicalBackupListItem';
import { PhysicalBackupStatus } from '../../model/physical-backups/PhysicalBackupStatus';
import { PhysicalBackupType } from '../../model/physical-backups/PhysicalBackupType';

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
