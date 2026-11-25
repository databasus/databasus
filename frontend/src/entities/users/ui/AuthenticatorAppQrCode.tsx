import { Alert, Button } from '@mantine/core';
import { Check, CircleAlert, Copy } from 'lucide-react';
import { QRCodeSVG } from 'qrcode.react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';

interface Props {
  otpauthUri: string;
  secret: string;
}

const QR_CODE_SIZE_PX = 200;
const QR_CODE_QUIET_ZONE_MODULES = 4;

export function AuthenticatorAppQrCode({ otpauthUri, secret }: Props) {
  const { t } = useTranslation();
  const [isCopied, setIsCopied] = useState(false);
  const [isCopyFailed, setIsCopyFailed] = useState(false);

  const copySecret = async () => {
    setIsCopyFailed(false);

    try {
      await BrowserClipboard.copyToClipboard(secret);
      setIsCopied(true);
    } catch {
      setIsCopyFailed(true);
    }
  };

  return (
    <div className="text-sm">
      <p>{t('authenticatorApp.setup.scan')}</p>

      {/* The library's own dark-on-light colors and quiet zone keep the code scannable in both themes. */}
      <QRCodeSVG
        className="mx-auto mt-3 block rounded-md"
        value={otpauthUri}
        size={QR_CODE_SIZE_PX}
        marginSize={QR_CODE_QUIET_ZONE_MODULES}
        title={t('authenticatorApp.setup.qrCodeAlt')}
        role="img"
        aria-label={t('authenticatorApp.setup.qrCodeAlt')}
      />

      <p className="mt-3 text-muted">{t('authenticatorApp.setup.secret')}</p>
      <div className="mt-1 flex flex-wrap items-center gap-2">
        <code className="min-w-0 rounded-md bg-code px-2 py-1 font-mono break-all">{secret}</code>
        <Button
          size="xs"
          variant="default"
          leftSection={isCopied ? <Check aria-hidden size={14} /> : <Copy aria-hidden size={14} />}
          onClick={() => void copySecret()}
        >
          {isCopied
            ? t('authenticatorApp.setup.secretCopied')
            : t('authenticatorApp.setup.copySecret')}
        </Button>
      </div>

      {isCopyFailed && (
        <Alert
          className="mt-3"
          color="danger"
          icon={<CircleAlert aria-hidden size={16} />}
          title={t('authenticatorApp.setup.copyFailed')}
        />
      )}
    </div>
  );
}
