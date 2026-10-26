import type { ApiComponents } from '../../../../shared/api';

// The storage of a backup config as /backup-configs names it; the storage's settings come from the
// storage operations.
export interface BackupConfigStorage {
  id: string;
  name: string;
  type: ApiComponents['schemas']['StorageSettings']['type'];
}
