import type { ApiComponents } from '../../../../shared/api';

export type VerificationStatus = ApiComponents['schemas']['VerificationStatus'];
export const VerificationStatus = {
  PENDING: 'PENDING',
  RUNNING: 'RUNNING',
  COMPLETED: 'COMPLETED',
  FAILED: 'FAILED',
  CANCELED: 'CANCELED',
} as const satisfies Record<string, VerificationStatus>;
