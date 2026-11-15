import { Alert, Button, Checkbox } from '@mantine/core';
import { CircleAlert, PauseCircle } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type InstanceRecovery, instanceRecoveryApi } from '@/entities/instance-backups';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { formatInstant } from '@/shared/time';
import { Dialog } from '@/shared/ui';

interface Props {
  recovery: InstanceRecovery | null | undefined;
  onActivated: () => Promise<unknown>;
}

// The pause ends only when the server acknowledges activation and a fresh status no longer reports
// it, so a failed or pending request never looks like resumed work.
export function RecoveryPauseNotice({ recovery, onActivated }: Props) {
  const { t } = useTranslation();
  const isCurrent = useSessionGuard();
  const [isConfirming, setIsConfirming] = useState(false);
  const [isOldInstanceStopped, setIsOldInstanceStopped] = useState(false);
  const [activationError, setActivationError] = useState<unknown>();
  const activateMutation = useSessionMutation({
    mutationFn: (recoveryId: string) => instanceRecoveryApi.activateRecovery(recoveryId),
  });
  const isActivating = activateMutation.isPending;

  const activate = async () => {
    if (!recovery || isActivating || !isOldInstanceStopped) return;

    setActivationError(undefined);

    try {
      await activateMutation.mutateAsync(recovery.id);
      if (!isCurrent()) return;

      await onActivated();
      if (isCurrent()) setIsConfirming(false);
    } catch (error: unknown) {
      if (isCurrent()) setActivationError(error);
    } finally {
      if (isCurrent()) activateMutation.reset();
    }
  };

  return (
    <>
      <Alert
        className="mb-3"
        color="warning"
        icon={<PauseCircle aria-hidden size={16} />}
        title={t('instanceRecovery.pause.title')}
      >
        <p>{t('instanceRecovery.pause.description')}</p>
        {recovery?.snapshotAt && recovery.sourceRelease && (
          <p className="mt-1">
            {t('instanceRecovery.pause.snapshot', {
              time: formatInstant(recovery.snapshotAt),
              version: recovery.sourceRelease,
            })}
          </p>
        )}
        {!!recovery?.discardedCleanupCount && (
          <p className="mt-1">
            {t('instanceRecovery.pause.discardedCleanup', {
              count: recovery.discardedCleanupCount,
            })}
          </p>
        )}
        <Button
          className="mt-2"
          size="xs"
          variant="filled"
          disabled={!recovery}
          onClick={() => {
            setIsOldInstanceStopped(false);
            setActivationError(undefined);
            setIsConfirming(true);
          }}
        >
          {t('instanceRecovery.pause.activate')}
        </Button>
      </Alert>

      {isConfirming && (
        <Dialog
          title={t('instanceRecovery.pause.confirmTitle')}
          opened
          onClose={() => {
            if (!isActivating) setIsConfirming(false);
          }}
        >
          <p className="text-sm">{t('instanceRecovery.pause.confirmDescription')}</p>
          <Checkbox
            className="mt-3"
            label={t('instanceRecovery.pause.oldInstanceStopped')}
            checked={isOldInstanceStopped}
            disabled={isActivating}
            onChange={(event) => setIsOldInstanceStopped(event.currentTarget.checked)}
          />
          {activationError !== undefined && (
            <Alert
              className="mt-3"
              color="danger"
              icon={<CircleAlert aria-hidden size={16} />}
              title={translateApiError(activationError, t)}
            />
          )}
          <div className="mt-5 flex flex-wrap justify-end gap-2">
            <Button disabled={isActivating} onClick={() => setIsConfirming(false)}>
              {t('common.actions.cancel')}
            </Button>
            <Button
              variant="filled"
              loading={isActivating}
              disabled={!isOldInstanceStopped || isActivating}
              onClick={() => void activate()}
            >
              {t('instanceRecovery.pause.activate')}
            </Button>
          </div>
        </Dialog>
      )}
    </>
  );
}
