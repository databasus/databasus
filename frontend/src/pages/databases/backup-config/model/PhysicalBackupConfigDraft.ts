import type { PhysicalBackupConfigInput, PhysicalBackupSettings } from '@/entities/backup-configs';
import type { ApiComponents } from '@/shared/api';

type PhysicalBackupRetentionDraft = Omit<PhysicalBackupSettings['retention'], 'fullBackups'> & {
  fullBackups: FullBackupsRetention;
};

// Keeps the values of the policies not chosen, so switching back shows them. The server's shape is
// built from it only for a request.
export type PhysicalBackupConfigDraft = Omit<PhysicalBackupConfigInput, 'settings'> & {
  settings: Omit<PhysicalBackupSettings, 'retention'> & {
    retention: PhysicalBackupRetentionDraft;
  };
};

export type FullBackupsRetention = ApiComponents['schemas']['FullBackupsRetention'];
