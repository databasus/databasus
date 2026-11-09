import { AzureBlobAuthMethod, type StorageSettings, StorageType } from '@/entities/storages';

const DEFAULT_NAS_PORT = 445;
const DEFAULT_FTP_PORT = 21;
const DEFAULT_SFTP_PORT = 22;

// The empty settings a new storage of the type starts from.
export const createDefaultStorageSettings = (type: StorageType): StorageSettings => {
  switch (type) {
    case StorageType.LOCAL:
      return { type };
    case StorageType.S3:
      return { type, s3Bucket: '', isVirtualHostedStyle: false, isSkipTlsVerify: false };
    case StorageType.NAS:
      return { type, host: '', port: DEFAULT_NAS_PORT, share: '', username: '', isSsl: false };
    case StorageType.AZURE_BLOB:
      return { type, authMethod: AzureBlobAuthMethod.ACCOUNT_KEY, containerName: '' };
    case StorageType.FTP:
      return {
        type,
        host: '',
        port: DEFAULT_FTP_PORT,
        username: '',
        isSsl: false,
        isSkipTlsVerify: false,
      };
    case StorageType.SFTP:
      return {
        type,
        host: '',
        port: DEFAULT_SFTP_PORT,
        username: '',
        authMethods: ['PASSWORD'],
        isSkipHostKeyVerify: false,
      };
  }
};
