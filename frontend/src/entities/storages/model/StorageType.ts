import type { StorageSettings } from '@/entities/storages/model/StorageSettings';

export type StorageType = StorageSettings['type'];

// The values the screens compare against. satisfies keeps them within the contract's enum.
export const StorageType = {
  LOCAL: 'LOCAL',
  // eslint-disable-next-line i18next/no-literal-string -- enum value, not copy
  S3: 'S3',
  GOOGLE_DRIVE: 'GOOGLE_DRIVE',
  NAS: 'NAS',
  AZURE_BLOB: 'AZURE_BLOB',
  FTP: 'FTP',
  SFTP: 'SFTP',
  RCLONE: 'RCLONE',
} as const satisfies Record<string, StorageType>;
