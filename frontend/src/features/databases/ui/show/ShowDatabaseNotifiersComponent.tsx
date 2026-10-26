import { Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database } from '../../../../entity/databases';
import { type Notifier, getNotifierLogoFromType, notifierApi } from '../../../../entity/notifiers';
import { translateApiError } from '../../../../shared/i18n';

interface Props {
  database: Database;
}

// The database holds only the IDs of its notifiers; their names and types come from the
// workspace's notifier list.
export const ShowDatabaseNotifiersComponent = ({ database }: Props) => {
  const { t } = useTranslation();

  const [workspaceNotifiers, setWorkspaceNotifiers] = useState<Notifier[]>();

  useEffect(() => {
    notifierApi
      .listAllNotifiers(database.workspaceId)
      .then(setWorkspaceNotifiers)
      .catch((e) => alert(translateApiError(e, t)));
  }, [database.workspaceId]);

  if (!workspaceNotifiers) return <Spin size="small" />;

  const databaseNotifiers = workspaceNotifiers.filter((notifier) =>
    database.notifierIds.includes(notifier.id),
  );

  return (
    <div>
      <div className="flex w-full">
        <div className="min-w-[150px] pr-2">{t('databases.notifiers.notifyTo')}</div>

        <div>
          {databaseNotifiers.length > 0 ? (
            databaseNotifiers.map((notifier) => (
              <div className="flex items-center" key={notifier.id}>
                <div>- {notifier.name}</div>
                <img
                  src={getNotifierLogoFromType(notifier.settings.type)}
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
