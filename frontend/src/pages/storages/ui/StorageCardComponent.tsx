import { useTranslation } from 'react-i18next';

import { type Storage, StorageType, getStorageLogoFromType } from '../../../entities/storages';
import type { TranslationKey } from '../../../shared/i18n';

interface Props {
  storage: Storage;
  selectedStorageId?: string;
  setSelectedStorageId: (storageId: string) => void;
}

// One sentence per type rather than STORAGE_TYPE_LABEL_KEYS: inside the sentence the type is
// written in lower case, and some languages inflect it
const STORAGE_CARD_TYPE_KEYS: Record<StorageType, TranslationKey> = {
  [StorageType.LOCAL]: 'storages.card.type.local',
  [StorageType.S3]: 'storages.card.type.s3',
  [StorageType.GOOGLE_DRIVE]: 'storages.card.type.googleDrive',
  [StorageType.NAS]: 'storages.card.type.nas',
  [StorageType.AZURE_BLOB]: 'storages.card.type.azureBlob',
  [StorageType.FTP]: 'storages.card.type.ftp',
  [StorageType.SFTP]: 'storages.card.type.sftp',
  [StorageType.RCLONE]: 'storages.card.type.rclone',
};

export const StorageCardComponent = ({
  storage,
  selectedStorageId,
  setSelectedStorageId,
}: Props) => {
  const { t } = useTranslation();

  return (
    <button
      type="button"
      aria-label={storage.name}
      aria-pressed={selectedStorageId === storage.id}
      className={`mb-3 w-full cursor-pointer rounded p-3 text-left break-words shadow ${selectedStorageId === storage.id ? 'bg-selected' : 'bg-surface'}`}
      onClick={() => setSelectedStorageId(storage.id)}
    >
      <div className="mb-1 font-bold">{storage.name}</div>

      <div className="flex items-center">
        <div className="text-sm text-muted">{t(STORAGE_CARD_TYPE_KEYS[storage.settings.type])}</div>

        <img src={getStorageLogoFromType(storage.settings.type)} alt="" className="ml-1 h-4 w-4" />
      </div>
    </button>
  );
};
