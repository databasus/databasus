import dayjs, { Dayjs } from 'dayjs';
import customParseFormat from 'dayjs/plugin/customParseFormat';
import utc from 'dayjs/plugin/utc';

dayjs.extend(utc);
dayjs.extend(customParseFormat);

// Detect whether the user's locale prefers 12-hour time
export const getUserTimeFormat = () => {
  const locale = navigator.language || 'en-US';
  return /[AP]M/.test(new Date(2023, 0, 1, 13).toLocaleTimeString(locale, { hour: 'numeric' }));
};

// UTC weekday (1–7) + HH:mm → local weekday (1–7)
export const getLocalWeekday = (utcWeekday: number, utcTime: string): number => {
  const [h, m] = utcTime.split(':').map(Number);
  const local = dayjs
    .utc()
    .day(utcWeekday % 7)
    .hour(h)
    .minute(m)
    .local();
  return ((local.day() + 6) % 7) + 1; // 1 = Mon … 7 = Sun
};

// Local weekday (1–7) + local time → UTC weekday (1–7)
export const getUtcWeekday = (localWeekday: number, localTime: Dayjs): number => {
  const utc = dayjs()
    .day(localWeekday % 7)
    .hour(localTime.hour())
    .minute(localTime.minute())
    .utc();
  return ((utc.day() + 6) % 7) + 1;
};

const MINUTES_IN_DAY = 24 * 60;
const DAYS_IN_MONTHLY_CYCLE = 31;

// A monthly day wraps on a 31-day cycle instead of a calendar month, so the length of the current
// month never changes a stored day and every day survives being shown and saved again.
const shiftDayOfMonth = ({
  dayOfMonth,
  minutesAfterStartOfDay,
}: {
  dayOfMonth: number;
  minutesAfterStartOfDay: number;
}): number => {
  const dayShift = Math.floor(minutesAfterStartOfDay / MINUTES_IN_DAY);

  return ((dayOfMonth + dayShift - 1 + DAYS_IN_MONTHLY_CYCLE) % DAYS_IN_MONTHLY_CYCLE) + 1;
};

export const getLocalDayOfMonth = (utcDayOfMonth: number, utcTime: string): number => {
  const [h, m] = utcTime.split(':').map(Number);
  const utcOffset = dayjs.utc(utcTime, 'HH:mm').local().utcOffset();

  return shiftDayOfMonth({
    dayOfMonth: utcDayOfMonth,
    minutesAfterStartOfDay: h * 60 + m + utcOffset,
  });
};

export const getUtcDayOfMonth = (localDayOfMonth: number, localTime: Dayjs): number =>
  shiftDayOfMonth({
    dayOfMonth: localDayOfMonth,
    minutesAfterStartOfDay: localTime.hour() * 60 + localTime.minute() - localTime.utcOffset(),
  });
