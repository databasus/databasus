import { Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import { formatBackupSize } from '@/pages/databases/lib/formatBackupSize';
import { getPhysicalBackupDuration } from '@/pages/databases/lib/physical-backups/getPhysicalBackupDuration';
import { type PhysicalBackupListItem } from '@/pages/databases/model/physical-backups/PhysicalBackupListItem';
import {
  PhysicalBackupActions,
  type PhysicalBackupRowActions,
} from '@/pages/databases/ui/physical-backups/PhysicalBackupActions';
import { PhysicalBackupCreatedAt } from '@/pages/databases/ui/physical-backups/PhysicalBackupCreatedAt';
import { PhysicalBackupStatusBadge } from '@/pages/databases/ui/physical-backups/PhysicalBackupStatusBadge';
import { PhysicalBackupTypeCellComponent } from '@/pages/databases/ui/physical-backups/PhysicalBackupTypeCellComponent';
import { useLocale } from '@/shared/i18n';
import { formatCompactSizeMb } from '@/shared/lib';

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

  const columns: ColumnsType<PhysicalBackupListItem> = [
    {
      title: t('common.fields.type'),
      dataIndex: 'type',
      key: 'type',
      render: (_, backup) => <PhysicalBackupTypeCellComponent backup={backup} />,
    },
    {
      title: t('common.fields.status'),
      dataIndex: 'status',
      key: 'status',
      render: (_, backup) => (
        <PhysicalBackupStatusBadge backup={backup} onShowError={onShowError} />
      ),
    },
    {
      title: (
        <span className="inline-flex items-center gap-1 whitespace-nowrap">
          {t('backups.list.columns.size')}
          {totalUsageMb !== undefined && (
            <span className="text-xs font-normal text-muted">
              {formatCompactSizeMb(totalUsageMb, formatNumber)}
            </span>
          )}
        </span>
      ),
      dataIndex: 'sizeMb',
      key: 'sizeMb',
      width: 150,
      render: (sizeMb: number) => formatBackupSize(sizeMb, formatNumber),
    },
    {
      title: t('backups.list.columns.duration'),
      key: 'duration',
      width: 150,
      render: (_, backup) => getPhysicalBackupDuration(backup),
    },
    {
      title: t('backups.list.columns.created'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (_, backup) => <PhysicalBackupCreatedAt backup={backup} />,
    },
    {
      title: t('backups.list.columns.actions'),
      key: 'actions',
      render: (_, backup) => <PhysicalBackupActions backup={backup} actions={actions} />,
    },
  ];

  return (
    <Table
      bordered
      columns={columns}
      dataSource={backups}
      rowKey="id"
      loading={isLoading}
      size="small"
      pagination={false}
    />
  );
};
