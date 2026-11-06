import { useEffect, useState } from 'react';

import { type BackupCleanup, backupConfigApi } from '@/entities/backup-configs';
import { useSession, useSessionGuard } from '@/shared/auth';

interface CleanupSource {
  databaseId: string;
  workspaceId: string;
  initial: BackupCleanup | undefined;
}

export const useBackupCleanup = ({ databaseId, workspaceId, initial }: CleanupSource) => {
  const [state, setState] = useState<{
    databaseId: string;
    workspaceId: string;
    initial: BackupCleanup | undefined;
    cleanup: BackupCleanup | undefined;
    error?: unknown;
    shouldReadImmediately: boolean;
  }>();

  const retry = () =>
    setState({ databaseId, workspaceId, initial, cleanup, shouldReadImmediately: true });

  const isCurrentSession = useSessionGuard();
  const session = useSession();
  const isSameSource =
    state?.databaseId === databaseId &&
    state?.workspaceId === workspaceId &&
    state?.initial === initial;
  const cleanup = isSameSource ? state?.cleanup : initial;
  const error = isSameSource ? state?.error : undefined;
  const shouldReadImmediately = isSameSource && state?.shouldReadImmediately;

  useEffect(() => {
    if (!cleanup?.pendingFileCount || error !== undefined || !isCurrentSession()) return;

    let isActive = true;
    const timer = setTimeout(
      () => {
        backupConfigApi
          .getBackupConfig(databaseId)
          .then((config) => {
            if (!isActive || !isCurrentSession()) return;

            setState({
              databaseId,
              workspaceId,
              initial,
              cleanup: config.cleanup,
              shouldReadImmediately: false,
            });
          })
          .catch((failure: unknown) => {
            if (!isActive || !isCurrentSession()) return;

            setState({
              databaseId,
              workspaceId,
              initial,
              cleanup,
              error: failure,
              shouldReadImmediately: false,
            });
          });
      },
      shouldReadImmediately ? 0 : 2000,
    );

    return () => {
      isActive = false;
      clearTimeout(timer);
    };
  }, [
    databaseId,
    workspaceId,
    initial,
    cleanup,
    error,
    shouldReadImmediately,
    isCurrentSession,
    session,
  ]);

  return { cleanup, cleanupError: error, retryCleanup: retry };
};
