import { Button, TextInput } from '@mantine/core';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useFeedback } from '@/shared/lib/feedback';
import { Dialog } from '@/shared/ui';
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
  const notify = useFeedback();
  const [name, setName] = useState('');

  const submitName = () => {
    const trimmedName = name.trim();

    if (!trimmedName) {
      notify({ kind: 'error', title: t('verification.agents.create.nameRequired') });
      return;
    }

    if (!isCreating) onCreate(trimmedName);
  };

  return (
    <Dialog
      title={t('verification.agents.create.title')}
      opened={isOpen}
      onClose={onCancel}
      onExitTransitionEnd={() => setName('')}
    >
      <p className="mb-2 text-sm text-muted">{t('verification.agents.create.tokenNotice')}</p>
      <TextInput
        aria-label={t('common.fields.name')}
        placeholder={t('verification.agents.create.namePlaceholder')}
        value={name}
        onChange={(event) => setName(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            submitName();
          }
        }}
        maxLength={200}
        data-autofocus
      />
      <div className="mt-5 flex flex-wrap justify-end gap-2">
        {[
          <Button key="cancel" onClick={onCancel}>
            {t('common.actions.cancel')}
          </Button>,
          <Button
            key="create"
            variant="filled"
            loading={isCreating}
            onClick={submitName}
            aria-busy={isCreating}
          >
            {t('common.actions.create')}
          </Button>,
        ]}
      </div>
    </Dialog>
  );
};
