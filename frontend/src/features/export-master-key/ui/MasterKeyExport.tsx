import { Alert, Button, PasswordInput, TextInput } from '@mantine/core';
import { CircleAlert, ShieldAlert } from 'lucide-react';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type IdentityProof,
  type IdentityProofMethod,
  isInsecureTransfer,
  masterKeyApi,
} from '@/entities/instance-backups';
import { MasterKeyHostInstructions } from '@/features/export-master-key/ui/MasterKeyHostInstructions';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { followFileLink } from '@/shared/lib';
import { useFeedback } from '@/shared/lib/feedback';

interface Props {
  identityProofMethods: IdentityProofMethod[];
  masterKeyPath: string;
  isAcknowledged: boolean;
  onAcknowledged: () => void;
}

// Nothing reaches the server before the owner accepts the HTTP warning, so opening or dismissing it
// never produces key material or a usable download token.
export function MasterKeyExport({
  identityProofMethods,
  masterKeyPath,
  isAcknowledged,
  onAcknowledged,
}: Props) {
  const { t, i18n } = useTranslation();
  const notify = useFeedback();
  const isCurrent = useSessionGuard();
  const passwordId = useId();
  const codeId = useId();
  const isInsecure = isInsecureTransfer();
  const [isInsecureTransferAccepted, setIsInsecureTransferAccepted] = useState(false);
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [codeExpiresAt, setCodeExpiresAt] = useState<string>();
  const [error, setError] = useState<unknown>();
  const [isDownloaded, setIsDownloaded] = useState(false);
  const canUsePassword = identityProofMethods.includes('PASSWORD');
  const canUseEmailCode = identityProofMethods.includes('EMAIL_CODE');

  const confirmationMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof masterKeyApi.requestConfirmation>) =>
      masterKeyApi.requestConfirmation(...parameters),
  });
  const exportMutation = useSessionMutation({
    mutationFn: (proof: IdentityProof) => masterKeyApi.createExport(proof, isInsecure),
  });
  const acknowledgeMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof masterKeyApi.acknowledge>) =>
      masterKeyApi.acknowledge(...parameters),
  });

  const requestCode = async () => {
    if (confirmationMutation.isPending) return;

    setError(undefined);

    try {
      const confirmation = await confirmationMutation.mutateAsync([]);
      if (isCurrent()) setCodeExpiresAt(confirmation.expiresAt);
    } catch (failure: unknown) {
      if (isCurrent()) setError(failure);
    } finally {
      if (isCurrent()) confirmationMutation.reset();
    }
  };

  const download = async (proof: IdentityProof) => {
    if (exportMutation.isPending) return;

    setError(undefined);

    try {
      const keyExport = await exportMutation.mutateAsync(proof);
      if (!isCurrent()) return;

      followFileLink(masterKeyApi.getDownloadUrl(keyExport.token), keyExport.fileName);
      setPassword('');
      setCode('');
      setIsDownloaded(true);
    } catch (failure: unknown) {
      if (isCurrent()) setError(failure);
    } finally {
      if (isCurrent()) exportMutation.reset();
    }
  };

  const acknowledge = async () => {
    if (acknowledgeMutation.isPending) return;

    setError(undefined);

    try {
      await acknowledgeMutation.mutateAsync([true]);
      if (!isCurrent()) return;

      notify({ kind: 'success', title: i18n.t('masterKey.acknowledged') });
      onAcknowledged();
    } catch (failure: unknown) {
      if (isCurrent()) setError(failure);
    } finally {
      if (isCurrent()) acknowledgeMutation.reset();
    }
  };

  const isProofStepShown = !isInsecure || isInsecureTransferAccepted;

  return (
    <div className="text-sm">
      <p>{t('masterKey.description')}</p>

      {isInsecure && !isInsecureTransferAccepted && (
        <Alert
          className="mt-3"
          color="warning"
          icon={<ShieldAlert aria-hidden size={16} />}
          title={t('masterKey.insecure.title')}
        >
          <p>{t('masterKey.insecure.description')}</p>
          <Button
            className="mt-2 !h-auto min-h-8 max-w-full !whitespace-normal"
            size="xs"
            variant="outline"
            onClick={() => setIsInsecureTransferAccepted(true)}
          >
            {t('masterKey.insecure.continue')}
          </Button>
        </Alert>
      )}

      {isProofStepShown && (canUsePassword || canUseEmailCode) && (
        <div className="mt-3 space-y-3">
          {canUsePassword && (
            <div>
              <PasswordInput
                id={passwordId}
                size="xs"
                label={t('masterKey.proof.password')}
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.currentTarget.value)}
              />
              <Button
                className="mt-2"
                size="xs"
                variant="filled"
                disabled={!password || exportMutation.isPending}
                loading={exportMutation.isPending && !code}
                onClick={() => void download({ password })}
              >
                {t('masterKey.download')}
              </Button>
            </div>
          )}

          {canUseEmailCode && (
            <div>
              <Button
                size="xs"
                variant="outline"
                loading={confirmationMutation.isPending}
                onClick={() => void requestCode()}
              >
                {codeExpiresAt ? t('masterKey.proof.resendCode') : t('masterKey.proof.sendCode')}
              </Button>
              {codeExpiresAt && (
                <>
                  <TextInput
                    id={codeId}
                    className="mt-2"
                    size="xs"
                    label={t('masterKey.proof.code')}
                    autoComplete="one-time-code"
                    value={code}
                    onChange={(event) => setCode(event.currentTarget.value)}
                  />
                  <Button
                    className="mt-2"
                    size="xs"
                    variant="filled"
                    disabled={!code || exportMutation.isPending}
                    loading={exportMutation.isPending && !!code}
                    onClick={() => void download({ code })}
                  >
                    {t('masterKey.download')}
                  </Button>
                </>
              )}
            </div>
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

      {isDownloaded && <p className="mt-3 text-success">{t('masterKey.downloaded')}</p>}

      <div className="mt-4">
        {!canUsePassword && !canUseEmailCode && (
          <p className="mb-2 font-medium">{t('masterKey.host.onlyOption')}</p>
        )}
        <MasterKeyHostInstructions masterKeyPath={masterKeyPath} />
      </div>

      {!isAcknowledged && (
        <Button
          className="mt-4 !h-auto min-h-8 max-w-full !whitespace-normal"
          size="xs"
          variant={isDownloaded ? 'filled' : 'default'}
          loading={acknowledgeMutation.isPending}
          onClick={() => void acknowledge()}
        >
          {t('masterKey.acknowledge')}
        </Button>
      )}
    </div>
  );
}
