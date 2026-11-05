import { Button, Input, Modal } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

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
    <Modal
      title={t('app.clipboardPaste.title')}
      open={open}
      onCancel={handleCancel}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={handleCancel}>{t('common.actions.cancel')}</Button>
          <Button type="primary" disabled={!value.trim()} onClick={handleSubmit}>
            {t('app.clipboardPaste.submit')}
          </Button>
        </div>
      }
    >
      <p className="mb-2 text-sm text-muted">{t('app.clipboardPaste.description')}</p>
      <Input.TextArea
        aria-label={t('app.clipboardPaste.title')}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder={t('app.clipboardPaste.placeholder')}
        rows={4}
        autoFocus
      />
    </Modal>
  );
}
