import { Button, Modal } from 'antd';
import type { JSX } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import type { VerificationAgent } from '@/entities/verification-agents';
import {
  type AgentArchitecture,
  TOKEN_PLACEHOLDER,
} from '@/pages/instance-settings/lib/verificationAgentCommands';
import { InstallCommands } from '@/pages/instance-settings/ui/verification-agents/InstallCommands';
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
    <Modal
      title={t('verification.agents.install.title')}
      open={viewingAgent !== undefined}
      onCancel={onClose}
      width={640}
      footer={
        <Button type="primary" onClick={onClose}>
          {t('common.actions.close')}
        </Button>
      }
      destroyOnHidden
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
    </Modal>
  );
};
