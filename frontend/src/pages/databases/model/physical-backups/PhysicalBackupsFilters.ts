import type { PhysicalBackupStatus } from '@/pages/databases/model/physical-backups/PhysicalBackupStatus';
import type { PhysicalBackupType } from '@/pages/databases/model/physical-backups/PhysicalBackupType';

export interface PhysicalBackupsFilters {
  types?: PhysicalBackupType[];
  statuses?: PhysicalBackupStatus[];
  createdBefore?: string;
}
