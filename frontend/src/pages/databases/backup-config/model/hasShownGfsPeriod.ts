import type { LogicalBackupSettings } from '@/entities/backup-configs';
import { isGfsHoursShown } from '@/pages/databases/backup-config/model/isGfsHoursShown';

// A schedule of a day or longer keeps a hidden gfsHours the user never chose, so it does not count.
export const hasShownGfsPeriod = ({ schedule, retention }: LogicalBackupSettings): boolean =>
  retention.gfsDays > 0 ||
  retention.gfsWeeks > 0 ||
  retention.gfsMonths > 0 ||
  retention.gfsYears > 0 ||
  (isGfsHoursShown(schedule) && retention.gfsHours > 0);
