import type { ApiComponents } from '../../../../shared/api';

export type WalStreamSourceState = ApiComponents['schemas']['WalStreamSourceState'];

export const WalStreamSourceState = {
  REACHABLE: 'REACHABLE',
  UNREACHABLE: 'UNREACHABLE',
  SLOT_MISSING: 'SLOT_MISSING',
} as const satisfies Record<string, WalStreamSourceState>;
