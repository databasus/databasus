import { Form, Button, Input } from 'antd';
import { useId } from 'react';
import { type JSX, useEffect, useRef, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { userApi } from '@/entities/users';
import { useCloudflareTurnstile } from '@/pages/auth/model/useCloudflareTurnstile';
import { CloudflareTurnstileWidget } from '@/pages/auth/ui/CloudflareTurnstileWidget';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { FormValidator } from '@/shared/lib/FormValidator';

interface RequestResetPasswordComponentProps {
  onSwitchToSignIn?: () => void;
  onSwitchToResetPassword?: (email: string) => void;
}

export function RequestResetPasswordComponent({
  onSwitchToSignIn,
  onSwitchToResetPassword,
}: RequestResetPasswordComponentProps): JSX.Element {
  const [email, setEmail] = useState('');
  const [isLoading, setLoading] = useState(false);
  const [isEmailError, setEmailError] = useState(false);
  const [sendCodeError, setSendCodeError] = useState<unknown>();
  const [isCodeSent, setIsCodeSent] = useState(false);

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
    if (isLoading || isCodeSent) return;
    setSendCodeError(undefined);
    setIsCodeSent(false);

    if (validateEmail()) {
      setLoading(true);

      try {
        await userApi.sendResetPasswordCode({
          email,
          cloudflareTurnstileToken: token,
        });
        if (!isCurrent()) return;
        setIsCodeSent(true);

        navigationTimer.current = setTimeout(() => {
          if (!isCurrent()) return;
          if (onSwitchToResetPassword) {
            onSwitchToResetPassword(email);
          }
        }, 2000);
      } catch (e) {
        if (!isCurrent()) return;
        setSendCodeError(e);
        resetCloudflareTurnstile();
      }

      setLoading(false);
    }
  };
  const controlId = useId();
  const isCurrent = useSessionGuard();
  const navigationTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const { t } = useTranslation();

  const { token, containerRef, resetCloudflareTurnstile } = useCloudflareTurnstile();

  useEffect(
    () => () => {
      if (navigationTimer.current) clearTimeout(navigationTimer.current);
    },
    [],
  );

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-5 text-center text-2xl font-bold">
        {t('users.requestPasswordReset.title')}
      </div>

      <div className="mb-4 text-center text-sm text-muted">
        {t('users.requestPasswordReset.description')}
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
          onPressEnter={() => {
            onSendCode();
          }}
        />
      </Form.Item>

      <div className="mt-3" />

      <CloudflareTurnstileWidget containerRef={containerRef} />

      <Button
        aria-label={t('users.requestPasswordReset.submit')}
        disabled={isLoading || isCodeSent}
        loading={isLoading}
        className="!h-auto min-h-8 w-full max-w-full !whitespace-normal"
        onClick={() => {
          onSendCode();
        }}
        type="primary"
      >
        {t('users.requestPasswordReset.submit')}
      </Button>

      {sendCodeError !== undefined && (
        <div className="mt-3 flex justify-center text-center text-sm text-danger">
          {translateApiError(sendCodeError, t)}
        </div>
      )}

      {isCodeSent && (
        <div className="mt-3 flex justify-center text-center text-sm text-success">
          {t('users.requestPasswordReset.codeSent')}
        </div>
      )}

      {onSwitchToSignIn && (
        <div className="mt-4 text-center text-sm text-muted">
          <Trans
            i18nKey="users.requestPasswordReset.rememberPassword"
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
