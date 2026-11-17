import { Alert, Button } from '@mantine/core';
import { Check, CircleAlert, Copy, Download, KeyRound } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { MasterKeyFile } from '@/entities/instance-backups';
import { getMasterKeyPreview } from '@/features/export-master-key/lib/getMasterKeyPreview';
import { saveTextFile } from '@/shared/lib';
import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';

interface Props {
  keyFile: MasterKeyFile;
  onKeySaved?: () => void;
}

export function MasterKeySaveOptions({ keyFile, onKeySaved }: Props) {
  const { t } = useTranslation();
  const copyButtonRef = useRef<HTMLButtonElement>(null);
  const [isCopied, setIsCopied] = useState(false);
  const [isDownloaded, setIsDownloaded] = useState(false);
  const [isCopyFailed, setIsCopyFailed] = useState(false);

  // The key arrives after the dialog or the panel action took focus, so the first action takes it
  // over instead of leaving it on the dialog itself or on a button that is gone.
  useEffect(() => copyButtonRef.current?.focus(), []);

  const copy = async () => {
    setIsCopyFailed(false);

    try {
      await BrowserClipboard.copyToClipboard(keyFile.content);
      setIsCopied(true);
      onKeySaved?.();
    } catch {
      setIsCopyFailed(true);
    }
  };

  const download = () => {
    saveTextFile(keyFile.content, keyFile.fileName);
    setIsDownloaded(true);
    onKeySaved?.();
  };

  return (
    <div>
      <div className="flex items-center gap-2 rounded-md bg-code px-3 py-2">
        <KeyRound aria-hidden size={16} className="shrink-0 text-muted" />
        <code className="min-w-0 truncate font-mono text-sm">
          {getMasterKeyPreview(keyFile.content)}
        </code>
      </div>

      <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
        <Button
          ref={copyButtonRef}
          variant="default"
          leftSection={isCopied ? <Check aria-hidden size={16} /> : <Copy aria-hidden size={16} />}
          onClick={() => void copy()}
        >
          {isCopied ? t('masterKey.copied') : t('masterKey.copy')}
        </Button>
        <Button
          variant="default"
          leftSection={
            isDownloaded ? <Check aria-hidden size={16} /> : <Download aria-hidden size={16} />
          }
          onClick={download}
        >
          {isDownloaded ? t('masterKey.downloaded') : t('masterKey.download')}
        </Button>
      </div>

      {isCopyFailed && (
        <Alert
          className="mt-3"
          color="danger"
          icon={<CircleAlert aria-hidden size={16} />}
          title={t('masterKey.copyFailed')}
        />
      )}
    </div>
  );
}
