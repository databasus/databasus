import type { ApiComponents } from '../../../../shared/api';

export type FullBackupsRetentionPolicy = ApiComponents['schemas']['FullBackupsRetentionPolicy'];

export const FullBackupsRetentionPolicy = {
  LAST_N: 'LAST_N',
  GFS: 'GFS',
} as const satisfies Record<string, FullBackupsRetentionPolicy>;
