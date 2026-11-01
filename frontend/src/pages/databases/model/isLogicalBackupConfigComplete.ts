import type {
  LogicalBackupConfigInput,
  LogicalBackupSettings,
} from '../../../entities/backup-configs';
import { isScheduleComplete } from '../../../entities/schedules';
import { hasShownGfsPeriod } from './hasShownGfsPeriod';
import { LogicalRetentionPolicy } from './LogicalRetentionPolicy';

const isRetentionComplete = (settings: LogicalBackupSettings): boolean => {
  const { retention } = settings;

  switch (retention.policy) {
    case LogicalRetentionPolicy.TIME_PERIOD:
      return Boolean(retention.period);
    case LogicalRetentionPolicy.COUNT:
      return retention.count > 0;
    case LogicalRetentionPolicy.GFS:
      return hasShownGfsPeriod(settings);
  }
};

// The server validates retention with backups off too, while a schedule and encryption apply only
// to running backups. A storage stays out, because a saved configuration may clear it.
export const isLogicalBackupConfigComplete = (config: LogicalBackupConfigInput): boolean => {
  if (!isRetentionComplete(config.settings)) return false;
  if (!config.isBackupsEnabled) return true;

  return Boolean(config.encryption) && isScheduleComplete(config.settings.schedule);
};
