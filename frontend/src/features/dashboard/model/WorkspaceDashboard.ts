import type { DashboardDatabase } from './DashboardDatabase';
import type { DashboardTotals } from './DashboardTotals';

export interface WorkspaceDashboard {
  databases: DashboardDatabase[];
  totals: DashboardTotals;
}
