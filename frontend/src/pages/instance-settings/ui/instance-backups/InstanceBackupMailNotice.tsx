import { Alert, Button } from '@mantine/core';
import { Mail } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { InstanceBackupStatus } from '@/entities/instance-backups';

interface Props {
  status: InstanceBackupStatus;
  onConfigureSmtp: () => void;
}

export function InstanceBackupMailNotice({ status, onConfigureSmtp }: Props) {
  const { t } = useTranslation();

  if (!status.isEmailConfigured) {
    return (
      <Alert
        className="mt-3"
        color="warning"
        icon={<Mail aria-hidden size={16} />}
        title={t('instanceBackups.smtp.missingTitle')}
      >
        <p>{t('instanceBackups.smtp.missingDescription')}</p>
        <Button className="mt-2" size="xs" variant="outline" onClick={onConfigureSmtp}>
          {t('instanceBackups.smtp.configure')}
        </Button>
      </Alert>
    );
  }

  if (!status.isRootEmailUsable) {
    return (
      <Alert
        className="mt-3"
        color="warning"
        icon={<Mail aria-hidden size={16} />}
        title={t('instanceBackups.smtp.addressMissing')}
      />
    );
  }

  return null;
}
