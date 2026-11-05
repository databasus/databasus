import { BackupEncryption, BackupSettingsType } from '../../../../entities/backup-configs';
import { PhysicalDatabaseBackupType } from '../../../../entities/databases';
import { ScheduleType } from '../../../../entities/schedules';
import { DEFAULT_FULL_BACKUPS_RETENTION } from './defaultFullBackupsRetention';
import { hasIncrementalBackups } from './hasIncrementalBackups';
import type { PhysicalBackupConfigDraft } from './PhysicalBackupConfigDraft';
import { PhysicalNotificationTrigger } from './PhysicalNotificationTrigger';
import { PhysicalRetentionPolicy } from './PhysicalRetentionPolicy';

const DEFAULT_WAL_LAG_THRESHOLD_BYTES = 256 * 1024 * 1024;

export const createDefaultPhysicalBackupConfigDraft = (
  backupType: PhysicalDatabaseBackupType,
): PhysicalBackupConfigDraft => ({
  isBackupsEnabled: true,
  storageId: null,
  encryption: BackupEncryption.NONE,
  settings: {
    type: BackupSettingsType.PHYSICAL,
    fullBackupSchedule: { type: ScheduleType.WEEKLY, timeOfDay: '00:00', weekday: 1 },
    incrementalBackupSchedule: hasIncrementalBackups(backupType)
      ? { type: ScheduleType.DAILY, timeOfDay: '00:00' }
      : null,
    retention: {
      policy: hasIncrementalBackups(backupType)
        ? PhysicalRetentionPolicy.CHAINS_AND_FULL_BACKUPS
        : PhysicalRetentionPolicy.FULL_BACKUPS,
      chainsCount: 7,
      fullBackups: DEFAULT_FULL_BACKUPS_RETENTION,
    },
    walLagThresholdBytes:
      backupType === PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM
        ? DEFAULT_WAL_LAG_THRESHOLD_BYTES
        : 0,
    notificationTriggers: [PhysicalNotificationTrigger.BACKUP_FAILED],
  },
});
