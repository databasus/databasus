import type { BackupConfigStorage } from '../../../entity/backups/shared';
import type { Storage } from '../../../entity/storages';

export const toBackupConfigStorage = (storage: Storage): BackupConfigStorage => ({
  id: storage.id,
  name: storage.name,
  type: storage.settings.type,
});
