import type { ApiComponents } from '../../../../shared/api';

export type WalStreamReasonLevel = ApiComponents['schemas']['WalStreamReasonLevel'];

export const WalStreamReasonLevel = {
  BROKEN: 'BROKEN',
  NEEDS_ATTENTION: 'NEEDS_ATTENTION',
  INFO: 'INFO',
} as const satisfies Record<string, WalStreamReasonLevel>;
