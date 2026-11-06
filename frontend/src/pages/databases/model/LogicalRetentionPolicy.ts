import type { ApiComponents } from '@/shared/api';

export type LogicalRetentionPolicy = ApiComponents['schemas']['LogicalRetentionPolicy'];

export const LogicalRetentionPolicy = {
  TIME_PERIOD: 'TIME_PERIOD',
  COUNT: 'COUNT',
  GFS: 'GFS',
} as const satisfies Record<string, LogicalRetentionPolicy>;
