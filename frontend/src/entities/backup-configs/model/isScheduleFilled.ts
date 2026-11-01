import type { Schedule } from './Schedule';
import { ScheduleType } from './ScheduleType';

export const isScheduleFilled = (schedule?: Schedule | null): boolean => {
  if (!schedule?.type) return false;

  if (schedule.type === ScheduleType.WEEKLY) return Boolean(schedule.weekday);
  if (schedule.type === ScheduleType.MONTHLY) return Boolean(schedule.dayOfMonth);
  if (schedule.type === ScheduleType.CRON) return Boolean(schedule.cronExpression);

  return true;
};
