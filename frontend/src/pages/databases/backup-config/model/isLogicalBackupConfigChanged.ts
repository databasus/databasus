import type { LogicalBackupConfigInput, LogicalBackupSettings } from '@/entities/backup-configs';
import { isScheduleChanged } from '@/entities/schedules';
import { isGfsHoursShown } from '@/pages/databases/backup-config/model/isGfsHoursShown';
import { LogicalRetentionPolicy } from '@/pages/databases/backup-config/model/LogicalRetentionPolicy';

interface LogicalBackupConfigComparison {
  acknowledged: LogicalBackupConfigInput;
  shown: LogicalBackupConfigInput;
}

interface NotificationTriggersComparison {
  acknowledged: LogicalBackupSettings['notificationTriggers'];
  shown: LogicalBackupSettings['notificationTriggers'];
}

const isRetentionChanged = (
  acknowledged: LogicalBackupSettings['retention'],
  shownSettings: LogicalBackupSettings,
): boolean => {
  const { retention: shown, schedule } = shownSettings;

  if (acknowledged.policy !== shown.policy) return true;

  switch (shown.policy) {
    case LogicalRetentionPolicy.TIME_PERIOD:
      return acknowledged.period !== shown.period;
    case LogicalRetentionPolicy.COUNT:
      return acknowledged.count !== shown.count;
    case LogicalRetentionPolicy.GFS:
      return (
        acknowledged.gfsDays !== shown.gfsDays ||
        acknowledged.gfsWeeks !== shown.gfsWeeks ||
        acknowledged.gfsMonths !== shown.gfsMonths ||
        acknowledged.gfsYears !== shown.gfsYears ||
        (isGfsHoursShown(schedule) && acknowledged.gfsHours !== shown.gfsHours)
      );
  }
};

const areTriggersChanged = ({ acknowledged, shown }: NotificationTriggersComparison): boolean =>
  acknowledged.length !== shown.length || shown.some((trigger) => !acknowledged.includes(trigger));

// Compares what the form shows, so values of an unchosen policy and fields hidden with backups off
// never count as a difference.
export const isLogicalBackupConfigChanged = ({
  acknowledged,
  shown,
}: LogicalBackupConfigComparison): boolean => {
  if (
    acknowledged.isBackupsEnabled !== shown.isBackupsEnabled ||
    acknowledged.storageId !== shown.storageId ||
    acknowledged.encryption !== shown.encryption ||
    isRetentionChanged(acknowledged.settings.retention, shown.settings)
  ) {
    return true;
  }

  if (!shown.isBackupsEnabled) return false;

  return (
    isScheduleChanged({
      acknowledged: acknowledged.settings.schedule,
      shown: shown.settings.schedule,
    }) ||
    areTriggersChanged({
      acknowledged: acknowledged.settings.notificationTriggers,
      shown: shown.settings.notificationTriggers,
    }) ||
    acknowledged.settings.isRetryIfFailed !== shown.settings.isRetryIfFailed ||
    (shown.settings.isRetryIfFailed &&
      acknowledged.settings.maxFailedTriesCount !== shown.settings.maxFailedTriesCount)
  );
};
