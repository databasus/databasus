import type { BackupConfig } from '@/entities/backup-configs/model/BackupConfig';
import { BackupSettingsType } from '@/entities/backup-configs/model/BackupSettingsType';
import type { LogicalBackupConfigInput } from '@/entities/backup-configs/model/LogicalBackupConfigInput';

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
