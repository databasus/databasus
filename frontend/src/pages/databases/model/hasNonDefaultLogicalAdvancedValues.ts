import type { LogicalBackupSettings } from '../../../entities/backup-configs';
import { createDefaultLogicalBackupConfig } from './createDefaultLogicalBackupConfig';

export const hasNonDefaultLogicalAdvancedValues = (settings: LogicalBackupSettings): boolean => {
  const defaults = createDefaultLogicalBackupConfig().settings;

  return (
    settings.isRetryIfFailed !== defaults.isRetryIfFailed ||
    settings.maxFailedTriesCount !== defaults.maxFailedTriesCount
  );
};
