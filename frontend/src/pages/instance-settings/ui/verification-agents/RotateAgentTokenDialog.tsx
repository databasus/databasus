import { Button } from '@mantine/core';
import type { JSX } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import type { VerificationAgent } from '@/entities/verification-agents';
import { Dialog } from '@/shared/ui';
interface Props {
  rotatingAgent?: VerificationAgent;
  isRotating: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}

export const RotateAgentTokenDialog = ({
  rotatingAgent,
  isRotating,
  onCancel,
  onConfirm,
}: Props): JSX.Element => {
  const { t } = useTranslation();

  return (
    <Dialog
      title={t('verification.agents.rotate.title')}
      opened={rotatingAgent !== undefined}
      onClose={onCancel}
    >
      <p className="text-sm">
        <Trans
          i18nKey="verification.agents.rotate.description"
          components={{ agentName: <strong>{rotatingAgent?.name}</strong> }}
        />
      </p>
      <div className="mt-5 flex flex-wrap justify-end gap-2">
        {[
          <Button key="cancel" onClick={onCancel}>
            {t('common.actions.cancel')}
          </Button>,
          <Button
            key="rotate"
            variant="filled"
            color="danger"
            loading={isRotating}
            onClick={onConfirm}
            aria-busy={isRotating}
          >
            {t('verification.agents.actions.rotateToken')}
          </Button>,
        ]}
      </div>
    </Dialog>
  );
};
