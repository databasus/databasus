import { Button, PasswordInput, TextInput } from '@mantine/core';
import { useId } from 'react';
import { type JSX, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { userApi } from '@/entities/users';
import { OAuthSignInButtons } from '@/features/oauth-sign-in';
import { useCloudflareTurnstile } from '@/pages/auth/model/useCloudflareTurnstile';
import { CloudflareTurnstileWidget } from '@/pages/auth/ui/CloudflareTurnstileWidget';
import { useSessionMutation } from '@/shared/api';
import { useAuthOperation } from '@/shared/auth';
import { CLOUDFLARE_TURNSTILE_SITE_KEY, GITHUB_CLIENT_ID, GOOGLE_CLIENT_ID } from '@/shared/config';
import { translateApiError } from '@/shared/i18n';
import { FormValidator } from '@/shared/lib/FormValidator';

interface SignUpComponentProps {
  onSwitchToSignIn?: () => void;
  isClaimingInstance?: boolean;
}

export function SignUpComponent({
  onSwitchToSignIn,
  isClaimingInstance,
}: SignUpComponentProps): JSX.Element {
  const signUpMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof userApi.signUp>) => userApi.signUp(...parameters),
  });

  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isPasswordVisible, setPasswordVisible] = useState(false);
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isConfirmPasswordVisible, setConfirmPasswordVisible] = useState(false);

  const [isLoading, setLoading] = useState(false);

  const [isNameError, setNameError] = useState(false);
  const [isEmailError, setEmailError] = useState(false);
  const [isPasswordError, setPasswordError] = useState(false);
  const [isConfirmPasswordError, setConfirmPasswordError] = useState(false);

  const [signUpError, setSignUpError] = useState<unknown>();

  const validateFieldsForSignUp = (): boolean => {
    if (!name || name.trim() === '') {
      setNameError(true);

      return false;
    }
    setNameError(false);

    if (!email) {
      setEmailError(true);
      return false;
    }

    if (!FormValidator.isValidEmail(email)) {
      setEmailError(true);
      return false;
    }

    if (!password) {
      setPasswordError(true);
      return false;
    }

    if (password.length < 8) {
      setPasswordError(true);

      return false;
    }
    setPasswordError(false);

    if (!confirmPassword) {
      setConfirmPasswordError(true);
      return false;
    }
    if (password !== confirmPassword) {
      setConfirmPasswordError(true);
      return false;
    }
    setConfirmPasswordError(false);

    return true;
  };

  const onSignUp = async () => {
    if (isLoading) return;
    const operation = authOperation.start();
    setSignUpError(undefined);

    if (validateFieldsForSignUp()) {
      setLoading(true);

      try {
        const session = await signUpMutation.mutateAsync([
          {
            email,
            password,
            name,
            cloudflareTurnstileToken: token,
          },
        ]);
        authOperation.accept({ operation, session });
      } catch (e) {
        if (!authOperation.isCurrent(operation)) return;
        setSignUpError(e);
        resetCloudflareTurnstile();
      }
    }

    setLoading(false);
  };
  const controlId = useId();
  const { t } = useTranslation();

  const authOperation = useAuthOperation();

  const { token, containerRef, resetCloudflareTurnstile } = useCloudflareTurnstile();

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-4 text-center text-xl font-semibold">{t('users.signUp.title')}</div>

      {isClaimingInstance && (
        <div className="mb-5 text-center text-sm text-muted">
          {t('users.signUp.administersInstance')}
        </div>
      )}

      {(GOOGLE_CLIENT_ID || GITHUB_CLIENT_ID) && (
        <>
          <OAuthSignInButtons />

          <div className="relative my-4">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-border"></div>
            </div>
            <div className="relative flex justify-center text-sm">
              <span className="bg-surface px-2 text-muted">{t('users.oauth.divider')}</span>
            </div>
          </div>
        </>
      )}

      <div className="mb-3 min-w-0">
        <label htmlFor={controlId + 'field-1'} className="mb-1 block text-sm text-content">
          {t('users.fields.yourName')}
        </label>
        <TextInput
          aria-invalid={isNameError}
          id={controlId + 'field-1'}
          placeholder={t('users.signUp.namePlaceholder')}
          value={name}
          onChange={(e) => {
            setNameError(false);
            setName(e.currentTarget.value);
          }}
          error={isNameError ? <span>{t('users.validation.nameRequired')}</span> : undefined}
        />
      </div>

      <div className="mb-3 min-w-0">
        <label htmlFor={controlId + 'field-2'} className="mb-1 block text-sm text-content">
          {t('users.fields.yourEmail')}
        </label>
        <TextInput
          aria-invalid={isEmailError}
          id={controlId + 'field-2'}
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

      <div className="mb-3 min-w-0">
        <label htmlFor={controlId + 'field-3'} className="mb-1 block text-sm text-content">
          {t('common.fields.password')}
        </label>
        <PasswordInput
          aria-invalid={isPasswordError}
          id={controlId + 'field-3'}
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

      <div className="mb-3 min-w-0">
        <label htmlFor={controlId + 'field-4'} className="mb-1 block text-sm text-content">
          {t('users.fields.confirmPassword')}
        </label>
        <PasswordInput
          aria-invalid={isConfirmPasswordError}
          id={controlId + 'field-4'}
          placeholder="********"
          value={confirmPassword}
          onChange={(e) => {
            setConfirmPasswordError(false);
            setConfirmPassword(e.currentTarget.value);
          }}
          visible={isConfirmPasswordVisible}
          onVisibilityChange={setConfirmPasswordVisible}
          visibilityToggleButtonProps={{
            'aria-label': t('common.actions.togglePasswordVisibility'),
          }}
          error={
            isConfirmPasswordError ? (
              <span>{t('users.validation.passwordsDoNotMatch')}</span>
            ) : undefined
          }
        />
      </div>

      <CloudflareTurnstileWidget containerRef={containerRef} />

      <Button
        aria-label={t('users.signUp.submit')}
        disabled={isLoading || (!!CLOUDFLARE_TURNSTILE_SITE_KEY && !token)}
        loading={isLoading}
        className="!h-auto min-h-8 w-full max-w-full !whitespace-normal"
        onClick={() => {
          onSignUp();
        }}
        variant="filled"
        aria-busy={isLoading}
      >
        {t('users.signUp.submit')}
      </Button>

      {signUpError !== undefined && (
        <div className="mt-3 flex justify-center text-center text-sm text-danger">
          {translateApiError(signUpError, t)}
        </div>
      )}

      {onSwitchToSignIn && (
        <div className="mt-4 text-center text-sm text-muted">
          <Trans
            i18nKey="users.signUp.hasAccount"
            components={{
              signInLink: (
                <button
                  type="button"
                  onClick={onSwitchToSignIn}
                  className="cursor-pointer font-medium text-accent hover:text-accent"
                />
              ),
            }}
          />
        </div>
      )}
    </div>
  );
}
