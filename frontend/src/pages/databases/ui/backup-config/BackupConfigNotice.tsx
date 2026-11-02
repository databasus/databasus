import { Alert, Button } from 'antd';
import { useTranslation } from 'react-i18next';

import type { BackupConfig } from '../../../../entities/backup-configs';
import { translateApiError } from '../../../../shared/i18n';

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
        <Alert
          className="mt-3"
          type="error"
          showIcon
          message={translateApiError(backupConfigError, t)}
          action={<Button onClick={onRetry}>{t('common.actions.retry')}</Button>}
        />
      )}

      {backupConfig && !backupConfig.isBackupsEnabled && (
        <div className="text-sm text-danger">{t('backups.list.scheduledBackupsDisabled')}</div>
      )}
    </>
  );
};
