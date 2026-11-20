import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

type Schemas = ApiComponents['schemas'];

export const INSTANCE_BACKUP_STATUS_LABEL_KEYS: Record<
  Schemas['InstanceBackupStatusValue'],
  TranslationKey
> = {
  IN_PROGRESS: 'instanceBackups.backupStatuses.inProgress',
  COMPLETED: 'instanceBackups.backupStatuses.completed',
  PARTIAL: 'instanceBackups.backupStatuses.partial',
  FAILED: 'instanceBackups.backupStatuses.failed',
  INTERRUPTED: 'instanceBackups.backupStatuses.interrupted',
};

export const INSTANCE_BACKUP_COPY_STATUS_LABEL_KEYS: Record<
  Schemas['InstanceBackupCopyStatus'],
  TranslationKey
> = {
  PENDING: 'instanceBackups.copyStatuses.pending',
  UPLOADING: 'instanceBackups.copyStatuses.uploading',
  RETRYING: 'instanceBackups.copyStatuses.retrying',
  STORED: 'instanceBackups.copyStatuses.stored',
  FAILED: 'instanceBackups.copyStatuses.failed',
  INTERRUPTED: 'instanceBackups.copyStatuses.interrupted',
  DELETED: 'instanceBackups.copyStatuses.deleted',
};
