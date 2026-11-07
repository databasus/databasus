import { Button, Textarea } from '@mantine/core';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { Dialog } from '@/shared/ui/Dialog';

interface Props {
  open: boolean;
  onSubmit(text: string): void;
  onCancel(): void;
}

export function ClipboardPasteModalComponent({ open, onSubmit, onCancel }: Props) {
  const { t } = useTranslation();
  const [value, setValue] = useState('');

  const handleSubmit = () => {
    const trimmed = value.trim();
    if (!trimmed) return;

    onSubmit(trimmed);
    setValue('');
  };

  const handleCancel = () => {
    setValue('');
    onCancel();
  };

  return (
    <Dialog
      title={t('app.clipboardPaste.title')}
      opened={open}
      onClose={handleCancel}
      closeButtonProps={{ 'aria-label': t('common.actions.close') }}
    >
      <p className="mb-2 text-sm text-muted">{t('app.clipboardPaste.description')}</p>
      <Textarea
        aria-label={t('app.clipboardPaste.title')}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder={t('app.clipboardPaste.placeholder')}
        rows={4}
        data-autofocus
      />
      <div className="mt-5 flex flex-wrap justify-end gap-2">
        <Button onClick={handleCancel}>{t('common.actions.cancel')}</Button>
        <Button variant="filled" disabled={!value.trim()} onClick={handleSubmit}>
          {t('app.clipboardPaste.submit')}
        </Button>
      </div>
    </Dialog>
  );
}
