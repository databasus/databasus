import { ActionIcon, Tooltip } from '@mantine/core';
import { Eye, RotateCw, Trash2 } from 'lucide-react';
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
import { AnchoredConfirmation } from '@/shared/ui';
import tableStyles from '@/shared/ui/nativeTable.module.css';

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

  return (
    <div
      className="relative overflow-x-auto"
      role="region"
      aria-label={t('verification.agents.title')}
      tabIndex={0}
    >
      <table className={`${tableStyles.table} min-w-[640px]`}>
        <thead>
          <tr>
            <th scope="col">{t('common.fields.name')}</th>
            <th scope="col" style={{ width: 180 }}>
              {t('common.fields.status')}
            </th>
            <th scope="col">{t('verification.agents.columns.capacity')}</th>
            <th scope="col" style={{ width: 110 }}>
              {t('verification.agents.columns.created')}
            </th>
            <th scope="col" style={{ width: 100, textAlign: 'right' }}>
              {''}
            </th>
          </tr>
        </thead>
        <tbody>
          {agents.map((agent) => (
            <tr key={agent.id}>
              <td>
                <span className="font-medium">{agent.name}</span>
              </td>
              <td>
                {(() => {
                  const status = getAgentStatus(agent.lastSeenAt, currentTimeMs);

                  return (
                    <div className="flex items-center gap-2">
                      <span className={`font-medium ${AGENT_STATUS_TEXT_CLASSES[status]}`}>
                        {t(AGENT_STATUS_LABEL_KEYS[status])}
                      </span>
                      {agent.lastSeenAt && (
                        <span className="text-xs text-muted">
                          {formatRelativeTime(agent.lastSeenAt)}
                        </span>
                      )}
                    </div>
                  );
                })()}
              </td>
              <td>
                {(() => {
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
                })()}
              </td>
              <td>
                <span className="text-xs text-muted">{formatRelativeTime(agent.createdAt)}</span>
              </td>
              <td style={{ textAlign: 'right' }}>
                <div className="flex items-center justify-end gap-1">
                  <Tooltip label={t('verification.agents.actions.viewInstallCommands')}>
                    <ActionIcon
                      variant="subtle"
                      size="xs"
                      aria-label={t('verification.agents.actions.viewInstallCommands')}
                      onClick={() => onViewInstallCommands(agent)}
                    >
                      <Eye size="1em" className="inline-block shrink-0 align-middle" />
                    </ActionIcon>
                  </Tooltip>

                  <Tooltip label={t('verification.agents.actions.rotateToken')}>
                    <ActionIcon
                      variant="subtle"
                      size="xs"
                      aria-label={t('verification.agents.actions.rotateToken')}
                      onClick={() => onRotateToken(agent)}
                    >
                      <RotateCw size="1em" className="inline-block shrink-0 align-middle" />
                    </ActionIcon>
                  </Tooltip>

                  <AnchoredConfirmation
                    title={t('verification.agents.deleteConfirmation')}
                    confirmLabel={t('common.actions.delete')}
                    isConfirming={deletingAgentId === agent.id}
                    onConfirm={() => onDelete(agent)}
                  >
                    <Tooltip label={t('common.actions.delete')}>
                      <ActionIcon
                        variant="subtle"
                        size="xs"
                        color="danger"
                        aria-label={t('common.actions.delete')}
                      >
                        <Trash2 size="1em" className="inline-block shrink-0 align-middle" />
                      </ActionIcon>
                    </Tooltip>
                  </AnchoredConfirmation>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
