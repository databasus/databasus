import { InfoCircleOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import dayjs from 'dayjs';
import { useMemo } from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type LogicalBackupConfigInput,
  type LogicalBackupSettings,
  SCHEDULE_TYPE_LABEL_KEYS,
  ScheduleType,
  backupConfigApi,
  StorageSwitchHistoryComponent,
  toLogicalBackupConfigInput,
} from '../../../../entities/backups/config';
import {
  LOGICAL_NOTIFICATION_TRIGGER_LABEL_KEYS,
  LogicalRetentionPolicy,
  RETENTION_PERIOD_LABEL_KEYS,
} from '../../../../entities/backups/logical';
import { BACKUP_ENCRYPTION_LABEL_KEYS } from '../../../../entities/backups/shared';
import type { Database } from '../../../../entities/databases';
import { WEEKDAY_LABEL_KEYS, type Weekday } from '../../../../entities/intervals';
import { type Storage, getStorageLogoFromType, storageApi } from '../../../../entities/storages';
import {
  type LocalizedText,
  translateApiError,
  translateLocalizedText,
} from '../../../../shared/i18n';
import {
  getUserTimeFormat as getIs12Hour,
  getLocalDayOfMonth,
  getLocalWeekday,
} from '../../../../shared/time/utils';

interface Props {
  database: Database;
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

export const ShowLogicalBackupConfigComponent = ({ database }: Props) => {
  const { t } = useTranslation();
  const listSeparator = t('common.listSeparator');
  const [backupConfig, setBackupConfig] = useState<LogicalBackupConfigInput>();
  const [storage, setStorage] = useState<Storage>();

  const describeGfsRetention = (retention: LogicalBackupSettings['retention']): string => {
    const texts = getGfsRetentionTexts(retention);
    if (texts.length === 0) return t('backups.config.gfs.notConfigured');

    return texts.map((text) => translateLocalizedText(text, t)).join(listSeparator);
  };

  const timeFormat = useMemo(() => {
    const is12Hour = getIs12Hour();
    return {
      use12Hours: is12Hour,
      format: is12Hour ? 'h:mm A' : 'HH:mm',
    };
  }, []);

  useEffect(() => {
    let isActive = true;
    const loadBackupConfig = async () => {
      try {
        const loadedBackupConfig = toLogicalBackupConfigInput(
          await backupConfigApi.getBackupConfig(database.id),
        );
        if (!isActive) return;
        setBackupConfig(loadedBackupConfig);
        const storageId = loadedBackupConfig?.storageId;
        const loadedStorage = storageId ? await storageApi.getStorage(storageId) : undefined;
        if (isActive) setStorage(loadedStorage);
      } catch (error) {
        if (isActive) alert(translateApiError(error, t));
      }
    };
    if (database.id) void loadBackupConfig();
    return () => {
      isActive = false;
    };
  }, [database.id, t]);

  if (!backupConfig) return <div />;

  const { schedule, retention } = backupConfig.settings;

  const localTime = schedule.timeOfDay ? dayjs.utc(schedule.timeOfDay, 'HH:mm').local() : undefined;

  const formattedTime = localTime ? localTime.format(timeFormat.format) : '';

  const displayedWeekday: number | undefined =
    schedule.type === ScheduleType.WEEKLY && schedule.weekday && schedule.timeOfDay
      ? getLocalWeekday(schedule.weekday, schedule.timeOfDay)
      : schedule.weekday;

  const displayedDayOfMonth: number | undefined =
    schedule.type === ScheduleType.MONTHLY && schedule.dayOfMonth && schedule.timeOfDay
      ? getLocalDayOfMonth(schedule.dayOfMonth, schedule.timeOfDay)
      : schedule.dayOfMonth;

  const retentionPolicy = retention.policy;

  return (
    <div>
      <StorageSwitchHistoryComponent workspaceId={database.workspaceId} databaseId={database.id} />
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('backups.config.backupsEnabled')}</div>
        <div className={backupConfig.isBackupsEnabled ? '' : 'font-bold text-red-600'}>
          {backupConfig.isBackupsEnabled ? t('common.answers.yes') : t('common.answers.no')}
        </div>
      </div>

      {backupConfig.isBackupsEnabled ? (
        <>
          <div className="mt-4 mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('backups.logical.config.interval')}</div>
            <div>{t(SCHEDULE_TYPE_LABEL_KEYS[schedule.type])}</div>
          </div>

          {schedule.type === ScheduleType.WEEKLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">{t('backups.logical.config.weekday')}</div>
              <div>
                {displayedWeekday ? t(WEEKDAY_LABEL_KEYS[displayedWeekday as Weekday]) : ''}
              </div>
            </div>
          )}

          {schedule.type === ScheduleType.MONTHLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">{t('backups.logical.config.dayOfMonth')}</div>
              <div>{displayedDayOfMonth || ''}</div>
            </div>
          )}

          {schedule.type === ScheduleType.CRON && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">
                {t('backups.config.schedule.cronExpression')}
              </div>
              <code className="rounded bg-gray-100 px-2 py-0.5 text-sm dark:bg-gray-700">
                {schedule.cronExpression || ''}
              </code>
            </div>
          )}

          {schedule.type !== ScheduleType.HOURLY && schedule.type !== ScheduleType.CRON && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">{t('backups.logical.config.timeOfDay')}</div>
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
                    <InfoCircleOutlined style={{ color: 'gray' }} />
                  </Tooltip>
                </span>
              )}
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

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('backups.config.encryption')}</div>
            <div>{t(BACKUP_ENCRYPTION_LABEL_KEYS[backupConfig.encryption])}</div>

            <Tooltip
              className="cursor-pointer"
              title={t('backups.logical.config.encryptionTooltip')}
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
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
