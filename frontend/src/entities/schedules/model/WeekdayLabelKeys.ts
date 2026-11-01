import type { TranslationKey } from '../../../shared/i18n';
import type { Weekday } from './Weekday';

// Short weekday names. Not taken from Intl or dayjs: several languages return them in lower case or
// with a trailing period, which would need a casing fix-up per language.
export const WEEKDAY_LABEL_KEYS: Record<Weekday, TranslationKey> = {
  1: 'intervals.weekdays.mon',
  2: 'intervals.weekdays.tue',
  3: 'intervals.weekdays.wed',
  4: 'intervals.weekdays.thu',
  5: 'intervals.weekdays.fri',
  6: 'intervals.weekdays.sat',
  7: 'intervals.weekdays.sun',
};
