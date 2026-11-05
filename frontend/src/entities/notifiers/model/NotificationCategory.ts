import type { ApiComponents } from '../../../shared/api';

export type NotificationCategory = ApiComponents['schemas']['NotificationCategory'];

// The values the screens compare against. satisfies keeps them within the contract's enum.
export const NotificationCategory = {
  ALL: 'ALL',
  BACKUP_SUCCESS: 'BACKUP_SUCCESS',
  BACKUP_FAILED: 'BACKUP_FAILED',
  HEALTHCHECK_SUCCESS: 'HEALTHCHECK_SUCCESS',
  HEALTHCHECK_FAILED: 'HEALTHCHECK_FAILED',
  VERIFICATION_SUCCESS: 'VERIFICATION_SUCCESS',
  VERIFICATION_FAILED: 'VERIFICATION_FAILED',
} as const satisfies Record<string, NotificationCategory>;
