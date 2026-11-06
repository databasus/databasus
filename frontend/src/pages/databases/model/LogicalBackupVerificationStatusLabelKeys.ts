import { LogicalBackupVerificationStatus } from '@/pages/databases/model/LogicalBackupVerificationStatus';
import type { TranslationKey } from '@/shared/i18n';

// NOT_VERIFIED has no label: a backup that was never restore-verified shows no tag at all.
// It is an explicit null rather than a missing entry, so adding a status still fails the build.
export const LOGICAL_BACKUP_VERIFICATION_STATUS_LABEL_KEYS: Record<
  LogicalBackupVerificationStatus,
  TranslationKey | null
> = {
  [LogicalBackupVerificationStatus.NOT_VERIFIED]: null,
  [LogicalBackupVerificationStatus.VERIFIED_SUCCESSFUL]:
    'status.restoreVerification.verifiedSuccessful',
  [LogicalBackupVerificationStatus.VERIFICATION_FAILED]:
    'status.restoreVerification.verificationFailed',
};
