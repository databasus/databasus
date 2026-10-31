import {
  CopyOutlined,
  DeleteOutlined,
  EyeOutlined,
  LoadingOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { App, Button, Input, Modal, Popconfirm, Spin, Table, Tag, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffectEvent, useEffect, useRef, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  type VerificationAgent,
  verificationAgentApi,
} from '../../../../entities/verification/agents';
import { useSessionGuard } from '../../../../shared/auth';
import {
  type LocalizedText,
  getWebsitePageUrl,
  translateApiError,
  translateLocalizedText,
  useLocale,
} from '../../../../shared/i18n';
import { ClipboardHelper } from '../../../../shared/lib/ClipboardHelper';
import { formatBytes } from '../../../../shared/lib/formatBytes';
import { InlineCodeComponent } from '../../../../shared/ui';
import {
  type AgentArchitecture,
  TOKEN_PLACEHOLDER,
  buildInstallCommand,
  buildLaunchCommand,
} from '../lib/verificationAgentCommands';
import { AGENT_STATUS_COLORS, AGENT_STATUS_LABEL_KEYS, getAgentStatus } from '../model/agentStatus';

const LIST_REFRESH_MS = 15_000;
// eslint-disable-next-line i18next/no-literal-string -- Tailwind classes
const MUTED_TEXT_CLASS = 'text-gray-400 dark:text-gray-500';

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

export const VerificationAgentsComponent = () => {
  const [agents, setAgents] = useState<VerificationAgent[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [newAgentName, setNewAgentName] = useState('');
  const [isCreating, setIsCreating] = useState(false);

  const [revealedToken, setRevealedToken] = useState<string | null>(null);
  const [revealedTokenAgentName, setRevealedTokenAgentName] = useState<string>('');
  const [revealedTokenAgentId, setRevealedTokenAgentId] = useState<string>('');
  const [selectedArch, setSelectedArch] = useState<AgentArchitecture>('amd64');

  const [rotatingAgent, setRotatingAgent] = useState<VerificationAgent | null>(null);
  const [isRotating, setIsRotating] = useState(false);

  const [viewingInstallAgent, setViewingInstallAgent] = useState<VerificationAgent | null>(null);

  const [deletingAgentId, setDeletingAgentId] = useState<string | null>(null);
  const [currentTimeMs, setCurrentTimeMs] = useState<number>(Date.now());

  const copyToClipboard = async (text: string) => {
    try {
      await ClipboardHelper.copyToClipboard(text);
      if (isCurrent()) message.success(i18n.t('common.messages.copiedToClipboard'));
    } catch {
      if (isCurrent()) message.error(i18n.t('verification.agents.copyFailed'));
    }
  };

  const closeRevealedTokenModal = () => {
    setRevealedToken(null);
    setRevealedTokenAgentName('');
    setRevealedTokenAgentId('');
  };

  const renderCodeBlock = (code: string) => (
    <div className="relative mt-2">
      <pre className="rounded-md bg-gray-900 p-4 pr-10 font-mono text-sm break-all whitespace-pre-wrap text-gray-100">
        {code}
      </pre>
      <Tooltip title={t('common.actions.copy')}>
        <button
          className="absolute top-2 right-2 cursor-pointer rounded p-1 text-gray-400 hover:text-white"
          onClick={() => copyToClipboard(code)}
        >
          <CopyOutlined />
        </button>
      </Tooltip>
    </div>
  );

  const renderArchButton = (arch: AgentArchitecture) => (
    <Button
      type="primary"
      ghost={selectedArch !== arch}
      onClick={() => setSelectedArch(arch)}
      className="mr-2"
    >
      {arch}
    </Button>
  );

  const renderAgentIdRow = (agentId: string) => (
    <div className="mt-3 flex items-center text-sm text-gray-500 dark:text-gray-400">
      <span className="mr-1">{t('verification.agents.agentId')}</span>
      <code className="rounded bg-gray-100 px-2 py-0.5 text-xs dark:bg-gray-700">{agentId}</code>
      <Tooltip title={t('common.actions.copy')}>
        <button
          className="ml-1 cursor-pointer rounded p-1 text-gray-400 hover:text-gray-700 dark:hover:text-white"
          onClick={() => copyToClipboard(agentId)}
        >
          <CopyOutlined style={{ fontSize: 12 }} />
        </button>
      </Tooltip>
    </div>
  );

  const renderArchitecturePicker = () => (
    <div className="mt-4">
      <div className="mb-1 text-sm font-medium text-gray-700 dark:text-gray-300">
        {t('verification.agents.install.architecture')}
      </div>
      <div className="flex">
        {renderArchButton('amd64')}
        {renderArchButton('arm64')}
      </div>
    </div>
  );

  const renderInstallAndLaunchSteps = ({ agentId, token }: { agentId: string; token: string }) => (
    <>
      <div className="mt-4 font-semibold dark:text-white">
        {t('verification.agents.install.stepInstall')}
      </div>
      {renderCodeBlock(buildInstallCommand(selectedArch))}

      <div className="mt-4 font-semibold dark:text-white">
        {t('verification.agents.install.stepLaunch')}
      </div>
      <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
        <Trans
          i18nKey="verification.agents.install.capacityDefaults"
          components={{ code: <InlineCodeComponent /> }}
        />
      </p>
      {renderCodeBlock(buildLaunchCommand(agentId, token))}

      <div className="mt-4 font-semibold dark:text-white">
        {t('verification.agents.install.afterInstallation')}
      </div>
      <ul className="mt-1 list-disc space-y-1 pl-5 text-sm text-gray-600 dark:text-gray-400">
        <li>
          <Trans
            i18nKey="verification.agents.install.runsInBackground"
            components={{ code: <InlineCodeComponent /> }}
          />
        </li>
        <li>
          <Trans
            i18nKey="verification.agents.install.checkStatus"
            components={{ code: <InlineCodeComponent /> }}
          />
        </li>
        <li>
          <Trans
            i18nKey="verification.agents.install.viewLogs"
            components={{ code: <InlineCodeComponent /> }}
          />
        </li>
        <li>
          <Trans
            i18nKey="verification.agents.install.stop"
            components={{ code: <InlineCodeComponent /> }}
          />
        </li>
      </ul>
    </>
  );

  const closeViewingInstallModal = () => {
    setViewingInstallAgent(null);
  };

  const handleCreate = async () => {
    if (!isCurrent() || isCreating) return;
    const name = newAgentName.trim();
    if (!name) {
      message.error(t('verification.agents.create.nameRequired'));
      return;
    }

    setIsCreating(true);

    try {
      const response = await verificationAgentApi.createAgent(name);
      if (!isCurrent()) return;
      setAgents((prev) => [response.agent, ...prev]);
      setRevealedToken(response.token);
      setRevealedTokenAgentName(response.agent.name);
      setRevealedTokenAgentId(response.agent.id);
      setIsCreateModalOpen(false);
      setNewAgentName('');
    } catch (e) {
      if (isCurrent()) message.error(translateApiError(e, i18n.t));
    } finally {
      if (isCurrent()) setIsCreating(false);
    }
  };

  const handleConfirmRotate = async () => {
    if (!isCurrent() || !rotatingAgent || isRotating) return;

    setIsRotating(true);

    try {
      const response = await verificationAgentApi.rotateToken(rotatingAgent.id);
      if (!isCurrent()) return;
      setRevealedToken(response.token);
      setRevealedTokenAgentName(rotatingAgent.name);
      setRevealedTokenAgentId(rotatingAgent.id);
      setRotatingAgent(null);
    } catch (e) {
      if (isCurrent()) message.error(translateApiError(e, i18n.t));
    } finally {
      if (isCurrent()) setIsRotating(false);
    }
  };

  const handleDelete = async (agent: VerificationAgent) => {
    if (!isCurrent() || deletingAgentId) return;
    setDeletingAgentId(agent.id);

    try {
      await verificationAgentApi.deleteAgent(agent.id);
      if (!isCurrent()) return;
      setAgents((prev) => prev.filter((a) => a.id !== agent.id));
      message.success(i18n.t('verification.agents.deleted', { name: agent.name }));
    } catch (e) {
      if (isCurrent()) message.error(translateApiError(e, i18n.t));
    } finally {
      if (isCurrent()) setDeletingAgentId(null);
    }
  };
  const { t, i18n } = useTranslation();
  const { locale, formatRelativeTime, formatNumber } = useLocale();
  const { message } = App.useApp();

  // Pause background polling while any modal is open so it doesn't trample state.
  const hasOpenModal = useRef(false);
  hasOpenModal.current =
    isCreateModalOpen ||
    revealedToken !== null ||
    rotatingAgent !== null ||
    viewingInstallAgent !== null;

  const isCurrent = useSessionGuard();
  const loadRevision = useRef(0);
  const loadAgents = useEffectEvent(async () => {
    if (!isCurrent()) return;
    const requestRevision = ++loadRevision.current;
    try {
      const list = await verificationAgentApi.listAgents();
      if (isCurrent() && loadRevision.current === requestRevision) setAgents(list);
    } catch (e) {
      if (isCurrent() && loadRevision.current === requestRevision)
        message.error(translateApiError(e, i18n.t));
    } finally {
      if (isCurrent() && loadRevision.current === requestRevision) setIsLoading(false);
    }
  });

  useEffect(() => {
    setIsLoading(true);
    void loadAgents();
    return () => {
      loadRevision.current++;
    };
  }, []);

  useEffect(() => {
    const interval = window.setInterval(() => {
      if (!isCurrent()) return;
      setCurrentTimeMs(Date.now());

      if (!hasOpenModal.current) {
        loadAgents();
      }
    }, LIST_REFRESH_MS);

    return () => window.clearInterval(interval);
  }, [isCurrent]);

  const columns: ColumnsType<VerificationAgent> = [
    {
      title: t('common.fields.name'),
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => (
        <span className="font-medium text-gray-900 dark:text-white">{name}</span>
      ),
    },
    {
      title: t('common.fields.status'),
      key: 'status',
      width: 180,
      render: (_, record) => {
        const status = getAgentStatus(record.lastSeenAt, currentTimeMs);
        return (
          <div className="flex items-center gap-2">
            <Tag color={AGENT_STATUS_COLORS[status]} className="!m-0">
              {t(AGENT_STATUS_LABEL_KEYS[status])}
            </Tag>
            {record.lastSeenAt && (
              <span className={`text-xs ${MUTED_TEXT_CLASS}`}>
                {formatRelativeTime(record.lastSeenAt)}
              </span>
            )}
          </div>
        );
      },
    },
    {
      title: t('verification.agents.columns.capacity'),
      key: 'capacity',
      render: (_, record) => {
        const capacityParts = getCapacityParts(record, formatNumber);
        return capacityParts.length > 0 ? (
          <span className="text-xs text-gray-700 dark:text-gray-300">
            {capacityParts.map((part) => translateLocalizedText(part, t)).join(' · ')}
          </span>
        ) : (
          <span className={`text-xs ${MUTED_TEXT_CLASS}`}>
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
        <span className={`text-xs ${MUTED_TEXT_CLASS}`}>{formatRelativeTime(createdAt)}</span>
      ),
    },
    {
      title: '',
      key: 'actions',
      width: 100,
      align: 'right',
      render: (_, record) => (
        <div className="flex items-center justify-end gap-1">
          <Tooltip title={t('verification.agents.actions.viewInstallCommands')}>
            <Button
              type="text"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => setViewingInstallAgent(record)}
            />
          </Tooltip>

          <Tooltip title={t('verification.agents.actions.rotateToken')}>
            <Button
              type="text"
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => setRotatingAgent(record)}
            />
          </Tooltip>

          <Popconfirm
            title={t('verification.agents.deleteConfirmation')}
            okText={t('common.actions.delete')}
            okButtonProps={{ danger: true, loading: deletingAgentId === record.id }}
            cancelText={t('common.actions.cancel')}
            onConfirm={() => handleDelete(record)}
          >
            <Tooltip title={t('common.actions.delete')}>
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </div>
      ),
    },
  ];

  return (
    <section className="my-8 max-w-[800px]">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-baseline gap-3">
          <h2 className="text-xl font-bold dark:text-white">{t('verification.agents.title')}</h2>
        </div>

        <div className="flex items-center gap-2">
          <Button
            type="primary"
            size="small"
            icon={<PlusOutlined />}
            className="!h-auto min-h-8 max-w-full !whitespace-normal"
            onClick={() => setIsCreateModalOpen(true)}
          >
            {t('common.actions.create')}
          </Button>
        </div>
      </div>

      <p className="mb-4 max-w-2xl text-sm text-gray-500 dark:text-gray-400">
        <Trans
          i18nKey="verification.agents.description"
          components={{
            docsLink: (
              <a
                href={getWebsitePageUrl('restoreVerification', locale)}
                target="_blank"
                rel="noopener noreferrer"
              />
            ),
          }}
        />
      </p>

      {isLoading ? (
        <div className="py-4">
          <Spin indicator={<LoadingOutlined spin />} />
        </div>
      ) : agents.length === 0 ? (
        <p className="text-sm text-gray-500 dark:text-gray-400">{t('verification.agents.empty')}</p>
      ) : (
        <Table
          columns={columns}
          dataSource={agents}
          pagination={false}
          rowKey="id"
          size="small"
          className="[&_.ant-table-tbody_.ant-table-cell]:align-top"
        />
      )}

      <Modal
        title={t('verification.agents.create.title')}
        open={isCreateModalOpen}
        onCancel={() => {
          setIsCreateModalOpen(false);
          setNewAgentName('');
        }}
        footer={[
          <Button
            key="cancel"
            onClick={() => {
              setIsCreateModalOpen(false);
              setNewAgentName('');
            }}
          >
            {t('common.actions.cancel')}
          </Button>,
          <Button key="create" type="primary" loading={isCreating} onClick={handleCreate}>
            {t('common.actions.create')}
          </Button>,
        ]}
      >
        <p className="mb-2 text-sm text-gray-500 dark:text-gray-400">
          {t('verification.agents.create.tokenNotice')}
        </p>
        <Input
          placeholder={t('verification.agents.create.namePlaceholder')}
          value={newAgentName}
          onChange={(e) => setNewAgentName(e.target.value)}
          onPressEnter={handleCreate}
          maxLength={200}
          autoFocus
        />
      </Modal>

      <Modal
        title={t('verification.agents.rotate.title')}
        open={rotatingAgent !== null}
        onCancel={() => setRotatingAgent(null)}
        footer={[
          <Button key="cancel" onClick={() => setRotatingAgent(null)}>
            {t('common.actions.cancel')}
          </Button>,
          <Button
            key="rotate"
            type="primary"
            danger
            loading={isRotating}
            onClick={handleConfirmRotate}
          >
            {t('verification.agents.actions.rotateToken')}
          </Button>,
        ]}
      >
        <p className="text-sm text-gray-700 dark:text-gray-300">
          <Trans
            i18nKey="verification.agents.rotate.description"
            components={{ agentName: <strong>{rotatingAgent?.name}</strong> }}
          />
        </p>
      </Modal>

      {revealedToken !== null && (
        <Modal
          title={t('verification.agents.token.title')}
          open={revealedToken !== null}
          onCancel={closeRevealedTokenModal}
          width={640}
          footer={
            <Button type="primary" onClick={closeRevealedTokenModal}>
              {t('verification.agents.token.confirmSaved')}
            </Button>
          }
          maskClosable={false}
        >
          <p className="text-sm text-gray-700 dark:text-gray-300">
            <Trans
              i18nKey="verification.agents.token.tokenFor"
              components={{ agentName: <strong>{revealedTokenAgentName}</strong> }}
            />
          </p>
          {renderCodeBlock(revealedToken ?? '')}

          {renderAgentIdRow(revealedTokenAgentId)}
          {renderArchitecturePicker()}
          {renderInstallAndLaunchSteps({
            agentId: revealedTokenAgentId,
            token: revealedToken ?? '',
          })}

          <p className="mt-3 text-sm text-amber-600 dark:text-amber-400">
            {t('verification.agents.token.shownOnce')}
          </p>
        </Modal>
      )}

      <Modal
        title={t('verification.agents.install.title')}
        open={viewingInstallAgent !== null}
        onCancel={closeViewingInstallModal}
        width={640}
        footer={
          <Button type="primary" onClick={closeViewingInstallModal}>
            {t('common.actions.close')}
          </Button>
        }
      >
        <p className="text-sm text-gray-700 dark:text-gray-300">
          <Trans
            i18nKey="verification.agents.install.description"
            components={{
              agentName: <strong>{viewingInstallAgent?.name}</strong>,
              tokenPlaceholder: (
                <code className="rounded bg-gray-100 px-1 text-xs dark:bg-gray-700">
                  {TOKEN_PLACEHOLDER}
                </code>
              ),
            }}
          />
        </p>

        {viewingInstallAgent && (
          <>
            {renderAgentIdRow(viewingInstallAgent.id)}
            {renderArchitecturePicker()}
            {renderInstallAndLaunchSteps({
              agentId: viewingInstallAgent.id,
              token: TOKEN_PLACEHOLDER,
            })}
          </>
        )}
      </Modal>
    </section>
  );
};
