import type { ApiComponents } from '@/shared/api';

export type BackupEncryption = ApiComponents['schemas']['BackupEncryption'];

export const BackupEncryption = {
  NONE: 'NONE',
  ENCRYPTED: 'ENCRYPTED',
} as const satisfies Record<string, BackupEncryption>;
