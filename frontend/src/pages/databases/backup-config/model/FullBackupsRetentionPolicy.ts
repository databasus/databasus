import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type FullBackupsRetentionPolicy = ApiComponents['schemas']['FullBackupsRetentionPolicy'];

export const FullBackupsRetentionPolicy = {
  LAST_N: 'LAST_N',
  GFS: 'GFS',
} as const satisfies Record<string, FullBackupsRetentionPolicy>;

export const FULL_BACKUPS_RETENTION_POLICY_LABEL_KEYS: Record<
  FullBackupsRetentionPolicy,
  TranslationKey
> = {
  [FullBackupsRetentionPolicy.LAST_N]: 'backups.physical.fullBackupsPolicies.lastN',
  [FullBackupsRetentionPolicy.GFS]: 'backups.physical.fullBackupsPolicies.gfs',
};
