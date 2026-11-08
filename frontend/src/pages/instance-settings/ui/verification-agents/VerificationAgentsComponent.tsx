import { Button, Loader } from '@mantine/core';
import { Plus } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { type VerificationAgent, verificationAgentApi } from '@/entities/verification-agents';
import type { AgentArchitecture } from '@/pages/instance-settings/lib/verificationAgentCommands';
import { AgentInstallCommandsDialog } from '@/pages/instance-settings/ui/verification-agents/AgentInstallCommandsDialog';
import { CreateAgentDialog } from '@/pages/instance-settings/ui/verification-agents/CreateAgentDialog';
import {
  type RevealedAgentToken,
  RevealedAgentTokenDialog,
} from '@/pages/instance-settings/ui/verification-agents/RevealedAgentTokenDialog';
import { RotateAgentTokenDialog } from '@/pages/instance-settings/ui/verification-agents/RotateAgentTokenDialog';
import { VerificationAgentsTable } from '@/pages/instance-settings/ui/verification-agents/VerificationAgentsTable';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { usePolledPagedList } from '@/shared/hooks';
import { getWebsitePageUrl, translateApiError, useLocale } from '@/shared/i18n';
import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';
import { useFeedback } from '@/shared/lib/feedback';
import { LoadFailure } from '@/shared/ui';
const LIST_REFRESH_MS = 15000;
// eslint-disable-next-line i18next/no-literal-string -- the list has one key, it never resets
const AGENTS_LIST_KEY = 'verification-agents';

const fetchAgentsPage = async () => ({ items: await verificationAgentApi.listAgents() });

const getAgentKey = (agent: VerificationAgent) => agent.id;

export const VerificationAgentsComponent = () => {
  const createAgentMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof verificationAgentApi.createAgent>) =>
      verificationAgentApi.createAgent(...parameters),
  });
  const rotateTokenMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof verificationAgentApi.rotateToken>) =>
      verificationAgentApi.rotateToken(...parameters),
  });
  const deleteAgentMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof verificationAgentApi.deleteAgent>) =>
      verificationAgentApi.deleteAgent(...parameters),
  });

  const { t } = useTranslation();
  const { locale } = useLocale();
  const notify = useFeedback();
  const isCurrent = useSessionGuard();

  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const isCreating = createAgentMutation.isPending;

  const [revealedToken, setRevealedToken] = useState<RevealedAgentToken | undefined>();
  const [selectedArch, setSelectedArch] = useState<AgentArchitecture>('amd64');

  const [rotatingAgent, setRotatingAgent] = useState<VerificationAgent | undefined>();
  const isRotating = rotateTokenMutation.isPending;

  const [viewingInstallAgent, setViewingInstallAgent] = useState<VerificationAgent | undefined>();

  const [deletingAgentId, setDeletingAgentId] = useState<string | undefined>();
  const [currentTimeMs, setCurrentTimeMs] = useState(() => Date.now());

  const showFailure = (error: unknown) => {
    if (isCurrent()) notify({ kind: 'error', title: translateApiError(error, t) });
  };

  const copyToClipboard = async (text: string) => {
    try {
      await BrowserClipboard.copyToClipboard(text);
      if (isCurrent()) notify({ kind: 'success', title: t('common.messages.copiedToClipboard') });
    } catch {
      if (isCurrent()) notify({ kind: 'error', title: t('verification.agents.copyFailed') });
    }
  };

  const createAgent = async (name: string) => {
    try {
      const { agent, token } = await createAgentMutation.mutateAsync([name]);
      if (!isCurrent()) return;

      replaceOrPrependItem(agent);
      setRevealedToken({ token, agentId: agent.id, agentName: agent.name });
      setIsCreateDialogOpen(false);
    } catch (error) {
      showFailure(error);
    }

    createAgentMutation.reset();
  };

  const rotateToken = async () => {
    if (!rotatingAgent || isRotating) return;

    try {
      const { token } = await rotateTokenMutation.mutateAsync([rotatingAgent.id]);
      if (!isCurrent()) return;

      setRevealedToken({ token, agentId: rotatingAgent.id, agentName: rotatingAgent.name });
      setRotatingAgent(undefined);
    } catch (error) {
      showFailure(error);
    }

    rotateTokenMutation.reset();
  };

  const deleteAgent = async (agent: VerificationAgent) => {
    if (deletingAgentId) return;

    setDeletingAgentId(agent.id);

    try {
      await deleteAgentMutation.mutateAsync([agent.id]);

      if (isCurrent()) {
        notify({ kind: 'success', title: t('verification.agents.deleted', { name: agent.name }) });
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
          variant="filled"
          size="xs"
          leftSection={<Plus size="1em" className="inline-block shrink-0 align-middle" />}
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
          <Loader role="status" />
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
