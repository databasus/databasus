import { PhysicalDatabaseBackupType } from '@/entities/databases';
import { hasIncrementalBackups } from '@/pages/databases/backup-config/model/hasIncrementalBackups';
import { PhysicalNotificationTrigger } from '@/pages/databases/backup-config/model/PhysicalNotificationTrigger';

export const getShownPhysicalNotificationTriggers = (
  backupType: PhysicalDatabaseBackupType,
): PhysicalNotificationTrigger[] => [
  PhysicalNotificationTrigger.BACKUP_SUCCESS,
  PhysicalNotificationTrigger.BACKUP_FAILED,
  ...(hasIncrementalBackups(backupType) ? [PhysicalNotificationTrigger.CHAIN_BROKEN] : []),
  ...(backupType === PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM
    ? [PhysicalNotificationTrigger.WAL_GAP]
    : []),
];
