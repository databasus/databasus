import type { TranslationKey } from '../../../../shared/i18n';
import { PhysicalRetentionPolicy } from './PhysicalRetentionPolicy';

export const PHYSICAL_RETENTION_POLICY_LABEL_KEYS: Record<PhysicalRetentionPolicy, TranslationKey> =
  {
    [PhysicalRetentionPolicy.CHAINS]: 'backups.physical.retentions.chains',
    [PhysicalRetentionPolicy.FULL_BACKUPS]: 'backups.physical.retentions.fullBackups',
    [PhysicalRetentionPolicy.CHAINS_AND_FULL_BACKUPS]:
      'backups.physical.retentions.chainsAndFullBackups',
  };
