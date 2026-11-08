import type { PhysicalDatabaseBackupType } from '@/entities/databases';
import { type Schedule, isScheduleChanged } from '@/entities/schedules';
import { FullBackupsRetentionPolicy } from '@/pages/databases/backup-config/model/FullBackupsRetentionPolicy';
import { getShownPhysicalNotificationTriggers } from '@/pages/databases/backup-config/model/getShownPhysicalNotificationTriggers';
import { hasIncrementalBackups } from '@/pages/databases/backup-config/model/hasIncrementalBackups';
import type {
  FullBackupsRetention,
  PhysicalBackupConfigDraft,
} from '@/pages/databases/backup-config/model/PhysicalBackupConfigDraft';
import { PhysicalRetentionPolicy } from '@/pages/databases/backup-config/model/PhysicalRetentionPolicy';

type PhysicalRetentionDraft = PhysicalBackupConfigDraft['settings']['retention'];
type PhysicalNotificationTriggers = PhysicalBackupConfigDraft['settings']['notificationTriggers'];

interface Comparison<TValue> {
  acknowledged: TValue;
  shown: TValue;
}

interface PhysicalBackupConfigComparison extends Comparison<PhysicalBackupConfigDraft> {
  backupType: PhysicalDatabaseBackupType;
}

const isFullBackupsRetentionChanged = ({
  acknowledged,
  shown,
}: Comparison<FullBackupsRetention>): boolean => {
  if (acknowledged.policy !== shown.policy) return true;
  if (shown.policy === FullBackupsRetentionPolicy.LAST_N) return acknowledged.count !== shown.count;

  return (
    acknowledged.gfsHours !== shown.gfsHours ||
    acknowledged.gfsDays !== shown.gfsDays ||
    acknowledged.gfsWeeks !== shown.gfsWeeks ||
    acknowledged.gfsMonths !== shown.gfsMonths ||
    acknowledged.gfsYears !== shown.gfsYears
  );
};

const isRetentionChanged = ({
  acknowledged,
  shown,
}: Comparison<PhysicalRetentionDraft>): boolean => {
  if (acknowledged.policy !== shown.policy) return true;

  return (
    (shown.policy !== PhysicalRetentionPolicy.FULL_BACKUPS &&
      acknowledged.chainsCount !== shown.chainsCount) ||
    (shown.policy !== PhysicalRetentionPolicy.CHAINS &&
      isFullBackupsRetentionChanged({
        acknowledged: acknowledged.fullBackups,
        shown: shown.fullBackups,
      }))
  );
};

const isOptionalScheduleChanged = ({
  acknowledged,
  shown,
}: Comparison<Schedule | null>): boolean =>
  acknowledged && shown ? isScheduleChanged({ acknowledged, shown }) : acknowledged !== shown;

const areShownTriggersChanged = ({
  acknowledged,
  shown,
  backupType,
}: Comparison<PhysicalNotificationTriggers> & {
  backupType: PhysicalDatabaseBackupType;
}): boolean =>
  getShownPhysicalNotificationTriggers(backupType).some(
    (trigger) => acknowledged.includes(trigger) !== shown.includes(trigger),
  );

// Compares what the form shows, so values of an unchosen policy, of another backup type and fields
// hidden with backups off never count as a difference.
export const isPhysicalBackupConfigChanged = ({
  acknowledged,
  shown,
  backupType,
}: PhysicalBackupConfigComparison): boolean => {
  if (
    acknowledged.isBackupsEnabled !== shown.isBackupsEnabled ||
    acknowledged.storageId !== shown.storageId ||
    acknowledged.encryption !== shown.encryption ||
    isRetentionChanged({
      acknowledged: acknowledged.settings.retention,
      shown: shown.settings.retention,
    })
  ) {
    return true;
  }

  if (!shown.isBackupsEnabled) return false;

  return (
    isScheduleChanged({
      acknowledged: acknowledged.settings.fullBackupSchedule,
      shown: shown.settings.fullBackupSchedule,
    }) ||
    (hasIncrementalBackups(backupType) &&
      isOptionalScheduleChanged({
        acknowledged: acknowledged.settings.incrementalBackupSchedule,
        shown: shown.settings.incrementalBackupSchedule,
      })) ||
    areShownTriggersChanged({
      acknowledged: acknowledged.settings.notificationTriggers,
      shown: shown.settings.notificationTriggers,
      backupType,
    })
  );
};
