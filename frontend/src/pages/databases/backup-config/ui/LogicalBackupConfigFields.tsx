import { Tooltip } from '@mantine/core';
import { Info } from 'lucide-react';
import { type ReactNode, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  BACKUP_ENCRYPTION_OPTION_LABEL_KEYS,
  BackupEncryption,
  type LogicalBackupConfigInput,
  type LogicalBackupSettings,
} from '@/entities/backup-configs';
import { type Schedule, ScheduleEditor } from '@/entities/schedules';
import { getRetentionForSchedule } from '@/pages/databases/backup-config/model/getRetentionForSchedule';
import { hasNonDefaultLogicalAdvancedValues } from '@/pages/databases/backup-config/model/hasNonDefaultLogicalAdvancedValues';
import { BackupStorageSelection } from '@/pages/databases/backup-config/ui/BackupStorageSelection';
import { LogicalAdvancedFields } from '@/pages/databases/backup-config/ui/LogicalAdvancedFields';
import { LogicalNotificationTriggerFields } from '@/pages/databases/backup-config/ui/LogicalNotificationTriggerFields';
import { LogicalRetentionFields } from '@/pages/databases/backup-config/ui/LogicalRetentionFields';
import { WrappingSelect, AdvancedSettingsToggleComponent } from '@/shared/ui';

interface Props {
  config: LogicalBackupConfigInput;
  onChange: (config: LogicalBackupConfigInput) => void;
  workspaceId: string;
  canClearStorage: boolean;
  storageCleanupNotice?: ReactNode;
}

export const LogicalBackupConfigFields = ({
  config,
  onChange,
  workspaceId,
  canClearStorage,
  storageCleanupNotice,
}: Props) => {
  const { t } = useTranslation();
  const [isShowAdvanced, setIsShowAdvanced] = useState(() =>
    hasNonDefaultLogicalAdvancedValues(config.settings),
  );
  const advancedRegionId = useId();

  const { settings } = config;

  const updateSettings = (patch: Partial<LogicalBackupSettings>) =>
    onChange({ ...config, settings: { ...settings, ...patch } });

  const changeSchedule = (schedule: Schedule) =>
    updateSettings({ schedule, retention: getRetentionForSchedule(settings, schedule) });

  const encryptionOptions = Object.values(BackupEncryption).map((encryption) => ({
    value: encryption,
    label: t(BACKUP_ENCRYPTION_OPTION_LABEL_KEYS[encryption]),
  }));

  return (
    <>
      {config.isBackupsEnabled && (
        <>
          <ScheduleEditor
            label={t('backups.logical.config.interval')}
            schedule={settings.schedule}
            onChange={changeSchedule}
          />

          <div className="mb-3" />
        </>
      )}

      <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
          {t('backups.config.storage')}
        </div>
        <div className="flex w-full min-w-0 items-center">
          <BackupStorageSelection
            workspaceId={workspaceId}
            selectedStorageId={config.storageId ?? undefined}
            canClear={canClearStorage}
            onSelect={(storageId) => onChange({ ...config, storageId: storageId ?? null })}
          />
        </div>
      </div>

      {storageCleanupNotice}

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
          {t('backups.config.encryption')}
        </div>
        <div className="flex w-full min-w-0 items-center">
          <WrappingSelect
            value={config.encryption}
            onChange={(encryption) => onChange({ ...config, encryption })}
            size="xs"
            className="w-full max-w-[200px]"
            data={encryptionOptions}
            aria-label={t('backups.config.encryption')}
          />

          <Tooltip className="cursor-pointer" label={t('backups.logical.config.encryptionTooltip')}>
            <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
          </Tooltip>
        </div>
      </div>

      <LogicalRetentionFields
        settings={settings}
        onChange={(retention) => updateSettings({ retention })}
      />

      {config.isBackupsEnabled && (
        <>
          <LogicalNotificationTriggerFields
            triggers={settings.notificationTriggers}
            onChange={(notificationTriggers) => updateSettings({ notificationTriggers })}
          />

          <AdvancedSettingsToggleComponent
            label={t('backups.logical.config.advancedSettings')}
            isShowAdvanced={isShowAdvanced}
            onToggle={() => setIsShowAdvanced(!isShowAdvanced)}
            regionId={advancedRegionId}
          />

          {isShowAdvanced && (
            <div id={advancedRegionId}>
              <LogicalAdvancedFields settings={settings} onChange={updateSettings} />
            </div>
          )}
        </>
      )}
    </>
  );
};
