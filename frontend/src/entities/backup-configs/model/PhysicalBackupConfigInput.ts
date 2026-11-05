import type { BackupConfigInput } from './BackupConfigInput';
import type { PhysicalBackupSettings } from './PhysicalBackupSettings';

export type PhysicalBackupConfigInput = Omit<BackupConfigInput, 'settings'> & {
  settings: PhysicalBackupSettings;
};
