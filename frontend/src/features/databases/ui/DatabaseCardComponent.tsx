import { InfoCircleOutlined } from '@ant-design/icons';
import { Trans, useTranslation } from 'react-i18next';

import { type Database, HEALTH_STATUS_LABEL_KEYS, HealthStatus } from '../../../entity/databases';
import { type Storage, getStorageLogoFromType } from '../../../entity/storages';
import { useLocale } from '../../../shared/i18n';

interface Props {
  database: Database;
  storage?: Storage;
  selectedDatabaseId?: string;
  setSelectedDatabaseId: (databaseId: string) => void;
}

export const DatabaseCardComponent = ({
  database,
  storage,
  selectedDatabaseId,
  setSelectedDatabaseId,
}: Props) => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();

  return (
    <div
      className={`mb-3 cursor-pointer rounded p-3 shadow ${selectedDatabaseId === database.id ? 'bg-blue-100 dark:bg-blue-800' : 'bg-white dark:bg-gray-800'}`}
      onClick={() => setSelectedDatabaseId(database.id)}
    >
      <div className="flex">
        <div className="mb-1 min-w-0 font-bold break-words">{database.name}</div>

        {database.healthStatus && (
          <div className="ml-auto shrink-0 pl-1">
            <div
              className={`rounded px-[6px] py-[2px] text-[10px] text-white ${
                database.healthStatus === HealthStatus.AVAILABLE ? 'bg-green-500' : 'bg-red-500'
              }`}
            >
              {t(HEALTH_STATUS_LABEL_KEYS[database.healthStatus])}
            </div>
          </div>
        )}
      </div>

      {storage && (
        <div className="text-sm text-gray-500 dark:text-gray-400">
          <Trans
            i18nKey="databases.card.storage"
            components={{
              storageName: (
                <span className="inline-flex items-center">
                  {storage.name}{' '}
                  <img
                    src={getStorageLogoFromType(storage.settings.type)}
                    alt=""
                    className="ml-1 h-4 w-4"
                  />
                </span>
              ),
            }}
          />
        </div>
      )}

      {database.lastBackupTime && (
        <div className="text-gray-500 dark:text-gray-400">
          {t('databases.card.lastBackup', {
            relativeTime: formatRelativeTime(database.lastBackupTime),
          })}
        </div>
      )}

      {database.lastBackupErrorMessage && (
        <div className="mt-1 flex items-center text-sm text-red-600 underline dark:text-red-400">
          <InfoCircleOutlined className="mr-1" style={{ color: 'red' }} />
          {t('databases.card.hasBackupError')}
        </div>
      )}
    </div>
  );
};
