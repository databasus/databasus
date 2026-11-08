import { Button, PasswordInput, TextInput } from '@mantine/core';
import { useId } from 'react';
import { type JSX, useEffect, useRef, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { userApi } from '@/entities/users';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
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
  const resetPasswordMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof userApi.resetPassword>) =>
      userApi.resetPassword(...parameters),
  });

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
        await resetPasswordMutation.mutateAsync([
          {
            email,
            code,
            newPassword,
          },
        ]);

        if (!isCurrent()) return;
        setIsResetComplete(true);
        notify({ kind: 'success', title: i18n.t('users.resetPassword.succeeded') });
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
  const notify = useFeedback();
  useEffect(
    () => () => {
      if (navigationTimer.current) clearTimeout(navigationTimer.current);
    },
    [],
  );

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-4 text-center text-xl font-semibold">{t('users.resetPassword.title')}</div>

      <div className="mb-4 text-center text-sm text-muted">
        {t('users.resetPassword.description')}
      </div>

      <div className="mb-3 min-w-0">
        <label htmlFor={controlId + 'field-1'} className="mb-1 block text-sm text-content">
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

      <div className="mb-3 min-w-0">
        <label htmlFor={controlId + 'field-2'} className="mb-1 block text-sm text-content">
          {t('users.resetPassword.code')}
        </label>
        <TextInput
          aria-invalid={isCodeError}
          id={controlId + 'field-2'}
          placeholder="123456"
          value={code}
          onChange={(e) => {
            setCodeError(false);
            const value = e.currentTarget.value.replace(/\D/g, '').slice(0, 6);
            setCode(value);
          }}
          maxLength={6}
          error={isCodeError ? <span>{t('users.resetPassword.invalidCode')}</span> : undefined}
        />
      </div>

      <div className="mb-3 min-w-0">
        <label htmlFor={controlId + 'field-3'} className="mb-1 block text-sm text-content">
          {t('users.fields.newPassword')}
        </label>
        <PasswordInput
          aria-invalid={isPasswordError}
          id={controlId + 'field-3'}
          placeholder="********"
          value={newPassword}
          onChange={(e) => {
            setPasswordError(false);
            setNewPassword(e.currentTarget.value);
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
          {t('users.resetPassword.confirmPassword')}
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

      <Button
        aria-label={t('users.resetPassword.submit')}
        disabled={isLoading || isResetComplete}
        loading={isLoading}
        className="!h-auto min-h-8 w-full max-w-full !whitespace-normal"
        onClick={() => {
          onResetPassword();
        }}
        variant="filled"
        aria-busy={isLoading}
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
