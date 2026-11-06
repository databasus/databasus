import type { Weekday } from '@/entities/schedules/model/Weekday';
import type { TranslationKey } from '@/shared/i18n';

// Short weekday names. Not taken from Intl or dayjs: several languages return them in lower case or
// with a trailing period, which would need a casing fix-up per language.
export const WEEKDAY_LABEL_KEYS: Record<Weekday, TranslationKey> = {
  1: 'schedules.weekdays.mon',
  2: 'schedules.weekdays.tue',
  3: 'schedules.weekdays.wed',
  4: 'schedules.weekdays.thu',
  5: 'schedules.weekdays.fri',
  6: 'schedules.weekdays.sat',
  7: 'schedules.weekdays.sun',
};
