import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export const INSTANCE_RECOVERY_FAILURE_LABEL_KEYS: Record<
  ApiComponents['schemas']['InstanceRecoveryFailure'],
  TranslationKey
> = {
  ARCHIVE_INVALID: 'instanceRecovery.failures.archiveInvalid',
  ARCHIVE_INCOMPLETE: 'instanceRecovery.failures.archiveIncomplete',
  MASTER_KEY_INVALID: 'instanceRecovery.failures.masterKeyInvalid',
  RELEASE_MISMATCH: 'instanceRecovery.failures.releaseMismatch',
  SCHEMA_INCOMPATIBLE: 'instanceRecovery.failures.schemaIncompatible',
  INSUFFICIENT_DISK_SPACE: 'instanceRecovery.failures.insufficientDiskSpace',
  IMPORT_FAILED: 'instanceRecovery.failures.importFailed',
  EXPIRED: 'instanceRecovery.failures.expired',
  QUIESCENCE_FAILED: 'instanceRecovery.failures.quiescenceFailed',
  INTERRUPTED: 'instanceRecovery.failures.interrupted',
};
