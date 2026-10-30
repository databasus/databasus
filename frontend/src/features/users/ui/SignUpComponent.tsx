import { EyeInvisibleOutlined, EyeTwoTone } from '@ant-design/icons';
import { App, Button, Input } from 'antd';
import { type JSX, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  CLOUDFLARE_TURNSTILE_SITE_KEY,
  GITHUB_CLIENT_ID,
  GOOGLE_CLIENT_ID,
} from '../../../constants';
import { userApi } from '../../../entities/users';
import { useCloudflareTurnstile } from '../../../shared/hooks/useCloudflareTurnstile';
import { translateApiError } from '../../../shared/i18n';
import { FormValidator } from '../../../shared/lib/FormValidator';
import { CloudflareTurnstileWidget } from '../../../shared/ui/CloudflareTurnstileWidget';
import { GithubOAuthComponent } from './oauth/GithubOAuthComponent';
import { GoogleOAuthComponent } from './oauth/GoogleOAuthComponent';

interface SignUpComponentProps {
  onSwitchToSignIn?: () => void;
  isClaimingInstance?: boolean;
}

export function SignUpComponent({
  onSwitchToSignIn,
  isClaimingInstance,
}: SignUpComponentProps): JSX.Element {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [passwordVisible, setPasswordVisible] = useState(false);
  const [confirmPassword, setConfirmPassword] = useState('');
  const [confirmPasswordVisible, setConfirmPasswordVisible] = useState(false);

  const [isLoading, setLoading] = useState(false);

  const [nameError, setNameError] = useState(false);
  const [isEmailError, setEmailError] = useState(false);
  const [passwordError, setPasswordError] = useState(false);
  const [confirmPasswordError, setConfirmPasswordError] = useState(false);

  const [signUpError, setSignUpError] = useState<unknown>();

  const { token, containerRef, resetCloudflareTurnstile } = useCloudflareTurnstile();

  const validateFieldsForSignUp = (): boolean => {
    if (!name || name.trim() === '') {
      setNameError(true);
      message.error(t('users.validation.nameRequired'));
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
      message.error(t('users.validation.passwordTooShort', { minLength: 8 }));
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
    setSignUpError(undefined);

    if (validateFieldsForSignUp()) {
      setLoading(true);

      try {
        await userApi.signUp({
          email,
          password,
          name,
          cloudflareTurnstileToken: token,
        });
      } catch (e) {
        setSignUpError(e);
        resetCloudflareTurnstile();
      }
    }

    setLoading(false);
  };

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-5 text-center text-2xl font-bold">{t('users.signUp.title')}</div>

      {isClaimingInstance && (
        <div className="mb-5 text-center text-sm text-gray-600 dark:text-gray-400">
          {t('users.signUp.administersInstance')}
        </div>
      )}

      <div className="mt-4">
        <div className="space-y-2">
          <GithubOAuthComponent />
          <GoogleOAuthComponent />
        </div>
      </div>

      {(GOOGLE_CLIENT_ID || GITHUB_CLIENT_ID) && (
        <div className="relative my-6">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-gray-300"></div>
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="bg-white px-2 text-gray-500 dark:bg-gray-900 dark:text-gray-400">
              {t('users.oauth.divider')}
            </span>
          </div>
        </div>
      )}

      <div className="my-1 text-xs font-semibold">{t('users.fields.yourName')}</div>
      <Input
        placeholder={t('users.signUp.namePlaceholder')}
        value={name}
        onChange={(e) => {
          setNameError(false);
          setName(e.currentTarget.value);
        }}
        status={nameError ? 'error' : undefined}
      />

      <div className="my-1 text-xs font-semibold">{t('users.fields.yourEmail')}</div>
      <Input
        placeholder="your@email.com"
        value={email}
        onChange={(e) => {
          setEmailError(false);
          setEmail(e.currentTarget.value.trim().toLowerCase());
        }}
        status={isEmailError ? 'error' : undefined}
        type="email"
      />

      <div className="my-1 text-xs font-semibold">{t('common.fields.password')}</div>
      <Input.Password
        placeholder="********"
        value={password}
        onChange={(e) => {
          setPasswordError(false);
          setPassword(e.currentTarget.value);
        }}
        status={passwordError ? 'error' : undefined}
        iconRender={(visible) => (visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />)}
        visibilityToggle={{ visible: passwordVisible, onVisibleChange: setPasswordVisible }}
      />

      <div className="my-1 text-xs font-semibold">{t('users.fields.confirmPassword')}</div>
      <Input.Password
        placeholder="********"
        value={confirmPassword}
        status={confirmPasswordError ? 'error' : undefined}
        onChange={(e) => {
          setConfirmPasswordError(false);
          setConfirmPassword(e.currentTarget.value);
        }}
        iconRender={(visible) => (visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />)}
        visibilityToggle={{
          visible: confirmPasswordVisible,
          onVisibleChange: setConfirmPasswordVisible,
        }}
      />

      <div className="mt-3" />

      <CloudflareTurnstileWidget containerRef={containerRef} />

      <Button
        disabled={isLoading || (!!CLOUDFLARE_TURNSTILE_SITE_KEY && !token)}
        loading={isLoading}
        className="w-full"
        onClick={() => {
          onSignUp();
        }}
        type="primary"
      >
        {t('users.signUp.submit')}
      </Button>

      {signUpError !== undefined && (
        <div className="mt-3 flex justify-center text-center text-sm text-red-600">
          {translateApiError(signUpError, t)}
        </div>
      )}

      {onSwitchToSignIn && (
        <div className="mt-4 text-center text-sm text-gray-600 dark:text-gray-400">
          <Trans
            i18nKey="users.signUp.hasAccount"
            components={{
              signInLink: (
                <button
                  type="button"
                  onClick={onSwitchToSignIn}
                  className="cursor-pointer font-medium text-blue-600 hover:text-blue-700 dark:!text-blue-500"
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
