import { useTranslation } from 'react-i18next';

import { type Database } from '../../../../entity/databases';
import { getNotifierLogoFromType } from '../../../../entity/notifiers';

interface Props {
  database: Database;
}

export const ShowDatabaseNotifiersComponent = ({ database }: Props) => {
  const { t } = useTranslation();

  return (
    <div>
      <div className="flex w-full">
        <div className="min-w-[150px] pr-2">{t('databases.notifiers.notifyTo')}</div>

        <div>
          {database.notifiers && database.notifiers.length > 0 ? (
            database.notifiers.map((notifier) => (
              <div className="flex items-center" key={notifier.id}>
                <div>- {notifier.name}</div>
                <img
                  src={getNotifierLogoFromType(notifier.notifierType)}
                  className="ml-1 h-4 w-4"
                />
              </div>
            ))
          ) : (
            <div className="text-gray-500 dark:text-gray-400">{t('databases.notifiers.empty')}</div>
          )}
        </div>
      </div>
    </div>
  );
};
