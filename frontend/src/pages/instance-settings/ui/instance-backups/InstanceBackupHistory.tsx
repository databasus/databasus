import { Alert, Button, Loader } from '@mantine/core';
import { CircleAlert, Download } from 'lucide-react';
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  INSTANCE_BACKUP_COPY_STATUS_LABEL_KEYS,
  INSTANCE_BACKUP_STATUS_LABEL_KEYS,
  INSTANCE_BACKUP_TRIGGER_LABEL_KEYS,
  type InstanceBackup,
  type InstanceBackupCopy,
  instanceBackupApi,
} from '@/entities/instance-backups';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { usePolledPagedList } from '@/shared/hooks';
import { translateApiError, useLocale } from '@/shared/i18n';
import { formatBytes, followFileLink } from '@/shared/lib';
import { useFeedback } from '@/shared/lib/feedback';
import { formatInstant } from '@/shared/time';
import { LoadFailure } from '@/shared/ui';

const HISTORY_PAGE_SIZE = 10;
const HISTORY_REFRESH_INTERVAL_MS = 15_000;

interface Props {
  hasDestinations: boolean;
  onStarted: () => void;
}

export function InstanceBackupHistory({ hasDestinations, onStarted }: Props) {
  const { t, i18n } = useTranslation();
  const { formatNumber } = useLocale();
  const notify = useFeedback();
  const isCurrent = useSessionGuard();
  const [loadError, setLoadError] = useState<unknown>();
  const [startError, setStartError] = useState<unknown>();
  const [downloadingCopyId, setDownloadingCopyId] = useState<string>();

  const fetchPage = useCallback(async (after: string | undefined) => {
    setLoadError(undefined);
    return instanceBackupApi.listBackups({ limit: HISTORY_PAGE_SIZE, after });
  }, []);
  const getItemKey = useCallback((backup: InstanceBackup) => backup.id, []);
  const history = usePolledPagedList<InstanceBackup, string>({
    fetchPage,
    getItemKey,
    onError: setLoadError,
    resetKey: '',
    refreshIntervalMs: HISTORY_REFRESH_INTERVAL_MS,
  });

  const startMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof instanceBackupApi.startBackup>) =>
      instanceBackupApi.startBackup(...parameters),
  });
  const downloadMutation = useSessionMutation({
    mutationFn: ({ backupId, copyId }: { backupId: string; copyId: string }) =>
      instanceBackupApi.createDownloadToken(backupId, copyId),
  });
  const isStarting = startMutation.isPending;

  const start = async () => {
    if (isStarting) return;

    setStartError(undefined);

    try {
      const backup = await startMutation.mutateAsync([]);
      if (!isCurrent()) return;

      history.replaceOrPrependItem(backup);
      onStarted();
      notify({ kind: 'success', title: i18n.t('instanceBackups.history.started') });
    } catch (error: unknown) {
      if (isCurrent()) setStartError(error);
    } finally {
      if (isCurrent()) startMutation.reset();
    }
  };

  const download = async (backup: InstanceBackup, copy: InstanceBackupCopy) => {
    if (downloadingCopyId) return;

    setDownloadingCopyId(copy.id);

    try {
      const token = await downloadMutation.mutateAsync({ backupId: backup.id, copyId: copy.id });
      if (isCurrent())
        followFileLink(instanceBackupApi.getDownloadUrl(token.token), token.fileName);
    } catch (error: unknown) {
      if (isCurrent()) notify({ kind: 'error', title: translateApiError(error, i18n.t) });
    } finally {
      if (isCurrent()) {
        setDownloadingCopyId(undefined);
        downloadMutation.reset();
      }
    }
  };

  return (
    <div className="mt-6">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-base font-semibold">{t('instanceBackups.history.title')}</h3>
        <Button
          className="!h-auto min-h-8 max-w-full !whitespace-normal"
          size="xs"
          variant="outline"
          onClick={() => void start()}
          loading={isStarting}
          disabled={isStarting || !hasDestinations}
          aria-busy={isStarting}
        >
          {t('instanceBackups.history.start')}
        </Button>
      </div>
      {!hasDestinations && (
        <p className="mt-1 text-xs text-muted">
          {t('instanceBackups.history.startNeedsDestination')}
        </p>
      )}
      <p className="mt-1 text-xs text-muted">{t('instanceBackups.history.downloadOnly')}</p>

      {startError !== undefined && (
        <Alert
          className="mt-3"
          color="danger"
          icon={<CircleAlert aria-hidden size={16} />}
          title={translateApiError(startError, t)}
        />
      )}

      {history.isLoading ? (
        <Loader className="mt-3" role="status" />
      ) : loadError !== undefined && history.items.length === 0 ? (
        <LoadFailure className="mt-3" error={loadError} onRetry={() => void history.reload()} />
      ) : history.items.length === 0 ? (
        <p className="mt-3 text-muted">{t('instanceBackups.history.empty')}</p>
      ) : (
        <ul className="mt-3 space-y-3">
          {history.items.map((backup) => (
            <li key={backup.id} className="rounded border border-border px-3 py-2">
              <div className="flex flex-wrap items-baseline justify-between gap-x-3">
                <span className="font-medium">
                  {t(INSTANCE_BACKUP_STATUS_LABEL_KEYS[backup.status])}
                </span>
                <span className="text-xs text-muted">
                  {t(INSTANCE_BACKUP_TRIGGER_LABEL_KEYS[backup.trigger])}
                </span>
              </div>
              <div className="mt-1 text-xs text-muted">
                {t('instanceBackups.history.snapshot', {
                  time: formatInstant(backup.snapshotAt ?? backup.startedAt),
                  version: backup.sourceRelease,
                })}
              </div>
              {typeof backup.sizeBytes === 'number' && (
                <div className="text-xs text-muted">
                  {t('instanceBackups.history.size', {
                    size: formatBytes(backup.sizeBytes, formatNumber),
                  })}
                </div>
              )}
              {backup.failureReason && (
                <div className="mt-1 text-xs break-words text-danger">{backup.failureReason}</div>
              )}
              <ul className="mt-2 space-y-1">
                {backup.copies.map((copy) => (
                  <li key={copy.id} className="flex flex-wrap items-center gap-x-2 text-xs">
                    <span className="min-w-0 wrap-anywhere">
                      {t('instanceBackups.history.copy', {
                        storage: copy.storageName,
                        workspace: copy.workspaceName,
                        status: t(INSTANCE_BACKUP_COPY_STATUS_LABEL_KEYS[copy.status]),
                      })}
                    </span>
                    {copy.isDownloadable && (
                      <Button
                        size="compact-xs"
                        variant="subtle"
                        leftSection={<Download aria-hidden size={12} />}
                        loading={downloadingCopyId === copy.id}
                        disabled={downloadingCopyId !== undefined}
                        aria-label={t('instanceBackups.history.downloadFrom', {
                          storage: copy.storageName,
                        })}
                        onClick={() => void download(backup, copy)}
                      >
                        {t('instanceBackups.history.download')}
                      </Button>
                    )}
                    {copy.lastError && (
                      <span className="w-full break-words text-danger">{copy.lastError}</span>
                    )}
                  </li>
                ))}
              </ul>
            </li>
          ))}
        </ul>
      )}

      {history.hasMore && (
        <Button
          className="mt-3"
          size="xs"
          variant="default"
          loading={history.isLoadingMore}
          onClick={() => void history.loadNextPage()}
        >
          {t('instanceBackups.history.showMore')}
        </Button>
      )}
    </div>
  );
}
