import { getNextCronRun } from './getNextCronRun';
import type { Schedule } from './Schedule';
import { ScheduleType } from './ScheduleType';

export const isScheduleComplete = (schedule: Schedule | null): boolean => {
  if (!schedule) return false;

  const hasTimeOfDay = Boolean(schedule.timeOfDay);

  switch (schedule.type) {
    case ScheduleType.HOURLY:
      return true;
    case ScheduleType.DAILY:
      return hasTimeOfDay;
    case ScheduleType.WEEKLY:
      return hasTimeOfDay && typeof schedule.weekday === 'number';
    case ScheduleType.MONTHLY:
      return hasTimeOfDay && typeof schedule.dayOfMonth === 'number';
    case ScheduleType.CRON:
      return getNextCronRun(schedule.cronExpression ?? '') !== null;
  }
};
