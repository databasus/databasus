import { VerificationNotificationType } from '@/pages/databases/model/verification/VerificationNotificationType';
import type { TranslationKey } from '@/shared/i18n';

export const VERIFICATION_NOTIFICATION_TYPE_LABEL_KEYS: Record<
  VerificationNotificationType,
  TranslationKey
> = {
  [VerificationNotificationType.VerificationSuccess]:
    'verification.notificationTypes.verificationSuccess',
  [VerificationNotificationType.VerificationFailed]:
    'verification.notificationTypes.verificationFailed',
};
