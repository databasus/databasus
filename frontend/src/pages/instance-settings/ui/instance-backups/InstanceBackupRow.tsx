import { CircleAlert, CircleCheck, CircleX, type LucideIcon, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import {
  INSTANCE_BACKUP_COPY_STATUS_LABEL_KEYS,
  INSTANCE_BACKUP_STATUS_LABEL_KEYS,
  type InstanceBackup,
} from '@/entities/instance-backups';
import { InstanceBackupDownloadAction } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupDownloadAction';
import { useLocale } from '@/shared/i18n';
import { formatBytes } from '@/shared/lib';

const OUTCOME_PRESENTATIONS: Record<
  InstanceBackup['status'],
  { Icon: LucideIcon; className: string }
> = {
  IN_PROGRESS: { Icon: RefreshCw, className: 'text-accent [&>svg]:animate-spin' },
  COMPLETED: { Icon: CircleCheck, className: 'text-success' },
  PARTIAL: { Icon: CircleAlert, className: 'text-warning' },
  FAILED: { Icon: CircleX, className: 'text-danger' },
  INTERRUPTED: { Icon: CircleX, className: 'text-danger' },
};

interface Props {
  backup: InstanceBackup;
}

export function InstanceBackupRow({ backup }: Props) {
  const { t } = useTranslation();
  const { formatNumber, formatDateTime, formatDateTimeWithRelativeTime } = useLocale();
  const instant = backup.snapshotAt ?? backup.startedAt;
  const time = formatDateTime(instant);
  const { Icon, className } = OUTCOME_PRESENTATIONS[backup.status];
  // A retrying copy keeps the error of its last attempt but has not failed, so it is not listed.
  const failedCopies = backup.copies.filter(
    (copy) => copy.status === 'FAILED' || copy.status === 'INTERRUPTED',
  );

  return (
    <li className="px-3 py-2">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
        <time dateTime={instant}>{formatDateTimeWithRelativeTime(instant)}</time>
        <span className={`inline-flex items-center gap-1 ${className}`}>
          <Icon aria-hidden size={14} className="shrink-0" />
          {t(INSTANCE_BACKUP_STATUS_LABEL_KEYS[backup.status])}
        </span>
        {typeof backup.sizeBytes === 'number' && (
          <span className="text-muted">{formatBytes(backup.sizeBytes, formatNumber)}</span>
        )}
        <span className="ml-auto flex items-center">
          <InstanceBackupDownloadAction
            backup={backup}
            label={t('instanceBackups.history.downloadBackup', { time })}
          />
        </span>
      </div>

      {(backup.failureReason || failedCopies.length > 0) && (
        <div className="mt-1 space-y-1 text-xs break-words text-danger">
          {backup.failureReason && <p>{backup.failureReason}</p>}
          {failedCopies.map((copy) => (
            <p key={copy.id} className="wrap-anywhere">
              {t('instanceBackups.history.copy', {
                storage: copy.storageName,
                workspace: copy.workspaceName,
                status: t(INSTANCE_BACKUP_COPY_STATUS_LABEL_KEYS[copy.status]),
              })}
              <span className="block">{copy.lastError}</span>
            </p>
          ))}
        </div>
      )}
    </li>
  );
}
