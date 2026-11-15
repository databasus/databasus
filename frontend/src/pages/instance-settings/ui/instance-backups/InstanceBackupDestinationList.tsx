import { ActionIcon } from '@mantine/core';
import { TriangleAlert, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import {
  INSTANCE_BACKUP_HEALTH_LABEL_KEYS,
  type InstanceBackupDestinationChoice,
  type InstanceBackupDestinationHealth,
} from '@/entities/instance-backups';
import { STORAGE_TYPE_LABEL_KEYS, StorageType, getStorageLogoFromType } from '@/entities/storages';
import { formatInstant } from '@/shared/time';

interface Props {
  storageIds: string[];
  choices: InstanceBackupDestinationChoice[];
  destinationHealth: InstanceBackupDestinationHealth[];
  isDisabled: boolean;
  onRemove: (storageId: string) => void;
}

export function InstanceBackupDestinationList({
  storageIds,
  choices,
  destinationHealth,
  isDisabled,
  onRemove,
}: Props) {
  const { t } = useTranslation();

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

        return (
          <li
            key={storageId}
            className="flex items-start gap-2 rounded border border-border px-3 py-2"
          >
            {storageType && (
              <img src={getStorageLogoFromType(storageType)} alt="" className="mt-0.5 h-4 w-4" />
            )}
            <div className="min-w-0 flex-1">
              <div className="font-medium text-content">
                {storageName ?? t('instanceBackups.destinations.unknownStorage')}
              </div>
              {workspaceName !== undefined && storageType && (
                <div className="text-xs text-muted">
                  {t('instanceBackups.destinations.location', {
                    workspace: workspaceName,
                    provider: t(STORAGE_TYPE_LABEL_KEYS[storageType]),
                  })}
                </div>
              )}
              <div className="mt-1 text-xs">
                {health ? (
                  <>
                    <span className="font-medium">
                      {t(INSTANCE_BACKUP_HEALTH_LABEL_KEYS[health.health])}
                    </span>
                    {health.lastSuccessfulSnapshotAt && (
                      <span className="block text-muted">
                        {t('instanceBackups.destinations.lastSuccess', {
                          time: formatInstant(health.lastSuccessfulSnapshotAt),
                        })}
                      </span>
                    )}
                  </>
                ) : (
                  <span className="text-muted">{t('instanceBackups.destinations.unsaved')}</span>
                )}
              </div>
              {health?.lastError && (
                <div className="mt-1 text-xs break-words text-danger">{health.lastError}</div>
              )}
              {storageType === StorageType.LOCAL && (
                <div className="mt-1 flex items-start gap-1 text-xs text-warning">
                  <TriangleAlert aria-hidden size={14} className="mt-0.5 shrink-0" />
                  <span>{t('instanceBackups.destinations.localWarning')}</span>
                </div>
              )}
            </div>
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
          </li>
        );
      })}
    </ul>
  );
}
