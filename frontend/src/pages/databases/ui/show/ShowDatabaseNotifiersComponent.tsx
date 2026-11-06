import { Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database } from '@/entities/databases';
import { type Notifier, getNotifierLogoFromType, notifierApi } from '@/entities/notifiers';
import { useSessionGuard } from '@/shared/auth';
import { LoadFailure } from '@/shared/ui';

interface Props {
  database: Database;
}

// The database holds only the IDs of its notifiers; their names and types come from the
// workspace's notifier list.
export const ShowDatabaseNotifiersComponent = ({ database }: Props) => {
  const { t } = useTranslation();
  const isActive = useSessionGuard();

  const [workspaceNotifiers, setWorkspaceNotifiers] = useState<Notifier[]>();
  const [loadError, setLoadError] = useState<unknown>();
  const [loadAttempt, setLoadAttempt] = useState(0);

  useEffect(() => {
    let isCurrentLoad = true;

    setWorkspaceNotifiers(undefined);
    setLoadError(undefined);

    notifierApi
      .listAllNotifiers(database.workspaceId)
      .then((listedNotifiers) => {
        if (isCurrentLoad && isActive()) setWorkspaceNotifiers(listedNotifiers);
      })
      .catch((error: unknown) => {
        if (isCurrentLoad && isActive()) setLoadError(error);
      });

    return () => {
      isCurrentLoad = false;
    };
  }, [database.workspaceId, loadAttempt, isActive]);

  if (loadError !== undefined) {
    return (
      <LoadFailure error={loadError} onRetry={() => setLoadAttempt((attempt) => attempt + 1)} />
    );
  }

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
