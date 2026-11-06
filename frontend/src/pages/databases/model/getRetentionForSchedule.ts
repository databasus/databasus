import type { LogicalBackupSettings } from '@/entities/backup-configs';
import { type Schedule } from '@/entities/schedules';
import { isGfsHoursShown } from '@/pages/databases/model/isGfsHoursShown';
import { LogicalRetentionPolicy } from '@/pages/databases/model/LogicalRetentionPolicy';

export const getRetentionForSchedule = (
  { schedule, retention }: LogicalBackupSettings,
  nextSchedule: Schedule,
): LogicalBackupSettings['retention'] => {
  const shouldResetGfsHours =
    nextSchedule.type !== schedule.type &&
    !isGfsHoursShown(nextSchedule) &&
    retention.policy === LogicalRetentionPolicy.GFS;

  return shouldResetGfsHours ? { ...retention, gfsHours: 24 } : retention;
};
