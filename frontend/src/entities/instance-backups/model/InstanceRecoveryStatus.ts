import type { ApiComponents } from '@/shared/api';

export type InstanceRecoveryStatus = ApiComponents['schemas']['InstanceRecoveryStatus'];

export const InstanceRecoveryStatus = {
  VALIDATING: 'VALIDATING',
  READY: 'READY',
  APPLYING: 'APPLYING',
  PAUSED: 'PAUSED',
  ACTIVE: 'ACTIVE',
  FAILED: 'FAILED',
  CANCELLED: 'CANCELLED',
} as const satisfies Record<string, InstanceRecoveryStatus>;
