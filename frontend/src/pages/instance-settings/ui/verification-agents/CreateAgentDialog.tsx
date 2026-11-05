import { App, Button, Input, Modal } from 'antd';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

interface Props {
  isOpen: boolean;
  isCreating: boolean;
  onCancel: () => void;
  onCreate: (name: string) => void;
}

export const CreateAgentDialog = ({
  isOpen,
  isCreating,
  onCancel,
  onCreate,
}: Props): JSX.Element => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  const [name, setName] = useState('');

  const submitName = () => {
    const trimmedName = name.trim();

    if (!trimmedName) {
      message.error(t('verification.agents.create.nameRequired'));
      return;
    }

    if (!isCreating) onCreate(trimmedName);
  };

  return (
    <Modal
      title={t('verification.agents.create.title')}
      open={isOpen}
      onCancel={onCancel}
      afterOpenChange={(isNowOpen) => {
        if (!isNowOpen) setName('');
      }}
      footer={[
        <Button key="cancel" onClick={onCancel}>
          {t('common.actions.cancel')}
        </Button>,
        <Button key="create" type="primary" loading={isCreating} onClick={submitName}>
          {t('common.actions.create')}
        </Button>,
      ]}
      destroyOnHidden
    >
      <p className="mb-2 text-sm text-muted">{t('verification.agents.create.tokenNotice')}</p>
      <Input
        aria-label={t('common.fields.name')}
        placeholder={t('verification.agents.create.namePlaceholder')}
        value={name}
        onChange={(event) => setName(event.target.value)}
        onPressEnter={submitName}
        maxLength={200}
        autoFocus
      />
    </Modal>
  );
};
