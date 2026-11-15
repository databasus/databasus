import type { ApiComponents } from '@/shared/api';

export type InstanceBackupHealth = ApiComponents['schemas']['InstanceBackupHealth'];

export const InstanceBackupHealth = {
  NOT_CONFIGURED: 'NOT_CONFIGURED',
  INITIALIZING: 'INITIALIZING',
  HEALTHY: 'HEALTHY',
  PARTIAL: 'PARTIAL',
  FAILED: 'FAILED',
  OVERDUE: 'OVERDUE',
  STOPPED: 'STOPPED',
  RESUMING: 'RESUMING',
} as const satisfies Record<string, InstanceBackupHealth>;
