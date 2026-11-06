import { type Schedule, ScheduleType } from '@/entities/schedules';

export const isGfsHoursShown = (schedule: Schedule): boolean =>
  schedule.type === ScheduleType.HOURLY || schedule.type === ScheduleType.CRON;
