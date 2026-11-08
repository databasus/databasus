import type { LogicalBackupStatus } from '@/pages/databases/logical-backups/model/LogicalBackupStatus';

// Narrows the backup list; both conditions apply to every page.
export interface LogicalBackupListFilter {
  statuses?: LogicalBackupStatus[];
  createdBefore?: string;
}
