import type { ApiComponents } from '@/shared/api';

export type SftpAuthMethod = ApiComponents['schemas']['SftpAuthMethod'];

export const SftpAuthMethod = {
  PASSWORD: 'PASSWORD',
  PRIVATE_KEY: 'PRIVATE_KEY',
} as const satisfies Record<string, SftpAuthMethod>;
