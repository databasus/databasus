import { EyeInvisibleOutlined, EyeTwoTone } from '@ant-design/icons';
import { Form, Button, Input } from 'antd';
import { useId } from 'react';
import { type JSX, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { type PendingSignIn, userApi } from '@/entities/users';
import { OAuthSignInButtons } from '@/features/oauth-sign-in';
import { useCloudflareTurnstile } from '@/pages/auth/model/useCloudflareTurnstile';
import { CloudflareTurnstileWidget } from '@/pages/auth/ui/CloudflareTurnstileWidget';
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
        const outcome = await userApi.signIn({
          email,
          password,
          cloudflareTurnstileToken: token,
        });

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

      <Form.Item
        htmlFor={controlId + 'field-1'}
        label={t('users.fields.yourEmail')}
        layout="vertical"
        validateStatus={isEmailError ? 'error' : undefined}
        help={
          isEmailError ? (
            <span id={controlId + 'field-1' + '-error'}>{t('users.validation.emailInvalid')}</span>
          ) : undefined
        }
      >
        <Input
          aria-invalid={isEmailError}
          aria-describedby={controlId + 'field-1' + '-error'}
          id={controlId + 'field-1'}
          placeholder="your@email.com"
          value={email}
          onChange={(e) => {
            setEmailError(false);
            setEmail(e.currentTarget.value.trim().toLowerCase());
          }}
          status={isEmailError ? 'error' : undefined}
          type="email"
        />
      </Form.Item>

      <Form.Item
        htmlFor={controlId + 'field-2'}
        label={t('common.fields.password')}
        layout="vertical"
        validateStatus={isPasswordError ? 'error' : undefined}
        help={
          isPasswordError ? (
            <span id={controlId + 'field-2' + '-error'}>
              {t('users.validation.passwordTooShort', { minLength: 8 })}
            </span>
          ) : undefined
        }
      >
        <Input.Password
          aria-invalid={isPasswordError}
          aria-describedby={controlId + 'field-2' + '-error'}
          id={controlId + 'field-2'}
          placeholder="********"
          value={password}
          onChange={(e) => {
            setPasswordError(false);
            setPassword(e.currentTarget.value);
          }}
          status={isPasswordError ? 'error' : undefined}
          iconRender={(visible) => (visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />)}
          visibilityToggle={{ visible: isPasswordVisible, onVisibleChange: setPasswordVisible }}
        />
      </Form.Item>

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
        type="primary"
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
