import { type Database, DatabaseType } from '@/entities/databases';
import { isMonitoringTabShown } from '@/pages/databases/lib/wal-stream-monitoring/monitoringTab';
import type { DatabaseTab } from '@/pages/databases/model/DatabaseTab';

export const getDatabaseTab = (database: Database, tab?: string): DatabaseTab => {
  if (tab === 'config' || tab === 'backups') return tab;

  if (tab === 'verifications' && database.settings.type === DatabaseType.POSTGRES_LOGICAL)
    return tab;

  if (tab === 'monitoring' && isMonitoringTabShown(database)) return tab;

  return 'backups';
};
