import dayjs, { type Dayjs } from 'dayjs';

import type { Schedule } from '@/entities/schedules/model/Schedule';
import {
  getLocalDayOfMonth,
  getLocalWeekday,
  getUtcDayOfMonth,
  getUtcWeekday,
} from '@/entities/schedules/model/scheduleDayConversion';
import type { Weekday } from '@/entities/schedules/model/Weekday';

interface LocalScheduleParts {
  localTime?: Dayjs;
  weekday?: Weekday;
  dayOfMonth?: number;
}

const getLocalTime = (timeOfDay: string): Dayjs => dayjs.utc(timeOfDay, 'HH:mm').local();

const getShownWeekday = ({ weekday, timeOfDay }: Schedule): Weekday | undefined => {
  if (typeof weekday !== 'number') return undefined;
  if (timeOfDay) return getLocalWeekday(weekday, timeOfDay) as Weekday;

  return (weekday === 0 ? 7 : weekday) as Weekday;
};

const getShownDayOfMonth = ({ dayOfMonth, timeOfDay }: Schedule): number | undefined => {
  if (typeof dayOfMonth !== 'number') return undefined;

  return timeOfDay ? getLocalDayOfMonth(dayOfMonth, timeOfDay) : dayOfMonth;
};

// Without a time of day a schedule has no UTC day yet, so its day stays as the user chose it until
// withLocalTimeOfDay converts both together.
export const getLocalScheduleParts = (schedule: Schedule): LocalScheduleParts => ({
  localTime: schedule.timeOfDay ? getLocalTime(schedule.timeOfDay) : undefined,
  weekday: getShownWeekday(schedule),
  dayOfMonth: getShownDayOfMonth(schedule),
});

export const withLocalWeekday = (schedule: Schedule, localWeekday: Weekday): Schedule => ({
  ...schedule,
  weekday: schedule.timeOfDay
    ? getUtcWeekday(localWeekday, getLocalTime(schedule.timeOfDay))
    : localWeekday,
});

export const withLocalDayOfMonth = (schedule: Schedule, localDayOfMonth: number): Schedule => ({
  ...schedule,
  dayOfMonth: schedule.timeOfDay
    ? getUtcDayOfMonth(localDayOfMonth, getLocalTime(schedule.timeOfDay))
    : localDayOfMonth,
});

export const withLocalTimeOfDay = (schedule: Schedule, localTime: Dayjs): Schedule => {
  const { weekday, dayOfMonth } = getLocalScheduleParts(schedule);
  const scheduleWithTime: Schedule = { ...schedule, timeOfDay: localTime.utc().format('HH:mm') };

  if (weekday !== undefined) scheduleWithTime.weekday = getUtcWeekday(weekday, localTime);

  if (dayOfMonth !== undefined) {
    scheduleWithTime.dayOfMonth = getUtcDayOfMonth(dayOfMonth, localTime);
  }

  return scheduleWithTime;
};
