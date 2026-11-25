import { Button } from '@mantine/core';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { userApi, SecondFactorCodeField, isAppCodeFormat } from '@/entities/users';
import { useCloudflareTurnstile } from '@/pages/auth/model/useCloudflareTurnstile';
import { CloudflareTurnstileWidget } from '@/pages/auth/ui/CloudflareTurnstileWidget';
import { ApiError, useSessionMutation } from '@/shared/api';
import { useAuthOperation } from '@/shared/auth';
import { CLOUDFLARE_TURNSTILE_SITE_KEY } from '@/shared/config';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';

interface Props {
  pendingSignInId: string;
  onPendingSignInLost: () => void;
  onSwitchToSignIn: () => void;
}

// eslint-disable-next-line i18next/no-literal-string -- backend error code
const PENDING_SIGN_IN_NOT_USABLE_CODE = 'pending_sign_in_not_usable';

// The app computes the code itself, so this step has nothing to resend.
export function SignInAuthenticatorCodeComponent({
  pendingSignInId,
  onPendingSignInLost,
  onSwitchToSignIn,
}: Props): JSX.Element {
  const { t, i18n } = useTranslation();
  const notify = useFeedback();
  const authOperation = useAuthOperation();
  const { token, containerRef, resetCloudflareTurnstile } = useCloudflareTurnstile();

  const verifyMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof userApi.verifySignInCode>) =>
      userApi.verifySignInCode(...parameters),
  });

  const [code, setCode] = useState('');
  const [isRecoveryCode, setIsRecoveryCode] = useState(false);
  const [isCodeFormatError, setIsCodeFormatError] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [failure, setFailure] = useState<unknown>();

  const verify = async () => {
    if (isLoading) return;
    const operation = authOperation.start();
    setFailure(undefined);

    if (isRecoveryCode ? !code : !isAppCodeFormat(code)) {
      setIsCodeFormatError(true);

      return;
    }

    setIsLoading(true);

    try {
      const session = await verifyMutation.mutateAsync([
        pendingSignInId,
        { code, cloudflareTurnstileToken: token },
      ]);
      authOperation.accept({ operation, session });
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

  return (
    <form
      className="w-full max-w-[300px]"
      onSubmit={(event) => {
        event.preventDefault();
        void verify();
      }}
    >
      <div className="mb-4 text-center text-xl font-semibold">
        {t('users.signInAuthenticatorCode.title')}
      </div>

      <div className="mb-4 text-center text-sm text-muted">
        {isRecoveryCode
          ? t('users.signInAuthenticatorCode.recoveryDescription')
          : t('users.signInAuthenticatorCode.description')}
      </div>

      <SecondFactorCodeField
        code={code}
        onCodeChange={(enteredCode) => {
          setIsCodeFormatError(false);
          setFailure(undefined);
          setCode(enteredCode);
        }}
        isRecoveryCode={isRecoveryCode}
        onRecoveryCodeToggle={(isRecoveryCodeChosen) => {
          setIsRecoveryCode(isRecoveryCodeChosen);
          setCode('');
          setIsCodeFormatError(false);
          setFailure(undefined);
        }}
        error={isCodeFormatError ? t('authenticatorApp.code.invalidAppCode') : undefined}
        isDisabled={isLoading}
      />

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
        {t('users.signInAuthenticatorCode.submit')}
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
          {t('users.signInAuthenticatorCode.backToSignIn')}
        </button>
      </div>
    </form>
  );
}
