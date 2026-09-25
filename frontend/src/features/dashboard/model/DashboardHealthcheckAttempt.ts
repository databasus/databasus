import type { HealthStatus } from '../../../entity/databases';

export interface DashboardHealthcheckAttempt {
  status: HealthStatus;
  createdAt: Date;
}
