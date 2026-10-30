import type { TranslationKey } from '../../../shared/i18n';
import { IntervalType } from './IntervalType';

export const INTERVAL_TYPE_LABEL_KEYS: Record<IntervalType, TranslationKey> = {
  [IntervalType.HOURLY]: 'intervals.types.hourly',
  [IntervalType.DAILY]: 'intervals.types.daily',
  [IntervalType.WEEKLY]: 'intervals.types.weekly',
  [IntervalType.MONTHLY]: 'intervals.types.monthly',
  [IntervalType.CRON]: 'intervals.types.cron',
};
