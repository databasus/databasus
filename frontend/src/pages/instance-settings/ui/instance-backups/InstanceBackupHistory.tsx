import { Alert, Button, Loader } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { instanceBackupApi } from '@/entities/instance-backups';
import { useInstanceBackupList } from '@/pages/instance-settings/model/useInstanceBackupList';
import { InstanceBackupRow } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupRow';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { LoadFailure } from '@/shared/ui';

const RECENT_BACKUP_COUNT = 5;
const PAGE_SIZE = 10;

interface Props {
  hasDestinations: boolean;
  onStarted: () => void;
}

export function InstanceBackupHistory({ hasDestinations, onStarted }: Props) {
  const { t, i18n } = useTranslation();
  const notify = useFeedback();
  const isCurrent = useSessionGuard();
  const history = useInstanceBackupList({ pageSize: PAGE_SIZE });
  const [startError, setStartError] = useState<unknown>();
  const [isShowingAll, setIsShowingAll] = useState(false);
  const hasOlderBackups = history.hasMore || history.items.length > RECENT_BACKUP_COUNT;
  const shownBackups = isShowingAll ? history.items : history.items.slice(0, RECENT_BACKUP_COUNT);

  const startMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof instanceBackupApi.startBackup>) =>
      instanceBackupApi.startBackup(...parameters),
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

  return (
    <div className="mt-6">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-base font-medium">{t('instanceBackups.history.title')}</h3>
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
      ) : history.loadError !== undefined && shownBackups.length === 0 ? (
        <LoadFailure
          className="mt-3"
          error={history.loadError}
          onRetry={() => void history.reload()}
        />
      ) : shownBackups.length === 0 ? (
        <p className="mt-2 text-muted">{t('instanceBackups.history.empty')}</p>
      ) : (
        <ul className="mt-2 divide-y divide-border rounded border border-border">
          {shownBackups.map((backup) => (
            <InstanceBackupRow key={backup.id} backup={backup} />
          ))}
        </ul>
      )}

      {!isShowingAll && hasOlderBackups && (
        <Button className="mt-2" size="xs" variant="subtle" onClick={() => setIsShowingAll(true)}>
          {t('instanceBackups.history.showAll')}
        </Button>
      )}

      {isShowingAll && history.hasMore && (
        <Button
          className="mt-2"
          size="xs"
          variant="subtle"
          loading={history.isLoadingMore}
          onClick={() => void history.loadNextPage()}
        >
          {t('instanceBackups.history.showMore')}
        </Button>
      )}
    </div>
  );
}
