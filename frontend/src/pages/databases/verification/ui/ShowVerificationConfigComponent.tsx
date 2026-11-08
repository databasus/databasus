import { Loader } from '@mantine/core';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';

import {
  SCHEDULE_TYPE_LABEL_KEYS,
  ScheduleType,
  WEEKDAY_LABEL_KEYS,
  getLocalScheduleParts,
  getNextCronRun,
} from '@/entities/schedules';
import { verificationConfigApi } from '@/pages/databases/verification/api/verificationConfigApi';
import { VERIFICATION_NOTIFICATION_TYPE_LABEL_KEYS } from '@/pages/databases/verification/model/VerificationNotificationType';
import { VerificationScheduleType } from '@/pages/databases/verification/model/VerificationScheduleType';
import { useSessionQuery } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { getUserClockFormat, getUserTimeFormat } from '@/shared/time';
import { LoadFailure } from '@/shared/ui';

interface Props {
  databaseId: string;
}

export const ShowVerificationConfigComponent = ({ databaseId }: Props) => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();

  const query = useSessionQuery({
    queryKey: ['verification-config', databaseId],
    queryFn: (signal) => verificationConfigApi.getByDatabaseId(databaseId, { signal }),
  });
  const config = query.data;
  const loadError = query.error ?? undefined;

  if (loadError !== undefined) {
    return <LoadFailure error={loadError} onRetry={() => void query.refetch()} />;
  }

  if (!config) {
    return <Loader size="xs" role="status" />;
  }

  const { schedule } = config;

  const { localTime, weekday, dayOfMonth } = getLocalScheduleParts(schedule);
  const formattedTime = localTime ? localTime.format(getUserClockFormat().format) : '';
  const nextCronRun = getNextCronRun(schedule.cronExpression ?? '');

  const isAfterBackup = config.scheduleType === VerificationScheduleType.AFTER_BACKUP;

  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('verification.config.scheduledVerification')}</div>
        <div className={config.isAutomaticVerificationEnabled ? '' : 'text-muted'}>
          {config.isAutomaticVerificationEnabled ? t('common.answers.yes') : t('common.answers.no')}
        </div>
      </div>

      {config.isAutomaticVerificationEnabled && (
        <>
          <div className="mt-5 mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('verification.config.interval')}</div>
            <div>
              {isAfterBackup
                ? t('verification.config.afterBackup')
                : t(SCHEDULE_TYPE_LABEL_KEYS[schedule.type])}
            </div>
          </div>

          {!isAfterBackup && schedule.type === ScheduleType.WEEKLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">{t('schedules.weekday')}</div>
              <div>{weekday ? t(WEEKDAY_LABEL_KEYS[weekday]) : ''}</div>
            </div>
          )}

          {!isAfterBackup && schedule.type === ScheduleType.MONTHLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">{t('schedules.dayOfMonth')}</div>
              <div>{dayOfMonth || ''}</div>
            </div>
          )}

          {!isAfterBackup && schedule.type === ScheduleType.CRON && (
            <>
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px] pr-2">{t('schedules.cronExpression')}</div>
                <code className="rounded bg-code px-2 py-0.5 text-sm">
                  {schedule.cronExpression || ''}
                </code>
              </div>

              {nextCronRun && (
                <div className="mb-1 flex w-full items-center text-xs text-muted">
                  <div className="min-w-[150px] pr-2" />
                  <div>
                    {t('schedules.nextRun', {
                      time: dayjs(nextCronRun).local().format(getUserTimeFormat().format),
                    })}
                    <br />({formatRelativeTime(nextCronRun)})
                  </div>
                </div>
              )}
            </>
          )}

          {!isAfterBackup &&
            schedule.type !== ScheduleType.HOURLY &&
            schedule.type !== ScheduleType.CRON && (
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px] pr-2">{t('schedules.timeOfDay')}</div>
                <div>{formattedTime}</div>
              </div>
            )}

          <div className="mt-5 mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('verification.config.notifications')}</div>
            <div>
              {config.notificationTriggers.length > 0
                ? config.notificationTriggers
                    .map((type) => t(VERIFICATION_NOTIFICATION_TYPE_LABEL_KEYS[type]))
                    .join(t('common.listSeparator'))
                : t('verification.config.noNotifications')}
            </div>
          </div>
        </>
      )}
    </div>
  );
};
