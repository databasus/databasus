import { InfoCircleOutlined } from '@ant-design/icons';
import { Table, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';

import { formatBackupSize } from '@/pages/databases/lib/formatBackupSize';
import type { LogicalBackup } from '@/pages/databases/model/LogicalBackup';
import {
  LogicalBackupActions,
  type LogicalBackupRowActions,
} from '@/pages/databases/ui/logical-backups/LogicalBackupActions';
import { LogicalBackupStatusLabel } from '@/pages/databases/ui/logical-backups/LogicalBackupStatusLabel';
import { useLocale } from '@/shared/i18n';
import { formatDuration } from '@/shared/lib';
import { getUserTimeFormat } from '@/shared/time';

interface Props {
  backups: LogicalBackup[];
  isLoading: boolean;
  actions: LogicalBackupRowActions;
  onShowError: (backup: LogicalBackup) => void;
}

export const LogicalBackupsTable = ({ backups, isLoading, actions, onShowError }: Props) => {
  const { t } = useTranslation();
  const { formatNumber, formatRelativeTime } = useLocale();

  const columns: ColumnsType<LogicalBackup> = [
    {
      title: t('backups.list.columns.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (createdAt: string) => (
        <div>
          {dayjs.utc(createdAt).local().format(getUserTimeFormat().format)} <br />
          <span className="text-muted">({formatRelativeTime(dayjs.utc(createdAt).local())})</span>
        </div>
      ),
      sorter: (a, b) => dayjs(a.createdAt).unix() - dayjs(b.createdAt).unix(),
      defaultSortOrder: 'descend',
    },
    {
      title: t('common.fields.status'),
      dataIndex: 'status',
      key: 'status',
      render: (_, backup: LogicalBackup) => (
        <LogicalBackupStatusLabel backup={backup} onShowError={onShowError} />
      ),
    },
    {
      title: (
        <div className="flex items-center">
          {t('backups.list.columns.size')}
          <Tooltip className="ml-1" title={t('backups.logical.list.sizeTooltip')}>
            <InfoCircleOutlined />
          </Tooltip>
        </div>
      ),
      dataIndex: 'sizeMb',
      key: 'sizeMb',
      width: 150,
      render: (sizeMb: number, backup: LogicalBackup) => (
        <div>
          <div>{formatBackupSize(sizeMb, formatNumber)}</div>
          {backup.rawDatabaseSizeMb > 0 && (
            <div className="text-xs text-muted">
              {t('backups.logical.list.databaseSize', {
                size: formatBackupSize(backup.rawDatabaseSizeMb, formatNumber),
              })}
            </div>
          )}
        </div>
      ),
    },
    {
      title: t('backups.list.columns.duration'),
      dataIndex: 'durationMs',
      key: 'durationMs',
      width: 150,
      render: (durationMs: number) => formatDuration(durationMs),
    },
    {
      title: t('backups.list.columns.actions'),
      dataIndex: '',
      key: '',
      render: (_, backup: LogicalBackup) => (
        <LogicalBackupActions backup={backup} actions={actions} />
      ),
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
