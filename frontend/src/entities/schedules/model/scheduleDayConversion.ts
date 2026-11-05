import dayjs, { Dayjs } from 'dayjs';
import customParseFormat from 'dayjs/plugin/customParseFormat';
import utc from 'dayjs/plugin/utc';

dayjs.extend(utc);
dayjs.extend(customParseFormat);

const MINUTES_IN_DAY = 24 * 60;
const DAYS_IN_WEEK = 7;
const DAYS_IN_MONTHLY_CYCLE = 31;

// A day wraps on a fixed cycle instead of a calendar week or month, so neither the length of the current
// month nor an offset change on another date moves a stored day, and every day survives being shown and
// saved again.
const shiftDay = ({
  day,
  minutesAfterStartOfDay,
  daysInCycle,
}: {
  day: number;
  minutesAfterStartOfDay: number;
  daysInCycle: number;
}): number => {
  const dayShift = Math.floor(minutesAfterStartOfDay / MINUTES_IN_DAY);

  return ((day + dayShift - 1 + daysInCycle) % daysInCycle) + 1;
};

const getMinutesAfterLocalStartOfDay = (utcTime: string): number => {
  const [h, m] = utcTime.split(':').map(Number);
  const utcOffset = dayjs.utc(utcTime, 'HH:mm').local().utcOffset();

  return h * 60 + m + utcOffset;
};

const getMinutesAfterUtcStartOfDay = (localTime: Dayjs): number =>
  localTime.hour() * 60 + localTime.minute() - localTime.utcOffset();

export const getLocalWeekday = (utcWeekday: number, utcTime: string): number =>
  shiftDay({
    day: utcWeekday,
    minutesAfterStartOfDay: getMinutesAfterLocalStartOfDay(utcTime),
    daysInCycle: DAYS_IN_WEEK,
  });

export const getUtcWeekday = (localWeekday: number, localTime: Dayjs): number =>
  shiftDay({
    day: localWeekday,
    minutesAfterStartOfDay: getMinutesAfterUtcStartOfDay(localTime),
    daysInCycle: DAYS_IN_WEEK,
  });

export const getLocalDayOfMonth = (utcDayOfMonth: number, utcTime: string): number =>
  shiftDay({
    day: utcDayOfMonth,
    minutesAfterStartOfDay: getMinutesAfterLocalStartOfDay(utcTime),
    daysInCycle: DAYS_IN_MONTHLY_CYCLE,
  });

export const getUtcDayOfMonth = (localDayOfMonth: number, localTime: Dayjs): number =>
  shiftDay({
    day: localDayOfMonth,
    minutesAfterStartOfDay: getMinutesAfterUtcStartOfDay(localTime),
    daysInCycle: DAYS_IN_MONTHLY_CYCLE,
  });
