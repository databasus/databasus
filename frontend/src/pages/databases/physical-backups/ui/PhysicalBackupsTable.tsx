import { Loader } from '@mantine/core';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import { formatBackupSize } from '@/pages/databases/lib/formatBackupSize';
import { getPhysicalBackupDuration } from '@/pages/databases/physical-backups/lib/getPhysicalBackupDuration';
import { type PhysicalBackupListItem } from '@/pages/databases/physical-backups/model/PhysicalBackupListItem';
import {
  PhysicalBackupActions,
  type PhysicalBackupRowActions,
} from '@/pages/databases/physical-backups/ui/PhysicalBackupActions';
import { PhysicalBackupCreatedAt } from '@/pages/databases/physical-backups/ui/PhysicalBackupCreatedAt';
import { PhysicalBackupStatusBadge } from '@/pages/databases/physical-backups/ui/PhysicalBackupStatusBadge';
import { PhysicalBackupTypeCellComponent } from '@/pages/databases/physical-backups/ui/PhysicalBackupTypeCellComponent';
import { useLocale } from '@/shared/i18n';
import { formatCompactSizeMb } from '@/shared/lib';
import tableStyles from '@/shared/ui/nativeTable.module.css';

interface Props {
  totalUsageMb: number | undefined;
  backups: PhysicalBackupListItem[];
  isLoading: boolean;
  actions: PhysicalBackupRowActions;
  onShowError: (backup: PhysicalBackupListItem) => void;
}

export const PhysicalBackupsTable = ({
  totalUsageMb,
  backups,
  isLoading,
  actions,
  onShowError,
}: Props): JSX.Element => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();

  return (
    <div className="relative overflow-x-auto" aria-busy={isLoading}>
      <table className={`${tableStyles.table} ${tableStyles.bordered}`}>
        <thead>
          <tr>
            <th scope="col">{t('common.fields.type')}</th>
            <th scope="col">{t('common.fields.status')}</th>
            <th scope="col" style={{ width: 150 }}>
              <span className="inline-flex items-center gap-1 whitespace-nowrap">
                {t('backups.list.columns.size')}
                {totalUsageMb !== undefined && (
                  <span className="text-xs font-normal text-muted">
                    {formatCompactSizeMb(totalUsageMb, formatNumber)}
                  </span>
                )}
              </span>
            </th>
            <th scope="col" style={{ width: 150 }}>
              {t('backups.list.columns.duration')}
            </th>
            <th scope="col">{t('backups.list.columns.created')}</th>
            <th scope="col">{t('backups.list.columns.actions')}</th>
          </tr>
        </thead>
        <tbody>
          {backups.map((backup) => (
            <tr key={backup.id}>
              <td>
                <PhysicalBackupTypeCellComponent backup={backup} />
              </td>
              <td>
                <PhysicalBackupStatusBadge backup={backup} onShowError={onShowError} />
              </td>
              <td>{formatBackupSize(backup.sizeMb, formatNumber)}</td>
              <td>{getPhysicalBackupDuration(backup)}</td>
              <td>
                <PhysicalBackupCreatedAt backup={backup} />
              </td>
              <td>
                <PhysicalBackupActions backup={backup} actions={actions} />
              </td>
            </tr>
          ))}
          {backups.length === 0 && !isLoading && (
            <tr>
              <td colSpan={6}>
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
