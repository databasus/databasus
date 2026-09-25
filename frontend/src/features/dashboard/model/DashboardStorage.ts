import type { StorageType } from '../../../entity/storages';

export interface DashboardStorage {
  id: string;
  name: string;
  type: StorageType;
}
