import { NumberInput } from '@mantine/core';
import { TimePicker } from '@mantine/dates';
import dayjs from 'dayjs';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import {
  getLocalScheduleParts,
  withLocalDayOfMonth,
  withLocalTimeOfDay,
  withLocalWeekday,
} from '@/entities/schedules/model/localSchedule';
import type { Schedule } from '@/entities/schedules/model/Schedule';
import { ScheduleType } from '@/entities/schedules/model/ScheduleType';
import { WEEKDAYS } from '@/entities/schedules/model/Weekday';
import { WEEKDAY_LABEL_KEYS } from '@/entities/schedules/model/WeekdayLabelKeys';
import { CronExpressionField } from '@/entities/schedules/ui/CronExpressionField';
import { getUserClockFormat } from '@/shared/time';
import { WrappingSelect } from '@/shared/ui';

interface Props {
  schedule: Schedule;
  onChange: (schedule: Schedule) => void;
}

export const ScheduleDetailsFields = ({ schedule, onChange }: Props) => {
  const { t } = useTranslation();
  const weekdayControlId = useId();
  const dayOfMonthControlId = useId();
  const timeOfDayControlId = useId();

  const { localTime, weekday, dayOfMonth } = getLocalScheduleParts(schedule);
  const clockFormat = getUserClockFormat();

  const weekdayOptions = WEEKDAYS.map((weekdayOption) => ({
    value: weekdayOption,
    label: t(WEEKDAY_LABEL_KEYS[weekdayOption]),
  }));

  return (
    <>
      {schedule.type === ScheduleType.WEEKLY && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <label
            htmlFor={weekdayControlId}
            className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2"
          >
            {t('schedules.weekday')}
          </label>
          <WrappingSelect
            id={weekdayControlId}
            value={weekday}
            onChange={(localWeekday) => onChange(withLocalWeekday(schedule, localWeekday))}
            size="xs"
            className="w-full max-w-[200px] grow"
            data={weekdayOptions}
          />
        </div>
      )}

      {schedule.type === ScheduleType.MONTHLY && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <label
            htmlFor={dayOfMonthControlId}
            className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2"
          >
            {t('schedules.dayOfMonth')}
          </label>
          <NumberInput
            id={dayOfMonthControlId}
            min={1}
            max={31}
            value={dayOfMonth}
            onChange={(localDayOfMonth) => {
              if (typeof localDayOfMonth !== 'number' || !localDayOfMonth) return;

              onChange(withLocalDayOfMonth(schedule, localDayOfMonth));
            }}
            size="xs"
            className="w-full max-w-[200px] grow"
            role="spinbutton"
            aria-valuenow={typeof dayOfMonth === 'number' ? dayOfMonth : undefined}
            aria-valuemin={1}
            aria-valuemax={31}
          />
        </div>
      )}

      {schedule.type === ScheduleType.CRON && (
        <CronExpressionField
          cronExpression={schedule.cronExpression ?? ''}
          onChange={(cronExpression) => onChange({ ...schedule, cronExpression })}
        />
      )}

      {schedule.type !== ScheduleType.HOURLY && schedule.type !== ScheduleType.CRON && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <label
            htmlFor={timeOfDayControlId}
            className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2"
          >
            {t('schedules.timeOfDay')}
          </label>
          <TimePicker
            value={localTime?.format('HH:mm:ss') ?? ''}
            format={clockFormat.use12Hours ? '12h' : '24h'}
            withDropdown
            hoursInputProps={{ id: timeOfDayControlId }}
            hoursInputLabel={t('schedules.hours')}
            minutesInputLabel={t('schedules.minutes')}
            amPmInputLabel={t('schedules.timePeriod')}
            amPmLabels={{ am: dayjs().hour(9).format('A'), pm: dayjs().hour(15).format('A') }}
            pasteSplit={({ time, format, amPmLabels }) => {
              const parsedTime = dayjs(time, [clockFormat.format, 'HH:mm', 'HH:mm:ss'], true);

              if (!parsedTime.isValid())
                return { hours: null, minutes: null, seconds: null, amPm: null };

              return {
                hours: format === '12h' ? parsedTime.hour() % 12 || 12 : parsedTime.hour(),
                minutes: parsedTime.minute(),
                seconds: parsedTime.second(),
                amPm:
                  format === '12h'
                    ? parsedTime.hour() >= 12
                      ? amPmLabels.pm
                      : amPmLabels.am
                    : null,
              };
            }}
            size="xs"
            className="w-full grow sm:max-w-[200px]"
            onChange={(pickedTime) => {
              if (!pickedTime) return;

              const localTime = dayjs(pickedTime, ['HH:mm:ss', 'HH:mm'], true);

              if (localTime.isValid()) onChange(withLocalTimeOfDay(schedule, localTime));
            }}
          />
        </div>
      )}
    </>
  );
};
