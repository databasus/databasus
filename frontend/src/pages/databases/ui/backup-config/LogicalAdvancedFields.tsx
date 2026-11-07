import { NumberInput, Switch, Tooltip } from '@mantine/core';
import { Info } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { LogicalBackupSettings } from '@/entities/backup-configs';

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
            size="xs"
            checked={settings.isRetryIfFailed}
            aria-label={t('backups.logical.config.retryIfFailed')}
            onChange={(event) => onChange({ isRetryIfFailed: event.currentTarget.checked })}
          />

          <Tooltip
            className="cursor-pointer"
            label={t('backups.logical.config.retryIfFailedTooltip')}
          >
            <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
          </Tooltip>
        </div>
      </div>

      {settings.isRetryIfFailed && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
            {t('backups.logical.config.maxFailedTries')}
          </div>
          <div className="flex items-center">
            <NumberInput
              min={1}
              max={10}
              value={settings.maxFailedTriesCount}
              aria-label={t('backups.logical.config.maxFailedTries')}
              onChange={(value) =>
                onChange({ maxFailedTriesCount: typeof value === 'number' ? value : 1 })
              }
              size="xs"
              className="w-full max-w-[75px] grow"
              role="spinbutton"
              aria-valuenow={
                typeof settings.maxFailedTriesCount === 'number'
                  ? settings.maxFailedTriesCount
                  : undefined
              }
              aria-valuemin={1}
              aria-valuemax={10}
            />

            <Tooltip
              className="cursor-pointer"
              label={t('backups.logical.config.maxFailedTriesTooltip')}
            >
              <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
            </Tooltip>
          </div>
        </div>
      )}
    </>
  );
};
