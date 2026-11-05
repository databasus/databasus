import type { ApiComponents } from '../../../../shared/api';

export type VerificationTrigger = ApiComponents['schemas']['VerificationTrigger'];
export const VerificationTrigger = {
  MANUAL: 'MANUAL',
  SCHEDULED: 'SCHEDULED',
  AFTER_BACKUP: 'AFTER_BACKUP',
} as const satisfies Record<string, VerificationTrigger>;
