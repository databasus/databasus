import { InfoCircleOutlined } from '@ant-design/icons';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { logicalBackupConfigApi } from '../../../entity/backups/logical';
import { physicalBackupConfigApi } from '../../../entity/backups/physical';
import { type Database, DatabaseType, HealthStatusBadgeComponent } from '../../../entity/databases';
import type { Storage } from '../../../entity/storages';
import { getStorageLogoFromType } from '../../../entity/storages/models/getStorageLogoFromType';
import { useLocale } from '../../../shared/i18n';

interface Props {
  database: Database;
  selectedDatabaseId?: string;
  setSelectedDatabaseId: (databaseId: string) => void;
}

export const DatabaseCardComponent = ({
  database,
  selectedDatabaseId,
  setSelectedDatabaseId,
}: Props) => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const [storage, setStorage] = useState<Storage | undefined>();

  useEffect(() => {
    if (!database.id) return;

    const backupConfigRequest =
      database.type === DatabaseType.POSTGRES_PHYSICAL
        ? physicalBackupConfigApi.getPhysicalBackupConfigByDbId(database.id)
        : logicalBackupConfigApi.getBackupConfigByDbID(database.id);

    backupConfigRequest.then((config) => setStorage(config?.storage));
  }, [database.id, database.type]);

  return (
    <div
      className={`mb-3 cursor-pointer rounded p-3 shadow ${selectedDatabaseId === database.id ? 'bg-blue-100 dark:bg-blue-800' : 'bg-white dark:bg-gray-800'}`}
      onClick={() => setSelectedDatabaseId(database.id)}
    >
      <div className="flex">
        <div className="mb-1 min-w-0 font-bold break-words">{database.name}</div>

        {database.healthStatus && (
          <div className="ml-auto shrink-0 pl-1">
            <HealthStatusBadgeComponent healthStatus={database.healthStatus} />
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
                  {storage.type && (
                    <img
                      src={getStorageLogoFromType(storage.type)}
                      alt=""
                      className="ml-1 h-4 w-4"
                    />
                  )}
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
