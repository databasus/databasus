import {
  BackupEncryption,
  BackupSettingsType,
  type LogicalBackupConfigInput,
} from '@/entities/backup-configs';
import { ScheduleType } from '@/entities/schedules';
import { LogicalNotificationTrigger } from '@/pages/databases/backup-config/model/LogicalNotificationTrigger';
import { LogicalRetentionPolicy } from '@/pages/databases/backup-config/model/LogicalRetentionPolicy';
import { RetentionPeriod } from '@/pages/databases/backup-config/model/RetentionPeriod';

export const createDefaultLogicalBackupConfig = (): LogicalBackupConfigInput => ({
  isBackupsEnabled: true,
  storageId: null,
  encryption: BackupEncryption.ENCRYPTED,
  settings: {
    type: BackupSettingsType.LOGICAL,
    schedule: { type: ScheduleType.DAILY, timeOfDay: '00:00' },
    retention: {
      policy: LogicalRetentionPolicy.TIME_PERIOD,
      period: RetentionPeriod.THREE_MONTHS,
      count: 100,
      gfsHours: 24,
      gfsDays: 7,
      gfsWeeks: 4,
      gfsMonths: 12,
      gfsYears: 3,
    },
    isRetryIfFailed: true,
    maxFailedTriesCount: 3,
    notificationTriggers: [LogicalNotificationTrigger.BACKUP_FAILED],
  },
});
