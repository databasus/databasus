import {
  BackupSettingsType,
  type LogicalBackupConfigInput,
  type PhysicalBackupConfigInput,
} from '@/entities/backup-configs';
import { logicalBackupApi } from '@/pages/databases/logical-backups/api/logicalBackupApi';

export const startFirstBackup = async (
  databaseId: string,
  backupConfig: LogicalBackupConfigInput | PhysicalBackupConfigInput,
): Promise<void> => {
  if (!backupConfig.isBackupsEnabled || backupConfig.settings.type === BackupSettingsType.PHYSICAL)
    return;

  await logicalBackupApi.startLogicalBackup(databaseId);
};
