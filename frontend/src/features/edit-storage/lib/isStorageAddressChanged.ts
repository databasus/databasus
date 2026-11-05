import { type StorageSettings, StorageType } from '../../../entities/storages';
import { isOptionalValueChanged } from '../../../shared/lib';

// Mirrors the resource address each storage provider compares on the backend, which stays the
// authority: a stale list here only makes the user see the generic refusal.
export const isStorageAddressChanged = (
  settings: StorageSettings,
  savedSettings?: StorageSettings,
): boolean => {
  if (!savedSettings) return false;

  switch (settings.type) {
    case StorageType.S3:
      return (
        savedSettings.type === StorageType.S3 &&
        isOptionalValueChanged(settings.s3Endpoint, savedSettings.s3Endpoint)
      );
    case StorageType.AZURE_BLOB:
      // The backend keeps the stored account name when the request leaves it out.
      return (
        savedSettings.type === StorageType.AZURE_BLOB &&
        (isOptionalValueChanged(settings.endpoint, savedSettings.endpoint) ||
          (!!settings.accountName && settings.accountName !== savedSettings.accountName))
      );
    case StorageType.FTP:
    case StorageType.NAS:
    case StorageType.SFTP:
      return (
        savedSettings.type === settings.type &&
        (isOptionalValueChanged(settings.host, savedSettings.host) ||
          isOptionalValueChanged(settings.port, savedSettings.port))
      );
    default:
      return false;
  }
};
