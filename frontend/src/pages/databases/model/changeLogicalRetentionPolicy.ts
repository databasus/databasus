import type { LogicalBackupSettings } from '@/entities/backup-configs';
import { createDefaultLogicalBackupConfig } from '@/pages/databases/model/createDefaultLogicalBackupConfig';
import { hasShownGfsPeriod } from '@/pages/databases/model/hasShownGfsPeriod';
import { LogicalRetentionPolicy } from '@/pages/databases/model/LogicalRetentionPolicy';

type LogicalRetention = LogicalBackupSettings['retention'];

// The server returns zeros for a policy that was never used, so only such a policy gets defaults.
export const changeLogicalRetentionPolicy = (
  settings: LogicalBackupSettings,
  policy: LogicalRetentionPolicy,
): LogicalRetention => {
  const { retention } = settings;
  const defaults = createDefaultLogicalBackupConfig().settings.retention;

  if (policy === LogicalRetentionPolicy.COUNT && retention.count <= 0) {
    return { ...retention, policy, count: defaults.count };
  }

  if (policy === LogicalRetentionPolicy.GFS && !hasShownGfsPeriod(settings)) {
    return {
      ...retention,
      policy,
      gfsHours: defaults.gfsHours,
      gfsDays: defaults.gfsDays,
      gfsWeeks: defaults.gfsWeeks,
      gfsMonths: defaults.gfsMonths,
      gfsYears: defaults.gfsYears,
    };
  }

  return { ...retention, policy };
};
