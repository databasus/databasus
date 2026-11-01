import type { TranslationKey } from '../../../../shared/i18n';
import { FullBackupsRetentionPolicy } from './FullBackupsRetentionPolicy';

export const FULL_BACKUPS_RETENTION_POLICY_LABEL_KEYS: Record<
  FullBackupsRetentionPolicy,
  TranslationKey
> = {
  [FullBackupsRetentionPolicy.LAST_N]: 'backups.physical.fullBackupsPolicies.lastN',
  [FullBackupsRetentionPolicy.GFS]: 'backups.physical.fullBackupsPolicies.gfs',
};
