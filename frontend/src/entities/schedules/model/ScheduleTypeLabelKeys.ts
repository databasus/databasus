import type { TranslationKey } from '../../../shared/i18n';
import { ScheduleType } from './ScheduleType';

export const SCHEDULE_TYPE_LABEL_KEYS: Record<ScheduleType, TranslationKey> = {
  [ScheduleType.HOURLY]: 'schedules.types.hourly',
  [ScheduleType.DAILY]: 'schedules.types.daily',
  [ScheduleType.WEEKLY]: 'schedules.types.weekly',
  [ScheduleType.MONTHLY]: 'schedules.types.monthly',
  [ScheduleType.CRON]: 'schedules.types.cron',
};
