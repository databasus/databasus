import { Alert, Button, Loader, TextInput } from '@mantine/core';
import { CircleAlert, KeyRound, Mail } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { masterKeyApi } from '@/entities/instance-backups';
import { SecondFactorCodeField, useIsAuthenticatorAppConnected } from '@/entities/users';
import { MasterKeyHostInstructions } from '@/features/export-master-key/ui/MasterKeyHostInstructions';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';

interface Props {
  isConfirming: boolean;
  onConfirm: (code: string) => void;
}

// eslint-disable-next-line i18next/no-literal-string -- contract enum value
const AUTHENTICATOR_APP_METHOD = 'AUTHENTICATOR_APP';

// Root admin with an app confirms with its code and nothing is sent. Otherwise the code is
// emailed, and when its delivery fails, copying the key from the server is the way left.
export function MasterKeyCodeConfirmation({ isConfirming, onConfirm }: Props) {
  const { t } = useTranslation();
  const isCurrent = useSessionGuard();
  const isAppConnected = useIsAuthenticatorAppConnected();
  const [code, setCode] = useState('');
  const [isRecoveryCode, setIsRecoveryCode] = useState(false);
  const [isAppCodeAnswered, setIsAppCodeAnswered] = useState(false);
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
      if (!isCurrent()) return;
      if (confirmation.method === AUTHENTICATOR_APP_METHOD) setIsAppCodeAnswered(true);
      else setCodeExpiresAt(confirmation.expiresAt);
    } catch (failure: unknown) {
      if (isCurrent()) setCodeRequestError(failure);
    } finally {
      if (isCurrent()) confirmationMutation.reset();
    }
  };

  const submit = () => {
    if (code) onConfirm(code);
  };

  if (isAppConnected === undefined) return <Loader size="xs" role="status" />;

  if (isAppConnected || isAppCodeAnswered) {
    return (
      <div className="text-sm">
        <div className="flex gap-3">
          <KeyRound aria-hidden size={18} className="mt-0.5 shrink-0 text-accent" />
          <p>{t('masterKey.appCodeRequired')}</p>
        </div>

        <form
          className="mt-3 flex flex-wrap items-start gap-2"
          onSubmit={(event) => {
            event.preventDefault();
            submit();
          }}
        >
          <div className="min-w-40 flex-1">
            <SecondFactorCodeField
              code={code}
              onCodeChange={setCode}
              isRecoveryCode={isRecoveryCode}
              onRecoveryCodeToggle={(isRecoveryCodeChosen) => {
                setIsRecoveryCode(isRecoveryCodeChosen);
                setCode('');
              }}
              isDisabled={isConfirming}
            />
          </div>
          <Button className="mt-6" type="submit" disabled={!code} loading={isConfirming}>
            {t('masterKey.proof.confirm')}
          </Button>
        </form>
      </div>
    );
  }

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
          submit();
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
            <MasterKeyHostInstructions />
          </div>
        </div>
      )}
    </div>
  );
}
