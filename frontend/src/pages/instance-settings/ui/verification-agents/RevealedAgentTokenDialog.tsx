import { Button } from '@mantine/core';
import type { JSX } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import type { AgentArchitecture } from '@/pages/instance-settings/lib/verificationAgentCommands';
import { CopyableCodeBlock } from '@/pages/instance-settings/ui/verification-agents/CopyableCodeBlock';
import { InstallCommands } from '@/pages/instance-settings/ui/verification-agents/InstallCommands';
import { Dialog } from '@/shared/ui';

export interface RevealedAgentToken {
  token: string;
  agentId: string;
  agentName: string;
}

interface Props {
  revealedToken?: RevealedAgentToken;
  selectedArch: AgentArchitecture;
  onSelectArch: (arch: AgentArchitecture) => void;
  onCopy: (text: string) => void;
  onClose: () => void;
}

export const RevealedAgentTokenDialog = ({
  revealedToken,
  selectedArch,
  onSelectArch,
  onCopy,
  onClose,
}: Props): JSX.Element => {
  const { t } = useTranslation();

  return (
    <Dialog
      title={t('verification.agents.token.title')}
      opened={revealedToken !== undefined}
      onClose={onClose}
      size={640}
      closeOnClickOutside={false}
    >
      {revealedToken && (
        <>
          <p className="text-sm">
            <Trans
              i18nKey="verification.agents.token.tokenFor"
              components={{ agentName: <strong>{revealedToken.agentName}</strong> }}
            />
          </p>
          <CopyableCodeBlock code={revealedToken.token} onCopy={onCopy} />

          <InstallCommands
            agentId={revealedToken.agentId}
            token={revealedToken.token}
            selectedArch={selectedArch}
            onSelectArch={onSelectArch}
            onCopy={onCopy}
          />

          <p className="mt-3 text-sm text-warning">{t('verification.agents.token.shownOnce')}</p>
        </>
      )}
      <div className="mt-5 flex flex-wrap justify-end gap-2">
        <Button variant="filled" onClick={onClose}>
          {t('verification.agents.token.confirmSaved')}
        </Button>
      </div>
    </Dialog>
  );
};
