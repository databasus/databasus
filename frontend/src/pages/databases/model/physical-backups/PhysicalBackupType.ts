import type { ApiComponents } from '@/shared/api';

export type PhysicalBackupType = ApiComponents['schemas']['PhysicalBackupListRowType'];

export const PhysicalBackupType = {
  FULL: 'FULL',
  INCREMENTAL: 'INCREMENTAL',
  WAL: 'WAL',
} as const satisfies Record<string, PhysicalBackupType>;
