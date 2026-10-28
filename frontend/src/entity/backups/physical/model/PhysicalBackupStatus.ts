import type { ApiComponents } from '../../../../shared/api';

export type PhysicalBackupStatus = ApiComponents['schemas']['PhysicalBackupStatus'];

export const PhysicalBackupStatus = {
  IN_PROGRESS: 'IN_PROGRESS',
  COMPLETED: 'COMPLETED',
  ERROR: 'ERROR',
  CHAIN_BROKEN: 'CHAIN_BROKEN',
  CANCELED: 'CANCELED',
} as const satisfies Record<string, PhysicalBackupStatus>;
