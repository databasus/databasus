import { Tooltip } from '@mantine/core';
import { Info } from 'lucide-react';
import { type JSX } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type PhysicalBackupConfigInput,
  type BackupCleanup,
  BACKUP_ENCRYPTION_LABEL_KEYS,
} from '@/entities/backup-configs';
import type { Database } from '@/entities/databases';
import {
  SCHEDULE_TYPE_LABEL_KEYS,
  type Schedule,
  ScheduleType,
  WEEKDAY_LABEL_KEYS,
  getLocalScheduleParts,
} from '@/entities/schedules';
import { getStorageLogoFromType, storageApi } from '@/entities/storages';
import { FullBackupsRetentionPolicy } from '@/pages/databases/backup-config/model/FullBackupsRetentionPolicy';
import { type FullBackupsRetention } from '@/pages/databases/backup-config/model/PhysicalBackupConfigDraft';
import { PHYSICAL_NOTIFICATION_TRIGGER_LABEL_KEYS } from '@/pages/databases/backup-config/model/PhysicalNotificationTrigger';
import {
  PhysicalRetentionPolicy,
  PHYSICAL_RETENTION_POLICY_LABEL_KEYS,
} from '@/pages/databases/backup-config/model/PhysicalRetentionPolicy';
import { StorageCleanupNotice } from '@/pages/databases/backup-config/ui/StorageCleanupNotice';
import { useSessionQuery } from '@/shared/api';
import { type LocalizedText, translateLocalizedText } from '@/shared/i18n';
import { getUserClockFormat } from '@/shared/time';
import { LoadFailure } from '@/shared/ui';

interface Props {
  database: Database;
  cleanup?: BackupCleanup;
  cleanupError?: unknown;
  onRetryCleanup?: () => void;
  backupConfig: PhysicalBackupConfigInput;
}

const getGfsRetentionTexts = (retention: FullBackupsRetention): LocalizedText[] => {
  const texts: LocalizedText[] = [];

  if (retention.gfsHours > 0) {
    texts.push({ key: 'backups.config.gfs.hourly', params: { count: retention.gfsHours } });
  }
  if (retention.gfsDays > 0) {
    texts.push({ key: 'backups.config.gfs.daily', params: { count: retention.gfsDays } });
  }
  if (retention.gfsWeeks > 0) {
    texts.push({ key: 'backups.config.gfs.weekly', params: { count: retention.gfsWeeks } });
  }
  if (retention.gfsMonths > 0) {
    texts.push({ key: 'backups.config.gfs.monthly', params: { count: retention.gfsMonths } });
  }
  if (retention.gfsYears > 0) {
    texts.push({ key: 'backups.config.gfs.yearly', params: { count: retention.gfsYears } });
  }

  return texts;
};

export const ShowPhysicalBackupConfigComponent = ({
  backupConfig,
  cleanup,
  cleanupError,
  onRetryCleanup,
}: Props): JSX.Element => {
  const { t } = useTranslation();
  const listSeparator = t('common.listSeparator');

  const describeGfsRetention = (retention: FullBackupsRetention): string => {
    const texts = getGfsRetentionTexts(retention);
    if (texts.length === 0) return t('backups.config.gfs.notConfigured');

    return texts.map((text) => translateLocalizedText(text, t)).join(listSeparator);
  };

  // Time of day, weekday and day of month are stored in UTC and shown in local time.
  const renderSchedule = (label: string, schedule: Schedule | null): JSX.Element | null => {
    if (!schedule) return null;

    const { localTime, weekday, dayOfMonth } = getLocalScheduleParts(schedule);
    const formattedTime = localTime ? localTime.format(getUserClockFormat().format) : '';

    return (
      <>
        <div className="mt-4 mb-1 flex w-full items-center">
          <div className="max-w-[180px] min-w-[180px] pr-2">{label}</div>
          <div>{t(SCHEDULE_TYPE_LABEL_KEYS[schedule.type])}</div>
        </div>

        {schedule.type === ScheduleType.WEEKLY && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('schedules.weekday')}</div>
            <div>{weekday ? t(WEEKDAY_LABEL_KEYS[weekday]) : ''}</div>
          </div>
        )}

        {schedule.type === ScheduleType.MONTHLY && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('schedules.dayOfMonth')}</div>
            <div>{dayOfMonth || ''}</div>
          </div>
        )}

        {schedule.type === ScheduleType.CRON && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('schedules.cronExpression')}</div>
            <code className="rounded bg-code px-2 py-0.5 text-sm">
              {schedule.cronExpression || ''}
            </code>
          </div>
        )}

        {schedule.type !== ScheduleType.HOURLY && schedule.type !== ScheduleType.CRON && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('schedules.timeOfDay')}</div>
            <div>{formattedTime}</div>
          </div>
        )}
      </>
    );
  };

  const { storageId } = backupConfig;

  const storageQuery = useSessionQuery({
    queryKey: ['storage', storageId],
    enabled: !!storageId,
    queryFn: (signal) => storageApi.getStorage(storageId!, { signal }),
  });
  const storage = storageQuery.data;
  const storageLoadError = storageQuery.error ?? undefined;

  const { settings } = backupConfig;
  const { retention } = settings;
  const fullBackupsRetention = retention.fullBackups;

  const isShowChainsCount =
    retention.policy === PhysicalRetentionPolicy.CHAINS ||
    retention.policy === PhysicalRetentionPolicy.CHAINS_AND_FULL_BACKUPS;

  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[180px] pr-2">{t('backups.config.backupsEnabled')}</div>
        <div className={backupConfig.isBackupsEnabled ? '' : 'font-bold text-danger'}>
          {backupConfig.isBackupsEnabled ? t('common.answers.yes') : t('common.answers.no')}
        </div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[180px] pr-2">{t('backups.config.storage')}</div>
        <div className="flex items-center">
          <div>{storage?.name || '-'}</div>
          {storage && (
            <img
              src={getStorageLogoFromType(storage.settings.type)}
              alt=""
              className="ml-1 h-4 w-4"
            />
          )}
        </div>
      </div>

      {storageLoadError !== undefined && (
        <LoadFailure
          className="mb-1"
          error={storageLoadError}
          onRetry={() => void storageQuery.refetch()}
        />
      )}

      <StorageCleanupNotice cleanup={cleanup} error={cleanupError} onRetry={onRetryCleanup} />

      {backupConfig.isBackupsEnabled && (
        <>
          {renderSchedule(
            t('backups.physical.config.fullBackupCadence'),
            settings.fullBackupSchedule,
          )}
          {renderSchedule(
            t('backups.physical.config.incrementalBackupCadence'),
            settings.incrementalBackupSchedule,
          )}

          <div className="mt-4 mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.physical.config.retention')}</div>
            <div>{t(PHYSICAL_RETENTION_POLICY_LABEL_KEYS[retention.policy])}</div>
          </div>

          {isShowChainsCount && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[180px] pr-2">{t('backups.physical.config.chainsKept')}</div>
              <div>{retention.chainsCount}</div>
            </div>
          )}

          {fullBackupsRetention && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[180px] pr-2">
                {t('backups.physical.config.fullBackupsKept')}
              </div>
              <div className="flex items-center gap-1">
                {fullBackupsRetention.policy === FullBackupsRetentionPolicy.LAST_N ? (
                  <span>
                    {t('backups.physical.config.fullBackupsCountValue', {
                      count: fullBackupsRetention.count,
                    })}
                  </span>
                ) : (
                  <span className="flex items-center gap-1">
                    {describeGfsRetention(fullBackupsRetention)}
                    <Tooltip label={t('backups.physical.config.gfsTooltip')}>
                      <Info className="inline-block shrink-0 align-middle text-muted" size="1em" />
                    </Tooltip>
                  </span>
                )}
              </div>
            </div>
          )}

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.config.encryption')}</div>
            <div>{t(BACKUP_ENCRYPTION_LABEL_KEYS[backupConfig.encryption])}</div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.config.notifications')}</div>
            <div>
              {settings.notificationTriggers.length > 0
                ? settings.notificationTriggers
                    .map((trigger) => t(PHYSICAL_NOTIFICATION_TRIGGER_LABEL_KEYS[trigger]))
                    .join(listSeparator)
                : t('backups.config.noNotifications')}
            </div>
          </div>
        </>
      )}
    </div>
  );
};
