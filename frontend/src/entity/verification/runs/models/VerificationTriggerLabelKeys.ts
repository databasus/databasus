import type { TranslationKey } from '../../../../shared/i18n';
import { VerificationTrigger } from './VerificationTrigger';

export const VERIFICATION_TRIGGER_LABEL_KEYS: Record<VerificationTrigger, TranslationKey> = {
  [VerificationTrigger.MANUAL]: 'verification.triggers.manual',
  [VerificationTrigger.AFTER_BACKUP]: 'verification.triggers.afterBackup',
  [VerificationTrigger.SCHEDULED]: 'verification.triggers.scheduled',
};
