import type { ApiComponents } from '@/shared/api';

export type PhysicalNotificationTrigger = ApiComponents['schemas']['NotificationTrigger'];

export const PhysicalNotificationTrigger = {
  BACKUP_SUCCESS: 'BACKUP_SUCCESS',
  BACKUP_FAILED: 'BACKUP_FAILED',
  CHAIN_BROKEN: 'CHAIN_BROKEN',
  WAL_GAP: 'WAL_GAP',
} as const satisfies Record<string, PhysicalNotificationTrigger>;
