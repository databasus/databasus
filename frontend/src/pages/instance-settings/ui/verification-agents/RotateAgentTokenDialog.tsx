import { Button, Modal } from 'antd';
import type { JSX } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import type { VerificationAgent } from '../../../../entities/verification-agents';

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
    <Modal
      title={t('verification.agents.rotate.title')}
      open={rotatingAgent !== undefined}
      onCancel={onCancel}
      footer={[
        <Button key="cancel" onClick={onCancel}>
          {t('common.actions.cancel')}
        </Button>,
        <Button key="rotate" type="primary" danger loading={isRotating} onClick={onConfirm}>
          {t('verification.agents.actions.rotateToken')}
        </Button>,
      ]}
    >
      <p className="text-sm">
        <Trans
          i18nKey="verification.agents.rotate.description"
          components={{ agentName: <strong>{rotatingAgent?.name}</strong> }}
        />
      </p>
    </Modal>
  );
};
