import { LogicalBackupStatus } from '@/pages/databases/model/LogicalBackupStatus';
import type { TranslationKey } from '@/shared/i18n';

export const LOGICAL_BACKUP_STATUS_LABEL_KEYS: Record<LogicalBackupStatus, TranslationKey> = {
  [LogicalBackupStatus.IN_PROGRESS]: 'status.logicalBackup.inProgress',
  [LogicalBackupStatus.COMPLETED]: 'status.logicalBackup.completed',
  [LogicalBackupStatus.FAILED]: 'status.logicalBackup.failed',
  [LogicalBackupStatus.CANCELED]: 'status.logicalBackup.canceled',
};
