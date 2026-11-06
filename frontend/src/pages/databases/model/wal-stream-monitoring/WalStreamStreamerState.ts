import type { ApiComponents } from '@/shared/api';

export type WalStreamStreamerState = ApiComponents['schemas']['WalStreamStreamerState'];

export const WalStreamStreamerState = {
  RUNNING: 'RUNNING',
  NOT_RUNNING: 'NOT_RUNNING',
  NOT_STARTED: 'NOT_STARTED',
} as const satisfies Record<string, WalStreamStreamerState>;
