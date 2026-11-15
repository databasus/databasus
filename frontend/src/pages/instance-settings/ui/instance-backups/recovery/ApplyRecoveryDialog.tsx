import { Alert, Button, Checkbox, PasswordInput, TextInput } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type IdentityProof,
  type IdentityProofMethod,
  type InstanceRecovery,
  instanceRecoveryApi,
} from '@/entities/instance-backups';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { Dialog } from '@/shared/ui';

interface Props {
  recovery: InstanceRecovery;
  identityProofMethods: IdentityProofMethod[];
  onApplied: (recovery: InstanceRecovery) => void;
  onClose: () => void;
}

// Apply needs the replacement confirmation and a fresh identity proof bound to this recovery; a
// repeated request for the same recovery observes the one already accepted.
export function ApplyRecoveryDialog({ recovery, identityProofMethods, onApplied, onClose }: Props) {
  const { t } = useTranslation();
  const isCurrent = useSessionGuard();
  const [isReplacementConfirmed, setIsReplacementConfirmed] = useState(false);
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [isCodeSent, setIsCodeSent] = useState(false);
  const [error, setError] = useState<unknown>();
  const canUsePassword = identityProofMethods.includes('PASSWORD');
  const canUseEmailCode = identityProofMethods.includes('EMAIL_CODE');

  const confirmationMutation = useSessionMutation({
    mutationFn: (recoveryId: string) => instanceRecoveryApi.requestConfirmation(recoveryId),
  });
  const applyMutation = useSessionMutation({
    mutationFn: (proof: IdentityProof) => instanceRecoveryApi.applyRecovery(recovery.id, proof),
  });
  const isApplying = applyMutation.isPending;

  const requestCode = async () => {
    if (confirmationMutation.isPending) return;

    setError(undefined);

    try {
      await confirmationMutation.mutateAsync(recovery.id);
      if (isCurrent()) setIsCodeSent(true);
    } catch (failure: unknown) {
      if (isCurrent()) setError(failure);
    } finally {
      if (isCurrent()) confirmationMutation.reset();
    }
  };

  const apply = async (proof: IdentityProof) => {
    if (isApplying || !isReplacementConfirmed) return;

    setError(undefined);

    try {
      const appliedRecovery = await applyMutation.mutateAsync(proof);
      if (isCurrent()) onApplied(appliedRecovery);
    } catch (failure: unknown) {
      if (isCurrent()) setError(failure);
    } finally {
      if (isCurrent()) applyMutation.reset();
    }
  };

  return (
    <Dialog
      title={t('instanceRecovery.apply.title')}
      opened
      closeOnClickOutside={false}
      onClose={() => {
        if (!isApplying) onClose();
      }}
    >
      <p className="text-sm">{t('instanceRecovery.apply.description')}</p>
      <Checkbox
        className="mt-3"
        label={t('instanceRecovery.apply.confirmReplacement')}
        checked={isReplacementConfirmed}
        disabled={isApplying}
        onChange={(event) => setIsReplacementConfirmed(event.currentTarget.checked)}
      />

      {canUsePassword && (
        <PasswordInput
          className="mt-3"
          size="xs"
          label={t('masterKey.proof.password')}
          autoComplete="current-password"
          value={password}
          disabled={isApplying}
          onChange={(event) => setPassword(event.currentTarget.value)}
        />
      )}
      {canUseEmailCode && (
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
              disabled={isApplying}
              onChange={(event) => setCode(event.currentTarget.value)}
            />
          )}
        </div>
      )}
      {!canUsePassword && !canUseEmailCode && (
        <p className="mt-3 text-sm text-danger">{t('errors.identityProofUnavailable')}</p>
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
        <Button disabled={isApplying} onClick={onClose}>
          {t('common.actions.cancel')}
        </Button>
        <Button
          variant="filled"
          color="danger"
          loading={isApplying}
          disabled={!isReplacementConfirmed || isApplying || (!password && !code)}
          onClick={() => void apply(code ? { code } : { password })}
        >
          {t('instanceRecovery.apply.submit')}
        </Button>
      </div>
    </Dialog>
  );
}
