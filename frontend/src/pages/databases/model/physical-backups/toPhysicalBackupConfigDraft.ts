import type { PhysicalBackupConfigInput } from '../../../../entities/backup-configs';
import { DEFAULT_FULL_BACKUPS_RETENTION } from './defaultFullBackupsRetention';
import type { PhysicalBackupConfigDraft } from './PhysicalBackupConfigDraft';

export const toPhysicalBackupConfigDraft = (
  input: PhysicalBackupConfigInput,
): PhysicalBackupConfigDraft => ({
  ...input,
  settings: {
    ...input.settings,
    retention: {
      ...input.settings.retention,
      fullBackups: input.settings.retention.fullBackups ?? DEFAULT_FULL_BACKUPS_RETENTION,
    },
  },
});
