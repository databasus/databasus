import type { TranslationKey } from '../../../../shared/i18n';
import { VerificationStatus } from './VerificationStatus';

export const VERIFICATION_STATUS_LABEL_KEYS: Record<VerificationStatus, TranslationKey> = {
  [VerificationStatus.PENDING]: 'status.verification.pending',
  [VerificationStatus.RUNNING]: 'status.verification.running',
  [VerificationStatus.COMPLETED]: 'status.verification.completed',
  [VerificationStatus.FAILED]: 'status.verification.failed',
  [VerificationStatus.CANCELED]: 'status.verification.canceled',
};
