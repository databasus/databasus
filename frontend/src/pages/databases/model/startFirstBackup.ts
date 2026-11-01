import {
  BackupSettingsType,
  type LogicalBackupConfigInput,
  type PhysicalBackupConfigInput,
} from '../../../entities/backup-configs';
import { logicalBackupApi } from '../api/logicalBackupApi';
import { physicalBackupsApi } from '../api/physicalBackupsApi';

export const startFirstBackup = async (
  databaseId: string,
  backupConfig: LogicalBackupConfigInput | PhysicalBackupConfigInput,
): Promise<void> => {
  if (!backupConfig.isBackupsEnabled) return;

  if (backupConfig.settings.type === BackupSettingsType.PHYSICAL) {
    await physicalBackupsApi.triggerPhysicalBackup(databaseId, 'AUTO');
    return;
  }

  await logicalBackupApi.startLogicalBackup(databaseId);
};
