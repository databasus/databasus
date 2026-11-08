import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type VerificationStatus = ApiComponents['schemas']['VerificationStatus'];
export const VerificationStatus = {
  PENDING: 'PENDING',
  RUNNING: 'RUNNING',
  COMPLETED: 'COMPLETED',
  FAILED: 'FAILED',
  CANCELED: 'CANCELED',
} as const satisfies Record<string, VerificationStatus>;

export const VERIFICATION_STATUS_LABEL_KEYS: Record<VerificationStatus, TranslationKey> = {
  [VerificationStatus.PENDING]: 'status.verification.pending',
  [VerificationStatus.RUNNING]: 'status.verification.running',
  [VerificationStatus.COMPLETED]: 'status.verification.completed',
  [VerificationStatus.FAILED]: 'status.verification.failed',
  [VerificationStatus.CANCELED]: 'status.verification.canceled',
};
