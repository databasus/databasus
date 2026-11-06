import { InputNumber, Select, TimePicker } from 'antd';
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
          <label htmlFor={weekdayControlId} className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
            {t('schedules.weekday')}
          </label>
          <Select
            id={weekdayControlId}
            value={weekday}
            onChange={(localWeekday) => onChange(withLocalWeekday(schedule, localWeekday))}
            size="small"
            className="w-full max-w-[200px] grow"
            options={weekdayOptions}
          />
        </div>
      )}

      {schedule.type === ScheduleType.MONTHLY && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <label htmlFor={dayOfMonthControlId} className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
            {t('schedules.dayOfMonth')}
          </label>
          <InputNumber
            id={dayOfMonthControlId}
            min={1}
            max={31}
            value={dayOfMonth}
            onChange={(localDayOfMonth) => {
              if (!localDayOfMonth) return;

              onChange(withLocalDayOfMonth(schedule, localDayOfMonth));
            }}
            size="small"
            className="w-full max-w-[200px] grow"
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
          <label htmlFor={timeOfDayControlId} className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
            {t('schedules.timeOfDay')}
          </label>
          <TimePicker
            id={timeOfDayControlId}
            value={localTime}
            format={clockFormat.format}
            use12Hours={clockFormat.use12Hours}
            allowClear={false}
            size="small"
            className="w-full max-w-[200px] grow"
            onChange={(pickedTime) => {
              if (!pickedTime) return;

              onChange(withLocalTimeOfDay(schedule, pickedTime));
            }}
          />
        </div>
      )}
    </>
  );
};
