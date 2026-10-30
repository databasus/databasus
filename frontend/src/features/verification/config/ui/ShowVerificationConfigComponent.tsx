import { Spin } from 'antd';
import { CronExpressionParser } from 'cron-parser';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  INTERVAL_TYPE_LABEL_KEYS,
  IntervalType,
  WEEKDAY_LABEL_KEYS,
  type Weekday,
} from '../../../../entities/intervals';
import {
  type VerificationConfig,
  VERIFICATION_NOTIFICATION_TYPE_LABEL_KEYS,
  VerificationScheduleType,
  verificationConfigApi,
} from '../../../../entities/verification/config';
import { translateApiError, useLocale } from '../../../../shared/i18n';
import { getUserTimeFormat } from '../../../../shared/time';
import {
  getUserTimeFormat as getIs12Hour,
  getLocalDayOfMonth,
  getLocalWeekday,
} from '../../../../shared/time/utils';

interface Props {
  databaseId: string;
}

export const ShowVerificationConfigComponent = ({ databaseId }: Props) => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const [config, setConfig] = useState<VerificationConfig>();
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    setIsLoading(true);
    verificationConfigApi
      .getByDatabaseId(databaseId)
      .then(setConfig)
      .catch((error: unknown) => alert(translateApiError(error, t)))
      .finally(() => setIsLoading(false));
  }, [databaseId]);

  if (isLoading) {
    return <Spin size="small" />;
  }

  if (!config) return <div />;

  const is12Hour = getIs12Hour();
  const timeFormat = { use12Hours: is12Hour, format: is12Hour ? 'h:mm A' : 'HH:mm' };
  const dateTimeFormat = getUserTimeFormat();

  const { schedule } = config;

  const localTime = schedule?.timeOfDay
    ? dayjs.utc(schedule.timeOfDay, 'HH:mm').local()
    : undefined;

  const formattedTime = localTime ? localTime.format(timeFormat.format) : '';

  const displayedWeekday: number | undefined =
    schedule?.type === IntervalType.WEEKLY && schedule.weekday && schedule.timeOfDay
      ? getLocalWeekday(schedule.weekday, schedule.timeOfDay)
      : schedule?.weekday;

  const displayedDayOfMonth: number | undefined =
    schedule?.type === IntervalType.MONTHLY && schedule.dayOfMonth && schedule.timeOfDay
      ? getLocalDayOfMonth(schedule.dayOfMonth, schedule.timeOfDay)
      : schedule?.dayOfMonth;

  const isAfterBackup = config.scheduleType === VerificationScheduleType.AFTER_BACKUP;

  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[180px] pr-2">{t('verification.config.scheduledVerification')}</div>
        <div className={config.isAutomaticVerificationEnabled ? '' : 'text-gray-500'}>
          {config.isAutomaticVerificationEnabled ? t('common.answers.yes') : t('common.answers.no')}
        </div>
      </div>

      {config.isAutomaticVerificationEnabled && (
        <>
          <div className="mt-5 mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('verification.config.interval')}</div>
            <div>
              {isAfterBackup
                ? t('verification.config.afterBackup')
                : schedule?.type
                  ? t(INTERVAL_TYPE_LABEL_KEYS[schedule.type])
                  : ''}
            </div>
          </div>

          {!isAfterBackup && schedule?.type === IntervalType.WEEKLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[180px] pr-2">{t('verification.config.weekday')}</div>
              <div>
                {displayedWeekday ? t(WEEKDAY_LABEL_KEYS[displayedWeekday as Weekday]) : ''}
              </div>
            </div>
          )}

          {!isAfterBackup && schedule?.type === IntervalType.MONTHLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[180px] pr-2">{t('verification.config.dayOfMonth')}</div>
              <div>{displayedDayOfMonth || ''}</div>
            </div>
          )}

          {!isAfterBackup && schedule?.type === IntervalType.CRON && (
            <>
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[180px] pr-2">{t('verification.config.cron.expression')}</div>
                <code className="rounded bg-gray-100 px-2 py-0.5 text-sm dark:bg-gray-700">
                  {schedule?.cronExpression || ''}
                </code>
              </div>
              {schedule?.cronExpression &&
                (() => {
                  try {
                    const interval = CronExpressionParser.parse(schedule.cronExpression, {
                      tz: 'UTC',
                    });
                    const nextRun = interval.next().toDate();
                    return (
                      <div className="mb-1 flex w-full items-center text-xs text-gray-600 dark:text-gray-400">
                        <div className="min-w-[180px] pr-2" />
                        <div>
                          <Trans
                            i18nKey="verification.config.cron.nextRun"
                            values={{
                              dateTime: dayjs(nextRun).local().format(dateTimeFormat.format),
                              relativeTime: formatRelativeTime(nextRun),
                            }}
                            components={{ lineBreak: <br /> }}
                          />
                        </div>
                      </div>
                    );
                  } catch {
                    return null;
                  }
                })()}
            </>
          )}

          {!isAfterBackup &&
            schedule?.type !== IntervalType.HOURLY &&
            schedule?.type !== IntervalType.CRON && (
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[180px] pr-2">{t('verification.config.timeOfDay')}</div>
                <div>{formattedTime}</div>
              </div>
            )}

          <div className="mt-5 mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('verification.config.notifications')}</div>
            <div>
              {config.notificationTriggers.length > 0
                ? config.notificationTriggers
                    .map((type) => t(VERIFICATION_NOTIFICATION_TYPE_LABEL_KEYS[type]))
                    .join(', ')
                : t('verification.config.noNotifications')}
            </div>
          </div>
        </>
      )}
    </div>
  );
};
