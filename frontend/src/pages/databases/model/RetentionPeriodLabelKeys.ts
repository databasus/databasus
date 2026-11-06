import { RetentionPeriod } from '@/pages/databases/model/RetentionPeriod';
import type { TranslationKey } from '@/shared/i18n';

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
