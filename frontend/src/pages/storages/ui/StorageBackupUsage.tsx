import { Loader } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import { useStorageUsage } from '@/pages/storages/lib/useStorageUsage';
import { useLocale } from '@/shared/i18n';
import { formatCompactSizeMb } from '@/shared/lib';
import { LoadFailure } from '@/shared/ui';

interface Props {
  storageId: string;
  workspaceId: string;
}

export const StorageBackupUsage = ({ storageId, workspaceId }: Props) => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();
  const { totalUsageMb, usageError, retry } = useStorageUsage({ storageId, workspaceId });

  return (
    <div className="text-sm text-muted">
      {totalUsageMb !== undefined && (
        <div>
          {t('backups.list.totalUsage', { size: formatCompactSizeMb(totalUsageMb, formatNumber) })}
        </div>
      )}
      {usageError !== undefined ? (
        <div className="mt-1">
          <div>{t('storages.usageUnavailable')}</div>
          <LoadFailure error={usageError} onRetry={retry} />
        </div>
      ) : totalUsageMb === undefined ? (
        <Loader size="xs" role="status" />
      ) : null}
    </div>
  );
};
