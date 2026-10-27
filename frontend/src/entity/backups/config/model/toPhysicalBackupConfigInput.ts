import type { BackupConfig } from './BackupConfig';
import { BackupSettingsType } from './BackupSettingsType';
import type { PhysicalBackupConfigInput } from './PhysicalBackupConfigInput';

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
