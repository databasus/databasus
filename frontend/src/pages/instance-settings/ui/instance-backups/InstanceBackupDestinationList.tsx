import { ActionIcon } from '@mantine/core';
import { TriangleAlert, X } from 'lucide-react';
import { Trans, useTranslation } from 'react-i18next';

import {
  INSTANCE_BACKUP_HEALTH_LABEL_KEYS,
  type InstanceBackupDestinationChoice,
  type InstanceBackupDestinationHealth,
  InstanceBackupHealth,
} from '@/entities/instance-backups';
import { STORAGE_TYPE_LABEL_KEYS, StorageType, getStorageLogoFromType } from '@/entities/storages';
import { useLocale } from '@/shared/i18n';

interface Props {
  storageIds: string[];
  choices: InstanceBackupDestinationChoice[];
  destinationHealth: InstanceBackupDestinationHealth[];
  isDisabled?: boolean;
  onRemove?: (storageId: string) => void;
}

export function InstanceBackupDestinationList({
  storageIds,
  choices,
  destinationHealth,
  isDisabled = false,
  onRemove,
}: Props) {
  const { t } = useTranslation();
  const { formatDateTimeWithRelativeTime } = useLocale();

  if (storageIds.length === 0) {
    return <p className="mt-2 text-muted">{t('instanceBackups.destinations.none')}</p>;
  }

  return (
    <ul className="mt-2 space-y-2">
      {storageIds.map((storageId) => {
        const choice = choices.find((item) => item.storageId === storageId);
        const health = destinationHealth.find((item) => item.storageId === storageId);
        const storageName = choice?.storageName ?? health?.storageName;
        const workspaceName = choice?.workspaceName ?? health?.workspaceName;
        const storageType = choice?.storageType ?? health?.storageType;
        const shownStorageName = storageName ?? t('instanceBackups.destinations.unknownStorage');
        const unhealthyDestination =
          health && health.health !== InstanceBackupHealth.HEALTHY ? health : undefined;

        return (
          <li key={storageId} className="rounded-md border border-border p-2">
            <div className="flex items-center gap-2">
              {storageType && (
                <img
                  src={getStorageLogoFromType(storageType)}
                  alt={t(STORAGE_TYPE_LABEL_KEYS[storageType])}
                  title={t(STORAGE_TYPE_LABEL_KEYS[storageType])}
                  className="h-4 w-4 shrink-0"
                />
              )}
              <div className="min-w-0 flex-1 break-words">
                {workspaceName === undefined ? (
                  <span className="text-content">{shownStorageName}</span>
                ) : (
                  <Trans
                    i18nKey="instanceBackups.destinations.storageInWorkspace"
                    components={{
                      storage: <span className="text-content">{shownStorageName}</span>,
                      details: <span className="text-muted" />,
                      workspace: <span>{workspaceName}</span>,
                    }}
                  />
                )}
              </div>
              {onRemove && (
                <ActionIcon
                  size="sm"
                  variant="subtle"
                  disabled={isDisabled}
                  aria-label={t('instanceBackups.destinations.remove', {
                    storage: storageName ?? storageId,
                  })}
                  onClick={() => onRemove(storageId)}
                >
                  <X aria-hidden size={14} />
                </ActionIcon>
              )}
            </div>

            {unhealthyDestination && (
              <div className="mt-1 pl-6 text-xs">
                <span className="font-medium">
                  {t(INSTANCE_BACKUP_HEALTH_LABEL_KEYS[unhealthyDestination.health])}
                </span>
                {unhealthyDestination.lastSuccessfulSnapshotAt && (
                  <span className="block text-muted">
                    {t('instanceBackups.destinations.lastSuccess', {
                      time: formatDateTimeWithRelativeTime(
                        unhealthyDestination.lastSuccessfulSnapshotAt,
                      ),
                    })}
                  </span>
                )}
              </div>
            )}
            {health?.lastError && (
              <div className="mt-1 pl-6 text-xs break-words text-danger">{health.lastError}</div>
            )}
            {storageType === StorageType.LOCAL && (
              <div className="mt-1 flex items-start gap-1.5 pl-6 text-xs text-muted">
                <TriangleAlert aria-hidden size={13} className="mt-px shrink-0 text-warning" />
                <span>{t('instanceBackups.destinations.localWarning')}</span>
              </div>
            )}
          </li>
        );
      })}
    </ul>
  );
}
