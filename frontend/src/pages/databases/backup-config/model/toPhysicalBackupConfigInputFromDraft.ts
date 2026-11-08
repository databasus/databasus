import type { PhysicalBackupConfigInput } from '@/entities/backup-configs';
import { FullBackupsRetentionPolicy } from '@/pages/databases/backup-config/model/FullBackupsRetentionPolicy';
import type {
  FullBackupsRetention,
  PhysicalBackupConfigDraft,
} from '@/pages/databases/backup-config/model/PhysicalBackupConfigDraft';
import { PhysicalRetentionPolicy } from '@/pages/databases/backup-config/model/PhysicalRetentionPolicy';

const toFullBackupsRetentionInput = (retention: FullBackupsRetention): FullBackupsRetention =>
  retention.policy === FullBackupsRetentionPolicy.LAST_N
    ? { ...retention, gfsHours: 0, gfsDays: 0, gfsWeeks: 0, gfsMonths: 0, gfsYears: 0 }
    : { ...retention, count: 0 };

// The server refuses the retention values of a policy not chosen.
export const toPhysicalBackupConfigInputFromDraft = (
  draft: PhysicalBackupConfigDraft,
): PhysicalBackupConfigInput => {
  const { policy, chainsCount, fullBackups } = draft.settings.retention;

  return {
    ...draft,
    settings: {
      ...draft.settings,
      retention: {
        policy,
        chainsCount: policy === PhysicalRetentionPolicy.FULL_BACKUPS ? 0 : chainsCount,
        fullBackups:
          policy === PhysicalRetentionPolicy.CHAINS
            ? null
            : toFullBackupsRetentionInput(fullBackups),
      },
    },
  };
};
