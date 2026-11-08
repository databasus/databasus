import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type VerificationTrigger = ApiComponents['schemas']['VerificationTrigger'];
export const VerificationTrigger = {
  MANUAL: 'MANUAL',
  SCHEDULED: 'SCHEDULED',
  AFTER_BACKUP: 'AFTER_BACKUP',
} as const satisfies Record<string, VerificationTrigger>;

export const VERIFICATION_TRIGGER_LABEL_KEYS: Record<VerificationTrigger, TranslationKey> = {
  [VerificationTrigger.MANUAL]: 'verification.triggers.manual',
  [VerificationTrigger.AFTER_BACKUP]: 'verification.triggers.afterBackup',
  [VerificationTrigger.SCHEDULED]: 'verification.triggers.scheduled',
};
