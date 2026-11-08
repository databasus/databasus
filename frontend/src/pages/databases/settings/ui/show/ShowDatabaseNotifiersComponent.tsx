import { Loader } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import { type Database } from '@/entities/databases';
import { getNotifierLogoFromType, notifierApi } from '@/entities/notifiers';
import { useSessionQuery } from '@/shared/api';
import { LoadFailure } from '@/shared/ui';

interface Props {
  database: Database;
}

// The database holds only the IDs of its notifiers; their names and types come from the
// workspace's notifier list.
export const ShowDatabaseNotifiersComponent = ({ database }: Props) => {
  const { t } = useTranslation();

  const query = useSessionQuery({
    queryKey: ['notifiers', database.workspaceId],
    queryFn: (signal) => notifierApi.listAllNotifiers(database.workspaceId, { signal }),
  });
  const workspaceNotifiers = query.data;
  const loadError = query.error ?? undefined;

  if (loadError !== undefined) {
    return <LoadFailure error={loadError} onRetry={() => void query.refetch()} />;
  }
  if (!workspaceNotifiers) return <Loader size="xs" role="status" />;

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
                  alt=""
                  className="ml-1 h-4 w-4"
                />
              </div>
            ))
          ) : (
            <div className="text-muted">{t('databases.notifiers.empty')}</div>
          )}
        </div>
      </div>
    </div>
  );
};
