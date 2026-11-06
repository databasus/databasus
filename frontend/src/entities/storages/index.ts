export { storageApi } from '@/entities/storages/api/storageApi';
export type { Storage } from '@/entities/storages/model/Storage';
export type { StorageSettings } from '@/entities/storages/model/StorageSettings';
export type { StorageTestResult } from '@/entities/storages/model/StorageTestResult';
export { StorageType } from '@/entities/storages/model/StorageType';
export type {
  AzureBlobStorageSettings,
  FtpStorageSettings,
  GoogleDriveStorageSettings,
  NasStorageSettings,
  RcloneStorageSettings,
  S3StorageSettings,
  SftpStorageSettings,
} from '@/entities/storages/model/ProviderStorageSettings';
export { S3_STORAGE_CLASS_LABEL_KEYS } from '@/entities/storages/model/S3StorageClassLabelKeys';
export { AzureBlobAuthMethod } from '@/entities/storages/model/AzureBlobAuthMethod';
export { AZURE_BLOB_AUTH_METHOD_LABEL_KEYS } from '@/entities/storages/model/AzureBlobAuthMethodLabelKeys';
export { getStorageLogoFromType } from '@/entities/storages/model/getStorageLogoFromType';
export { STORAGE_TYPE_LABEL_KEYS } from '@/entities/storages/model/StorageTypeLabelKeys';

export { SftpAuthMethod } from '@/entities/storages/model/SftpAuthMethod';
export { SFTP_AUTH_METHOD_LABEL_KEYS } from '@/entities/storages/model/SftpAuthMethodLabelKeys';
