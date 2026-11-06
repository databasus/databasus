import type { ApiComponents } from '@/shared/api';

export type RestoreStatus = ApiComponents['schemas']['RestoreStatus'];

export const RestoreStatus = {
  IN_PROGRESS: 'IN_PROGRESS',
  COMPLETED: 'COMPLETED',
  FAILED: 'FAILED',
  CANCELED: 'CANCELED',
} as const satisfies Record<string, RestoreStatus>;
