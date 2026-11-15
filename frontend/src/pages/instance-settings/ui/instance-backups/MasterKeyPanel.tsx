import { Button } from '@mantine/core';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { InstanceBackupStatus } from '@/entities/instance-backups';
import { MasterKeyExport } from '@/features/export-master-key';

interface Props {
  status: InstanceBackupStatus;
  onAcknowledged: () => void;
}

export function MasterKeyPanel({ status, onAcknowledged }: Props) {
  const { t } = useTranslation();
  const [isExportOpen, setIsExportOpen] = useState(false);

  return (
    <div className="mt-4">
      <h3 className="text-base font-semibold">{t('masterKey.title')}</h3>
      <p className={`mt-1 ${status.isMasterKeyAcknowledged ? 'text-success' : 'text-warning'}`}>
        {status.isMasterKeyAcknowledged
          ? t('masterKey.statusSaved')
          : t('masterKey.statusNotSaved')}
      </p>
      {isExportOpen ? (
        <div className="mt-2 rounded border border-border p-3">
          <MasterKeyExport
            identityProofMethods={status.identityProofMethods}
            masterKeyPath={status.masterKeyPath}
            isAcknowledged={status.isMasterKeyAcknowledged}
            onAcknowledged={onAcknowledged}
          />
        </div>
      ) : (
        <Button
          className="mt-2"
          size="xs"
          variant="outline"
          aria-expanded={false}
          onClick={() => setIsExportOpen(true)}
        >
          {t('masterKey.onboarding.title')}
        </Button>
      )}
    </div>
  );
}
