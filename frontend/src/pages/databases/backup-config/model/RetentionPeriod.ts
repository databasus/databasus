import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type RetentionPeriod = ApiComponents['schemas']['RetentionPeriod'];

export const RetentionPeriod = {
  DAY: 'DAY',
  WEEK: 'WEEK',
  MONTH: 'MONTH',
  THREE_MONTHS: 'THREE_MONTHS',
  SIX_MONTHS: 'SIX_MONTHS',
  YEAR: 'YEAR',
  TWO_YEARS: 'TWO_YEARS',
  THREE_YEARS: 'THREE_YEARS',
  FOUR_YEARS: 'FOUR_YEARS',
  FIVE_YEARS: 'FIVE_YEARS',
  FOREVER: 'FOREVER',
} as const satisfies Record<string, RetentionPeriod>;

export const RETENTION_PERIOD_LABEL_KEYS: Record<RetentionPeriod, TranslationKey> = {
  [RetentionPeriod.DAY]: 'backups.logical.retentionPeriods.day',
  [RetentionPeriod.WEEK]: 'backups.logical.retentionPeriods.week',
  [RetentionPeriod.MONTH]: 'backups.logical.retentionPeriods.month',
  [RetentionPeriod.THREE_MONTHS]: 'backups.logical.retentionPeriods.threeMonths',
  [RetentionPeriod.SIX_MONTHS]: 'backups.logical.retentionPeriods.sixMonths',
  [RetentionPeriod.YEAR]: 'backups.logical.retentionPeriods.year',
  [RetentionPeriod.TWO_YEARS]: 'backups.logical.retentionPeriods.twoYears',
  [RetentionPeriod.THREE_YEARS]: 'backups.logical.retentionPeriods.threeYears',
  [RetentionPeriod.FOUR_YEARS]: 'backups.logical.retentionPeriods.fourYears',
  [RetentionPeriod.FIVE_YEARS]: 'backups.logical.retentionPeriods.fiveYears',
  [RetentionPeriod.FOREVER]: 'backups.logical.retentionPeriods.forever',
};
