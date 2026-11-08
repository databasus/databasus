import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type VerificationNotificationType =
  ApiComponents['schemas']['VerificationNotificationTrigger'];
export const VerificationNotificationType = {
  VerificationSuccess: 'VERIFICATION_SUCCESS',
  VerificationFailed: 'VERIFICATION_FAILED',
} as const satisfies Record<string, VerificationNotificationType>;

export const VERIFICATION_NOTIFICATION_TYPE_LABEL_KEYS: Record<
  VerificationNotificationType,
  TranslationKey
> = {
  [VerificationNotificationType.VerificationSuccess]:
    'verification.notificationTypes.verificationSuccess',
  [VerificationNotificationType.VerificationFailed]:
    'verification.notificationTypes.verificationFailed',
};
