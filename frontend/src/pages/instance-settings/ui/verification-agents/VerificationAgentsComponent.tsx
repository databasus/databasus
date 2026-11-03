import { LoadingOutlined, PlusOutlined } from '@ant-design/icons';
import { App, Button, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  type VerificationAgent,
  verificationAgentApi,
} from '../../../../entities/verification-agents';
import { useSessionGuard } from '../../../../shared/auth';
import { usePolledPagedList } from '../../../../shared/hooks';
import { getWebsitePageUrl, translateApiError, useLocale } from '../../../../shared/i18n';
import { BrowserClipboard } from '../../../../shared/lib/BrowserClipboard';
import { LoadFailure } from '../../../../shared/ui';
import type { AgentArchitecture } from '../../lib/verificationAgentCommands';
import { AgentInstallCommandsDialog } from './AgentInstallCommandsDialog';
import { CreateAgentDialog } from './CreateAgentDialog';
import { type RevealedAgentToken, RevealedAgentTokenDialog } from './RevealedAgentTokenDialog';
import { RotateAgentTokenDialog } from './RotateAgentTokenDialog';
import { VerificationAgentsTable } from './VerificationAgentsTable';

const LIST_REFRESH_MS = 15_000;
// eslint-disable-next-line i18next/no-literal-string -- the list has one key, it never resets
const AGENTS_LIST_KEY = 'verification-agents';

const fetchAgentsPage = async () => ({ items: await verificationAgentApi.listAgents() });

const getAgentKey = (agent: VerificationAgent) => agent.id;

export const VerificationAgentsComponent = () => {
  const { t } = useTranslation();
  const { locale } = useLocale();
  const { message } = App.useApp();
  const isCurrent = useSessionGuard();

  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [isCreating, setIsCreating] = useState(false);

  const [revealedToken, setRevealedToken] = useState<RevealedAgentToken | undefined>();
  const [selectedArch, setSelectedArch] = useState<AgentArchitecture>('amd64');

  const [rotatingAgent, setRotatingAgent] = useState<VerificationAgent | undefined>();
  const [isRotating, setIsRotating] = useState(false);

  const [viewingInstallAgent, setViewingInstallAgent] = useState<VerificationAgent | undefined>();

  const [deletingAgentId, setDeletingAgentId] = useState<string | undefined>();
  const [currentTimeMs, setCurrentTimeMs] = useState(() => Date.now());

  const showFailure = (error: unknown) => {
    if (isCurrent()) message.error(translateApiError(error, t));
  };

  const copyToClipboard = async (text: string) => {
    try {
      await BrowserClipboard.copyToClipboard(text);
      if (isCurrent()) message.success(t('common.messages.copiedToClipboard'));
    } catch {
      if (isCurrent()) message.error(t('verification.agents.copyFailed'));
    }
  };

  const createAgent = async (name: string) => {
    setIsCreating(true);

    try {
      const { agent, token } = await verificationAgentApi.createAgent(name);
      if (!isCurrent()) return;

      replaceOrPrependItem(agent);
      setRevealedToken({ token, agentId: agent.id, agentName: agent.name });
      setIsCreateDialogOpen(false);
    } catch (error) {
      showFailure(error);
    }

    setIsCreating(false);
  };

  const rotateToken = async () => {
    if (!rotatingAgent || isRotating) return;

    setIsRotating(true);

    try {
      const { token } = await verificationAgentApi.rotateToken(rotatingAgent.id);
      if (!isCurrent()) return;

      setRevealedToken({ token, agentId: rotatingAgent.id, agentName: rotatingAgent.name });
      setRotatingAgent(undefined);
    } catch (error) {
      showFailure(error);
    }

    setIsRotating(false);
  };

  const deleteAgent = async (agent: VerificationAgent) => {
    if (deletingAgentId) return;

    setDeletingAgentId(agent.id);

    try {
      await verificationAgentApi.deleteAgent(agent.id);

      if (isCurrent()) {
        message.success(t('verification.agents.deleted', { name: agent.name }));
        await reload();
      }
    } catch (error) {
      showFailure(error);
    }

    setDeletingAgentId(undefined);
  };

  const {
    items: agents,
    isLoading,
    firstLoadError,
    reload,
    replaceOrPrependItem,
  } = usePolledPagedList({
    fetchPage: fetchAgentsPage,
    getItemKey: getAgentKey,
    refreshIntervalMs: LIST_REFRESH_MS,
    resetKey: AGENTS_LIST_KEY,
    // The first-load failure renders in place of the list, and this list has no next page.
    onError: () => {},
  });

  useEffect(() => {
    const intervalId = setInterval(() => setCurrentTimeMs(Date.now()), LIST_REFRESH_MS);

    return () => clearInterval(intervalId);
  }, []);

  const isCreateDisabled =
    isLoading || firstLoadError !== undefined || deletingAgentId !== undefined;

  return (
    <section className="my-8 max-w-[800px]">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-xl font-bold">{t('verification.agents.title')}</h2>

        <Button
          type="primary"
          size="small"
          icon={<PlusOutlined />}
          className="!h-auto min-h-8 max-w-full !whitespace-normal"
          disabled={isCreateDisabled}
          onClick={() => setIsCreateDialogOpen(true)}
        >
          {t('common.actions.create')}
        </Button>
      </div>

      <p className="mb-4 max-w-2xl text-sm text-muted">
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

      {firstLoadError !== undefined ? (
        <LoadFailure error={firstLoadError} onRetry={() => void reload()} />
      ) : isLoading ? (
        <div className="py-4">
          <Spin indicator={<LoadingOutlined spin />} />
        </div>
      ) : agents.length === 0 ? (
        <p className="text-sm text-muted">{t('verification.agents.empty')}</p>
      ) : (
        <VerificationAgentsTable
          agents={agents}
          currentTimeMs={currentTimeMs}
          deletingAgentId={deletingAgentId}
          onViewInstallCommands={setViewingInstallAgent}
          onRotateToken={setRotatingAgent}
          onDelete={deleteAgent}
        />
      )}

      <CreateAgentDialog
        isOpen={isCreateDialogOpen}
        isCreating={isCreating}
        onCancel={() => setIsCreateDialogOpen(false)}
        onCreate={(name) => void createAgent(name)}
      />

      <RotateAgentTokenDialog
        rotatingAgent={rotatingAgent}
        isRotating={isRotating}
        onCancel={() => setRotatingAgent(undefined)}
        onConfirm={() => void rotateToken()}
      />

      <RevealedAgentTokenDialog
        revealedToken={revealedToken}
        selectedArch={selectedArch}
        onSelectArch={setSelectedArch}
        onCopy={(text) => void copyToClipboard(text)}
        onClose={() => setRevealedToken(undefined)}
      />

      <AgentInstallCommandsDialog
        viewingAgent={viewingInstallAgent}
        selectedArch={selectedArch}
        onSelectArch={setSelectedArch}
        onCopy={(text) => void copyToClipboard(text)}
        onClose={() => setViewingInstallAgent(undefined)}
      />
    </section>
  );
};
