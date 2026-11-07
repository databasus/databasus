import { ActionIcon, Button, Tooltip } from '@mantine/core';
import { Copy } from 'lucide-react';
import type { JSX } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  type AgentArchitecture,
  buildInstallCommand,
  buildLaunchCommand,
} from '@/pages/instance-settings/lib/verificationAgentCommands';
import { CopyableCodeBlock } from '@/pages/instance-settings/ui/verification-agents/CopyableCodeBlock';
import { InlineCodeComponent } from '@/shared/ui';

const ARCHITECTURES: AgentArchitecture[] = ['amd64', 'arm64'];
const AFTER_INSTALLATION_KEYS = [
  'verification.agents.install.runsInBackground',
  'verification.agents.install.checkStatus',
  'verification.agents.install.viewLogs',
  'verification.agents.install.stop',
] as const;

interface Props {
  agentId: string;
  token: string;
  selectedArch: AgentArchitecture;
  onSelectArch: (arch: AgentArchitecture) => void;
  onCopy: (text: string) => void;
}

export const InstallCommands = ({
  agentId,
  token,
  selectedArch,
  onSelectArch,
  onCopy,
}: Props): JSX.Element => {
  const { t } = useTranslation();

  return (
    <>
      <div className="mt-3 flex flex-wrap items-center gap-1 text-sm text-muted">
        <span>{t('verification.agents.agentId')}</span>
        <code className="rounded bg-code px-2 py-0.5 text-xs break-all">{agentId}</code>
        <Tooltip label={t('common.actions.copy')}>
          <ActionIcon
            variant="subtle"
            size="xs"
            aria-label={t('common.actions.copy')}
            onClick={() => onCopy(agentId)}
          >
            <Copy size="1em" className="inline-block shrink-0 align-middle" />
          </ActionIcon>
        </Tooltip>
      </div>

      <div className="mt-4">
        <div className="mb-1 text-sm font-medium">
          {t('verification.agents.install.architecture')}
        </div>
        <div className="flex gap-2">
          {ARCHITECTURES.map((arch) => (
            <Button
              key={arch}
              variant={selectedArch === arch ? 'filled' : 'outline'}
              aria-pressed={selectedArch === arch}
              onClick={() => onSelectArch(arch)}
            >
              {arch}
            </Button>
          ))}
        </div>
      </div>

      <div className="mt-4 font-semibold">{t('verification.agents.install.stepInstall')}</div>
      <CopyableCodeBlock code={buildInstallCommand(selectedArch)} onCopy={onCopy} />

      <div className="mt-4 font-semibold">{t('verification.agents.install.stepLaunch')}</div>
      <p className="mt-1 text-sm text-muted">
        <Trans
          i18nKey="verification.agents.install.capacityDefaults"
          components={{ code: <InlineCodeComponent /> }}
        />
      </p>
      <CopyableCodeBlock code={buildLaunchCommand(agentId, token)} onCopy={onCopy} />

      <div className="mt-4 font-semibold">{t('verification.agents.install.afterInstallation')}</div>
      <ul className="mt-1 list-disc space-y-1 pl-5 text-sm text-muted">
        {AFTER_INSTALLATION_KEYS.map((i18nKey) => (
          <li key={i18nKey}>
            <Trans i18nKey={i18nKey} components={{ code: <InlineCodeComponent /> }} />
          </li>
        ))}
      </ul>
    </>
  );
};
