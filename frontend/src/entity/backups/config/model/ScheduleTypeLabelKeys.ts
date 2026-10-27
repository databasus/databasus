import type { TranslationKey } from '../../../../shared/i18n';
import { ScheduleType } from './ScheduleType';

export const SCHEDULE_TYPE_LABEL_KEYS: Record<ScheduleType, TranslationKey> = {
  [ScheduleType.HOURLY]: 'intervals.types.hourly',
  [ScheduleType.DAILY]: 'intervals.types.daily',
  [ScheduleType.WEEKLY]: 'intervals.types.weekly',
  [ScheduleType.MONTHLY]: 'intervals.types.monthly',
  [ScheduleType.CRON]: 'intervals.types.cron',
};
