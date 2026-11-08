import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type PhysicalRetentionPolicy = ApiComponents['schemas']['PhysicalRetentionPolicy'];

export const PhysicalRetentionPolicy = {
  CHAINS: 'CHAINS',
  FULL_BACKUPS: 'FULL_BACKUPS',
  CHAINS_AND_FULL_BACKUPS: 'CHAINS_AND_FULL_BACKUPS',
} as const satisfies Record<string, PhysicalRetentionPolicy>;

export const PHYSICAL_RETENTION_POLICY_LABEL_KEYS: Record<PhysicalRetentionPolicy, TranslationKey> =
  {
    [PhysicalRetentionPolicy.CHAINS]: 'backups.physical.retentions.chains',
    [PhysicalRetentionPolicy.FULL_BACKUPS]: 'backups.physical.retentions.fullBackups',
    [PhysicalRetentionPolicy.CHAINS_AND_FULL_BACKUPS]:
      'backups.physical.retentions.chainsAndFullBackups',
  };
