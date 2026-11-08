import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type PhysicalNotificationTrigger = ApiComponents['schemas']['NotificationTrigger'];

export const PhysicalNotificationTrigger = {
  BACKUP_SUCCESS: 'BACKUP_SUCCESS',
  BACKUP_FAILED: 'BACKUP_FAILED',
  CHAIN_BROKEN: 'CHAIN_BROKEN',
  WAL_GAP: 'WAL_GAP',
} as const satisfies Record<string, PhysicalNotificationTrigger>;

export const PHYSICAL_NOTIFICATION_TRIGGER_LABEL_KEYS: Record<
  PhysicalNotificationTrigger,
  TranslationKey
> = {
  [PhysicalNotificationTrigger.BACKUP_SUCCESS]: 'backups.physical.notificationTypes.backupSuccess',
  [PhysicalNotificationTrigger.BACKUP_FAILED]: 'backups.physical.notificationTypes.backupFailed',
  [PhysicalNotificationTrigger.CHAIN_BROKEN]: 'backups.physical.notificationTypes.chainBroken',
  [PhysicalNotificationTrigger.WAL_GAP]: 'backups.physical.notificationTypes.walGap',
};
