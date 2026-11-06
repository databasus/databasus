import { InfoCircleOutlined } from '@ant-design/icons';
import { Select, Tooltip } from 'antd';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { BACKUP_ENCRYPTION_OPTION_LABEL_KEYS, BackupEncryption } from '@/entities/backup-configs';
import type { PhysicalDatabaseBackupType } from '@/entities/databases';
import { ScheduleEditor } from '@/entities/schedules';
import { hasIncrementalBackups } from '@/pages/databases/model/physical-backups/hasIncrementalBackups';
import type { PhysicalBackupConfigDraft } from '@/pages/databases/model/physical-backups/PhysicalBackupConfigDraft';
import { PhysicalNotificationTriggerFields } from '@/pages/databases/ui/backup-config/PhysicalNotificationTriggerFields';
import { PhysicalRetentionFields } from '@/pages/databases/ui/backup-config/PhysicalRetentionFields';
import { BackupStorageSelection } from '@/pages/databases/ui/BackupStorageSelection';

type PhysicalDraftSettings = PhysicalBackupConfigDraft['settings'];

interface Props {
  draft: PhysicalBackupConfigDraft;
  onChange: (draft: PhysicalBackupConfigDraft) => void;
  workspaceId: string;
  backupType: PhysicalDatabaseBackupType;
  canClearStorage: boolean;
  storageCleanupNotice?: ReactNode;
}

export const PhysicalBackupConfigFields = ({
  draft,
  onChange,
  workspaceId,
  backupType,
  canClearStorage,
  storageCleanupNotice,
}: Props) => {
  const { t } = useTranslation();

  const { settings } = draft;

  const updateSettings = (patch: Partial<PhysicalDraftSettings>) =>
    onChange({ ...draft, settings: { ...settings, ...patch } });

  const encryptionOptions = Object.values(BackupEncryption).map((encryption) => ({
    value: encryption,
    label: t(BACKUP_ENCRYPTION_OPTION_LABEL_KEYS[encryption]),
  }));

  return (
    <>
      {draft.isBackupsEnabled && (
        <>
          <ScheduleEditor
            label={t('backups.physical.config.fullBackupCadence')}
            schedule={settings.fullBackupSchedule}
            onChange={(fullBackupSchedule) => updateSettings({ fullBackupSchedule })}
          />

          {hasIncrementalBackups(backupType) && (
            <>
              <ScheduleEditor
                label={t('backups.physical.config.incrementalBackupCadence')}
                schedule={settings.incrementalBackupSchedule}
                onChange={(incrementalBackupSchedule) =>
                  updateSettings({ incrementalBackupSchedule })
                }
              />
              <div className="mt-1 mb-3 flex w-full flex-col items-start sm:flex-row sm:items-center">
                <div className="min-w-[150px] pr-2" />
                <div className="max-w-[320px] text-xs text-muted">
                  {t('backups.physical.config.incrementalMoreFrequent')}
                </div>
              </div>
            </>
          )}

          <div className="mb-3" />
        </>
      )}

      <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('backups.config.storage')}</div>
        <div className="flex w-full items-center">
          <BackupStorageSelection
            workspaceId={workspaceId}
            selectedStorageId={draft.storageId ?? undefined}
            canClear={canClearStorage}
            onSelect={(storageId) => onChange({ ...draft, storageId: storageId ?? null })}
          />
        </div>
      </div>

      {storageCleanupNotice}

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('backups.config.encryption')}</div>
        <div className="flex w-full items-center">
          <Select
            value={draft.encryption}
            onChange={(encryption) => onChange({ ...draft, encryption })}
            size="small"
            className="min-w-0 grow"
            options={encryptionOptions}
            aria-label={t('backups.config.encryption')}
          />

          <Tooltip
            className="cursor-pointer"
            title={t('backups.physical.config.encryptionTooltip')}
          >
            <InfoCircleOutlined className="ml-2 text-muted" />
          </Tooltip>
        </div>
      </div>

      <PhysicalRetentionFields
        retention={settings.retention}
        backupType={backupType}
        onChange={(retention) => updateSettings({ retention })}
      />

      {draft.isBackupsEnabled && (
        <PhysicalNotificationTriggerFields
          triggers={settings.notificationTriggers}
          backupType={backupType}
          onChange={(notificationTriggers) => updateSettings({ notificationTriggers })}
        />
      )}
    </>
  );
};
