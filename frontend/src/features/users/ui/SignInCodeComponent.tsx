import { App, Button, Input } from 'antd';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { CLOUDFLARE_TURNSTILE_SITE_KEY } from '../../../constants';
import { type PendingSignIn, userApi } from '../../../entities/users';
import { ApiError } from '../../../shared/api';
import { useCloudflareTurnstile } from '../../../shared/hooks/useCloudflareTurnstile';
import { translateApiError } from '../../../shared/i18n';
import { CloudflareTurnstileWidget } from '../../../shared/ui/CloudflareTurnstileWidget';

interface SignInCodeComponentProps {
  pendingSignIn: PendingSignIn;
  onPendingSignInReplaced: (pendingSignIn: PendingSignIn) => void;
  onPendingSignInLost: () => void;
  onSwitchToSignIn: () => void;
}

// The one refusal that ends the code step rather than leaving the user on it.
// eslint-disable-next-line i18next/no-literal-string -- backend error code
const PENDING_SIGN_IN_NOT_USABLE_CODE = 'pending_sign_in_not_usable';

export function SignInCodeComponent({
  pendingSignIn,
  onPendingSignInReplaced,
  onPendingSignInLost,
  onSwitchToSignIn,
}: SignInCodeComponentProps): JSX.Element {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [code, setCode] = useState('');
  const [isCodeError, setCodeError] = useState(false);
  const [isLoading, setLoading] = useState(false);
  const [isResending, setResending] = useState(false);
  const [failure, setFailure] = useState<unknown>();

  const { token, containerRef, resetCloudflareTurnstile } = useCloudflareTurnstile();

  const reportFailure = (error: unknown) => {
    if (error instanceof ApiError && error.code === PENDING_SIGN_IN_NOT_USABLE_CODE) {
      message.error(translateApiError(error, t));
      onPendingSignInLost();

      return;
    }

    setFailure(error);
  };

  const onVerify = async () => {
    setFailure(undefined);

    if (!/^\d{6}$/.test(code)) {
      setCodeError(true);

      return;
    }

    setLoading(true);

    try {
      await userApi.verifySignInCode(pendingSignIn.pendingSignInId, {
        code,
        cloudflareTurnstileToken: token,
      });
    } catch (e) {
      reportFailure(e);
      resetCloudflareTurnstile();
    }

    setLoading(false);
  };

  const onResend = async () => {
    setFailure(undefined);
    setResending(true);

    try {
      const resentSignIn = await userApi.resendSignInCode(pendingSignIn.pendingSignInId, {
        cloudflareTurnstileToken: token,
      });

      setCode('');
      onPendingSignInReplaced(resentSignIn);
      message.success(t('users.signInCode.codeResent'));
    } catch (e) {
      reportFailure(e);
    }

    // The challenge answer is single-use, and the resend spent it whether or not
    // it succeeded, so the next verification needs a fresh one.
    resetCloudflareTurnstile();

    setResending(false);
  };

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-5 text-center text-2xl font-bold">{t('users.signInCode.title')}</div>

      <div className="mb-4 text-center text-sm text-gray-600 dark:text-gray-400">
        {t('users.signInCode.description', { email: pendingSignIn.email })}
      </div>

      <div className="my-1 text-xs font-semibold">{t('users.signInCode.code')}</div>
      <Input
        placeholder="123456"
        value={code}
        onChange={(e) => {
          setCodeError(false);
          setFailure(undefined);
          setCode(e.currentTarget.value.replace(/\D/g, '').slice(0, 6));
        }}
        status={isCodeError ? 'error' : undefined}
        maxLength={6}
      />

      <div className="mt-3" />

      <CloudflareTurnstileWidget containerRef={containerRef} />

      <Button
        disabled={isLoading || (!!CLOUDFLARE_TURNSTILE_SITE_KEY && !token)}
        loading={isLoading}
        className="w-full"
        onClick={() => {
          onVerify();
        }}
        type="primary"
      >
        {t('users.signInCode.submit')}
      </Button>

      {(isCodeError || failure !== undefined) && (
        <div className="mt-3 flex justify-center text-center text-sm text-red-600">
          {isCodeError ? t('users.signInCode.invalidCode') : translateApiError(failure, t)}
        </div>
      )}

      <div className="mt-4 text-center text-sm text-gray-600 dark:text-gray-400">
        <button
          type="button"
          onClick={() => {
            onResend();
          }}
          disabled={isResending || (!!CLOUDFLARE_TURNSTILE_SITE_KEY && !token)}
          className="cursor-pointer font-medium text-blue-600 hover:text-blue-700 dark:!text-blue-500"
        >
          {t('users.signInCode.resend')}
        </button>
        <br />
        <button
          type="button"
          onClick={onSwitchToSignIn}
          className="cursor-pointer font-medium text-blue-600 hover:text-blue-700 dark:!text-blue-500"
        >
          {t('users.signInCode.backToSignIn')}
        </button>
      </div>

      <div className="mb-10" />
    </div>
  );
}
