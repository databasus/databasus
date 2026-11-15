import { Alert, Button, Checkbox, FileInput, Loader } from '@mantine/core';
import { CircleAlert, ShieldAlert } from 'lucide-react';
import { useEffect, useEffectEvent, useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  INSTANCE_RECOVERY_FAILURE_LABEL_KEYS,
  type IdentityProofMethod,
  type InstanceRecovery,
  InstanceRecoveryStatus,
  expectRestoredSignIn,
  instanceRecoveryApi,
  isInsecureTransfer,
} from '@/entities/instance-backups';
import { ApplyRecoveryDialog } from '@/pages/instance-settings/ui/instance-backups/recovery/ApplyRecoveryDialog';
import { MasterKeyTextField } from '@/pages/instance-settings/ui/instance-backups/recovery/MasterKeyTextField';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { formatInstant } from '@/shared/time';

const RECOVERY_POLL_INTERVAL_MS = 2_000;

interface Props {
  identityProofMethods: IdentityProofMethod[];
}

// The archive handle and the entered key live only in this component's state. Its owner remounts
// it on a session change, so neither outlives the account that entered them.
export function InstanceRecoveryImport({ identityProofMethods }: Props) {
  const { t } = useTranslation();
  const isCurrent = useSessionGuard();
  const archiveId = useId();
  const keyId = useId();
  const isInsecure = isInsecureTransfer();
  const [archive, setArchive] = useState<File | null>(null);
  const [masterKey, setMasterKey] = useState('');
  const [isInsecureTransferAccepted, setIsInsecureTransferAccepted] = useState(false);
  const [isSubmitted, setIsSubmitted] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [recovery, setRecovery] = useState<InstanceRecovery>();
  const [error, setError] = useState<unknown>();
  const [isCancelling, setIsCancelling] = useState(false);
  const [isApplyDialogOpen, setIsApplyDialogOpen] = useState(false);
  const uploadController = useRef<AbortController | null>(null);

  const isArchiveMissing = isSubmitted && !archive;
  const isKeyMissing = isSubmitted && masterKey.length === 0;
  const isPreparing = recovery?.status === InstanceRecoveryStatus.VALIDATING;
  const isApplying = recovery?.status === InstanceRecoveryStatus.APPLYING;

  const clearInputs = () => {
    setArchive(null);
    setMasterKey('');
    setIsSubmitted(false);
    setIsInsecureTransferAccepted(false);
  };

  const upload = async () => {
    setIsSubmitted(true);
    if (!archive || masterKey.length === 0 || isUploading) return;
    if (isInsecure && !isInsecureTransferAccepted) return;

    const controller = new AbortController();
    uploadController.current = controller;
    setError(undefined);
    setRecovery(undefined);
    setIsUploading(true);

    try {
      const createdRecovery = await instanceRecoveryApi.createRecovery(
        { archive, masterKey, isInsecureTransferAcknowledged: isInsecure },
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

  const cancel = async () => {
    if (uploadController.current) {
      uploadController.current.abort();
      uploadController.current = null;
      setIsUploading(false);
      clearInputs();
      return;
    }
    if (!recovery || isCancelling) return;

    setIsCancelling(true);
    setError(undefined);

    try {
      await instanceRecoveryApi.cancelRecovery(recovery.id);
      if (!isCurrent()) return;

      setRecovery(undefined);
      clearInputs();
    } catch (failure: unknown) {
      if (isCurrent()) setError(failure);
    } finally {
      if (isCurrent()) setIsCancelling(false);
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
  const shouldPoll = isPreparing || isApplying;

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

  const accept = (appliedRecovery: InstanceRecovery) => {
    expectRestoredSignIn();
    setIsApplyDialogOpen(false);
    clearInputs();
    setRecovery(appliedRecovery);
  };

  const failureText =
    recovery?.failure &&
    (recovery.failure === 'RELEASE_MISMATCH' && recovery.sourceRelease
      ? t('instanceRecovery.form.releaseMismatch', {
          sourceRelease: recovery.sourceRelease,
          runningRelease: recovery.runningRelease,
        })
      : t(INSTANCE_RECOVERY_FAILURE_LABEL_KEYS[recovery.failure]));

  return (
    <div className="mt-6">
      <h3 className="text-base font-semibold">{t('instanceRecovery.form.title')}</h3>
      <p className="mt-1 text-muted">{t('instanceRecovery.form.description')}</p>
      <p className="mt-1 text-muted">{t('instanceRecovery.form.sourceKey')}</p>

      {isApplying || recovery?.status === InstanceRecoveryStatus.PAUSED ? (
        <Alert className="mt-3" color="accent" icon={<Loader size={14} role="status" />}>
          <p>{t('instanceRecovery.form.applying')}</p>
          <p className="mt-1">{t('instanceRecovery.form.signInWithRestored')}</p>
        </Alert>
      ) : recovery?.status === InstanceRecoveryStatus.READY ? (
        <div className="mt-3 rounded border border-border p-3">
          <p className="font-medium">{t('instanceRecovery.form.ready')}</p>
          {recovery.snapshotAt && recovery.sourceRelease && (
            <p className="mt-1">
              {t('instanceRecovery.form.snapshot', {
                time: formatInstant(recovery.snapshotAt),
                version: recovery.sourceRelease,
              })}
            </p>
          )}
          <p className="mt-2 text-warning">{t('instanceRecovery.form.replacementWarning')}</p>
          <div className="mt-3 flex flex-wrap gap-2">
            <Button
              size="xs"
              variant="filled"
              color="danger"
              onClick={() => setIsApplyDialogOpen(true)}
            >
              {t('instanceRecovery.form.replace')}
            </Button>
            <Button
              size="xs"
              variant="default"
              loading={isCancelling}
              onClick={() => void cancel()}
            >
              {t('common.actions.cancel')}
            </Button>
          </div>
        </div>
      ) : (
        <div className="mt-3 max-w-md space-y-3">
          <FileInput
            id={archiveId}
            size="xs"
            label={t('instanceRecovery.form.archive')}
            placeholder={t('instanceRecovery.form.archivePlaceholder')}
            value={archive}
            clearable
            disabled={isUploading || isPreparing}
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

          {isInsecure && (
            <Alert
              color="warning"
              icon={<ShieldAlert aria-hidden size={16} />}
              title={t('instanceRecovery.form.insecureTitle')}
            >
              <p>{t('instanceRecovery.form.insecureDescription')}</p>
              <Checkbox
                className="mt-2"
                label={t('instanceRecovery.form.insecureAccept')}
                checked={isInsecureTransferAccepted}
                disabled={isUploading || isPreparing}
                onChange={(event) => setIsInsecureTransferAccepted(event.currentTarget.checked)}
              />
            </Alert>
          )}

          {isUploading || isPreparing ? (
            <div className="flex flex-wrap items-center gap-2" role="status">
              <Loader size="xs" />
              <span>
                {isUploading
                  ? t('instanceRecovery.form.uploading')
                  : t('instanceRecovery.form.preparing')}
              </span>
              <Button
                size="xs"
                variant="default"
                loading={isCancelling}
                onClick={() => void cancel()}
              >
                {t('common.actions.cancel')}
              </Button>
            </div>
          ) : (
            <Button
              size="xs"
              variant="filled"
              disabled={isInsecure && !isInsecureTransferAccepted}
              onClick={() => void upload()}
            >
              {t('instanceRecovery.form.upload')}
            </Button>
          )}
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
      {error !== undefined && (
        <Alert
          className="mt-3"
          color="danger"
          icon={<CircleAlert aria-hidden size={16} />}
          title={translateApiError(error, t)}
        />
      )}

      {isApplyDialogOpen && recovery && (
        <ApplyRecoveryDialog
          recovery={recovery}
          identityProofMethods={identityProofMethods}
          onApplied={accept}
          onClose={() => setIsApplyDialogOpen(false)}
        />
      )}
    </div>
  );
}
