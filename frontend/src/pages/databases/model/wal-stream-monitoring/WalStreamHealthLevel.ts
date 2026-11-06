import type { ApiComponents } from '@/shared/api';

export type WalStreamHealthLevel = ApiComponents['schemas']['WalStreamHealthLevel'];

export const WalStreamHealthLevel = {
  NOT_STREAMING: 'NOT_STREAMING',
  COLLECTING_DATA: 'COLLECTING_DATA',
  BROKEN: 'BROKEN',
  NEEDS_ATTENTION: 'NEEDS_ATTENTION',
  HEALTHY: 'HEALTHY',
} as const satisfies Record<string, WalStreamHealthLevel>;
