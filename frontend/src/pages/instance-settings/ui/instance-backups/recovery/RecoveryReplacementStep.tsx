import { Alert, Button, Checkbox, TextInput } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type InstanceRecovery, instanceRecoveryApi } from '@/entities/instance-backups';
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

// Apply needs the replacement confirmation and, while the instance requires two-factor sign-in, the
// code emailed for this recovery; a repeated request for the same recovery observes the one already
// accepted.
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
  const isCodeRejected =
    applyError instanceof ApiError && applyError.code === 'identity_proof_invalid';
  const error = isCodeRejected ? codeError : (applyError ?? codeError);
  const codeFieldError =
    isCodeRejected && isCodeRejectable ? t('masterKey.proof.wrongCode') : undefined;

  const confirmationMutation = useSessionMutation({
    mutationFn: (recoveryId: string) => instanceRecoveryApi.requestConfirmation(recoveryId),
  });

  const requestCode = async () => {
    if (confirmationMutation.isPending) return;

    setCodeError(undefined);

    try {
      await confirmationMutation.mutateAsync(recovery.id);
      if (isCurrent()) setIsCodeSent(true);
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

      {isCodeRequired && (
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
