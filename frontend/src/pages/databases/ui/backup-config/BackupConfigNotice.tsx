import { useTranslation } from 'react-i18next';

import type { BackupConfig } from '../../../../entities/backup-configs';
import { LoadFailure } from '../../../../shared/ui';

interface Props {
  backupConfig?: BackupConfig;
  backupConfigError: unknown;
  onRetry: () => void;
}

export const BackupConfigNotice = ({ backupConfig, backupConfigError, onRetry }: Props) => {
  const { t } = useTranslation();

  return (
    <>
      {backupConfigError !== undefined && (
        <LoadFailure className="mt-3" error={backupConfigError} onRetry={onRetry} />
      )}

      {backupConfig && !backupConfig.isBackupsEnabled && (
        <div className="text-sm text-danger">{t('backups.list.scheduledBackupsDisabled')}</div>
      )}
    </>
  );
};
