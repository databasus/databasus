import { TextInput } from '@mantine/core';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

interface Props {
  code: string;
  onCodeChange: (code: string) => void;
  isRecoveryCode: boolean;
  // Absent where only the app's code proves the step, such as confirming a new app.
  onRecoveryCodeToggle?: (isRecoveryCode: boolean) => void;
  error?: string;
  isDisabled?: boolean;
}

const APP_CODE_LENGTH = 6;
const RECOVERY_CODE_MAX_LENGTH = 32;

export function SecondFactorCodeField({
  code,
  onCodeChange,
  isRecoveryCode,
  onRecoveryCodeToggle,
  error,
  isDisabled = false,
}: Props) {
  const { t } = useTranslation();
  const fieldId = useId();

  return (
    <div>
      <TextInput
        id={fieldId}
        label={
          isRecoveryCode
            ? t('authenticatorApp.code.recoveryCode')
            : t('authenticatorApp.code.appCode')
        }
        // eslint-disable-next-line i18next/no-literal-string -- input pattern hint, not copy
        placeholder={isRecoveryCode ? 'xxxxx-xxxxx' : '123456'}
        autoComplete="one-time-code"
        inputMode={isRecoveryCode ? 'text' : 'numeric'}
        value={code}
        error={error}
        disabled={isDisabled}
        maxLength={isRecoveryCode ? RECOVERY_CODE_MAX_LENGTH : APP_CODE_LENGTH}
        onChange={(event) => {
          const enteredCode = event.currentTarget.value;
          onCodeChange(
            isRecoveryCode
              ? enteredCode.trim()
              : enteredCode.replace(/\D/g, '').slice(0, APP_CODE_LENGTH),
          );
        }}
      />

      {onRecoveryCodeToggle && (
        <button
          type="button"
          className="mt-2 cursor-pointer text-sm font-medium text-accent"
          disabled={isDisabled}
          onClick={() => onRecoveryCodeToggle(!isRecoveryCode)}
        >
          {isRecoveryCode
            ? t('authenticatorApp.code.useAppCode')
            : t('authenticatorApp.code.useRecoveryCode')}
        </button>
      )}
    </div>
  );
}
