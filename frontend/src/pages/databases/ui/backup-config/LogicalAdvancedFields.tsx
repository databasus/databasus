import { InfoCircleOutlined } from '@ant-design/icons';
import { InputNumber, Switch, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';

import type { LogicalBackupSettings } from '../../../../entities/backup-configs';

interface Props {
  settings: LogicalBackupSettings;
  onChange: (
    patch: Partial<Pick<LogicalBackupSettings, 'isRetryIfFailed' | 'maxFailedTriesCount'>>,
  ) => void;
}

export const LogicalAdvancedFields = ({ settings, onChange }: Props) => {
  const { t } = useTranslation();

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
          {t('backups.logical.config.retryIfFailed')}
        </div>
        <div className="flex items-center">
          <Switch
            size="small"
            checked={settings.isRetryIfFailed}
            aria-label={t('backups.logical.config.retryIfFailed')}
            onChange={(isRetryIfFailed) => onChange({ isRetryIfFailed })}
          />

          <Tooltip
            className="cursor-pointer"
            title={t('backups.logical.config.retryIfFailedTooltip')}
          >
            <InfoCircleOutlined className="ml-2 text-muted" />
          </Tooltip>
        </div>
      </div>

      {settings.isRetryIfFailed && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
            {t('backups.logical.config.maxFailedTries')}
          </div>
          <div className="flex items-center">
            <InputNumber
              min={1}
              max={10}
              value={settings.maxFailedTriesCount}
              aria-label={t('backups.logical.config.maxFailedTries')}
              onChange={(value) => onChange({ maxFailedTriesCount: value || 1 })}
              size="small"
              className="w-full max-w-[75px] grow"
            />

            <Tooltip
              className="cursor-pointer"
              title={t('backups.logical.config.maxFailedTriesTooltip')}
            >
              <InfoCircleOutlined className="ml-2 text-muted" />
            </Tooltip>
          </div>
        </div>
      )}
    </>
  );
};
