import type { DatabaseType, HealthStatus } from '../../../entity/databases';
import type { DashboardHealthcheckAttempt } from './DashboardHealthcheckAttempt';
import type { DashboardStorage } from './DashboardStorage';

export interface DashboardDatabase {
  id: string;
  name: string;
  type: DatabaseType;
  healthStatus?: HealthStatus;
  lastBackupTime?: Date;
  lastBackupErrorMessage?: string;
  storage?: DashboardStorage;
  backupsCount: number;
  completedBackupsCount: number;
  failedBackupsCount: number;
  meanBackupSizeMb: number;
  totalBackupSizeMb: number;
  recentHealthcheckAttempts: DashboardHealthcheckAttempt[];
}
