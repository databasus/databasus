import type { LogicalBackupConfigInput, LogicalBackupSettings } from '@/entities/backup-configs';
import { createDefaultLogicalBackupConfig } from '@/pages/databases/model/createDefaultLogicalBackupConfig';

const defaultConfig = createDefaultLogicalBackupConfig();

export const zeroRetention: LogicalBackupSettings['retention'] = {
  ...defaultConfig.settings.retention,
  count: 0,
  gfsHours: 0,
  gfsDays: 0,
  gfsWeeks: 0,
  gfsMonths: 0,
  gfsYears: 0,
};

export const buildLogicalBackupConfig = (
  change: Partial<LogicalBackupConfigInput> = {},
  settings: Partial<LogicalBackupSettings> = {},
  retention: Partial<LogicalBackupSettings['retention']> = {},
): LogicalBackupConfigInput => ({
  ...defaultConfig,
  ...change,
  settings: {
    ...defaultConfig.settings,
    ...settings,
    retention: { ...defaultConfig.settings.retention, ...settings.retention, ...retention },
  },
});
