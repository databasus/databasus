import { type Database, DatabaseType } from '@/entities/databases';
import type { DatabaseTab } from '@/pages/databases/model/DatabaseTab';
import { isMonitoringTabShown } from '@/pages/databases/wal-monitoring/lib/monitoringTab';

export const getDatabaseTab = (database: Database, tab?: string): DatabaseTab => {
  if (tab === 'config' || tab === 'backups') return tab;

  if (tab === 'verifications' && database.settings.type === DatabaseType.POSTGRES_LOGICAL)
    return tab;

  if (tab === 'monitoring' && isMonitoringTabShown(database)) return tab;

  return 'backups';
};
