import type { ApiComponents } from '../../../shared/api';

// An absent class means the bucket's default class.
export type S3StorageClass = ApiComponents['schemas']['S3StorageClass'];

// The values the screens compare against. satisfies keeps them within the contract's enum.
export const S3StorageClass = {
  STANDARD: 'STANDARD',
  STANDARD_IA: 'STANDARD_IA',
  ONEZONE_IA: 'ONEZONE_IA',
  INTELLIGENT_TIERING: 'INTELLIGENT_TIERING',
  REDUCED_REDUNDANCY: 'REDUCED_REDUNDANCY',
  GLACIER_IR: 'GLACIER_IR',
} as const satisfies Record<string, S3StorageClass>;
