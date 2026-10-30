import { AzureBlobAuthMethod, type StorageSettings, StorageType } from '../../../entities/storages';

// A saved storage may leave its secrets empty, because the backend keeps the stored ones.
export const isStorageSettingsFilled = (settings: StorageSettings, isSaved: boolean): boolean => {
  switch (settings.type) {
    case StorageType.LOCAL:
      return true;
    case StorageType.S3:
      return !!settings.s3Bucket && (isSaved || (!!settings.s3AccessKey && !!settings.s3SecretKey));
    case StorageType.GOOGLE_DRIVE:
      return !!settings.clientId && (isSaved || (!!settings.clientSecret && !!settings.tokenJson));
    case StorageType.NAS:
      return (
        !!settings.host &&
        !!settings.port &&
        !!settings.share &&
        !!settings.username &&
        (isSaved || !!settings.password)
      );
    case StorageType.AZURE_BLOB:
      if (!settings.containerName) {
        return false;
      }

      if (isSaved) {
        return true;
      }

      return settings.authMethod === AzureBlobAuthMethod.CONNECTION_STRING
        ? !!settings.connectionString
        : !!settings.accountName && !!settings.accountKey;
    case StorageType.FTP:
      return (
        !!settings.host &&
        !!settings.port &&
        !!settings.username &&
        (isSaved || !!settings.password)
      );
    case StorageType.SFTP:
      return (
        !!settings.host &&
        !!settings.port &&
        !!settings.username &&
        (isSaved || !!settings.password || !!settings.privateKey)
      );
    case StorageType.RCLONE:
      return isSaved || !!settings.configContent;
  }
};
