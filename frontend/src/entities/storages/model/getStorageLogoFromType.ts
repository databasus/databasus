import { StorageType } from './StorageType';

export const getStorageLogoFromType = (type: StorageType) => {
  switch (type) {
    case StorageType.LOCAL:
      return '/icons/storages/local.svg';
    case StorageType.S3:
      return '/icons/storages/s3.svg';
    case StorageType.GOOGLE_DRIVE:
      return '/icons/storages/google-drive.svg';
    case StorageType.NAS:
      return '/icons/storages/nas.svg';
    case StorageType.AZURE_BLOB:
      return '/icons/storages/azure.svg';
    case StorageType.FTP:
      return '/icons/storages/ftp.svg';
    case StorageType.SFTP:
      return '/icons/storages/sftp.svg';
    case StorageType.RCLONE:
      return '/icons/storages/rclone.svg';
    default:
      return '';
  }
};
