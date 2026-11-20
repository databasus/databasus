import { Alert, Tooltip } from '@mantine/core';
import type { ReactNode } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { InstanceBackupHealth, type InstanceBackupStatus } from '@/entities/instance-backups';
import { getInstanceBackupHealthPresentation } from '@/pages/instance-settings/model/getInstanceBackupHealthPresentation';
import { getNewestSuccessfulSnapshotAt } from '@/pages/instance-settings/model/getNewestSuccessfulSnapshotAt';
import { useLocale } from '@/shared/i18n';

interface Props {
  status: InstanceBackupStatus;
  children?: ReactNode;
}

export function InstanceBackupStatusBlock({ status, children }: Props) {
  const { t } = useTranslation();
  const { formatRelativeTime, formatDateTime, formatDateTimeWithRelativeTime } = useLocale();
  const { color, Icon, titleKey } = getInstanceBackupHealthPresentation(status.health);
  const newestSnapshotAt = getNewestSuccessfulSnapshotAt(status.destinations);
  const failingDestinations = status.destinations.filter((destination) => destination.lastError);
  const isNotConfigured = status.health === InstanceBackupHealth.NOT_CONFIGURED;
  const hasDestinations = status.destinations.length > 0;

  return (
    <Alert
      className="mt-4"
      role="status"
      color={color}
      icon={<Icon aria-hidden size={20} />}
      title={t(titleKey)}
    >
      <div className="space-y-1">
        {isNotConfigured && <p>{t('instanceBackups.setup.description')}</p>}

        {status.incident && (
          <p>
            {t('instanceBackups.incidents.since', {
              time: formatDateTimeWithRelativeTime(status.incident.startedAt),
            })}
          </p>
        )}

        {hasDestinations &&
          (newestSnapshotAt ? (
            <p>
              <Trans
                i18nKey="instanceBackups.status.lastBackup"
                components={{
                  time: (
                    <Tooltip label={formatDateTime(newestSnapshotAt)}>
                      <time
                        dateTime={newestSnapshotAt}
                        tabIndex={0}
                        className="underline decoration-dotted underline-offset-2"
                      >
                        {formatRelativeTime(newestSnapshotAt)}
                      </time>
                    </Tooltip>
                  ),
                }}
              />
            </p>
          ) : (
            <p>{t('instanceBackups.status.noSuccessfulBackup')}</p>
          ))}

        {failingDestinations.map((destination) => (
          <p key={destination.storageId} className="break-words">
            {t('instanceBackups.status.storageError', {
              storage: destination.storageName,
              error: destination.lastError,
            })}
          </p>
        ))}

        {status.incident?.lastMailError && (
          <p className="break-words">
            {t('instanceBackups.incidents.mailFailed', {
              error: status.incident.lastMailError,
            })}
          </p>
        )}
      </div>

      {children}
    </Alert>
  );
}
