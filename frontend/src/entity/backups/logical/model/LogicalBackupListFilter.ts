import type { LogicalBackupStatus } from './LogicalBackupStatus';

// Narrows the backup list; both conditions apply to every page.
export interface LogicalBackupListFilter {
  statuses?: LogicalBackupStatus[];
  createdBefore?: string;
}
