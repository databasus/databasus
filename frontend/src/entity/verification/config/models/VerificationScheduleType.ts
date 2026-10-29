import type { ApiComponents } from '../../../../shared/api';

export type VerificationScheduleType = ApiComponents['schemas']['VerificationScheduleType'];
export const VerificationScheduleType = {
  INTERVAL: 'INTERVAL',
  AFTER_BACKUP: 'AFTER_BACKUP',
} as const satisfies Record<string, VerificationScheduleType>;
