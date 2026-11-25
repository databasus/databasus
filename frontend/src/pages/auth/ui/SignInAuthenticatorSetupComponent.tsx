import { Button } from '@mantine/core';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type AuthenticatorAppSetupSession,
  type SignInAuthenticatorSetup,
  userApi,
  AuthenticatorAppQrCode,
  RecoveryCodesList,
  SecondFactorCodeField,
  isAppCodeFormat,
} from '@/entities/users';
import { useCloudflareTurnstile } from '@/pages/auth/model/useCloudflareTurnstile';
import { CloudflareTurnstileWidget } from '@/pages/auth/ui/CloudflareTurnstileWidget';
import { ApiError, useSessionMutation } from '@/shared/api';
import { type AuthIntent, useAuthOperation } from '@/shared/auth';
import { CLOUDFLARE_TURNSTILE_SITE_KEY } from '@/shared/config';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';

interface Props {
  setup: SignInAuthenticatorSetup;
  onPendingSignInLost: () => void;
  onSwitchToSignIn: () => void;
}

// eslint-disable-next-line i18next/no-literal-string -- backend error code
const PENDING_SIGN_IN_NOT_USABLE_CODE = 'pending_sign_in_not_usable';

interface ConnectedSetup {
  operation: AuthIntent;
  setupSession: AuthenticatorAppSetupSession;
}

// The instance issues the session together with the recovery codes, but the application opens only
// after the user confirms having saved them, so nobody walks past codes shown only once.
export function SignInAuthenticatorSetupComponent({
  setup,
  onPendingSignInLost,
  onSwitchToSignIn,
}: Props): JSX.Element {
  const { t, i18n } = useTranslation();
  const notify = useFeedback();
  const authOperation = useAuthOperation();
  const { token, containerRef, resetCloudflareTurnstile } = useCloudflareTurnstile();

  const confirmMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof userApi.confirmSignInAuthenticatorAppSetup>) =>
      userApi.confirmSignInAuthenticatorAppSetup(...parameters),
  });

  const [code, setCode] = useState('');
  const [isCodeFormatError, setIsCodeFormatError] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [failure, setFailure] = useState<unknown>();
  const [connectedSetup, setConnectedSetup] = useState<ConnectedSetup>();

  const confirm = async () => {
    if (isLoading) return;
    const operation = authOperation.start();
    setFailure(undefined);

    if (!isAppCodeFormat(code)) {
      setIsCodeFormatError(true);

      return;
    }

    setIsLoading(true);

    try {
      const setupSession = await confirmMutation.mutateAsync([
        setup.pendingSignInId,
        { code, cloudflareTurnstileToken: token },
      ]);
      if (!authOperation.isCurrent(operation)) return;
      setConnectedSetup({ operation, setupSession });
    } catch (e) {
      if (!authOperation.isCurrent(operation)) return;

      if (e instanceof ApiError && e.code === PENDING_SIGN_IN_NOT_USABLE_CODE) {
        notify({ kind: 'error', title: translateApiError(e, i18n.t) });
        onPendingSignInLost();

        return;
      }

      setCode('');
      setFailure(e);
      resetCloudflareTurnstile();
    }

    setIsLoading(false);
  };

  if (connectedSetup) {
    const { operation, setupSession } = connectedSetup;

    return (
      <div className="w-full max-w-[360px]">
        <RecoveryCodesList recoveryCodes={setupSession.recoveryCodes} />

        <Button
          className="mt-5 !h-auto min-h-8 w-full max-w-full !whitespace-normal"
          variant="filled"
          onClick={() =>
            authOperation.accept({
              operation,
              session: { userId: setupSession.userId, token: setupSession.token },
            })
          }
        >
          {t('authenticatorApp.recoveryCodes.saved')}
        </Button>
      </div>
    );
  }

  return (
    <form
      className="w-full max-w-[360px]"
      onSubmit={(event) => {
        event.preventDefault();
        void confirm();
      }}
    >
      <div className="mb-4 text-center text-xl font-semibold">
        {t('users.signInAuthenticatorSetup.title')}
      </div>

      <div className="mb-4 text-center text-sm text-muted">
        {t('users.signInAuthenticatorSetup.description')}
      </div>

      <AuthenticatorAppQrCode otpauthUri={setup.otpauthUri} secret={setup.secret} />

      <div className="mt-4">
        <SecondFactorCodeField
          code={code}
          onCodeChange={(enteredCode) => {
            setIsCodeFormatError(false);
            setFailure(undefined);
            setCode(enteredCode);
          }}
          isRecoveryCode={false}
          error={isCodeFormatError ? t('authenticatorApp.code.invalidAppCode') : undefined}
          isDisabled={isLoading}
        />
      </div>

      <div className="mt-3">
        <CloudflareTurnstileWidget containerRef={containerRef} />
      </div>

      <Button
        type="submit"
        disabled={isLoading || (!!CLOUDFLARE_TURNSTILE_SITE_KEY && !token)}
        loading={isLoading}
        className="!h-auto min-h-8 w-full max-w-full !whitespace-normal"
        variant="filled"
        aria-busy={isLoading}
      >
        {t('users.signInAuthenticatorSetup.submit')}
      </Button>

      {failure !== undefined && (
        <div role="alert" className="mt-3 flex justify-center text-center text-sm text-danger">
          {translateApiError(failure, t)}
        </div>
      )}

      <div className="mt-4 text-center text-sm text-muted">
        <button
          type="button"
          onClick={onSwitchToSignIn}
          className="cursor-pointer font-medium text-accent hover:text-accent"
        >
          {t('users.signInAuthenticatorSetup.backToSignIn')}
        </button>
      </div>
    </form>
  );
}
