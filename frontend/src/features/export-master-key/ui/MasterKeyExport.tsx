import { Alert, Button, Loader } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type MasterKeyFile, masterKeyApi } from '@/entities/instance-backups';
import { MasterKeyCodeConfirmation } from '@/features/export-master-key/ui/MasterKeyCodeConfirmation';
import { MasterKeySaveOptions } from '@/features/export-master-key/ui/MasterKeySaveOptions';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';

interface Props {
  isCodeRequired: boolean;
  onKeySaved?: () => void;
}

// Without two-factor sign-in the key is read as soon as this mounts, so callers mount it only when
// root admin asks for the key. Copying or downloading reports onKeySaved; acknowledging is the
// caller's decision.
export function MasterKeyExport({ isCodeRequired, onKeySaved }: Props) {
  const { t } = useTranslation();
  const isCurrent = useSessionGuard();
  const isKeyRequestedRef = useRef(false);
  const [keyFile, setKeyFile] = useState<MasterKeyFile>();
  const [keyError, setKeyError] = useState<unknown>();

  const keyMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof masterKeyApi.getKeyFile>) =>
      masterKeyApi.getKeyFile(...parameters),
  });

  const readKey = async (code?: string) => {
    if (keyMutation.isPending) return;

    setKeyError(undefined);

    try {
      const readKeyFile = await keyMutation.mutateAsync([code]);
      if (isCurrent()) setKeyFile(readKeyFile);
    } catch (failure: unknown) {
      if (isCurrent()) setKeyError(failure);
    } finally {
      if (isCurrent()) keyMutation.reset();
    }
  };

  const readKeyOnMount = useEffectEvent(() => void readKey());

  useEffect(() => {
    if (isCodeRequired || isKeyRequestedRef.current) return;

    isKeyRequestedRef.current = true;
    readKeyOnMount();
  }, [isCodeRequired]);

  if (keyFile) return <MasterKeySaveOptions keyFile={keyFile} onKeySaved={onKeySaved} />;

  return (
    <div>
      {isCodeRequired && (
        <MasterKeyCodeConfirmation
          isConfirming={keyMutation.isPending}
          onConfirm={(code) => void readKey(code)}
        />
      )}

      {!isCodeRequired && keyMutation.isPending && (
        <div className="flex items-center gap-2 text-sm text-muted">
          <Loader size="xs" />
          {t('masterKey.loading')}
        </div>
      )}

      {keyError !== undefined && (
        <Alert
          className="mt-3"
          color="danger"
          icon={<CircleAlert aria-hidden size={16} />}
          title={translateApiError(keyError, t)}
        >
          {!isCodeRequired && (
            <Button className="mt-2" size="xs" variant="default" onClick={() => void readKey()}>
              {t('common.actions.retry')}
            </Button>
          )}
        </Alert>
      )}
    </div>
  );
}
