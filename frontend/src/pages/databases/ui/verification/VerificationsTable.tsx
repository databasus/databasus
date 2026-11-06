import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Table, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import type { Verification } from '@/pages/databases/model/verification/Verification';
import { formatOptionalDuration } from '@/pages/databases/ui/verification/formatOptionalDuration';
import {
  VerificationActions,
  type VerificationRowActions,
} from '@/pages/databases/ui/verification/VerificationActions';
import { VerificationStatusLabel } from '@/pages/databases/ui/verification/VerificationStatusLabel';
import { VerificationTriggerTag } from '@/pages/databases/ui/verification/VerificationTriggerTag';
import { useLocale } from '@/shared/i18n';
import { getUserTimeFormat } from '@/shared/time';

interface Props {
  verifications: Verification[];
  isLoading: boolean;
  actions: VerificationRowActions;
}

export const VerificationsTable = ({ verifications, isLoading, actions }: Props): JSX.Element => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();

  const columns: ColumnsType<Verification> = [
    {
      title: t('verification.runs.fields.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 220,
      render: (createdAt: string) => (
        <div>
          {dayjs.utc(createdAt).local().format(getUserTimeFormat().format)} <br />
          <span className="text-muted">({formatRelativeTime(dayjs.utc(createdAt))})</span>
        </div>
      ),
    },
    {
      title: t('common.fields.status'),
      dataIndex: 'status',
      key: 'status',
      width: 160,
      render: (_, verification) => <VerificationStatusLabel status={verification.status} />,
    },
    {
      title: t('verification.runs.fields.trigger'),
      dataIndex: 'trigger',
      key: 'trigger',
      width: 110,
      render: (_, verification) => <VerificationTriggerTag trigger={verification.trigger} />,
    },
    {
      title: (
        <div className="flex items-center">
          {t('verification.runs.fields.duration')}
          <Tooltip className="ml-1" title={t('verification.runs.durationTooltip')}>
            <ExclamationCircleOutlined />
          </Tooltip>
        </div>
      ),
      key: 'duration',
      width: 130,
      render: (_, verification) => {
        const { restoreDurationMs, verifyDurationMs } = verification;

        if (restoreDurationMs === undefined && verifyDurationMs === undefined) return '-';

        const totalMs = (restoreDurationMs ?? 0) + (verifyDurationMs ?? 0);

        return (
          <div className="text-sm">
            <div>{formatOptionalDuration(totalMs)}</div>
            <div className="text-xs text-muted">
              {t('verification.runs.restoreDuration', {
                duration: formatOptionalDuration(restoreDurationMs),
              })}
            </div>
          </div>
        );
      },
    },
    {
      title: t('verification.runs.fields.actions'),
      key: 'actions',
      width: 120,
      render: (_, verification) => (
        <VerificationActions verification={verification} actions={actions} />
      ),
    },
  ];

  return (
    <Table
      bordered
      columns={columns}
      dataSource={verifications}
      rowKey="id"
      loading={isLoading}
      size="small"
      pagination={false}
    />
  );
};
