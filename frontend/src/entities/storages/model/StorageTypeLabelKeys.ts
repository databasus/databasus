import { StorageType } from '@/entities/storages/model/StorageType';
import type { TranslationKey } from '@/shared/i18n';

export const STORAGE_TYPE_LABEL_KEYS: Record<StorageType, TranslationKey> = {
  [StorageType.LOCAL]: 'storages.types.local',
  [StorageType.S3]: 'storages.types.s3',
  [StorageType.NAS]: 'storages.types.nas',
  [StorageType.AZURE_BLOB]: 'storages.types.azureBlob',
  [StorageType.FTP]: 'storages.types.ftp',
  [StorageType.SFTP]: 'storages.types.sftp',
};
