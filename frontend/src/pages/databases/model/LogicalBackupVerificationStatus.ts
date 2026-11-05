import type { ApiComponents } from '../../../shared/api';

export type LogicalBackupVerificationStatus =
  ApiComponents['schemas']['LogicalBackupVerificationStatus'];

export const LogicalBackupVerificationStatus = {
  NOT_VERIFIED: 'NOT_VERIFIED',
  VERIFIED_SUCCESSFUL: 'VERIFIED_SUCCESSFUL',
  VERIFICATION_FAILED: 'VERIFICATION_FAILED',
} as const satisfies Record<string, LogicalBackupVerificationStatus>;
