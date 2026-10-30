import { InfoCircleOutlined } from '@ant-design/icons';
import {
  Button,
  Checkbox,
  Input,
  InputNumber,
  Select,
  Spin,
  Switch,
  TimePicker,
  Tooltip,
} from 'antd';
import { CronExpressionParser } from 'cron-parser';
import dayjs, { type Dayjs } from 'dayjs';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  INTERVAL_TYPE_LABEL_KEYS,
  type Interval,
  IntervalType,
  WEEKDAYS,
  WEEKDAY_LABEL_KEYS,
} from '../../../../entities/intervals';
import {
  type VerificationConfig,
  VERIFICATION_NOTIFICATION_TYPE_LABEL_KEYS,
  VerificationNotificationType,
  VerificationScheduleType,
  verificationConfigApi,
} from '../../../../entities/verification/config';
import { getWebsitePageUrl, translateApiError, useLocale } from '../../../../shared/i18n';
import { getUserTimeFormat } from '../../../../shared/time';
import {
  getUserTimeFormat as getIs12Hour,
  getLocalDayOfMonth,
  getLocalWeekday,
  getUtcDayOfMonth,
  getUtcWeekday,
} from '../../../../shared/time/utils';

interface Props {
  databaseId: string;
  onClose: () => void;
}

export const EditVerificationConfigComponent = ({ databaseId, onClose }: Props) => {
  const { t } = useTranslation();
  const { locale, formatRelativeTime } = useLocale();
  const [config, setConfig] = useState<VerificationConfig>();
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isUnsaved, setIsUnsaved] = useState(false);

  const updateConfig = (patch: Partial<VerificationConfig>) => {
    setConfig((prev) => (prev ? { ...prev, ...patch } : prev));
    setIsUnsaved(true);
  };

  const saveInterval = (patch: Partial<Interval>) => {
    setConfig((prev) => {
      if (!prev) return prev;

      const updatedInterval = { ...prev.schedule, ...patch } as Interval;
      return { ...prev, schedule: updatedInterval };
    });
    setIsUnsaved(true);
  };

  const selectSchedule = (value: string) => {
    if (value === VerificationScheduleType.AFTER_BACKUP) {
      updateConfig({ scheduleType: VerificationScheduleType.AFTER_BACKUP });
      return;
    }

    setConfig((prev) =>
      prev
        ? {
            ...prev,
            scheduleType: VerificationScheduleType.INTERVAL,
            schedule: { ...prev.schedule, type: value as IntervalType },
          }
        : prev,
    );
    setIsUnsaved(true);
  };

  const toggleNotification = (type: VerificationNotificationType, checked: boolean) => {
    if (!config) return;

    const notifications = [...config.notificationTriggers];
    const index = notifications.indexOf(type);

    if (checked && index === -1) {
      notifications.push(type);
    } else if (!checked && index > -1) {
      notifications.splice(index, 1);
    }

    updateConfig({ notificationTriggers: notifications });
  };

  const handleSave = async () => {
    if (!config) return;

    setIsSaving(true);

    try {
      await verificationConfigApi.save(databaseId, config);
      setIsUnsaved(false);
      onClose();
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsSaving(false);
  };

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

  const localTime: Dayjs | undefined = schedule?.timeOfDay
    ? dayjs.utc(schedule.timeOfDay, 'HH:mm').local()
    : undefined;

  const displayedWeekday: number | undefined =
    schedule?.type === IntervalType.WEEKLY && schedule.weekday && schedule.timeOfDay
      ? getLocalWeekday(schedule.weekday, schedule.timeOfDay)
      : schedule?.weekday;

  const displayedDayOfMonth: number | undefined =
    schedule?.type === IntervalType.MONTHLY && schedule.dayOfMonth && schedule.timeOfDay
      ? getLocalDayOfMonth(schedule.dayOfMonth, schedule.timeOfDay)
      : schedule?.dayOfMonth;

  const isAfterBackup = config.scheduleType === VerificationScheduleType.AFTER_BACKUP;

  const isIntervalValid =
    !config.isAutomaticVerificationEnabled ||
    isAfterBackup ||
    (Boolean(schedule?.type) &&
      (schedule.type !== IntervalType.WEEKLY || displayedWeekday) &&
      (schedule.type !== IntervalType.MONTHLY || displayedDayOfMonth) &&
      (schedule.type !== IntervalType.CRON || schedule.cronExpression));

  const isAllFieldsFilled = isIntervalValid;

  return (
    <div className="space-y-3">
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[180px] sm:mb-0 sm:pr-2">
          {t('verification.config.scheduledVerification')}
        </div>
        <Switch
          checked={config.isAutomaticVerificationEnabled}
          onChange={(checked) => updateConfig({ isAutomaticVerificationEnabled: checked })}
          size="small"
        />
        <Tooltip
          className="cursor-pointer"
          title={
            <Trans
              i18nKey="verification.config.scheduledVerificationTooltip"
              components={{
                docsLink: (
                  <a
                    href={getWebsitePageUrl('restoreVerification', locale)}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="underline"
                  />
                ),
              }}
            />
          }
        >
          <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
        </Tooltip>
      </div>

      {config.isAutomaticVerificationEnabled && (
        <>
          <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[180px] sm:mb-0 sm:pr-2">
              {t('verification.config.interval')}
            </div>
            <Select
              value={isAfterBackup ? VerificationScheduleType.AFTER_BACKUP : schedule?.type}
              onChange={selectSchedule}
              size="small"
              className="w-full max-w-[200px] grow"
              options={[
                {
                  label: t('verification.config.afterBackup'),
                  value: VerificationScheduleType.AFTER_BACKUP,
                },
                ...Object.values(IntervalType).map((intervalType) => ({
                  label: t(INTERVAL_TYPE_LABEL_KEYS[intervalType]),
                  value: intervalType,
                })),
              ]}
            />
          </div>

          {isAfterBackup && (
            <div className="mb-1 flex w-full flex-col items-start text-xs text-gray-600 sm:flex-row sm:items-center dark:text-gray-400">
              <div className="mb-1 min-w-[180px] sm:mb-0 sm:pr-2" />
              <div>{t('verification.config.afterBackupHint')}</div>
            </div>
          )}

          {!isAfterBackup && schedule?.type === IntervalType.WEEKLY && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[180px] sm:mb-0 sm:pr-2">
                {t('verification.config.weekday')}
              </div>
              <Select
                value={displayedWeekday}
                onChange={(localWeekday) => {
                  if (!localWeekday) return;
                  const ref = localTime ?? dayjs();
                  saveInterval({ weekday: getUtcWeekday(localWeekday, ref) });
                }}
                size="small"
                className="w-full max-w-[200px] grow"
                options={WEEKDAYS.map((weekday) => ({
                  value: weekday,
                  label: t(WEEKDAY_LABEL_KEYS[weekday]),
                }))}
              />
            </div>
          )}

          {!isAfterBackup && schedule?.type === IntervalType.MONTHLY && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[180px] sm:mb-0 sm:pr-2">
                {t('verification.config.dayOfMonth')}
              </div>
              <InputNumber
                min={1}
                max={31}
                value={displayedDayOfMonth}
                onChange={(localDom) => {
                  if (!localDom) return;
                  const ref = localTime ?? dayjs();
                  saveInterval({ dayOfMonth: getUtcDayOfMonth(localDom, ref) });
                }}
                size="small"
                className="w-full max-w-[200px] grow"
              />
            </div>
          )}

          {!isAfterBackup && schedule?.type === IntervalType.CRON && (
            <>
              <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
                <div className="mb-1 min-w-[180px] sm:mb-0 sm:pr-2">
                  {t('verification.config.cron.expression')}
                </div>
                <div className="flex items-center">
                  <Input
                    value={schedule?.cronExpression || ''}
                    onChange={(e) => saveInterval({ cronExpression: e.target.value })}
                    placeholder="0 4 * * 0"
                    size="small"
                    className="w-full max-w-[200px] grow"
                  />
                  <Tooltip
                    className="cursor-pointer"
                    title={
                      <div>
                        <div className="font-bold">{t('verification.config.cron.format')}</div>
                        <div className="mt-1">{t('verification.config.cron.examples')}</div>
                        <div>
                          {t('verification.config.cron.exampleEverySunday', {
                            expression: '0 4 * * 0',
                          })}
                        </div>
                        <div>
                          {t('verification.config.cron.exampleEverySixHours', {
                            expression: '0 */6 * * *',
                          })}
                        </div>
                      </div>
                    }
                  >
                    <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                  </Tooltip>
                </div>
              </div>
              {schedule?.cronExpression &&
                (() => {
                  try {
                    const interval = CronExpressionParser.parse(schedule.cronExpression, {
                      tz: 'UTC',
                    });
                    const nextRun = interval.next().toDate();
                    return (
                      <div className="mb-1 flex w-full flex-col items-start text-xs text-gray-600 sm:flex-row sm:items-center dark:text-gray-400">
                        <div className="mb-1 min-w-[180px] sm:mb-0 sm:pr-2" />
                        <div className="text-gray-600 dark:text-gray-400">
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
                    return (
                      <div className="mb-1 flex w-full flex-col items-start text-red-500 sm:flex-row sm:items-center">
                        <div className="mb-1 min-w-[180px] sm:mb-0 sm:pr-2" />
                        <div className="text-red-500">{t('verification.config.cron.invalid')}</div>
                      </div>
                    );
                  }
                })()}
            </>
          )}

          {!isAfterBackup &&
            schedule?.type !== IntervalType.HOURLY &&
            schedule?.type !== IntervalType.CRON && (
              <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
                <div className="mb-1 min-w-[180px] sm:mb-0 sm:pr-2">
                  {t('verification.config.timeOfDay')}
                </div>
                <TimePicker
                  value={localTime}
                  format={timeFormat.format}
                  use12Hours={timeFormat.use12Hours}
                  allowClear={false}
                  size="small"
                  className="w-full max-w-[200px] grow"
                  onChange={(pickedTime) => {
                    if (!pickedTime) return;
                    const patch: Partial<Interval> = {
                      timeOfDay: pickedTime.utc().format('HH:mm'),
                    };

                    if (schedule?.type === IntervalType.WEEKLY && displayedWeekday) {
                      patch.weekday = getUtcWeekday(displayedWeekday, pickedTime);
                    }
                    if (schedule?.type === IntervalType.MONTHLY && displayedDayOfMonth) {
                      patch.dayOfMonth = getUtcDayOfMonth(displayedDayOfMonth, pickedTime);
                    }

                    saveInterval(patch);
                  }}
                />
              </div>
            )}

          <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
            <div className="mt-0 mb-1 min-w-[180px] sm:mt-1 sm:mb-0 sm:pr-2">
              {t('verification.config.notifications')}
            </div>
            <div className="flex flex-col space-y-2">
              {Object.values(VerificationNotificationType).map((notificationType) => (
                <Checkbox
                  key={notificationType}
                  checked={config.notificationTriggers.includes(notificationType)}
                  onChange={(e) => toggleNotification(notificationType, e.target.checked)}
                >
                  {t(VERIFICATION_NOTIFICATION_TYPE_LABEL_KEYS[notificationType])}
                </Checkbox>
              ))}
            </div>
          </div>
        </>
      )}

      <div className="mt-6 flex justify-end space-x-2">
        <Button onClick={onClose} disabled={isSaving}>
          {t('common.actions.cancel')}
        </Button>

        <Button
          type="primary"
          onClick={handleSave}
          loading={isSaving}
          disabled={!isUnsaved || !isAllFieldsFilled}
        >
          {t('common.actions.save')}
        </Button>
      </div>
    </div>
  );
};
