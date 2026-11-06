import type { ApiComponents } from '@/shared/api';

export type VerificationNotificationType =
  ApiComponents['schemas']['VerificationNotificationTrigger'];
export const VerificationNotificationType = {
  VerificationSuccess: 'VERIFICATION_SUCCESS',
  VerificationFailed: 'VERIFICATION_FAILED',
} as const satisfies Record<string, VerificationNotificationType>;
