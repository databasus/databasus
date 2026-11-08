import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type LogicalNotificationTrigger = ApiComponents['schemas']['LogicalNotificationTrigger'];

export const LogicalNotificationTrigger = {
  BACKUP_FAILED: 'BACKUP_FAILED',
  BACKUP_SUCCESS: 'BACKUP_SUCCESS',
} as const satisfies Record<string, LogicalNotificationTrigger>;

export const LOGICAL_NOTIFICATION_TRIGGER_LABEL_KEYS: Record<
  LogicalNotificationTrigger,
  TranslationKey
> = {
  [LogicalNotificationTrigger.BACKUP_SUCCESS]: 'backups.logical.notificationTypes.backupSuccess',
  [LogicalNotificationTrigger.BACKUP_FAILED]: 'backups.logical.notificationTypes.backupFailed',
};
