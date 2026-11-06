import type { BackupConfigInput } from '@/entities/backup-configs/model/BackupConfigInput';
import type { PhysicalBackupSettings } from '@/entities/backup-configs/model/PhysicalBackupSettings';

export type PhysicalBackupConfigInput = Omit<BackupConfigInput, 'settings'> & {
  settings: PhysicalBackupSettings;
};
