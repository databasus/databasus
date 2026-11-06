import type { ApiComponents } from '@/shared/api';

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
