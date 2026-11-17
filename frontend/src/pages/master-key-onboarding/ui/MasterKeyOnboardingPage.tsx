import { Alert, Button } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type InstanceBackupStatus, masterKeyApi } from '@/entities/instance-backups';
import { MasterKeyExport } from '@/features/export-master-key';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useScreenHeight } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';
import { BrandNavbar } from '@/shared/ui';

interface Props {
  status: InstanceBackupStatus;
  onAcknowledged: () => Promise<unknown>;
}

// The application opens only after root admin confirms saving the key, and this screen offers no
// way around that.
export function MasterKeyOnboardingPage({ status, onAcknowledged }: Props) {
  const { t } = useTranslation();
  const screenHeight = useScreenHeight();
  const isCurrent = useSessionGuard();
  const [isKeySaved, setIsKeySaved] = useState(false);
  const [acknowledgeError, setAcknowledgeError] = useState<unknown>();

  const acknowledgeMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof masterKeyApi.acknowledge>) =>
      masterKeyApi.acknowledge(...parameters),
  });

  const acknowledge = async () => {
    if (acknowledgeMutation.isPending) return;

    setAcknowledgeError(undefined);

    try {
      await acknowledgeMutation.mutateAsync([]);
      if (isCurrent()) await onAcknowledged();
    } catch (failure: unknown) {
      if (isCurrent()) setAcknowledgeError(failure);
    } finally {
      if (isCurrent()) acknowledgeMutation.reset();
    }
  };

  return (
    <div
      className="flex min-h-0 min-w-0 flex-col overflow-y-auto bg-surface text-content"
      style={{ height: screenHeight }}
    >
      <BrandNavbar />

      <main className="mt-8 flex flex-col items-center px-4 pb-8 sm:mt-[10vh]">
        <h1 className="text-center text-xl font-semibold">{t('masterKey.onboarding.title')}</h1>
        <p className="mt-3 max-w-[560px] text-center text-sm text-balance text-muted">
          {t('masterKey.description')}
        </p>

        <div className="mt-6 w-full max-w-[460px]">
          <MasterKeyExport
            isCodeRequired={status.isMasterKeyExportCodeRequired}
            masterKeyPath={status.masterKeyPath}
            onKeySaved={() => setIsKeySaved(true)}
          />

          {acknowledgeError !== undefined && (
            <Alert
              className="mt-4"
              color="danger"
              icon={<CircleAlert aria-hidden size={16} />}
              title={translateApiError(acknowledgeError, t)}
            />
          )}

          {/* A locked confirmation keeps the primary color, so the way forward stays obvious. */}
          <Button
            className="mt-6 data-disabled:opacity-50"
            style={{
              '--mantine-color-disabled': 'var(--button-bg)',
              '--mantine-color-disabled-color': 'var(--button-color)',
            }}
            variant="filled"
            fullWidth
            disabled={!isKeySaved}
            loading={acknowledgeMutation.isPending}
            onClick={() => void acknowledge()}
          >
            {t('masterKey.onboarding.close')}
          </Button>
        </div>
      </main>
    </div>
  );
}
