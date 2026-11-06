import type { PhysicalBackupConfigInput } from '@/entities/backup-configs';
import type { FullBackupsRetention } from '@/pages/databases/model/physical-backups/FullBackupsRetention';
import { FullBackupsRetentionPolicy } from '@/pages/databases/model/physical-backups/FullBackupsRetentionPolicy';
import type { PhysicalBackupConfigDraft } from '@/pages/databases/model/physical-backups/PhysicalBackupConfigDraft';
import { PhysicalRetentionPolicy } from '@/pages/databases/model/physical-backups/PhysicalRetentionPolicy';

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
