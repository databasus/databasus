import type { ApiComponents } from '@/shared/api';

export type PostgresSslMode = ApiComponents['schemas']['PostgresqlSslMode'];

export const PostgresSslMode = {
  Disable: 'disable',
  Require: 'require',
  VerifyCa: 'verify-ca',
  VerifyFull: 'verify-full',
} as const satisfies Record<string, PostgresSslMode>;
