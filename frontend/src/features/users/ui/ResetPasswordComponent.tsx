import { EyeInvisibleOutlined, EyeTwoTone } from '@ant-design/icons';
import { App, Button, Input } from 'antd';
import { type JSX, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { userApi } from '../../../entities/users';
import { translateApiError } from '../../../shared/i18n';
import { FormValidator } from '../../../shared/lib/FormValidator';

interface ResetPasswordComponentProps {
  onSwitchToSignIn?: () => void;
  onSwitchToRequestCode?: () => void;
  initialEmail?: string;
}

export function ResetPasswordComponent({
  onSwitchToSignIn,
  onSwitchToRequestCode,
  initialEmail = '',
}: ResetPasswordComponentProps): JSX.Element {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [email, setEmail] = useState(initialEmail);
  const [code, setCode] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordVisible, setPasswordVisible] = useState(false);
  const [confirmPasswordVisible, setConfirmPasswordVisible] = useState(false);

  const [isLoading, setLoading] = useState(false);

  const [isEmailError, setEmailError] = useState(false);
  const [isCodeError, setCodeError] = useState(false);
  const [passwordError, setPasswordError] = useState(false);
  const [confirmPasswordError, setConfirmPasswordError] = useState(false);

  const [resetError, setResetError] = useState<unknown>();

  const validateFields = (): boolean => {
    let isValid = true;

    if (!email) {
      setEmailError(true);
      isValid = false;
    } else if (!FormValidator.isValidEmail(email)) {
      setEmailError(true);
      isValid = false;
    } else {
      setEmailError(false);
    }

    if (!code) {
      setCodeError(true);
      isValid = false;
    } else if (!/^\d{6}$/.test(code)) {
      setCodeError(true);
      message.error(t('users.resetPassword.invalidCode'));
      isValid = false;
    } else {
      setCodeError(false);
    }

    if (!newPassword) {
      setPasswordError(true);
      isValid = false;
    } else if (newPassword.length < 8) {
      setPasswordError(true);
      message.error(t('users.validation.passwordTooShort', { minLength: 8 }));
      isValid = false;
    } else {
      setPasswordError(false);
    }

    if (!confirmPassword) {
      setConfirmPasswordError(true);
      isValid = false;
    } else if (newPassword !== confirmPassword) {
      setConfirmPasswordError(true);
      message.error(t('users.validation.passwordsDoNotMatch'));
      isValid = false;
    } else {
      setConfirmPasswordError(false);
    }

    return isValid;
  };

  const onResetPassword = async () => {
    setResetError(undefined);

    if (validateFields()) {
      setLoading(true);

      try {
        await userApi.resetPassword({
          email,
          code,
          newPassword,
        });

        message.success(t('users.resetPassword.succeeded'));

        // Redirect to sign in after successful reset
        setTimeout(() => {
          if (onSwitchToSignIn) {
            onSwitchToSignIn();
          }
        }, 2000);
      } catch (e) {
        setResetError(e);
      }

      setLoading(false);
    }
  };

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-5 text-center text-2xl font-bold">{t('users.resetPassword.title')}</div>

      <div className="mb-4 text-center text-sm text-gray-600 dark:text-gray-400">
        {t('users.resetPassword.description')}
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
      />

      <div className="my-1 text-xs font-semibold">{t('users.resetPassword.code')}</div>
      <Input
        placeholder="123456"
        value={code}
        onChange={(e) => {
          setCodeError(false);
          const value = e.currentTarget.value.replace(/\D/g, '').slice(0, 6);
          setCode(value);
        }}
        status={isCodeError ? 'error' : undefined}
        maxLength={6}
      />

      <div className="my-1 text-xs font-semibold">{t('users.fields.newPassword')}</div>
      <Input.Password
        placeholder="********"
        value={newPassword}
        onChange={(e) => {
          setPasswordError(false);
          setNewPassword(e.currentTarget.value);
        }}
        status={passwordError ? 'error' : undefined}
        iconRender={(visible) => (visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />)}
        visibilityToggle={{ visible: passwordVisible, onVisibleChange: setPasswordVisible }}
      />

      <div className="my-1 text-xs font-semibold">{t('users.resetPassword.confirmPassword')}</div>
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

      <Button
        disabled={isLoading}
        loading={isLoading}
        className="w-full"
        onClick={() => {
          onResetPassword();
        }}
        type="primary"
      >
        {t('users.resetPassword.submit')}
      </Button>

      {resetError !== undefined && (
        <div className="mt-3 flex justify-center text-center text-sm text-red-600">
          {translateApiError(resetError, t)}
        </div>
      )}

      <div className="mt-4 text-center text-sm text-gray-600 dark:text-gray-400">
        {onSwitchToRequestCode && (
          <>
            <Trans
              i18nKey="users.resetPassword.noCode"
              components={{
                requestCodeLink: (
                  <button
                    type="button"
                    onClick={onSwitchToRequestCode}
                    className="cursor-pointer font-medium text-blue-600 hover:text-blue-700 dark:!text-blue-500"
                  />
                ),
              }}
            />
            <br />
          </>
        )}
        {onSwitchToSignIn && (
          <button
            type="button"
            onClick={onSwitchToSignIn}
            className="cursor-pointer font-medium text-blue-600 hover:text-blue-700 dark:!text-blue-500"
          >
            {t('users.resetPassword.backToSignIn')}
          </button>
        )}
      </div>
    </div>
  );
}
