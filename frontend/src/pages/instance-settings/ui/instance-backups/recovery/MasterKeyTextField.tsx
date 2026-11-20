import { PasswordInput } from '@mantine/core';
import type { ClipboardEvent } from 'react';
import { useTranslation } from 'react-i18next';

interface Props {
  id: string;
  value: string;
  error?: string;
  isDisabled: boolean;
  onChange: (value: string) => void;
}

// The server compares the key byte for byte, but a single-line input drops line breaks from its
// value. A paste is therefore stored from the clipboard text itself, so the uploaded key keeps the
// exact whitespace and line endings it was saved with, while the field shows one masked line.
export function MasterKeyTextField({ id, value, error, isDisabled, onChange }: Props) {
  const { t } = useTranslation();

  const insertPastedText = (event: ClipboardEvent<HTMLInputElement>) => {
    // eslint-disable-next-line i18next/no-literal-string -- clipboard MIME type
    const pastedText = event.clipboardData.getData('text/plain');
    if (!pastedText) return;

    event.preventDefault();
    const field = event.currentTarget;
    const start = field.selectionStart ?? value.length;
    const end = field.selectionEnd ?? value.length;
    onChange(value.slice(0, start) + pastedText + value.slice(end));
  };

  return (
    <PasswordInput
      id={id}
      label={t('instanceRecovery.form.masterKey')}
      description={t('instanceRecovery.form.masterKeyDescription')}
      inputWrapperOrder={['label', 'input', 'description', 'error']}
      error={error}
      value={value}
      disabled={isDisabled}
      autoComplete="off"
      spellCheck={false}
      onPaste={insertPastedText}
      onChange={(event) => onChange(event.currentTarget.value)}
    />
  );
}
