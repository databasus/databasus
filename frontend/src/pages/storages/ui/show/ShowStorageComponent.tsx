import { useTranslation } from 'react-i18next';

import {
  STORAGE_TYPE_LABEL_KEYS,
  type Storage,
  StorageType,
  getStorageLogoFromType,
} from '../../../../entities/storages';
import { ShowAzureBlobStorageComponent } from './storages/ShowAzureBlobStorageComponent';
import { ShowFTPStorageComponent } from './storages/ShowFTPStorageComponent';
import { ShowGoogleDriveStorageComponent } from './storages/ShowGoogleDriveStorageComponent';
import { ShowNASStorageComponent } from './storages/ShowNASStorageComponent';
import { ShowRcloneStorageComponent } from './storages/ShowRcloneStorageComponent';
import { ShowS3StorageComponent } from './storages/ShowS3StorageComponent';
import { ShowSFTPStorageComponent } from './storages/ShowSFTPStorageComponent';

interface Props {
  storage?: Storage;
}

export function ShowStorageComponent({ storage }: Props) {
  const { t } = useTranslation();

  if (!storage) return null;

  const { settings } = storage;

  return (
    <div>
      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('common.fields.type')}</div>

        {t(STORAGE_TYPE_LABEL_KEYS[settings.type])}

        <img alt="" src={getStorageLogoFromType(settings.type)} className="ml-1 h-4 w-4" />
      </div>

      <div>
        {settings.type === StorageType.S3 && <ShowS3StorageComponent settings={settings} />}

        {settings.type === StorageType.GOOGLE_DRIVE && (
          <ShowGoogleDriveStorageComponent settings={settings} />
        )}

        {settings.type === StorageType.NAS && <ShowNASStorageComponent settings={settings} />}

        {settings.type === StorageType.AZURE_BLOB && (
          <ShowAzureBlobStorageComponent settings={settings} />
        )}

        {settings.type === StorageType.FTP && <ShowFTPStorageComponent settings={settings} />}

        {settings.type === StorageType.SFTP && <ShowSFTPStorageComponent settings={settings} />}

        {settings.type === StorageType.RCLONE && <ShowRcloneStorageComponent settings={settings} />}
      </div>
    </div>
  );
}
