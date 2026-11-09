import { AzureBlobAuthMethod, type StorageSettings, StorageType } from '@/entities/storages';
import { isStorageAddressChanged } from '@/features/edit-storage/lib/isStorageAddressChanged';

// SFTP can retain only credentials present in its acknowledged saved method set.
export const isStorageSettingsFilled = (
  settings: StorageSettings,
  isSaved: boolean,
  savedSettings?: StorageSettings,
): boolean => {
  const canKeepAddressBoundSecret = isSaved && !isStorageAddressChanged(settings, savedSettings);

  switch (settings.type) {
    case StorageType.LOCAL:
      return true;
    case StorageType.S3:
      return (
        !!settings.s3Bucket &&
        (canKeepAddressBoundSecret || (!!settings.s3AccessKey && !!settings.s3SecretKey))
      );
    case StorageType.NAS:
      return (
        !!settings.host &&
        !!settings.port &&
        !!settings.share &&
        !!settings.username &&
        (canKeepAddressBoundSecret || !!settings.password)
      );
    case StorageType.AZURE_BLOB:
      if (!settings.containerName) {
        return false;
      }

      if (settings.authMethod === AzureBlobAuthMethod.CONNECTION_STRING) {
        return isSaved || !!settings.connectionString;
      }

      return (
        canKeepAddressBoundSecret || ((isSaved || !!settings.accountName) && !!settings.accountKey)
      );
    case StorageType.FTP:
      return (
        !!settings.host &&
        !!settings.port &&
        !!settings.username &&
        (canKeepAddressBoundSecret || !!settings.password)
      );
    case StorageType.SFTP:
      return (
        !!settings.host &&
        !!settings.port &&
        !!settings.username &&
        settings.authMethods.length > 0 &&
        settings.authMethods.every(
          (method) =>
            !!(method === 'PASSWORD' ? settings.password : settings.privateKey) ||
            (canKeepAddressBoundSecret &&
              savedSettings?.type === StorageType.SFTP &&
              savedSettings.authMethods.includes(method)),
        )
      );
  }
};
