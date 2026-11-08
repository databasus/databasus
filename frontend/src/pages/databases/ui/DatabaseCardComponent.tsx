import { Info } from 'lucide-react';
import { Trans, useTranslation } from 'react-i18next';
import { Link } from 'react-router';

import { type Database, HEALTH_STATUS_LABEL_KEYS, HealthStatus } from '@/entities/databases';
import { type Storage, getStorageLogoFromType } from '@/entities/storages';
import { useLocale } from '@/shared/i18n';

interface Props {
  database: Database;
  storage?: Storage;
  selectedDatabaseId?: string;
  href: string;
}

export const DatabaseCardComponent = ({ database, storage, selectedDatabaseId, href }: Props) => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();

  return (
    <Link
      to={href}
      aria-current={selectedDatabaseId === database.id ? 'true' : 'false'}
      className={`mb-3 block w-full cursor-pointer rounded p-3 text-left shadow ${selectedDatabaseId === database.id ? 'bg-selected' : 'bg-surface'}`}
      onClick={(event) => {
        if (
          selectedDatabaseId === database.id &&
          !event.ctrlKey &&
          !event.metaKey &&
          !event.shiftKey &&
          !event.altKey
        )
          event.preventDefault();
      }}
    >
      <div className="flex">
        <div className="mb-1 min-w-0 font-bold break-words">{database.name}</div>

        {database.healthStatus && (
          <div className="ml-auto shrink-0 pl-1">
            <div
              className={`rounded px-[6px] py-[2px] text-[10px] ${
                database.healthStatus === HealthStatus.AVAILABLE
                  ? 'bg-success-fill text-on-success-fill'
                  : 'bg-danger text-surface'
              }`}
            >
              {t(HEALTH_STATUS_LABEL_KEYS[database.healthStatus])}
            </div>
          </div>
        )}
      </div>

      {storage && (
        <div className="text-sm text-muted">
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
        <div className="text-muted">
          {t('databases.card.lastBackup', {
            relativeTime: formatRelativeTime(database.lastBackupTime),
          })}
        </div>
      )}

      {database.lastBackupErrorMessage && (
        <div className="mt-1 flex items-center text-sm text-danger underline">
          <Info className="mr-1" size={14} />
          {t('databases.card.hasBackupError')}
        </div>
      )}
    </Link>
  );
};
