import { EyeInvisibleOutlined, EyeTwoTone } from '@ant-design/icons';
import { Form, App, Button, Input } from 'antd';
import { useId } from 'react';
import { type JSX, useEffect, useRef, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { userApi } from '@/entities/users';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { FormValidator } from '@/shared/lib/FormValidator';

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
  const [email, setEmail] = useState(initialEmail);
  const [code, setCode] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isPasswordVisible, setPasswordVisible] = useState(false);
  const [isConfirmPasswordVisible, setConfirmPasswordVisible] = useState(false);

  const [isLoading, setLoading] = useState(false);
  const [isResetComplete, setIsResetComplete] = useState(false);

  const [isEmailError, setEmailError] = useState(false);
  const [isCodeError, setCodeError] = useState(false);
  const [isPasswordError, setPasswordError] = useState(false);
  const [isConfirmPasswordError, setConfirmPasswordError] = useState(false);

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

      isValid = false;
    } else {
      setCodeError(false);
    }

    if (!newPassword) {
      setPasswordError(true);
      isValid = false;
    } else if (newPassword.length < 8) {
      setPasswordError(true);

      isValid = false;
    } else {
      setPasswordError(false);
    }

    if (!confirmPassword) {
      setConfirmPasswordError(true);
      isValid = false;
    } else if (newPassword !== confirmPassword) {
      setConfirmPasswordError(true);

      isValid = false;
    } else {
      setConfirmPasswordError(false);
    }

    return isValid;
  };

  const onResetPassword = async () => {
    if (isLoading || isResetComplete) return;
    setResetError(undefined);

    if (validateFields()) {
      setLoading(true);

      try {
        await userApi.resetPassword({
          email,
          code,
          newPassword,
        });

        if (!isCurrent()) return;
        setIsResetComplete(true);
        message.success(i18n.t('users.resetPassword.succeeded'));

        navigationTimer.current = setTimeout(() => {
          if (!isCurrent()) return;
          if (onSwitchToSignIn) {
            onSwitchToSignIn();
          }
        }, 2000);
      } catch (e) {
        if (!isCurrent()) return;
        setResetError(e);
      }

      setLoading(false);
    }
  };
  const controlId = useId();
  const isCurrent = useSessionGuard();
  const navigationTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const { t, i18n } = useTranslation();
  const { message } = App.useApp();

  useEffect(
    () => () => {
      if (navigationTimer.current) clearTimeout(navigationTimer.current);
    },
    [],
  );

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-5 text-center text-2xl font-bold">{t('users.resetPassword.title')}</div>

      <div className="mb-4 text-center text-sm text-muted">
        {t('users.resetPassword.description')}
      </div>

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
        label={t('users.resetPassword.code')}
        layout="vertical"
        validateStatus={isCodeError ? 'error' : undefined}
        help={
          isCodeError ? (
            <span id={controlId + 'field-2' + '-error'}>
              {t('users.resetPassword.invalidCode')}
            </span>
          ) : undefined
        }
      >
        <Input
          aria-invalid={isCodeError}
          aria-describedby={controlId + 'field-2' + '-error'}
          id={controlId + 'field-2'}
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
      </Form.Item>

      <Form.Item
        htmlFor={controlId + 'field-3'}
        label={t('users.fields.newPassword')}
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
          value={newPassword}
          onChange={(e) => {
            setPasswordError(false);
            setNewPassword(e.currentTarget.value);
          }}
          status={isPasswordError ? 'error' : undefined}
          iconRender={(visible) => (visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />)}
          visibilityToggle={{ visible: isPasswordVisible, onVisibleChange: setPasswordVisible }}
        />
      </Form.Item>

      <Form.Item
        htmlFor={controlId + 'field-4'}
        label={t('users.resetPassword.confirmPassword')}
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

      <Button
        aria-label={t('users.resetPassword.submit')}
        disabled={isLoading || isResetComplete}
        loading={isLoading}
        className="!h-auto min-h-8 w-full max-w-full !whitespace-normal"
        onClick={() => {
          onResetPassword();
        }}
        type="primary"
      >
        {t('users.resetPassword.submit')}
      </Button>

      {resetError !== undefined && (
        <div className="mt-3 flex justify-center text-center text-sm text-danger">
          {translateApiError(resetError, t)}
        </div>
      )}

      <div className="mt-4 text-center text-sm text-muted">
        {onSwitchToRequestCode && (
          <>
            <Trans
              i18nKey="users.resetPassword.noCode"
              components={{
                requestCodeLink: (
                  <button
                    type="button"
                    onClick={onSwitchToRequestCode}
                    className="cursor-pointer font-medium text-accent hover:text-accent"
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
            className="cursor-pointer font-medium text-accent hover:text-accent"
          >
            {t('users.resetPassword.backToSignIn')}
          </button>
        )}
      </div>
    </div>
  );
}
