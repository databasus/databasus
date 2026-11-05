import { LoadingOutlined, WarningOutlined } from '@ant-design/icons';
import { Button } from 'antd';
import { useTranslation } from 'react-i18next';

import type { BackupCleanup } from '../../../../entities/backup-configs';
import { translateApiError, useLocale } from '../../../../shared/i18n';

interface Props {
  cleanup?: BackupCleanup;
  error?: unknown;
  onRetry?: () => void;
}

export const StorageCleanupNotice = ({ cleanup, error, onRetry }: Props) => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();

  if (!cleanup?.pendingFileCount) return null;

  return (
    <div role="status" className="mb-3 max-w-full min-w-0 text-sm break-words">
      <div className="flex items-start gap-2 text-muted">
        <LoadingOutlined aria-hidden className="mt-1 shrink-0" />
        <span>
          {t('backups.cleanup.pending', { count: formatNumber(cleanup.pendingFileCount) })}
        </span>
      </div>
      {cleanup.failedFileCount > 0 && (
        <div className="mt-1 flex items-start gap-2 text-warning">
          <WarningOutlined aria-hidden className="mt-1 shrink-0" />
          <span>
            {t('backups.cleanup.retrying', { count: formatNumber(cleanup.failedFileCount) })}
          </span>
        </div>
      )}
      {error !== undefined && (
        <div className="mt-1 text-danger">
          <span>{translateApiError(error, t)}</span>
          <Button type="link" onClick={onRetry} className="!h-auto max-w-full !whitespace-normal">
            {t('backups.cleanup.retryStatus')}
          </Button>
        </div>
      )}
    </div>
  );
};
