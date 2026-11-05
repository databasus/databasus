import type { FullBackupsRetention } from './FullBackupsRetention';
import { FullBackupsRetentionPolicy } from './FullBackupsRetentionPolicy';

export const DEFAULT_FULL_BACKUPS_RETENTION: FullBackupsRetention = {
  policy: FullBackupsRetentionPolicy.GFS,
  count: 0,
  gfsHours: 0,
  gfsDays: 7,
  gfsWeeks: 4,
  gfsMonths: 12,
  gfsYears: 3,
};
