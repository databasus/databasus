import type { BackupConfigInput } from './BackupConfigInput';
import type { LogicalBackupSettings } from './LogicalBackupSettings';

export type LogicalBackupConfigInput = Omit<BackupConfigInput, 'settings'> & {
  settings: LogicalBackupSettings;
};
