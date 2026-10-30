import { Button, Input } from 'antd';
import { type JSX, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { userApi } from '../../../entities/users';
import { useCloudflareTurnstile } from '../../../shared/hooks/useCloudflareTurnstile';
import { translateApiError } from '../../../shared/i18n';
import { FormValidator } from '../../../shared/lib/FormValidator';
import { CloudflareTurnstileWidget } from '../../../shared/ui/CloudflareTurnstileWidget';

interface RequestResetPasswordComponentProps {
  onSwitchToSignIn?: () => void;
  onSwitchToResetPassword?: (email: string) => void;
}

export function RequestResetPasswordComponent({
  onSwitchToSignIn,
  onSwitchToResetPassword,
}: RequestResetPasswordComponentProps): JSX.Element {
  const { t } = useTranslation();
  const [email, setEmail] = useState('');
  const [isLoading, setLoading] = useState(false);
  const [isEmailError, setEmailError] = useState(false);
  const [sendCodeError, setSendCodeError] = useState<unknown>();
  const [isCodeSent, setIsCodeSent] = useState(false);

  const { token, containerRef, resetCloudflareTurnstile } = useCloudflareTurnstile();

  const validateEmail = (): boolean => {
    if (!email) {
      setEmailError(true);
      return false;
    }

    if (!FormValidator.isValidEmail(email)) {
      setEmailError(true);
      return false;
    }

    return true;
  };

  const onSendCode = async () => {
    setSendCodeError(undefined);
    setIsCodeSent(false);

    if (validateEmail()) {
      setLoading(true);

      try {
        await userApi.sendResetPasswordCode({
          email,
          cloudflareTurnstileToken: token,
        });
        setIsCodeSent(true);

        // After successful code send, switch to reset password form
        setTimeout(() => {
          if (onSwitchToResetPassword) {
            onSwitchToResetPassword(email);
          }
        }, 2000);
      } catch (e) {
        setSendCodeError(e);
        resetCloudflareTurnstile();
      }

      setLoading(false);
    }
  };

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-5 text-center text-2xl font-bold">
        {t('users.requestPasswordReset.title')}
      </div>

      <div className="mb-4 text-center text-sm text-gray-600 dark:text-gray-400">
        {t('users.requestPasswordReset.description')}
      </div>

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
        onPressEnter={() => {
          onSendCode();
        }}
      />

      <div className="mt-3" />

      <CloudflareTurnstileWidget containerRef={containerRef} />

      <Button
        disabled={isLoading}
        loading={isLoading}
        className="w-full"
        onClick={() => {
          onSendCode();
        }}
        type="primary"
      >
        {t('users.requestPasswordReset.submit')}
      </Button>

      {sendCodeError !== undefined && (
        <div className="mt-3 flex justify-center text-center text-sm text-red-600">
          {translateApiError(sendCodeError, t)}
        </div>
      )}

      {isCodeSent && (
        <div className="mt-3 flex justify-center text-center text-sm text-green-600">
          {t('users.requestPasswordReset.codeSent')}
        </div>
      )}

      {onSwitchToSignIn && (
        <div className="mt-4 text-center text-sm text-gray-600 dark:text-gray-400">
          <Trans
            i18nKey="users.requestPasswordReset.rememberPassword"
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
    </div>
  );
}
