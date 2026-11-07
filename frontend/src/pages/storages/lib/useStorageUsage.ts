import { useEffect, useState } from 'react';

import { storageUsageApi } from '@/pages/storages/api/storageUsageApi';
import { ApiError } from '@/shared/api';
import { isCurrentSession, type SessionSnapshot, useSession } from '@/shared/auth';

interface StorageUsageSource {
  storageId: string;
  workspaceId: string;
}

interface UsageState extends StorageUsageSource {
  session: SessionSnapshot;
  totalUsageMb: number | undefined;
  error: unknown;
}

export const useStorageUsage = ({ storageId, workspaceId }: StorageUsageSource) => {
  const session = useSession();
  const [state, setState] = useState<UsageState>();
  const [retryRevision, setRetryRevision] = useState(0);
  const isSameSource =
    state?.storageId === storageId &&
    state.workspaceId === workspaceId &&
    state.session === session;

  useEffect(() => {
    let isActive = true;
    let isInFlight = false;
    let hasFailed = false;

    setState((previous) => ({
      storageId,
      workspaceId,
      session,
      totalUsageMb:
        previous?.storageId === storageId &&
        previous.workspaceId === workspaceId &&
        previous.session === session
          ? previous.totalUsageMb
          : undefined,
      error: undefined,
    }));

    const readUsage = async () => {
      if (!isActive || isInFlight || hasFailed || !isCurrentSession(session)) return;

      isInFlight = true;

      try {
        const usage = await storageUsageApi.getUsage(storageId);

        if (!isActive || !isCurrentSession(session)) return;

        setState({
          storageId,
          workspaceId,
          session,
          totalUsageMb: usage.totalUsageMb,
          error: undefined,
        });
      } catch (error) {
        if (!isActive || !isCurrentSession(session)) return;

        hasFailed = true;
        const isUnauthorized =
          error instanceof ApiError && [401, 403, 404].includes(error.status ?? 0);

        setState((previous) => ({
          storageId,
          workspaceId,
          session,
          totalUsageMb: isUnauthorized ? undefined : previous?.totalUsageMb,
          error,
        }));
      } finally {
        isInFlight = false;
      }
    };

    const refreshIfVisible = () => {
      // eslint-disable-next-line i18next/no-literal-string -- document visibility state
      if (document.visibilityState !== 'hidden') void readUsage();
    };

    void readUsage();
    const timer = setInterval(refreshIfVisible, 1000);
    // eslint-disable-next-line i18next/no-literal-string -- DOM event name
    document.addEventListener('visibilitychange', refreshIfVisible);

    return () => {
      isActive = false;
      clearInterval(timer);
      // eslint-disable-next-line i18next/no-literal-string -- DOM event name
      document.removeEventListener('visibilitychange', refreshIfVisible);
    };
  }, [storageId, workspaceId, session, retryRevision]);

  return {
    totalUsageMb: isSameSource ? state.totalUsageMb : undefined,
    usageError: isSameSource ? state.error : undefined,
    retry: () => setRetryRevision((revision) => revision + 1),
  };
};
