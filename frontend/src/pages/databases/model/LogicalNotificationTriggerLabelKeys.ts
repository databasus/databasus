import { LogicalNotificationTrigger } from '@/pages/databases/model/LogicalNotificationTrigger';
import type { TranslationKey } from '@/shared/i18n';

export const LOGICAL_NOTIFICATION_TRIGGER_LABEL_KEYS: Record<
  LogicalNotificationTrigger,
  TranslationKey
> = {
  [LogicalNotificationTrigger.BACKUP_SUCCESS]: 'backups.logical.notificationTypes.backupSuccess',
  [LogicalNotificationTrigger.BACKUP_FAILED]: 'backups.logical.notificationTypes.backupFailed',
};
