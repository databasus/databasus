import { Button, Textarea } from '@mantine/core';
import { type ClipboardEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';

interface Props {
  id: string;
  value: string;
  error?: string;
  isDisabled: boolean;
  onChange: (value: string) => void;
}

// The server compares the key byte for byte. A single-line password input drops line breaks from a
// paste, and a textarea reports CRLF as LF, so a paste is inserted from the clipboard text itself.
export function MasterKeyTextField({ id, value, error, isDisabled, onChange }: Props) {
  const { t } = useTranslation();
  const [isShown, setIsShown] = useState(false);

  const insertPastedText = (event: ClipboardEvent<HTMLTextAreaElement>) => {
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
    <div>
      <Textarea
        id={id}
        size="xs"
        label={t('instanceRecovery.form.masterKey')}
        description={t('instanceRecovery.form.masterKeyDescription')}
        error={error}
        value={value}
        disabled={isDisabled}
        autosize
        minRows={2}
        maxRows={6}
        autoComplete="off"
        spellCheck={false}
        styles={{ input: isShown ? undefined : { WebkitTextSecurity: 'disc' } }}
        onPaste={insertPastedText}
        onChange={(event) => onChange(event.currentTarget.value)}
      />
      <Button
        className="mt-1"
        size="compact-xs"
        variant="subtle"
        aria-pressed={isShown}
        onClick={() => setIsShown((current) => !current)}
      >
        {isShown ? t('instanceRecovery.form.hideKey') : t('instanceRecovery.form.showKey')}
      </Button>
    </div>
  );
}
