import { InfoCircleOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import dayjs from 'dayjs';
import { type JSX, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type PhysicalBackupConfigInput,
  SCHEDULE_TYPE_LABEL_KEYS,
  type Schedule,
  ScheduleType,
  backupConfigApi,
  StorageSwitchHistoryComponent,
  toPhysicalBackupConfigInput,
  BACKUP_ENCRYPTION_LABEL_KEYS,
} from '../../../../entities/backup-configs';
import {
  type FullBackupsRetention,
  FullBackupsRetentionPolicy,
  PHYSICAL_NOTIFICATION_TRIGGER_LABEL_KEYS,
  PHYSICAL_RETENTION_POLICY_LABEL_KEYS,
  PhysicalRetentionPolicy,
} from '../../../../entities/backups/physical';
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

export const ShowPhysicalBackupConfigComponent = ({ database }: Props): JSX.Element => {
  const { t } = useTranslation();
  const listSeparator = t('common.listSeparator');
  const [backupConfig, setBackupConfig] = useState<PhysicalBackupConfigInput>();
  const [storage, setStorage] = useState<Storage>();

  const describeGfsRetention = (retention: FullBackupsRetention): string => {
    const texts = getGfsRetentionTexts(retention);
    if (texts.length === 0) return t('backups.config.gfs.notConfigured');

    return texts.map((text) => translateLocalizedText(text, t)).join(listSeparator);
  };

  const timeFormat = useMemo(() => {
    const is12Hour = getIs12Hour();
    return { use12Hours: is12Hour, format: is12Hour ? 'h:mm A' : 'HH:mm' };
  }, []);

  // Time of day, weekday and day of month are stored in UTC and shown in local time.
  const renderSchedule = (label: string, schedule: Schedule | null): JSX.Element | null => {
    if (!schedule) return null;

    const localTime = schedule.timeOfDay
      ? dayjs.utc(schedule.timeOfDay, 'HH:mm').local()
      : undefined;
    const formattedTime = localTime ? localTime.format(timeFormat.format) : '';

    const displayedWeekday =
      schedule.type === ScheduleType.WEEKLY && schedule.weekday && schedule.timeOfDay
        ? getLocalWeekday(schedule.weekday, schedule.timeOfDay)
        : schedule.weekday;

    const displayedDayOfMonth =
      schedule.type === ScheduleType.MONTHLY && schedule.dayOfMonth && schedule.timeOfDay
        ? getLocalDayOfMonth(schedule.dayOfMonth, schedule.timeOfDay)
        : schedule.dayOfMonth;

    return (
      <>
        <div className="mt-4 mb-1 flex w-full items-center">
          <div className="max-w-[150px] min-w-[150px] pr-2">{label}</div>
          <div>{t(SCHEDULE_TYPE_LABEL_KEYS[schedule.type])}</div>
        </div>

        {schedule.type === ScheduleType.WEEKLY && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.config.schedule.weekday')}</div>
            <div>{displayedWeekday ? t(WEEKDAY_LABEL_KEYS[displayedWeekday as Weekday]) : ''}</div>
          </div>
        )}

        {schedule.type === ScheduleType.MONTHLY && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.config.schedule.dayOfMonth')}</div>
            <div>{displayedDayOfMonth || ''}</div>
          </div>
        )}

        {schedule.type === ScheduleType.CRON && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.config.schedule.cronExpression')}</div>
            <code className="rounded bg-gray-100 px-2 py-0.5 text-sm dark:bg-gray-700">
              {schedule.cronExpression || ''}
            </code>
          </div>
        )}

        {schedule.type !== ScheduleType.HOURLY && schedule.type !== ScheduleType.CRON && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.config.schedule.timeOfDay')}</div>
            <div>{formattedTime}</div>
          </div>
        )}
      </>
    );
  };

  useEffect(() => {
    let isActive = true;

    const loadBackupConfig = async () => {
      try {
        const loadedBackupConfig = toPhysicalBackupConfigInput(
          await backupConfigApi.getBackupConfig(database.id),
        );
        if (!isActive) return;
        setBackupConfig(loadedBackupConfig);

        const storageId = loadedBackupConfig?.storageId;
        const loadedStorage = storageId ? await storageApi.getStorage(storageId) : undefined;
        if (isActive) setStorage(loadedStorage);
      } catch (e) {
        if (isActive) alert(translateApiError(e, t));
      }
    };

    if (database.id) void loadBackupConfig();

    return () => {
      isActive = false;
    };
  }, [database.id, t]);

  if (!backupConfig) return <div />;

  const { settings } = backupConfig;
  const { retention } = settings;
  const fullBackupsRetention = retention.fullBackups;

  const isShowChainsCount =
    retention.policy === PhysicalRetentionPolicy.CHAINS ||
    retention.policy === PhysicalRetentionPolicy.CHAINS_AND_FULL_BACKUPS;

  return (
    <div>
      <StorageSwitchHistoryComponent workspaceId={database.workspaceId} databaseId={database.id} />
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[180px] pr-2">{t('backups.config.backupsEnabled')}</div>
        <div className={backupConfig.isBackupsEnabled ? '' : 'font-bold text-red-600'}>
          {backupConfig.isBackupsEnabled ? t('common.answers.yes') : t('common.answers.no')}
        </div>
      </div>

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
                    <Tooltip title={t('backups.physical.config.gfsTooltip')}>
                      <InfoCircleOutlined style={{ color: 'gray' }} />
                    </Tooltip>
                  </span>
                )}
              </div>
            </div>
          )}

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
