import type { BackupConfig } from '@/entities/backup-configs/model/BackupConfig';
import { BackupSettingsType } from '@/entities/backup-configs/model/BackupSettingsType';
import type { PhysicalBackupConfigInput } from '@/entities/backup-configs/model/PhysicalBackupConfigInput';

export const toPhysicalBackupConfigInput = (
  backupConfig: BackupConfig,
): PhysicalBackupConfigInput | undefined => {
  const { settings } = backupConfig;
  if (settings.type !== BackupSettingsType.PHYSICAL) return undefined;

  return {
    isBackupsEnabled: backupConfig.isBackupsEnabled,
    storageId: backupConfig.storageId,
    encryption: backupConfig.encryption,
    settings,
  };
};
