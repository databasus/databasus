import { EyeInvisibleOutlined, EyeTwoTone } from '@ant-design/icons';
import { Form, Button, Input } from 'antd';
import { useId } from 'react';
import { type JSX, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { userApi } from '@/entities/users';
import { OAuthSignInButtons } from '@/features/oauth-sign-in';
import { useAuthOperation } from '@/shared/auth';
import { CLOUDFLARE_TURNSTILE_SITE_KEY, GITHUB_CLIENT_ID, GOOGLE_CLIENT_ID } from '@/shared/config';
import { translateApiError } from '@/shared/i18n';
import { FormValidator } from '@/shared/lib/FormValidator';

import { useCloudflareTurnstile } from '../model/useCloudflareTurnstile';
import { CloudflareTurnstileWidget } from './CloudflareTurnstileWidget';

interface SignUpComponentProps {
  onSwitchToSignIn?: () => void;
  isClaimingInstance?: boolean;
}

export function SignUpComponent({
  onSwitchToSignIn,
  isClaimingInstance,
}: SignUpComponentProps): JSX.Element {
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
        const session = await userApi.signUp({
          email,
          password,
          name,
          cloudflareTurnstileToken: token,
        });
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
      <div className="mb-5 text-center text-2xl font-bold">{t('users.signUp.title')}</div>

      {isClaimingInstance && (
        <div className="mb-5 text-center text-sm text-muted">
          {t('users.signUp.administersInstance')}
        </div>
      )}

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
        label={t('users.fields.yourName')}
        layout="vertical"
        validateStatus={isNameError ? 'error' : undefined}
        help={
          isNameError ? (
            <span id={controlId + 'field-1' + '-error'}>{t('users.validation.nameRequired')}</span>
          ) : undefined
        }
      >
        <Input
          aria-invalid={isNameError}
          aria-describedby={controlId + 'field-1' + '-error'}
          id={controlId + 'field-1'}
          placeholder={t('users.signUp.namePlaceholder')}
          value={name}
          onChange={(e) => {
            setNameError(false);
            setName(e.currentTarget.value);
          }}
          status={isNameError ? 'error' : undefined}
        />
      </Form.Item>

      <Form.Item
        htmlFor={controlId + 'field-2'}
        label={t('users.fields.yourEmail')}
        layout="vertical"
        validateStatus={isEmailError ? 'error' : undefined}
        help={
          isEmailError ? (
            <span id={controlId + 'field-2' + '-error'}>{t('users.validation.emailInvalid')}</span>
          ) : undefined
        }
      >
        <Input
          aria-invalid={isEmailError}
          aria-describedby={controlId + 'field-2' + '-error'}
          id={controlId + 'field-2'}
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
        htmlFor={controlId + 'field-3'}
        label={t('common.fields.password')}
        layout="vertical"
        validateStatus={isPasswordError ? 'error' : undefined}
        help={
          isPasswordError ? (
            <span id={controlId + 'field-3' + '-error'}>
              {t('users.validation.passwordTooShort', { minLength: 8 })}
            </span>
          ) : undefined
        }
      >
        <Input.Password
          aria-invalid={isPasswordError}
          aria-describedby={controlId + 'field-3' + '-error'}
          id={controlId + 'field-3'}
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

      <Form.Item
        htmlFor={controlId + 'field-4'}
        label={t('users.fields.confirmPassword')}
        layout="vertical"
        validateStatus={isConfirmPasswordError ? 'error' : undefined}
        help={
          isConfirmPasswordError ? (
            <span id={controlId + 'field-4' + '-error'}>
              {t('users.validation.passwordsDoNotMatch')}
            </span>
          ) : undefined
        }
      >
        <Input.Password
          aria-invalid={isConfirmPasswordError}
          aria-describedby={controlId + 'field-4' + '-error'}
          id={controlId + 'field-4'}
          placeholder="********"
          value={confirmPassword}
          status={isConfirmPasswordError ? 'error' : undefined}
          onChange={(e) => {
            setConfirmPasswordError(false);
            setConfirmPassword(e.currentTarget.value);
          }}
          iconRender={(visible) => (visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />)}
          visibilityToggle={{
            visible: isConfirmPasswordVisible,
            onVisibleChange: setConfirmPasswordVisible,
          }}
        />
      </Form.Item>

      <div className="mt-3" />

      <CloudflareTurnstileWidget containerRef={containerRef} />

      <Button
        aria-label={t('users.signUp.submit')}
        disabled={isLoading || (!!CLOUDFLARE_TURNSTILE_SITE_KEY && !token)}
        loading={isLoading}
        className="!h-auto min-h-8 w-full max-w-full !whitespace-normal"
        onClick={() => {
          onSignUp();
        }}
        type="primary"
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

      <div className="mb-10" />
    </div>
  );
}
