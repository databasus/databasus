import { DeleteOutlined, EyeOutlined, ReloadOutlined } from '@ant-design/icons';
import { Button, Popconfirm, Table, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import type { VerificationAgent } from '@/entities/verification-agents';
import {
  AGENT_STATUS_LABEL_KEYS,
  AGENT_STATUS_TEXT_CLASSES,
  getAgentStatus,
} from '@/pages/instance-settings/model/agentStatus';
import { type LocalizedText, translateLocalizedText, useLocale } from '@/shared/i18n';
import { formatBytes } from '@/shared/lib/formatBytes';

interface Props {
  agents: VerificationAgent[];
  currentTimeMs: number;
  deletingAgentId?: string;
  onViewInstallCommands: (agent: VerificationAgent) => void;
  onRotateToken: (agent: VerificationAgent) => void;
  onDelete: (agent: VerificationAgent) => Promise<void>;
}

const getCapacityParts = (
  agent: VerificationAgent,
  formatNumber: ReturnType<typeof useLocale>['formatNumber'],
): LocalizedText[] => {
  const parts: LocalizedText[] = [];

  if (agent.maxCpu > 0) {
    parts.push({ key: 'verification.agents.capacity.cpu', params: { count: agent.maxCpu } });
  }
  if (agent.maxRamMb > 0) {
    parts.push({
      key: 'verification.agents.capacity.ram',
      params: { size: formatBytes(agent.maxRamMb * 1024 * 1024, formatNumber) },
    });
  }
  if (agent.maxDiskGb > 0) {
    parts.push({ key: 'verification.agents.capacity.disk', params: { count: agent.maxDiskGb } });
  }
  if (agent.maxConcurrentJobs > 0) {
    parts.push({
      key: 'verification.agents.capacity.jobs',
      params: { count: agent.maxConcurrentJobs },
    });
  }

  return parts;
};

export const VerificationAgentsTable = ({
  agents,
  currentTimeMs,
  deletingAgentId,
  onViewInstallCommands,
  onRotateToken,
  onDelete,
}: Props): JSX.Element => {
  const { t } = useTranslation();
  const { formatRelativeTime, formatNumber } = useLocale();

  const columns: ColumnsType<VerificationAgent> = [
    {
      title: t('common.fields.name'),
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => <span className="font-medium">{name}</span>,
    },
    {
      title: t('common.fields.status'),
      key: 'status',
      width: 180,
      render: (_, agent) => {
        const status = getAgentStatus(agent.lastSeenAt, currentTimeMs);

        return (
          <div className="flex items-center gap-2">
            <span className={`font-medium ${AGENT_STATUS_TEXT_CLASSES[status]}`}>
              {t(AGENT_STATUS_LABEL_KEYS[status])}
            </span>
            {agent.lastSeenAt && (
              <span className="text-xs text-muted">{formatRelativeTime(agent.lastSeenAt)}</span>
            )}
          </div>
        );
      },
    },
    {
      title: t('verification.agents.columns.capacity'),
      key: 'capacity',
      render: (_, agent) => {
        const capacityParts = getCapacityParts(agent, formatNumber);

        return capacityParts.length > 0 ? (
          <span className="text-xs">
            {capacityParts.map((part) => translateLocalizedText(part, t)).join(' · ')}
          </span>
        ) : (
          <span className="text-xs text-muted">
            {t('verification.agents.capacity.notReported')}
          </span>
        );
      },
    },
    {
      title: t('verification.agents.columns.created'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 110,
      render: (createdAt: string) => (
        <span className="text-xs text-muted">{formatRelativeTime(createdAt)}</span>
      ),
    },
    {
      title: '',
      key: 'actions',
      width: 100,
      align: 'right',
      render: (_, agent) => (
        <div className="flex items-center justify-end gap-1">
          <Tooltip title={t('verification.agents.actions.viewInstallCommands')}>
            <Button
              type="text"
              size="small"
              aria-label={t('verification.agents.actions.viewInstallCommands')}
              icon={<EyeOutlined />}
              onClick={() => onViewInstallCommands(agent)}
            />
          </Tooltip>

          <Tooltip title={t('verification.agents.actions.rotateToken')}>
            <Button
              type="text"
              size="small"
              aria-label={t('verification.agents.actions.rotateToken')}
              icon={<ReloadOutlined />}
              onClick={() => onRotateToken(agent)}
            />
          </Tooltip>

          <Popconfirm
            title={t('verification.agents.deleteConfirmation')}
            okText={t('common.actions.delete')}
            okButtonProps={{ danger: true, loading: deletingAgentId === agent.id }}
            cancelText={t('common.actions.cancel')}
            onConfirm={() => onDelete(agent)}
          >
            <Tooltip title={t('common.actions.delete')}>
              <Button
                type="text"
                size="small"
                danger
                aria-label={t('common.actions.delete')}
                icon={<DeleteOutlined />}
              />
            </Tooltip>
          </Popconfirm>
        </div>
      ),
    },
  ];

  return (
    <Table
      columns={columns}
      dataSource={agents}
      pagination={false}
      rowKey="id"
      size="small"
      className="[&_.ant-table-tbody_.ant-table-cell]:align-top"
    />
  );
};
