import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type LogicalBackupStatus = ApiComponents['schemas']['LogicalBackupStatus'];

export const LogicalBackupStatus = {
  IN_PROGRESS: 'IN_PROGRESS',
  COMPLETED: 'COMPLETED',
  FAILED: 'FAILED',
  CANCELED: 'CANCELED',
} as const satisfies Record<string, LogicalBackupStatus>;

export const LOGICAL_BACKUP_STATUS_LABEL_KEYS: Record<LogicalBackupStatus, TranslationKey> = {
  [LogicalBackupStatus.IN_PROGRESS]: 'status.logicalBackup.inProgress',
  [LogicalBackupStatus.COMPLETED]: 'status.logicalBackup.completed',
  [LogicalBackupStatus.FAILED]: 'status.logicalBackup.failed',
  [LogicalBackupStatus.CANCELED]: 'status.logicalBackup.canceled',
};
