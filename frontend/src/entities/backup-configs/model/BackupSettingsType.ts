import type { ApiComponents } from '@/shared/api';

export type BackupSettingsType = ApiComponents['schemas']['BackupSettings']['type'];

export const BackupSettingsType = {
  LOGICAL: 'LOGICAL',
  PHYSICAL: 'PHYSICAL',
} as const satisfies Record<string, BackupSettingsType>;
