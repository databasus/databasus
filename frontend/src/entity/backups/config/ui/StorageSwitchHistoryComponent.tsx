import { Alert, Button, Collapse, Progress, Spin } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { translateApiError } from '../../../../shared/i18n';
import { storageSwitchApi } from '../api/storageSwitchApi';
import type { StorageSwitch } from '../model/StorageSwitch';
import { STORAGE_SWITCH_STATUS_LABEL_KEYS } from '../model/StorageSwitchStatusLabelKeys';

interface Props {
  workspaceId: string;
  databaseId?: string;
  initialOperation?: StorageSwitch;
}

export const StorageSwitchHistoryComponent = ({
  workspaceId,
  databaseId,
  initialOperation,
}: Props) => {
  const { t } = useTranslation();
  const [operations, setOperations] = useState<StorageSwitch[]>(
    initialOperation ? [initialOperation] : [],
  );
  const [nextCursor, setNextCursor] = useState<string>();
  const [error, setError] = useState<unknown>();
  const [isLoading, setIsLoading] = useState(false);

  const loadMore = async () => {
    if (!nextCursor) return;
    const scope = scopeGeneration.current;
    setIsLoading(true);
    try {
      const page = await storageSwitchApi.listStorageSwitches({
        workspaceId,
        databaseId,
        after: nextCursor,
      });
      if (!isMounted.current || scope !== scopeGeneration.current) return;
      setOperations((current) => [...current, ...page.items]);
      setNextCursor(page.nextCursor);
      setError(undefined);
    } catch (e) {
      if (isMounted.current && scope === scopeGeneration.current) setError(e);
    } finally {
      if (isMounted.current && scope === scopeGeneration.current) setIsLoading(false);
    }
  };

  const refreshPending = async () => {
    const scope = scopeGeneration.current;
    try {
      const refreshed = await Promise.all(
        operations
          .filter((operation) => operation.status === 'CLEANUP_PENDING')
          .map((operation) => storageSwitchApi.getStorageSwitch(operation.id)),
      );
      if (!isMounted.current || scope !== scopeGeneration.current) return;
      setOperations((current) =>
        current.map((operation) =>
          operation.status === 'CLEANUP_PENDING'
            ? (refreshed.find((updated) => updated.id === operation.id) ?? operation)
            : operation,
        ),
      );
      setError(undefined);
    } catch (e) {
      if (isMounted.current && scope === scopeGeneration.current) setError(e);
    }
  };

  const loadHistory = async () => {
    const generation = ++historyReadGeneration.current;
    try {
      const page = await storageSwitchApi.listStorageSwitches({ workspaceId, databaseId });
      if (!isMounted.current || generation !== historyReadGeneration.current) return;
      setOperations(page.items);
      setNextCursor(page.nextCursor);
      setError(undefined);
    } catch (e) {
      if (isMounted.current && generation === historyReadGeneration.current) setError(e);
    }
  };

  const retryStatus = () => {
    if (operations.some((operation) => operation.status === 'CLEANUP_PENDING')) {
      void refreshPending();
    } else {
      void loadHistory();
    }
  };

  const isMounted = useRef(true);
  const historyReadGeneration = useRef(0);
  const scopeGeneration = useRef(0);

  useEffect(() => {
    isMounted.current = true;
    return () => {
      isMounted.current = false;
    };
  }, []);

  useEffect(() => {
    scopeGeneration.current++;
    setIsLoading(false);
    setOperations(initialOperation ? [initialOperation] : []);
    setNextCursor(undefined);
    setError(undefined);
  }, [workspaceId, databaseId]);

  useEffect(() => {
    if (initialOperation) {
      setOperations((current) => [
        initialOperation,
        ...current.filter((operation) => operation.id !== initialOperation.id),
      ]);
    }
    void loadHistory();
  }, [workspaceId, databaseId, initialOperation?.id]);

  useEffect(() => {
    if (
      error !== undefined ||
      !operations.some((operation) => operation.status === 'CLEANUP_PENDING')
    )
      return;
    const timer = setTimeout(refreshPending, 2000);
    return () => clearTimeout(timer);
  }, [operations, error]);

  return (
    <Collapse
      defaultActiveKey={['storage-switches']}
      className="my-3"
      items={[
        {
          key: 'storage-switches',
          label: t('backups.storageSwitch.history'),
          children: (
            <div className="space-y-3">
              {error !== undefined && (
                <Alert
                  type="error"
                  message={translateApiError(error, t)}
                  action={<Button onClick={retryStatus}>{t('backups.storageSwitch.retry')}</Button>}
                />
              )}
              {operations.map((operation) => (
                <div key={operation.id}>
                  <Alert
                    type={operation.remaining > 0 ? 'warning' : 'info'}
                    message={t(STORAGE_SWITCH_STATUS_LABEL_KEYS[operation.status])}
                    description={
                      <>
                        <div>
                          {t('backups.storageSwitch.target', {
                            storageId: operation.targetStorageId,
                          })}
                        </div>
                        <div>
                          {t('backups.storageSwitch.database', {
                            databaseId: operation.databaseId,
                          })}
                        </div>
                        <div>
                          {t('backups.storageSwitch.counts', {
                            total: operation.total,
                            pending: operation.pending,
                            deleted: operation.deleted,
                            remaining: operation.remaining,
                          })}
                        </div>
                        <Progress
                          percent={
                            operation.total === 0
                              ? 100
                              : Math.round(
                                  (100 * (operation.total - operation.pending)) / operation.total,
                                )
                          }
                          status={
                            operation.remaining > 0
                              ? 'exception'
                              : operation.status === 'CLEANUP_PENDING'
                                ? 'active'
                                : 'success'
                          }
                        />
                      </>
                    }
                  />
                </div>
              ))}
              {!operations.length && error === undefined && (
                <div>{t('backups.storageSwitch.empty')}</div>
              )}
              {nextCursor && (
                <Button onClick={loadMore} disabled={isLoading}>
                  {isLoading ? <Spin size="small" /> : t('backups.storageSwitch.more')}
                </Button>
              )}
            </div>
          ),
        },
      ]}
    />
  );
};
