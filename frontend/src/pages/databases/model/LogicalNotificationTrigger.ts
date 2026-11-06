import type { ApiComponents } from '@/shared/api';

export type LogicalNotificationTrigger = ApiComponents['schemas']['LogicalNotificationTrigger'];

export const LogicalNotificationTrigger = {
  BACKUP_FAILED: 'BACKUP_FAILED',
  BACKUP_SUCCESS: 'BACKUP_SUCCESS',
} as const satisfies Record<string, LogicalNotificationTrigger>;
