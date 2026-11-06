import type { PhysicalDatabaseBackupType } from '@/entities/databases';
import { isScheduleComplete } from '@/entities/schedules';
import type { FullBackupsRetention } from '@/pages/databases/model/physical-backups/FullBackupsRetention';
import { FullBackupsRetentionPolicy } from '@/pages/databases/model/physical-backups/FullBackupsRetentionPolicy';
import { hasIncrementalBackups } from '@/pages/databases/model/physical-backups/hasIncrementalBackups';
import type { PhysicalBackupConfigDraft } from '@/pages/databases/model/physical-backups/PhysicalBackupConfigDraft';
import { PhysicalRetentionPolicy } from '@/pages/databases/model/physical-backups/PhysicalRetentionPolicy';

const isFullBackupsRetentionComplete = (retention: FullBackupsRetention): boolean => {
  if (retention.policy === FullBackupsRetentionPolicy.LAST_N) return retention.count > 0;

  return (
    retention.gfsHours > 0 ||
    retention.gfsDays > 0 ||
    retention.gfsWeeks > 0 ||
    retention.gfsMonths > 0 ||
    retention.gfsYears > 0
  );
};

const isRetentionComplete = ({
  policy,
  chainsCount,
  fullBackups,
}: PhysicalBackupConfigDraft['settings']['retention']): boolean =>
  (policy === PhysicalRetentionPolicy.FULL_BACKUPS || chainsCount > 0) &&
  (policy === PhysicalRetentionPolicy.CHAINS || isFullBackupsRetentionComplete(fullBackups));

// The server validates retention with backups off too, while a schedule and encryption apply only
// to running backups. A storage stays out, because a saved configuration may clear it.
export const isPhysicalBackupConfigComplete = (
  draft: PhysicalBackupConfigDraft,
  backupType: PhysicalDatabaseBackupType,
): boolean => {
  const { retention, fullBackupSchedule, incrementalBackupSchedule } = draft.settings;

  if (!isRetentionComplete(retention)) return false;
  if (!draft.isBackupsEnabled) return true;

  return (
    Boolean(draft.encryption) &&
    isScheduleComplete(fullBackupSchedule) &&
    (!hasIncrementalBackups(backupType) || isScheduleComplete(incrementalBackupSchedule))
  );
};
