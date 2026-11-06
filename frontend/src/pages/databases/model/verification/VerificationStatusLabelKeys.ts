import { VerificationStatus } from '@/pages/databases/model/verification/VerificationStatus';
import type { TranslationKey } from '@/shared/i18n';

export const VERIFICATION_STATUS_LABEL_KEYS: Record<VerificationStatus, TranslationKey> = {
  [VerificationStatus.PENDING]: 'status.verification.pending',
  [VerificationStatus.RUNNING]: 'status.verification.running',
  [VerificationStatus.COMPLETED]: 'status.verification.completed',
  [VerificationStatus.FAILED]: 'status.verification.failed',
  [VerificationStatus.CANCELED]: 'status.verification.canceled',
};
