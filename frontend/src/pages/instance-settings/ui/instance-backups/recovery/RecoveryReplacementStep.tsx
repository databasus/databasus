import { Alert, Button, Checkbox, Loader, TextInput } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type InstanceRecovery, instanceRecoveryApi } from '@/entities/instance-backups';
import { SecondFactorCodeField, useIsAuthenticatorAppConnected } from '@/entities/users';
import { ApiError, useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError, useLocale } from '@/shared/i18n';

interface Props {
  recovery: InstanceRecovery;
  isCodeRequired: boolean;
  isApplying: boolean;
  isCancelling: boolean;
  applyError: unknown;
  onApply: (code?: string) => void;
  onCancel: () => void;
}

// eslint-disable-next-line i18next/no-literal-string -- contract enum value
const AUTHENTICATOR_APP_METHOD = 'AUTHENTICATOR_APP';

export function RecoveryReplacementStep({
  recovery,
  isCodeRequired,
  isApplying,
  isCancelling,
  applyError,
  onApply,
  onCancel,
}: Props) {
  const { t } = useTranslation();
  const { formatDateTimeWithRelativeTime } = useLocale();
  const isCurrent = useSessionGuard();
  const [isReplacementConfirmed, setIsReplacementConfirmed] = useState(false);
  const [code, setCode] = useState('');
  const [isCodeSent, setIsCodeSent] = useState(false);
  const [codeError, setCodeError] = useState<unknown>();
  const [isCodeRejectable, setIsCodeRejectable] = useState(false);
  const [isRecoveryCode, setIsRecoveryCode] = useState(false);
  const [isAppCodeAnswered, setIsAppCodeAnswered] = useState(false);
  const isAppConnected = useIsAuthenticatorAppConnected(isCodeRequired);
  const isAppCodeExpected = isAppConnected === true || isAppCodeAnswered;
  const wrongCodeMessage = isAppCodeExpected
    ? t('errors.secondFactorInvalid')
    : t('masterKey.proof.wrongCode');
  const isCodeRejected =
    applyError instanceof ApiError && applyError.code === 'identity_proof_invalid';
  const error = isCodeRejected ? codeError : (applyError ?? codeError);
  const codeFieldError = isCodeRejected && isCodeRejectable ? wrongCodeMessage : undefined;

  const confirmationMutation = useSessionMutation({
    mutationFn: (recoveryId: string) => instanceRecoveryApi.requestConfirmation(recoveryId),
  });

  const requestCode = async () => {
    if (confirmationMutation.isPending) return;

    setCodeError(undefined);

    try {
      const confirmation = await confirmationMutation.mutateAsync(recovery.id);
      if (!isCurrent()) return;
      if (confirmation.method === AUTHENTICATOR_APP_METHOD) setIsAppCodeAnswered(true);
      else setIsCodeSent(true);
    } catch (failure: unknown) {
      if (isCurrent()) setCodeError(failure);
    } finally {
      if (isCurrent()) confirmationMutation.reset();
    }
  };

  const apply = () => {
    if (isApplying || !isReplacementConfirmed) return;

    setCodeError(undefined);

    if (isCodeRequired) {
      setIsCodeRejectable(true);
      onApply(code);
    } else {
      onApply();
    }
  };

  return (
    <div className="text-sm">
      <p className="font-medium">{t('instanceRecovery.form.ready')}</p>
      {recovery.snapshotAt && recovery.sourceRelease && (
        <p className="mt-1">
          {t('instanceRecovery.form.snapshot', {
            time: formatDateTimeWithRelativeTime(recovery.snapshotAt),
            version: recovery.sourceRelease,
          })}
        </p>
      )}
      <p className="mt-3">{t('instanceRecovery.apply.description')}</p>

      <Checkbox
        className="mt-3"
        label={t('instanceRecovery.apply.confirmReplacement')}
        checked={isReplacementConfirmed}
        disabled={isApplying}
        onChange={(event) => setIsReplacementConfirmed(event.currentTarget.checked)}
      />

      {isCodeRequired && isAppConnected === undefined && (
        <div className="mt-3">
          <Loader size="xs" role="status" />
        </div>
      )}

      {isCodeRequired && isAppCodeExpected && (
        <div className="mt-3">
          <p className="mb-2">{t('masterKey.appCodeRequired')}</p>
          <SecondFactorCodeField
            code={code}
            onCodeChange={(enteredCode) => {
              setCode(enteredCode);
              setIsCodeRejectable(false);
            }}
            isRecoveryCode={isRecoveryCode}
            onRecoveryCodeToggle={(isRecoveryCodeChosen) => {
              setIsRecoveryCode(isRecoveryCodeChosen);
              setCode('');
              setIsCodeRejectable(false);
            }}
            error={codeFieldError}
            isDisabled={isApplying}
          />
        </div>
      )}

      {isCodeRequired && isAppConnected === false && !isAppCodeAnswered && (
        <div className="mt-3">
          <Button
            size="xs"
            variant="outline"
            loading={confirmationMutation.isPending}
            disabled={isApplying}
            onClick={() => void requestCode()}
          >
            {isCodeSent ? t('masterKey.proof.resendCode') : t('masterKey.proof.sendCode')}
          </Button>
          {isCodeSent && (
            <TextInput
              className="mt-2"
              size="xs"
              label={t('masterKey.proof.code')}
              autoComplete="one-time-code"
              value={code}
              error={codeFieldError}
              disabled={isApplying}
              onChange={(event) => {
                setCode(event.currentTarget.value);
                setIsCodeRejectable(false);
              }}
            />
          )}
        </div>
      )}

      {error !== undefined && (
        <Alert
          className="mt-3"
          color="danger"
          icon={<CircleAlert aria-hidden size={16} />}
          title={translateApiError(error, t)}
        />
      )}

      <div className="mt-5 flex flex-wrap justify-end gap-2">
        <Button disabled={isApplying} loading={isCancelling} onClick={onCancel}>
          {t('common.actions.cancel')}
        </Button>
        <Button
          variant="filled"
          color="danger"
          loading={isApplying}
          disabled={!isReplacementConfirmed || isApplying || (isCodeRequired && !code)}
          onClick={apply}
        >
          {t('instanceRecovery.apply.submit')}
        </Button>
      </div>
    </div>
  );
}
