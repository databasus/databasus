import { Loader, Tooltip } from '@mantine/core';
import dayjs from 'dayjs';
import { ArrowDown, ArrowUp, ArrowUpDown, Info } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { formatBackupSize } from '@/pages/databases/lib/formatBackupSize';
import type { LogicalBackup } from '@/pages/databases/logical-backups/model/logicalBackupContract';
import {
  LogicalBackupActions,
  type LogicalBackupRowActions,
} from '@/pages/databases/logical-backups/ui/LogicalBackupActions';
import { LogicalBackupStatusLabel } from '@/pages/databases/logical-backups/ui/LogicalBackupStatusLabel';
import { useLocale } from '@/shared/i18n';
import { formatCompactSizeMb, formatDuration } from '@/shared/lib';
import { getUserTimeFormat } from '@/shared/time';
import tableStyles from '@/shared/ui/nativeTable.module.css';

interface Props {
  totalUsageMb: number | undefined;
  backups: LogicalBackup[];
  isLoading: boolean;
  actions: LogicalBackupRowActions;
  onShowError: (backup: LogicalBackup) => void;
}

export const LogicalBackupsTable = ({
  totalUsageMb,
  backups,
  isLoading,
  actions,
  onShowError,
}: Props) => {
  const { t } = useTranslation();
  const { formatNumber, formatRelativeTime } = useLocale();

  const [sortDirection, setSortDirection] = useState<'ascend' | 'descend' | null>('descend');
  const SortIcon =
    sortDirection === 'ascend' ? ArrowUp : sortDirection === 'descend' ? ArrowDown : ArrowUpDown;
  const sortedBackups = sortDirection
    ? [...backups].sort(
        (first, second) =>
          (dayjs(first.createdAt).unix() - dayjs(second.createdAt).unix()) *
          (sortDirection === 'ascend' ? 1 : -1),
      )
    : backups;

  return (
    <div className="relative overflow-x-auto" aria-busy={isLoading}>
      <table className={`${tableStyles.table} ${tableStyles.bordered}`}>
        <thead>
          <tr>
            <th
              scope="col"
              aria-sort={
                sortDirection === 'ascend'
                  ? 'ascending'
                  : sortDirection === 'descend'
                    ? 'descending'
                    : 'none'
              }
            >
              <button
                type="button"
                className="inline-flex max-w-full items-center gap-1 text-start"
                onClick={() =>
                  setSortDirection((direction) =>
                    direction === 'ascend' ? 'descend' : direction === 'descend' ? null : 'ascend',
                  )
                }
              >
                {t('backups.list.columns.createdAt')}
                <SortIcon
                  aria-hidden
                  size={12}
                  className={sortDirection ? 'shrink-0 text-accent' : 'shrink-0 text-muted'}
                />
              </button>
            </th>
            <th scope="col">{t('common.fields.status')}</th>
            <th scope="col" style={{ width: 150 }}>
              <span className="inline-flex items-center gap-1 whitespace-nowrap">
                {t('backups.list.columns.size')}
                {totalUsageMb !== undefined && (
                  <span className="text-xs font-normal text-muted">
                    {formatCompactSizeMb(totalUsageMb, formatNumber)}
                  </span>
                )}
                <Tooltip className="ml-1" label={t('backups.logical.list.sizeTooltip')}>
                  <Info size="1em" className="inline-block shrink-0 align-middle" />
                </Tooltip>
              </span>
            </th>
            <th scope="col" style={{ width: 150 }}>
              {t('backups.list.columns.duration')}
            </th>
            <th scope="col">{t('backups.list.columns.actions')}</th>
          </tr>
        </thead>
        <tbody>
          {sortedBackups.map((backup) => (
            <tr key={backup.id}>
              <td>
                <div>
                  {dayjs.utc(backup.createdAt).local().format(getUserTimeFormat().format)} <br />
                  <span className="text-muted">
                    ({formatRelativeTime(dayjs.utc(backup.createdAt).local())})
                  </span>
                </div>
              </td>
              <td>
                <LogicalBackupStatusLabel backup={backup} onShowError={onShowError} />
              </td>
              <td>
                <div>
                  <div>{formatBackupSize(backup.sizeMb, formatNumber)}</div>
                  {backup.rawDatabaseSizeMb > 0 && (
                    <div className="text-xs text-muted">
                      {t('backups.logical.list.databaseSize', {
                        size: formatBackupSize(backup.rawDatabaseSizeMb, formatNumber),
                      })}
                    </div>
                  )}
                </div>
              </td>
              <td>{formatDuration(backup.durationMs)}</td>
              <td>
                <LogicalBackupActions backup={backup} actions={actions} />
              </td>
            </tr>
          ))}
          {backups.length === 0 && !isLoading && (
            <tr>
              <td colSpan={5}>
                <div className="py-8 text-center text-muted">{t('backups.list.empty')}</div>
              </td>
            </tr>
          )}
        </tbody>
      </table>
      {isLoading && (
        <div className="absolute inset-0 flex items-center justify-center bg-surface/50">
          <Loader size="sm" role="status" />
        </div>
      )}
    </div>
  );
};
