import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, InputNumber, Select, TimePicker, Tooltip } from 'antd';
import { CronExpressionParser } from 'cron-parser';
import dayjs, { Dayjs } from 'dayjs';
import { type JSX, useMemo } from 'react';
import { useTranslation } from 'react-i18next';

import {
  SCHEDULE_TYPE_LABEL_KEYS,
  type Schedule,
  ScheduleType,
} from '../../../../entities/backup-configs';
import { WEEKDAYS, WEEKDAY_LABEL_KEYS } from '../../../../entities/intervals';
import { useLocale } from '../../../../shared/i18n';
import { getUserTimeFormat } from '../../../../shared/time';
import {
  getUserTimeFormat as getIs12Hour,
  getLocalDayOfMonth,
  getLocalWeekday,
  getUtcDayOfMonth,
  getUtcWeekday,
} from '../../../../shared/time/utils';

interface Props {
  label: string;
  schedule?: Schedule | null;
  onChange: (patch: Partial<Schedule>) => void;
}

// Reusable interval sub-form for a single backup cadence. All times are stored in
// UTC (timeOfDay 'HH:mm', weekday/dayOfMonth as UTC values) and displayed in the
// user's local timezone - the getUtc*/getLocal* helpers translate across the date
// boundary so e.g. "Sunday 23:00 local" can map to "Monday 04:00 UTC".
export const PhysicalIntervalEditor = ({ label, schedule, onChange }: Props): JSX.Element => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();

  const timeFormat = useMemo(() => {
    const is12 = getIs12Hour();
    return { use12Hours: is12, format: is12 ? 'h:mm A' : 'HH:mm' };
  }, []);

  const dateTimeFormat = useMemo(() => getUserTimeFormat(), []);

  // UTC -> local conversions for display
  const localTime: Dayjs | undefined = schedule?.timeOfDay
    ? dayjs.utc(schedule.timeOfDay, 'HH:mm').local()
    : undefined;

  const displayedWeekday: number | undefined =
    schedule?.type === ScheduleType.WEEKLY && schedule.weekday && schedule.timeOfDay
      ? getLocalWeekday(schedule.weekday, schedule.timeOfDay)
      : schedule?.weekday;

  const displayedDayOfMonth: number | undefined =
    schedule?.type === ScheduleType.MONTHLY && schedule.dayOfMonth && schedule.timeOfDay
      ? getLocalDayOfMonth(schedule.dayOfMonth, schedule.timeOfDay)
      : schedule?.dayOfMonth;

  const scheduleTypeOptions = Object.values(ScheduleType).map((scheduleType) => ({
    value: scheduleType,
    label: t(SCHEDULE_TYPE_LABEL_KEYS[scheduleType]),
  }));

  const weekdayOptions = WEEKDAYS.map((weekday) => ({
    value: weekday,
    label: t(WEEKDAY_LABEL_KEYS[weekday]),
  }));

  return (
    <>
      <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 max-w-[150px] min-w-[150px] leading-4 sm:mb-0 sm:pr-2">{label}</div>
        <Select
          value={schedule?.type}
          onChange={(v) => onChange({ type: v })}
          size="small"
          className="w-full max-w-[200px] grow"
          options={scheduleTypeOptions}
        />
      </div>

      {schedule?.type === ScheduleType.WEEKLY && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
            {t('backups.config.schedule.weekday')}
          </div>
          <Select
            value={displayedWeekday}
            onChange={(localWeekday) => {
              if (!localWeekday) return;
              const ref = localTime ?? dayjs();
              onChange({ weekday: getUtcWeekday(localWeekday, ref) });
            }}
            size="small"
            className="w-full max-w-[200px] grow"
            options={weekdayOptions}
          />
        </div>
      )}

      {schedule?.type === ScheduleType.MONTHLY && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
            {t('backups.config.schedule.dayOfMonth')}
          </div>
          <InputNumber
            min={1}
            max={31}
            value={displayedDayOfMonth}
            onChange={(localDom) => {
              if (!localDom) return;
              const ref = localTime ?? dayjs();
              onChange({ dayOfMonth: getUtcDayOfMonth(localDom, ref) });
            }}
            size="small"
            className="w-full max-w-[200px] grow"
          />
        </div>
      )}

      {schedule?.type === ScheduleType.CRON && (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('backups.config.schedule.cronExpression')}
            </div>
            <div className="flex items-center">
              <Input
                value={schedule?.cronExpression || ''}
                onChange={(e) => onChange({ cronExpression: e.target.value })}
                placeholder="0 2 * * *"
                size="small"
                className="w-full max-w-[200px] grow"
              />
              <Tooltip
                className="cursor-pointer"
                title={
                  <div>
                    <div className="font-bold">{t('backups.config.schedule.cronHelp.format')}</div>
                    <div className="mt-1">{t('backups.config.schedule.cronHelp.examples')}</div>
                    <div>0 2 * * * - {t('backups.config.schedule.cronHelp.dailyAt2am')}</div>
                    <div>0 */6 * * * - {t('backups.config.schedule.cronHelp.every6Hours')}</div>
                    <div>0 3 * * 1 - {t('backups.config.schedule.cronHelp.mondaysAt3am')}</div>
                    <div>30 4 1,15 * * - {t('backups.config.schedule.cronHelp.twiceAMonth')}</div>
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
                const parsed = CronExpressionParser.parse(schedule.cronExpression, {
                  tz: 'UTC',
                });
                const nextRun = parsed.next().toDate();
                return (
                  <div className="mb-1 flex w-full flex-col items-start text-xs text-gray-600 sm:flex-row sm:items-center dark:text-gray-400">
                    <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2" />
                    <div className="text-gray-600 dark:text-gray-400">
                      {t('backups.config.schedule.nextRun', {
                        time: dayjs(nextRun).local().format(dateTimeFormat.format),
                      })}
                      <br />({formatRelativeTime(nextRun)})
                    </div>
                  </div>
                );
              } catch {
                return (
                  <div className="mb-1 flex w-full flex-col items-start text-red-500 sm:flex-row sm:items-center">
                    <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2" />
                    <div className="text-red-500">{t('backups.config.schedule.invalidCron')}</div>
                  </div>
                );
              }
            })()}
        </>
      )}

      {schedule?.type !== ScheduleType.HOURLY && schedule?.type !== ScheduleType.CRON && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
            {t('backups.config.schedule.timeOfDay')}
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
              const patch: Partial<Schedule> = { timeOfDay: pickedTime.utc().format('HH:mm') };

              if (schedule?.type === ScheduleType.WEEKLY && displayedWeekday) {
                patch.weekday = getUtcWeekday(displayedWeekday, pickedTime);
              }
              if (schedule?.type === ScheduleType.MONTHLY && displayedDayOfMonth) {
                patch.dayOfMonth = getUtcDayOfMonth(displayedDayOfMonth, pickedTime);
              }

              onChange(patch);
            }}
          />
        </div>
      )}
    </>
  );
};
