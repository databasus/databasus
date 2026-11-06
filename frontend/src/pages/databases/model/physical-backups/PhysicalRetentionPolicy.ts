import type { ApiComponents } from '@/shared/api';

export type PhysicalRetentionPolicy = ApiComponents['schemas']['PhysicalRetentionPolicy'];

export const PhysicalRetentionPolicy = {
  CHAINS: 'CHAINS',
  FULL_BACKUPS: 'FULL_BACKUPS',
  CHAINS_AND_FULL_BACKUPS: 'CHAINS_AND_FULL_BACKUPS',
} as const satisfies Record<string, PhysicalRetentionPolicy>;
