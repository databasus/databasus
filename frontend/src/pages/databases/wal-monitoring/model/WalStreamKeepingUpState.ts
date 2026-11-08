import type { ApiComponents } from '@/shared/api';

export type WalStreamKeepingUpState = ApiComponents['schemas']['WalStreamKeepingUpState'];

export const WalStreamKeepingUpState = {
  UNKNOWN: 'UNKNOWN',
  FALLING_BEHIND: 'FALLING_BEHIND',
  CATCHING_UP: 'CATCHING_UP',
  IDLE: 'IDLE',
  KEEPING_UP: 'KEEPING_UP',
} as const satisfies Record<string, WalStreamKeepingUpState>;
