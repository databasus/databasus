import { InfoCircleOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type LogicalBackupConfigInput,
  type LogicalBackupSettings,
  type BackupCleanup,
  BACKUP_ENCRYPTION_LABEL_KEYS,
} from '@/entities/backup-configs';
import type { Database } from '@/entities/databases';
import {
  SCHEDULE_TYPE_LABEL_KEYS,
  ScheduleType,
  WEEKDAY_LABEL_KEYS,
  getLocalScheduleParts,
} from '@/entities/schedules';
import { type Storage, getStorageLogoFromType, storageApi } from '@/entities/storages';
import { LOGICAL_NOTIFICATION_TRIGGER_LABEL_KEYS } from '@/pages/databases/model/LogicalNotificationTriggerLabelKeys';
import { LogicalRetentionPolicy } from '@/pages/databases/model/LogicalRetentionPolicy';
import { RETENTION_PERIOD_LABEL_KEYS } from '@/pages/databases/model/RetentionPeriodLabelKeys';
import { StorageCleanupNotice } from '@/pages/databases/ui/backup-config/StorageCleanupNotice';
import { type LocalizedText, translateLocalizedText } from '@/shared/i18n';
import { getUserClockFormat } from '@/shared/time';
import { LoadFailure } from '@/shared/ui';

interface Props {
  database: Database;
  cleanup?: BackupCleanup;
  cleanupError?: unknown;
  onRetryCleanup?: () => void;
  backupConfig: LogicalBackupConfigInput;
}

const getGfsRetentionTexts = (retention: LogicalBackupSettings['retention']): LocalizedText[] => {
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

export const ShowLogicalBackupConfigComponent = ({
  backupConfig,
  cleanup,
  cleanupError,
  onRetryCleanup,
}: Props) => {
  const { t } = useTranslation();
  const listSeparator = t('common.listSeparator');
  const [storage, setStorage] = useState<Storage>();
  const [storageLoadError, setStorageLoadError] = useState<unknown>();
  const [storageLoadAttempt, setStorageLoadAttempt] = useState(0);

  const describeGfsRetention = (retention: LogicalBackupSettings['retention']): string => {
    const texts = getGfsRetentionTexts(retention);
    if (texts.length === 0) return t('backups.config.gfs.notConfigured');

    return texts.map((text) => translateLocalizedText(text, t)).join(listSeparator);
  };

  const { storageId } = backupConfig;

  useEffect(() => {
    let isActive = true;

    setStorage(undefined);
    setStorageLoadError(undefined);

    if (!storageId) return;

    storageApi
      .getStorage(storageId)
      .then((loadedStorage) => {
        if (isActive) setStorage(loadedStorage);
      })
      .catch((error: unknown) => {
        if (isActive) setStorageLoadError(error);
      });

    return () => {
      isActive = false;
    };
  }, [storageId, storageLoadAttempt]);

  const { schedule, retention } = backupConfig.settings;

  const { localTime, weekday, dayOfMonth } = getLocalScheduleParts(schedule);
  const formattedTime = localTime ? localTime.format(getUserClockFormat().format) : '';

  const retentionPolicy = retention.policy;

  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('backups.config.backupsEnabled')}</div>
        <div className={backupConfig.isBackupsEnabled ? '' : 'font-bold text-danger'}>
          {backupConfig.isBackupsEnabled ? t('common.answers.yes') : t('common.answers.no')}
        </div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('backups.config.storage')}</div>
        <div className="flex items-center">
          <div>{storage?.name || ''}</div>
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
          onRetry={() => setStorageLoadAttempt((attempt) => attempt + 1)}
        />
      )}

      <StorageCleanupNotice cleanup={cleanup} error={cleanupError} onRetry={onRetryCleanup} />

      {backupConfig.isBackupsEnabled ? (
        <>
          <div className="mt-4 mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('backups.logical.config.interval')}</div>
            <div>{t(SCHEDULE_TYPE_LABEL_KEYS[schedule.type])}</div>
          </div>

          {schedule.type === ScheduleType.WEEKLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">{t('schedules.weekday')}</div>
              <div>{weekday ? t(WEEKDAY_LABEL_KEYS[weekday]) : ''}</div>
            </div>
          )}

          {schedule.type === ScheduleType.MONTHLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">{t('schedules.dayOfMonth')}</div>
              <div>{dayOfMonth || ''}</div>
            </div>
          )}

          {schedule.type === ScheduleType.CRON && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">{t('schedules.cronExpression')}</div>
              <code className="rounded bg-code px-2 py-0.5 text-sm">
                {schedule.cronExpression || ''}
              </code>
            </div>
          )}

          {schedule.type !== ScheduleType.HOURLY && schedule.type !== ScheduleType.CRON && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">{t('schedules.timeOfDay')}</div>
              <div>{formattedTime}</div>
            </div>
          )}

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">
              {t('backups.logical.config.retryIfFailedShort')}
            </div>
            <div>
              {backupConfig.settings.isRetryIfFailed
                ? t('common.answers.yes')
                : t('common.answers.no')}
            </div>
          </div>

          {backupConfig.settings.isRetryIfFailed && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">{t('backups.logical.config.maxFailedTries')}</div>
              <div>{backupConfig.settings.maxFailedTriesCount}</div>
            </div>
          )}

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('backups.logical.config.retentionPolicy')}</div>
            <div className="flex items-center gap-1">
              {retentionPolicy === LogicalRetentionPolicy.TIME_PERIOD && (
                <span>{t(RETENTION_PERIOD_LABEL_KEYS[retention.period])}</span>
              )}
              {retentionPolicy === LogicalRetentionPolicy.COUNT && (
                <span>
                  {t('backups.logical.config.retention.countValue', {
                    count: retention.count,
                  })}
                </span>
              )}
              {retentionPolicy === LogicalRetentionPolicy.GFS && (
                <span className="flex items-center gap-1">
                  {describeGfsRetention(retention)}
                  <Tooltip title={t('backups.logical.config.retention.gfsTooltip')}>
                    <InfoCircleOutlined className="text-muted" />
                  </Tooltip>
                </span>
              )}
            </div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('backups.config.encryption')}</div>
            <div>{t(BACKUP_ENCRYPTION_LABEL_KEYS[backupConfig.encryption])}</div>

            <Tooltip
              className="cursor-pointer"
              title={t('backups.logical.config.encryptionTooltip')}
            >
              <InfoCircleOutlined className="ml-2 text-muted" />
            </Tooltip>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('backups.config.notifications')}</div>
            <div>
              {backupConfig.settings.notificationTriggers.length > 0
                ? backupConfig.settings.notificationTriggers
                    .map((trigger) => t(LOGICAL_NOTIFICATION_TRIGGER_LABEL_KEYS[trigger]))
                    .join(listSeparator)
                : t('backups.config.noNotifications')}
            </div>
          </div>
        </>
      ) : (
        <div />
      )}
    </div>
  );
};
