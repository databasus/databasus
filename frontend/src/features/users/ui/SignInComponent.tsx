import { EyeInvisibleOutlined, EyeTwoTone } from '@ant-design/icons';
import { Button, Input } from 'antd';
import { type JSX, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { useCloudflareTurnstile } from '../../../shared/hooks/useCloudflareTurnstile';

import {
  CLOUDFLARE_TURNSTILE_SITE_KEY,
  GITHUB_CLIENT_ID,
  GOOGLE_CLIENT_ID,
  IS_EMAIL_CONFIGURED,
} from '../../../constants';
import { type PendingSignIn, userApi } from '../../../entity/users';
import { translateApiError } from '../../../shared/i18n';
import { FormValidator } from '../../../shared/lib/FormValidator';
import { CloudflareTurnstileWidget } from '../../../shared/ui/CloudflareTurnstileWidget';
import { GithubOAuthComponent } from './oauth/GithubOAuthComponent';
import { GoogleOAuthComponent } from './oauth/GoogleOAuthComponent';

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
  const { t } = useTranslation();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [passwordVisible, setPasswordVisible] = useState(false);

  const [isLoading, setLoading] = useState(false);

  const [isEmailError, setEmailError] = useState(false);
  const [passwordError, setPasswordError] = useState(false);

  const [signInError, setSignInError] = useState<unknown>();

  const { token, containerRef, resetCloudflareTurnstile } = useCloudflareTurnstile();

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
    setSignInError(undefined);

    if (validateFieldsForSignIn()) {
      setLoading(true);

      try {
        const outcome = await userApi.signIn({
          email,
          password,
          cloudflareTurnstileToken: token,
        });

        if (!outcome.isCompleted) {
          onCodeRequired?.(outcome.pendingSignIn);
        }
      } catch (e) {
        setSignInError(e);
        resetCloudflareTurnstile();
      }

      setLoading(false);
    }
  };

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-5 text-center text-2xl font-bold">{t('users.signIn.title')}</div>

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

      <div className="mt-3" />

      <CloudflareTurnstileWidget containerRef={containerRef} />

      <Button
        disabled={isLoading || (!!CLOUDFLARE_TURNSTILE_SITE_KEY && !token)}
        loading={isLoading}
        className="w-full"
        onClick={() => {
          onSignIn();
        }}
        type="primary"
      >
        {t('users.signIn.submit')}
      </Button>

      {signInError !== undefined && (
        <div className="mt-3 flex justify-center text-center text-sm text-red-600">
          {translateApiError(signInError, t)}
        </div>
      )}

      <div className="mt-4 text-center text-sm text-gray-600 dark:text-gray-400">
        {onSwitchToSignUp && (
          <>
            <Trans
              i18nKey="users.signIn.noAccount"
              components={{
                signUpLink: (
                  <button
                    type="button"
                    onClick={onSwitchToSignUp}
                    className="cursor-pointer font-medium text-blue-600 hover:text-blue-700 dark:!text-blue-500"
                  />
                ),
              }}
            />
            <br />
          </>
        )}
        {IS_EMAIL_CONFIGURED && (
          <button
            type="button"
            onClick={onSwitchToResetPassword}
            className="cursor-pointer font-medium text-blue-600 hover:text-blue-700 dark:!text-blue-500"
          >
            {t('users.signIn.forgotPassword')}
          </button>
        )}
      </div>

      <div className="mb-10" />
    </div>
  );
}
