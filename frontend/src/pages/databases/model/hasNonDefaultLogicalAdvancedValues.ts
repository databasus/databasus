import type { LogicalBackupSettings } from '@/entities/backup-configs';
import { createDefaultLogicalBackupConfig } from '@/pages/databases/model/createDefaultLogicalBackupConfig';

export const hasNonDefaultLogicalAdvancedValues = (settings: LogicalBackupSettings): boolean => {
  const defaults = createDefaultLogicalBackupConfig().settings;

  return (
    settings.isRetryIfFailed !== defaults.isRetryIfFailed ||
    settings.maxFailedTriesCount !== defaults.maxFailedTriesCount
  );
};
