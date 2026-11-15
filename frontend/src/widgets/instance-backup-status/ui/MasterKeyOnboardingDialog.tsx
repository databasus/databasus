import { Button } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import type { InstanceBackupStatus } from '@/entities/instance-backups';
import { MasterKeyExport } from '@/features/export-master-key';
import { Dialog } from '@/shared/ui';

interface Props {
  status: InstanceBackupStatus;
  onAcknowledged: () => void;
  onDefer: () => void;
}

export function MasterKeyOnboardingDialog({ status, onAcknowledged, onDefer }: Props) {
  const { t } = useTranslation();

  return (
    <Dialog
      title={t('masterKey.onboarding.title')}
      opened
      size="lg"
      closeOnClickOutside={false}
      onClose={onDefer}
    >
      <p className="mb-3 text-sm">{t('masterKey.onboarding.description')}</p>
      <MasterKeyExport
        identityProofMethods={status.identityProofMethods}
        masterKeyPath={status.masterKeyPath}
        isAcknowledged={status.isMasterKeyAcknowledged}
        onAcknowledged={onAcknowledged}
      />
      <div className="mt-5 flex flex-wrap justify-end gap-2">
        <Button variant="default" onClick={onDefer}>
          {t('masterKey.onboarding.later')}
        </Button>
      </div>
    </Dialog>
  );
}
