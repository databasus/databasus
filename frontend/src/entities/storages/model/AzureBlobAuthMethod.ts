import type { ApiComponents } from '@/shared/api';

export type AzureBlobAuthMethod = ApiComponents['schemas']['AzureBlobAuthMethod'];

// The values the screens compare against. satisfies keeps them within the contract's enum.
export const AzureBlobAuthMethod = {
  CONNECTION_STRING: 'CONNECTION_STRING',
  ACCOUNT_KEY: 'ACCOUNT_KEY',
} as const satisfies Record<string, AzureBlobAuthMethod>;
