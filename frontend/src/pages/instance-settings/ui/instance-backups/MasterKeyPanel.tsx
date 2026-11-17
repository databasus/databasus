import { Button } from '@mantine/core';
import { KeyRound } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { InstanceBackupStatus } from '@/entities/instance-backups';
import { MasterKeyExport, MasterKeyHostInstructions } from '@/features/export-master-key';

interface Props {
  status: InstanceBackupStatus;
}

// The key is read only after root admin asks for it, so opening the settings creates no export.
export function MasterKeyPanel({ status }: Props) {
  const { t } = useTranslation();
  const [isKeyRequested, setIsKeyRequested] = useState(false);

  return (
    <div className="mt-4">
      <h3 className="text-base font-semibold">{t('masterKey.title')}</h3>
      <p className={`mt-1 ${status.isMasterKeyAcknowledged ? 'text-success' : 'text-warning'}`}>
        {status.isMasterKeyAcknowledged
          ? t('masterKey.statusSaved')
          : t('masterKey.statusNotSaved')}
      </p>
      <div className="mt-2 rounded border border-border p-3">
        <p className="text-sm">{t('masterKey.description')}</p>
        <div className="mt-3">
          {isKeyRequested ? (
            <MasterKeyExport
              isCodeRequired={status.isMasterKeyExportCodeRequired}
              masterKeyPath={status.masterKeyPath}
            />
          ) : (
            <Button
              size="xs"
              variant="default"
              leftSection={<KeyRound aria-hidden size={14} />}
              onClick={() => setIsKeyRequested(true)}
            >
              {t('masterKey.saveAgain')}
            </Button>
          )}
        </div>
        <div className="mt-4">
          <MasterKeyHostInstructions masterKeyPath={status.masterKeyPath} />
        </div>
      </div>
    </div>
  );
}
