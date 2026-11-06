import { getNextCronRun } from '@/entities/schedules/model/getNextCronRun';
import type { Schedule } from '@/entities/schedules/model/Schedule';
import { ScheduleType } from '@/entities/schedules/model/ScheduleType';

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
