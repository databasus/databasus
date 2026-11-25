import { Alert, Button } from '@mantine/core';
import { Check, CircleAlert, Copy, Download } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { saveTextFile } from '@/shared/lib';
import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';

interface Props {
  recoveryCodes: string[];
}

// eslint-disable-next-line i18next/no-literal-string -- a file name, not interface copy
const RECOVERY_CODES_FILE_NAME = 'databasus-recovery-codes.txt';

// The codes exist only in the answer that issued them, so the caller drops them when this unmounts.
export function RecoveryCodesList({ recoveryCodes }: Props) {
  const { t } = useTranslation();
  const [isCopied, setIsCopied] = useState(false);
  const [isDownloaded, setIsDownloaded] = useState(false);
  const [isCopyFailed, setIsCopyFailed] = useState(false);
  const codesText = recoveryCodes.join('\n');

  const copy = async () => {
    setIsCopyFailed(false);

    try {
      await BrowserClipboard.copyToClipboard(codesText);
      setIsCopied(true);
    } catch {
      setIsCopyFailed(true);
    }
  };

  const download = () => {
    // eslint-disable-next-line i18next/no-literal-string -- the file's line break, not interface copy
    saveTextFile(`${codesText}\n`, RECOVERY_CODES_FILE_NAME);
    setIsDownloaded(true);
  };

  return (
    <div className="text-sm">
      <p className="font-medium">{t('authenticatorApp.recoveryCodes.title')}</p>
      <p className="mt-1">{t('authenticatorApp.recoveryCodes.description')}</p>

      <ul
        aria-label={t('authenticatorApp.recoveryCodes.title')}
        className="mt-3 grid grid-cols-2 gap-x-4 gap-y-1 rounded-md bg-code px-3 py-2 font-mono"
      >
        {recoveryCodes.map((recoveryCode) => (
          <li key={recoveryCode}>{recoveryCode}</li>
        ))}
      </ul>

      <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
        <Button
          variant="default"
          leftSection={isCopied ? <Check aria-hidden size={16} /> : <Copy aria-hidden size={16} />}
          onClick={() => void copy()}
        >
          {isCopied
            ? t('authenticatorApp.recoveryCodes.copied')
            : t('authenticatorApp.recoveryCodes.copy')}
        </Button>
        <Button
          variant="default"
          leftSection={
            isDownloaded ? <Check aria-hidden size={16} /> : <Download aria-hidden size={16} />
          }
          onClick={download}
        >
          {t('authenticatorApp.recoveryCodes.download')}
        </Button>
      </div>

      {isCopyFailed && (
        <Alert
          className="mt-3"
          color="danger"
          icon={<CircleAlert aria-hidden size={16} />}
          title={t('authenticatorApp.recoveryCodes.copyFailed')}
        />
      )}
    </div>
  );
}
