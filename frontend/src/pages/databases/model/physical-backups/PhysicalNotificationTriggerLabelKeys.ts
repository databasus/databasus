import { PhysicalNotificationTrigger } from '@/pages/databases/model/physical-backups/PhysicalNotificationTrigger';
import type { TranslationKey } from '@/shared/i18n';

export const PHYSICAL_NOTIFICATION_TRIGGER_LABEL_KEYS: Record<
  PhysicalNotificationTrigger,
  TranslationKey
> = {
  [PhysicalNotificationTrigger.BACKUP_SUCCESS]: 'backups.physical.notificationTypes.backupSuccess',
  [PhysicalNotificationTrigger.BACKUP_FAILED]: 'backups.physical.notificationTypes.backupFailed',
  [PhysicalNotificationTrigger.CHAIN_BROKEN]: 'backups.physical.notificationTypes.chainBroken',
  [PhysicalNotificationTrigger.WAL_GAP]: 'backups.physical.notificationTypes.walGap',
};
