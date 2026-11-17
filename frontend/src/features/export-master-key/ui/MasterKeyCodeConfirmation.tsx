import { Alert, Button, TextInput } from '@mantine/core';
import { CircleAlert, Mail } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { masterKeyApi } from '@/entities/instance-backups';
import { MasterKeyHostInstructions } from '@/features/export-master-key/ui/MasterKeyHostInstructions';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';

interface Props {
  masterKeyPath: string;
  isConfirming: boolean;
  onConfirm: (code: string) => void;
}

// When code delivery fails, copying the key from the server is the way left to root admin.
export function MasterKeyCodeConfirmation({ masterKeyPath, isConfirming, onConfirm }: Props) {
  const { t } = useTranslation();
  const isCurrent = useSessionGuard();
  const [code, setCode] = useState('');
  const [codeExpiresAt, setCodeExpiresAt] = useState<string>();
  const [codeRequestError, setCodeRequestError] = useState<unknown>();

  const confirmationMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof masterKeyApi.requestConfirmation>) =>
      masterKeyApi.requestConfirmation(...parameters),
  });

  const requestCode = async () => {
    if (confirmationMutation.isPending) return;

    setCodeRequestError(undefined);

    try {
      const confirmation = await confirmationMutation.mutateAsync([]);
      if (isCurrent()) setCodeExpiresAt(confirmation.expiresAt);
    } catch (failure: unknown) {
      if (isCurrent()) setCodeRequestError(failure);
    } finally {
      if (isCurrent()) confirmationMutation.reset();
    }
  };

  return (
    <div className="text-sm">
      <div className="flex gap-3">
        <Mail aria-hidden size={18} className="mt-0.5 shrink-0 text-accent" />
        <p>{t('masterKey.codeRequired')}</p>
      </div>

      <form
        className="mt-3 flex flex-wrap items-end gap-2"
        onSubmit={(event) => {
          event.preventDefault();
          if (code) onConfirm(code);
        }}
      >
        <Button
          variant={codeExpiresAt ? 'default' : 'filled'}
          loading={confirmationMutation.isPending}
          onClick={() => void requestCode()}
        >
          {codeExpiresAt ? t('masterKey.proof.resendCode') : t('masterKey.proof.sendCode')}
        </Button>
        {codeExpiresAt && (
          <>
            <TextInput
              className="min-w-40 flex-1"
              label={t('masterKey.proof.code')}
              autoComplete="one-time-code"
              value={code}
              onChange={(event) => setCode(event.currentTarget.value)}
            />
            <Button type="submit" disabled={!code} loading={isConfirming}>
              {t('masterKey.proof.confirm')}
            </Button>
          </>
        )}
      </form>

      {codeRequestError !== undefined && (
        <div className="mt-3">
          <Alert
            color="danger"
            icon={<CircleAlert aria-hidden size={16} />}
            title={translateApiError(codeRequestError, t)}
          >
            {t('masterKey.codeFailed')}
          </Alert>
          <div className="mt-3">
            <MasterKeyHostInstructions masterKeyPath={masterKeyPath} />
          </div>
        </div>
      )}
    </div>
  );
}
