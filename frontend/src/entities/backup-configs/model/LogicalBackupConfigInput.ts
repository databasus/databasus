import type { BackupConfigInput } from '@/entities/backup-configs/model/BackupConfigInput';
import type { LogicalBackupSettings } from '@/entities/backup-configs/model/LogicalBackupSettings';

export type LogicalBackupConfigInput = Omit<BackupConfigInput, 'settings'> & {
  settings: LogicalBackupSettings;
};
