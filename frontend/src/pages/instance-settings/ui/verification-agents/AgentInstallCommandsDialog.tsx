import { Button } from '@mantine/core';
import type { JSX } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import type { VerificationAgent } from '@/entities/verification-agents';
import {
  type AgentArchitecture,
  TOKEN_PLACEHOLDER,
} from '@/pages/instance-settings/lib/verificationAgentCommands';
import { InstallCommands } from '@/pages/instance-settings/ui/verification-agents/InstallCommands';
import { Dialog } from '@/shared/ui';
import { InlineCodeComponent } from '@/shared/ui';

interface Props {
  viewingAgent?: VerificationAgent;
  selectedArch: AgentArchitecture;
  onSelectArch: (arch: AgentArchitecture) => void;
  onCopy: (text: string) => void;
  onClose: () => void;
}

export const AgentInstallCommandsDialog = ({
  viewingAgent,
  selectedArch,
  onSelectArch,
  onCopy,
  onClose,
}: Props): JSX.Element => {
  const { t } = useTranslation();

  return (
    <Dialog
      title={t('verification.agents.install.title')}
      opened={viewingAgent !== undefined}
      onClose={onClose}
      size={640}
    >
      {viewingAgent && (
        <>
          <p className="text-sm">
            <Trans
              i18nKey="verification.agents.install.description"
              components={{
                agentName: <strong>{viewingAgent.name}</strong>,
                tokenPlaceholder: <InlineCodeComponent>{TOKEN_PLACEHOLDER}</InlineCodeComponent>,
              }}
            />
          </p>

          <InstallCommands
            agentId={viewingAgent.id}
            token={TOKEN_PLACEHOLDER}
            selectedArch={selectedArch}
            onSelectArch={onSelectArch}
            onCopy={onCopy}
          />
        </>
      )}
      <div className="mt-5 flex flex-wrap justify-end gap-2">
        <Button variant="filled" onClick={onClose}>
          {t('common.actions.close')}
        </Button>
      </div>
    </Dialog>
  );
};
