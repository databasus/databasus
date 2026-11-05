import { Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import { useLocale } from '../../../../shared/i18n';
import { formatBackupSize } from '../../lib/formatBackupSize';
import { getPhysicalBackupDuration } from '../../lib/physical-backups/getPhysicalBackupDuration';
import { type PhysicalBackupListItem } from '../../model/physical-backups/PhysicalBackupListItem';
import { PhysicalBackupActions, type PhysicalBackupRowActions } from './PhysicalBackupActions';
import { PhysicalBackupCreatedAt } from './PhysicalBackupCreatedAt';
import { PhysicalBackupStatusBadge } from './PhysicalBackupStatusBadge';
import { PhysicalBackupTypeCellComponent } from './PhysicalBackupTypeCellComponent';

interface Props {
  backups: PhysicalBackupListItem[];
  isLoading: boolean;
  actions: PhysicalBackupRowActions;
  onShowError: (backup: PhysicalBackupListItem) => void;
}

export const PhysicalBackupsTable = ({
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
      title: t('backups.list.columns.size'),
      dataIndex: 'sizeMb',
      key: 'sizeMb',
      width: 110,
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
