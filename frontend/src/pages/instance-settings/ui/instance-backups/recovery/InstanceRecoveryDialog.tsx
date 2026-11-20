import { Alert, Button, Loader } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useEffect, useEffectEvent, useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  INSTANCE_RECOVERY_FAILURE_LABEL_KEYS,
  type InstanceRecovery,
  InstanceRecoveryStatus,
  expectRestoredSignIn,
  instanceRecoveryApi,
} from '@/entities/instance-backups';
import { MasterKeyTextField } from '@/pages/instance-settings/ui/instance-backups/recovery/MasterKeyTextField';
import { RecoveryArchivePicker } from '@/pages/instance-settings/ui/instance-backups/recovery/RecoveryArchivePicker';
import { RecoveryReplacementStep } from '@/pages/instance-settings/ui/instance-backups/recovery/RecoveryReplacementStep';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { Dialog } from '@/shared/ui';

const RECOVERY_POLL_INTERVAL_MS = 2_000;

interface Props {
  isCodeRequired: boolean;
  onClose: () => void;
}

// The archive handle and the entered key live only in this dialog's state. Its owner mounts it only
// while it is open and remounts it on a session change, so neither outlives the dialog or the
// account that entered them. Closing before apply cancels the server-side recovery as well, because
// a pending one would block the next import until it expires.
export function InstanceRecoveryDialog({ isCodeRequired, onClose }: Props) {
  const { t } = useTranslation();
  const isCurrent = useSessionGuard();
  const keyId = useId();
  const [archive, setArchive] = useState<File | null>(null);
  const [masterKey, setMasterKey] = useState('');
  const [isSubmitted, setIsSubmitted] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [recovery, setRecovery] = useState<InstanceRecovery>();
  const [error, setError] = useState<unknown>();
  const [applyError, setApplyError] = useState<unknown>();
  const [isCancelling, setIsCancelling] = useState(false);
  const uploadController = useRef<AbortController | null>(null);

  const applyMutation = useSessionMutation({
    mutationFn: ({ recoveryId, code }: { recoveryId: string; code?: string }) =>
      instanceRecoveryApi.applyRecovery(recoveryId, code),
  });

  const isArchiveMissing = isSubmitted && !archive;
  const isKeyMissing = isSubmitted && masterKey.length === 0;
  const isPreparing = recovery?.status === InstanceRecoveryStatus.VALIDATING;
  const isReady = recovery?.status === InstanceRecoveryStatus.READY;
  const isRestoring =
    recovery?.status === InstanceRecoveryStatus.APPLYING ||
    recovery?.status === InstanceRecoveryStatus.COMPLETED;
  const isClosingBlocked = isRestoring || applyMutation.isPending || isCancelling;

  const upload = async () => {
    setIsSubmitted(true);
    if (!archive || masterKey.length === 0 || isUploading) return;

    const controller = new AbortController();
    uploadController.current = controller;
    setError(undefined);
    setRecovery(undefined);
    setIsUploading(true);

    try {
      const createdRecovery = await instanceRecoveryApi.createRecovery(
        { archive, masterKey },
        { signal: controller.signal },
      );
      if (!isCurrent() || controller.signal.aborted) return;

      setRecovery(createdRecovery);
    } catch (failure: unknown) {
      if (isCurrent() && !controller.signal.aborted) setError(failure);
    } finally {
      if (uploadController.current === controller) uploadController.current = null;
      if (isCurrent()) setIsUploading(false);
    }
  };

  const close = async () => {
    if (isClosingBlocked) return;

    if (uploadController.current) {
      uploadController.current.abort();
      uploadController.current = null;
      onClose();
      return;
    }

    if (!recovery || (!isPreparing && !isReady)) {
      onClose();
      return;
    }

    setIsCancelling(true);
    setError(undefined);

    try {
      await instanceRecoveryApi.cancelRecovery(recovery.id);
      if (isCurrent()) onClose();
    } catch (failure: unknown) {
      if (isCurrent()) setError(failure);
    } finally {
      if (isCurrent()) setIsCancelling(false);
    }
  };

  const apply = async (code?: string) => {
    if (!recovery || applyMutation.isPending) return;

    setApplyError(undefined);

    try {
      const appliedRecovery = await applyMutation.mutateAsync({ recoveryId: recovery.id, code });
      if (!isCurrent()) return;

      expectRestoredSignIn();
      setArchive(null);
      setMasterKey('');
      setRecovery(appliedRecovery);
    } catch (failure: unknown) {
      if (isCurrent()) setApplyError(failure);
    } finally {
      if (isCurrent()) applyMutation.reset();
    }
  };

  const refreshRecovery = useEffectEvent(async (signal: AbortSignal) => {
    if (!recovery) return;

    try {
      const refreshedRecovery = await instanceRecoveryApi.getRecovery(recovery.id, { signal });
      if (isCurrent() && !signal.aborted) setRecovery(refreshedRecovery);
    } catch {
      // Applying restarts the application, so an unreachable server is expected until sign-in.
    }
  });

  const recoveryId = recovery?.id;
  const shouldPoll = isPreparing || recovery?.status === InstanceRecoveryStatus.APPLYING;

  useEffect(() => {
    if (!recoveryId || !shouldPoll) return;

    const controller = new AbortController();
    const interval = setInterval(
      () => void refreshRecovery(controller.signal),
      RECOVERY_POLL_INTERVAL_MS,
    );

    return () => {
      clearInterval(interval);
      controller.abort();
    };
  }, [recoveryId, shouldPoll]);

  useEffect(
    () => () => {
      uploadController.current?.abort();
    },
    [],
  );

  const failureText =
    recovery?.failure &&
    (recovery.failure === 'RELEASE_MISMATCH' && recovery.sourceRelease
      ? t('instanceRecovery.form.releaseMismatch', {
          sourceRelease: recovery.sourceRelease,
          runningRelease: recovery.runningRelease,
        })
      : t(INSTANCE_RECOVERY_FAILURE_LABEL_KEYS[recovery.failure]));

  return (
    <Dialog
      opened
      size="lg"
      title={isReady ? t('instanceRecovery.apply.title') : t('instanceRecovery.form.title')}
      closeOnClickOutside={false}
      closeOnEscape={!isClosingBlocked}
      withCloseButton={!isRestoring}
      onClose={() => void close()}
    >
      {isRestoring ? (
        <Alert color="accent" icon={<Loader size={14} role="status" />}>
          <p>{t('instanceRecovery.form.applying')}</p>
          <p className="mt-1">{t('instanceRecovery.form.signInWithRestored')}</p>
        </Alert>
      ) : isReady && recovery ? (
        <RecoveryReplacementStep
          recovery={recovery}
          isCodeRequired={isCodeRequired}
          isApplying={applyMutation.isPending}
          isCancelling={isCancelling}
          applyError={applyError}
          onApply={(code) => void apply(code)}
          onCancel={() => void close()}
        />
      ) : (
        <div className="text-sm">
          <p className="text-muted">{t('instanceRecovery.form.description')}</p>

          <div className="mt-4 space-y-4">
            <RecoveryArchivePicker
              archive={archive}
              isDisabled={isUploading || isPreparing}
              error={isArchiveMissing ? t('instanceRecovery.form.archiveMissing') : undefined}
              onChange={setArchive}
            />
            <MasterKeyTextField
              id={keyId}
              value={masterKey}
              isDisabled={isUploading || isPreparing}
              error={isKeyMissing ? t('instanceRecovery.form.masterKeyMissing') : undefined}
              onChange={setMasterKey}
            />
          </div>

          {(isUploading || isPreparing) && (
            <div className="mt-4 flex items-center gap-2" role="status">
              <Loader size="xs" />
              <span>
                {isUploading
                  ? t('instanceRecovery.form.uploading')
                  : t('instanceRecovery.form.preparing')}
              </span>
            </div>
          )}

          {failureText && (
            <Alert
              className="mt-3"
              color="danger"
              icon={<CircleAlert aria-hidden size={16} />}
              title={failureText}
            >
              {t('instanceRecovery.form.correctAndRetry')}
            </Alert>
          )}

          <div className="mt-5 flex flex-wrap justify-end gap-2 border-t border-border pt-4">
            <Button loading={isCancelling} onClick={() => void close()}>
              {t('common.actions.cancel')}
            </Button>
            {!isUploading && !isPreparing && (
              <Button variant="filled" onClick={() => void upload()}>
                {t('instanceRecovery.form.upload')}
              </Button>
            )}
          </div>
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
    </Dialog>
  );
}
