import type { PhysicalBackupStatus } from '@/pages/databases/physical-backups/model/PhysicalBackupStatus';
import type { PhysicalBackupType } from '@/pages/databases/physical-backups/model/PhysicalBackupType';

export interface PhysicalBackupsFilters {
  types?: PhysicalBackupType[];
  statuses?: PhysicalBackupStatus[];
  createdBefore?: string;
}
