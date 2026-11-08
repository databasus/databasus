import { Button, PasswordInput, TextInput } from '@mantine/core';
import { useId } from 'react';
import { type JSX, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { type PendingSignIn, userApi } from '@/entities/users';
import { OAuthSignInButtons } from '@/features/oauth-sign-in';
import { useCloudflareTurnstile } from '@/pages/auth/model/useCloudflareTurnstile';
import { CloudflareTurnstileWidget } from '@/pages/auth/ui/CloudflareTurnstileWidget';
import { useSessionMutation } from '@/shared/api';
import { useAuthOperation } from '@/shared/auth';
import {
  CLOUDFLARE_TURNSTILE_SITE_KEY,
  GITHUB_CLIENT_ID,
  GOOGLE_CLIENT_ID,
  IS_EMAIL_CONFIGURED,
} from '@/shared/config';
import { translateApiError } from '@/shared/i18n';
import { FormValidator } from '@/shared/lib/FormValidator';

interface SignInComponentProps {
  onSwitchToSignUp?: () => void;
  onSwitchToResetPassword?: () => void;
  onCodeRequired?: (pendingSignIn: PendingSignIn) => void;
}

export function SignInComponent({
  onSwitchToSignUp,
  onSwitchToResetPassword,
  onCodeRequired,
}: SignInComponentProps): JSX.Element {
  const signInMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof userApi.signIn>) => userApi.signIn(...parameters),
  });

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isPasswordVisible, setPasswordVisible] = useState(false);

  const [isLoading, setLoading] = useState(false);

  const [isEmailError, setEmailError] = useState(false);
  const [isPasswordError, setPasswordError] = useState(false);

  const [signInError, setSignInError] = useState<unknown>();

  const validateFieldsForSignIn = (): boolean => {
    if (!email) {
      setEmailError(true);
      return false;
    }

    // 'admin' is not an address, but an instance created before the first account
    // administered it still signs in with that login until its owner replaces it.
    if (!FormValidator.isValidEmail(email) && email !== 'admin') {
      setEmailError(true);
      return false;
    }

    if (!password) {
      setPasswordError(true);
      return false;
    }
    setPasswordError(false);

    return true;
  };

  const onSignIn = async () => {
    if (isLoading) return;
    const operation = authOperation.start();
    setSignInError(undefined);

    if (validateFieldsForSignIn()) {
      setLoading(true);

      try {
        const outcome = await signInMutation.mutateAsync([
          {
            email,
            password,
            cloudflareTurnstileToken: token,
          },
        ]);

        if (!authOperation.isCurrent(operation)) return;
        if (outcome.outcome === 'COMPLETED') authOperation.accept({ operation, session: outcome });
        if (outcome.outcome === 'CODE_REQUIRED') {
          onCodeRequired?.({ pendingSignInId: outcome.pendingSignInId, email: outcome.email });
        }
      } catch (e) {
        if (!authOperation.isCurrent(operation)) return;
        setSignInError(e);
        resetCloudflareTurnstile();
      }

      setLoading(false);
    }
  };
  const controlId = useId();
  const { t } = useTranslation();

  const authOperation = useAuthOperation();

  const { token, containerRef, resetCloudflareTurnstile } = useCloudflareTurnstile();

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-5 text-center text-2xl font-bold">{t('users.signIn.title')}</div>

      <div className="mt-4">
        <div className="space-y-2">
          <OAuthSignInButtons />
        </div>
      </div>

      {(GOOGLE_CLIENT_ID || GITHUB_CLIENT_ID) && (
        <div className="relative my-6">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-border"></div>
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="bg-surface px-2 text-muted">{t('users.oauth.divider')}</span>
          </div>
        </div>
      )}

      <div className="mb-6 min-w-0">
        <label htmlFor={controlId + 'field-1'} className="mb-2 block text-sm text-content">
          {t('users.fields.yourEmail')}
        </label>
        <TextInput
          aria-invalid={isEmailError}
          id={controlId + 'field-1'}
          placeholder="your@email.com"
          value={email}
          onChange={(e) => {
            setEmailError(false);
            setEmail(e.currentTarget.value.trim().toLowerCase());
          }}
          type="email"
          error={isEmailError ? <span>{t('users.validation.emailInvalid')}</span> : undefined}
        />
      </div>

      <div className="mb-6 min-w-0">
        <label htmlFor={controlId + 'field-2'} className="mb-2 block text-sm text-content">
          {t('common.fields.password')}
        </label>
        <PasswordInput
          aria-invalid={isPasswordError}
          id={controlId + 'field-2'}
          placeholder="********"
          value={password}
          onChange={(e) => {
            setPasswordError(false);
            setPassword(e.currentTarget.value);
          }}
          visible={isPasswordVisible}
          onVisibilityChange={setPasswordVisible}
          visibilityToggleButtonProps={{
            'aria-label': t('common.actions.togglePasswordVisibility'),
          }}
          error={
            isPasswordError ? (
              <span>{t('users.validation.passwordTooShort', { minLength: 8 })}</span>
            ) : undefined
          }
        />
      </div>

      <div className="mt-3" />

      <CloudflareTurnstileWidget containerRef={containerRef} />

      <Button
        aria-label={t('users.signIn.submit')}
        disabled={isLoading || (!!CLOUDFLARE_TURNSTILE_SITE_KEY && !token)}
        loading={isLoading}
        className="!h-auto min-h-8 w-full max-w-full !whitespace-normal"
        onClick={() => {
          onSignIn();
        }}
        variant="filled"
        aria-busy={isLoading}
      >
        {t('users.signIn.submit')}
      </Button>

      {signInError !== undefined && (
        <div className="mt-3 flex justify-center text-center text-sm text-danger">
          {translateApiError(signInError, t)}
        </div>
      )}

      <div className="mt-4 text-center text-sm text-muted">
        <Trans
          i18nKey="users.signIn.noAccount"
          components={{
            signUpLink: (
              <button
                type="button"
                onClick={onSwitchToSignUp}
                className="cursor-pointer font-medium text-accent hover:text-accent"
              />
            ),
          }}
        />
        <br />
        {IS_EMAIL_CONFIGURED && (
          <button
            type="button"
            onClick={onSwitchToResetPassword}
            className="cursor-pointer font-medium text-accent hover:text-accent"
          >
            {t('users.signIn.forgotPassword')}
          </button>
        )}
      </div>

      <div className="mb-10" />
    </div>
  );
}
