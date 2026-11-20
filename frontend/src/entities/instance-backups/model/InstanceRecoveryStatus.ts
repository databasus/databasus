import type { ApiComponents } from '@/shared/api';

export type InstanceRecoveryStatus = ApiComponents['schemas']['InstanceRecoveryStatus'];

export const InstanceRecoveryStatus = {
  VALIDATING: 'VALIDATING',
  READY: 'READY',
  APPLYING: 'APPLYING',
  COMPLETED: 'COMPLETED',
  FAILED: 'FAILED',
  CANCELLED: 'CANCELLED',
} as const satisfies Record<string, InstanceRecoveryStatus>;
