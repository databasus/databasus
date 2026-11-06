import { VerificationTrigger } from '@/pages/databases/model/verification/VerificationTrigger';
import type { TranslationKey } from '@/shared/i18n';

export const VERIFICATION_TRIGGER_LABEL_KEYS: Record<VerificationTrigger, TranslationKey> = {
  [VerificationTrigger.MANUAL]: 'verification.triggers.manual',
  [VerificationTrigger.AFTER_BACKUP]: 'verification.triggers.afterBackup',
  [VerificationTrigger.SCHEDULED]: 'verification.triggers.scheduled',
};
