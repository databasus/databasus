import type { TranslationKey } from '../../../shared/i18n';
import { NotificationCategory } from './NotificationCategory';

export const NOTIFICATION_CATEGORY_LABEL_KEYS: Record<NotificationCategory, TranslationKey> = {
  [NotificationCategory.ALL]: 'notifiers.notificationTypes.all',
  [NotificationCategory.BACKUP_SUCCESS]: 'notifiers.notificationTypes.backupSuccess',
  [NotificationCategory.BACKUP_FAILED]: 'notifiers.notificationTypes.backupFailed',
  [NotificationCategory.HEALTHCHECK_SUCCESS]: 'notifiers.notificationTypes.healthcheckSuccess',
  [NotificationCategory.HEALTHCHECK_FAILED]: 'notifiers.notificationTypes.healthcheckFailed',
  [NotificationCategory.VERIFICATION_SUCCESS]: 'notifiers.notificationTypes.verificationSuccess',
  [NotificationCategory.VERIFICATION_FAILED]: 'notifiers.notificationTypes.verificationFailed',
};
