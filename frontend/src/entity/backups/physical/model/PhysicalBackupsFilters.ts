import type { PhysicalBackupStatus } from './PhysicalBackupStatus';
import type { PhysicalBackupType } from './PhysicalBackupType';

export interface PhysicalBackupsFilters {
  types?: PhysicalBackupType[];
  statuses?: PhysicalBackupStatus[];
  createdBefore?: string;
}
