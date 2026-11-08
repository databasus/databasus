import { Button, TextInput } from '@mantine/core';
import { useId } from 'react';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type PendingSignIn, userApi } from '@/entities/users';
import { useCloudflareTurnstile } from '@/pages/auth/model/useCloudflareTurnstile';
import { CloudflareTurnstileWidget } from '@/pages/auth/ui/CloudflareTurnstileWidget';
import { useSessionMutation } from '@/shared/api';
import { ApiError } from '@/shared/api';
import { useAuthOperation } from '@/shared/auth';
import { CLOUDFLARE_TURNSTILE_SITE_KEY } from '@/shared/config';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
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
  const verifySignInCodeMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof userApi.verifySignInCode>) =>
      userApi.verifySignInCode(...parameters),
  });
  const resendSignInCodeMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof userApi.resendSignInCode>) =>
      userApi.resendSignInCode(...parameters),
  });

  const [code, setCode] = useState('');
  const [isCodeError, setCodeError] = useState(false);
  const [isLoading, setLoading] = useState(false);
  const [isResending, setResending] = useState(false);
  const [failure, setFailure] = useState<unknown>();

  const reportFailure = (error: unknown) => {
    if (error instanceof ApiError && error.code === PENDING_SIGN_IN_NOT_USABLE_CODE) {
      notify({ kind: 'error', title: translateApiError(error, i18n.t) });
      onPendingSignInLost();

      return;
    }

    setFailure(error);
  };

  const onVerify = async () => {
    if (isLoading || isResending) return;
    const operation = authOperation.start();
    setFailure(undefined);

    if (!/^\d{6}$/.test(code)) {
      setCodeError(true);

      return;
    }

    setLoading(true);

    try {
      const session = await verifySignInCodeMutation.mutateAsync([
        pendingSignIn.pendingSignInId,
        {
          code,
          cloudflareTurnstileToken: token,
        },
      ]);
      authOperation.accept({ operation, session });
    } catch (e) {
      if (!authOperation.isCurrent(operation)) return;
      reportFailure(e);
      resetCloudflareTurnstile();
    }

    setLoading(false);
  };

  const onResend = async () => {
    if (isLoading || isResending) return;
    const operation = authOperation.start();
    setFailure(undefined);
    setResending(true);

    try {
      const resentSignIn = await resendSignInCodeMutation.mutateAsync([
        pendingSignIn.pendingSignInId,
        {
          cloudflareTurnstileToken: token,
        },
      ]);

      if (!authOperation.isCurrent(operation)) return;
      setCode('');
      onPendingSignInReplaced(resentSignIn);
      notify({ kind: 'success', title: i18n.t('users.signInCode.codeResent') });
    } catch (e) {
      if (!authOperation.isCurrent(operation)) return;
      reportFailure(e);
    }

    // The challenge answer is single-use, and the resend spent it whether or not
    // it succeeded, so the next verification needs a fresh one.
    resetCloudflareTurnstile();

    setResending(false);
  };
  const controlId = useId();
  const { t, i18n } = useTranslation();
  const notify = useFeedback();
  const authOperation = useAuthOperation();

  const { token, containerRef, resetCloudflareTurnstile } = useCloudflareTurnstile();

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-4 text-center text-xl font-semibold">{t('users.signInCode.title')}</div>

      <div className="mb-4 text-center text-sm text-muted">
        {t('users.signInCode.description', { email: pendingSignIn.email })}
      </div>

      <label className="mb-1 block text-sm text-content" htmlFor={controlId + 'field-1'}>
        {t('users.signInCode.code')}
      </label>
      <TextInput
        className="mb-3"
        id={controlId + 'field-1'}
        placeholder="123456"
        value={code}
        onChange={(e) => {
          setCodeError(false);
          setFailure(undefined);
          setCode(e.currentTarget.value.replace(/\D/g, '').slice(0, 6));
        }}
        error={(isCodeError ? 'error' : undefined) === 'error'}
        maxLength={6}
      />

      <CloudflareTurnstileWidget containerRef={containerRef} />

      <Button
        disabled={isLoading || (!!CLOUDFLARE_TURNSTILE_SITE_KEY && !token)}
        loading={isLoading}
        className="!h-auto min-h-8 w-full max-w-full !whitespace-normal"
        onClick={() => {
          onVerify();
        }}
        variant="filled"
        aria-busy={isLoading}
      >
        {t('users.signInCode.submit')}
      </Button>

      {(isCodeError || failure !== undefined) && (
        <div className="mt-3 flex justify-center text-center text-sm text-danger">
          {isCodeError ? t('users.signInCode.invalidCode') : translateApiError(failure, t)}
        </div>
      )}

      <div className="mt-4 text-center text-sm text-muted">
        <button
          type="button"
          onClick={() => {
            onResend();
          }}
          disabled={isResending || (!!CLOUDFLARE_TURNSTILE_SITE_KEY && !token)}
          className="cursor-pointer font-medium text-accent hover:text-accent"
        >
          {t('users.signInCode.resend')}
        </button>
        <br />
        <button
          type="button"
          onClick={onSwitchToSignIn}
          className="cursor-pointer font-medium text-accent hover:text-accent"
        >
          {t('users.signInCode.backToSignIn')}
        </button>
      </div>
    </div>
  );
}
