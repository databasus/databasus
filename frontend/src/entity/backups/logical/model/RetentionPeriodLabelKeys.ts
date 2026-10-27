import type { TranslationKey } from '../../../../shared/i18n';
import { RetentionPeriod } from './RetentionPeriod';

export const RETENTION_PERIOD_LABEL_KEYS: Record<RetentionPeriod, TranslationKey> = {
  [RetentionPeriod.DAY]: 'intervals.periods.day',
  [RetentionPeriod.WEEK]: 'intervals.periods.week',
  [RetentionPeriod.MONTH]: 'intervals.periods.month',
  [RetentionPeriod.THREE_MONTHS]: 'intervals.periods.threeMonths',
  [RetentionPeriod.SIX_MONTHS]: 'intervals.periods.sixMonths',
  [RetentionPeriod.YEAR]: 'intervals.periods.year',
  [RetentionPeriod.TWO_YEARS]: 'intervals.periods.twoYears',
  [RetentionPeriod.THREE_YEARS]: 'intervals.periods.threeYears',
  [RetentionPeriod.FOUR_YEARS]: 'intervals.periods.fourYears',
  [RetentionPeriod.FIVE_YEARS]: 'intervals.periods.fiveYears',
  [RetentionPeriod.FOREVER]: 'intervals.periods.forever',
};
