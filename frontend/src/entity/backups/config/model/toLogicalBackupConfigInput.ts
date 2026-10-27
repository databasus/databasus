import type { BackupConfig } from './BackupConfig';
import { BackupSettingsType } from './BackupSettingsType';
import type { LogicalBackupConfigInput } from './LogicalBackupConfigInput';

export const toLogicalBackupConfigInput = (
  backupConfig: BackupConfig,
): LogicalBackupConfigInput | undefined => {
  const { settings } = backupConfig;
  if (settings.type !== BackupSettingsType.LOGICAL) return undefined;

  return {
    isBackupsEnabled: backupConfig.isBackupsEnabled,
    storageId: backupConfig.storageId,
    encryption: backupConfig.encryption,
    settings,
  };
};
