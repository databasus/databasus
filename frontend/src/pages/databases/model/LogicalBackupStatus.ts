import type { ApiComponents } from '../../../shared/api';

export type LogicalBackupStatus = ApiComponents['schemas']['LogicalBackupStatus'];

export const LogicalBackupStatus = {
  IN_PROGRESS: 'IN_PROGRESS',
  COMPLETED: 'COMPLETED',
  FAILED: 'FAILED',
  CANCELED: 'CANCELED',
} as const satisfies Record<string, LogicalBackupStatus>;
