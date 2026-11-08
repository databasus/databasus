import { Loader } from '@mantine/core';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';

import { formatBackupSize } from '@/pages/databases/lib/formatBackupSize';
import type { LogicalBackup } from '@/pages/databases/logical-backups/model/logicalBackupContract';
import {
  LogicalBackupActions,
  type LogicalBackupRowActions,
} from '@/pages/databases/logical-backups/ui/LogicalBackupActions';
import { LogicalBackupStatusLabel } from '@/pages/databases/logical-backups/ui/LogicalBackupStatusLabel';
import { useLocale } from '@/shared/i18n';
import { formatDuration } from '@/shared/lib';
import { getUserTimeFormat } from '@/shared/time';

interface Props {
  backups: LogicalBackup[];
  isLoading: boolean;
  actions: LogicalBackupRowActions;
  onShowError: (backup: LogicalBackup) => void;
}

export const LogicalBackupCards = ({ backups, isLoading, actions, onShowError }: Props) => {
  const { t } = useTranslation();
  const { formatNumber, formatRelativeTime } = useLocale();

  if (isLoading) {
    return (
      <div className="flex justify-center py-8">
        <Loader role="status" />
      </div>
    );
  }

  return (
    <div>
      {backups.map((backup) => (
        <div
          key={backup.id}
          className="mb-2 rounded-lg border border-border bg-surface p-4 shadow-sm"
        >
          <div className="space-y-3">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <div className="text-xs text-muted">{t('backups.list.columns.createdAt')}</div>
                <div className="text-sm font-medium">
                  {dayjs.utc(backup.createdAt).local().format(getUserTimeFormat().format)}
                </div>
                <div className="text-xs text-muted">
                  ({formatRelativeTime(dayjs.utc(backup.createdAt).local())})
                </div>
              </div>
              <div>
                <LogicalBackupStatusLabel backup={backup} onShowError={onShowError} />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="text-xs text-muted">{t('backups.list.columns.size')}</div>
                <div className="text-sm font-medium">
                  {formatBackupSize(backup.sizeMb, formatNumber)}
                </div>
                {backup.rawDatabaseSizeMb > 0 && (
                  <div className="text-xs text-muted">
                    {t('backups.logical.list.databaseSize', {
                      size: formatBackupSize(backup.rawDatabaseSizeMb, formatNumber),
                    })}
                  </div>
                )}
              </div>
              <div>
                <div className="text-xs text-muted">{t('backups.list.columns.duration')}</div>
                <div className="text-sm font-medium">{formatDuration(backup.durationMs)}</div>
              </div>
            </div>

            <div className="flex items-center justify-end border-t border-border pt-3">
              <LogicalBackupActions backup={backup} actions={actions} />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
};
