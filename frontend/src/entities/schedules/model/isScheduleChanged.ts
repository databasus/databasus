import type { Schedule } from './Schedule';
import { ScheduleType } from './ScheduleType';

interface ScheduleComparison {
  acknowledged: Schedule;
  shown: Schedule;
}

// Compares only the fields the shown type uses, so values left from another type never count.
export const isScheduleChanged = ({ acknowledged, shown }: ScheduleComparison): boolean => {
  if (acknowledged.type !== shown.type) return true;

  switch (shown.type) {
    case ScheduleType.HOURLY:
      return false;
    case ScheduleType.DAILY:
      return acknowledged.timeOfDay !== shown.timeOfDay;
    case ScheduleType.WEEKLY:
      return acknowledged.timeOfDay !== shown.timeOfDay || acknowledged.weekday !== shown.weekday;
    case ScheduleType.MONTHLY:
      return (
        acknowledged.timeOfDay !== shown.timeOfDay || acknowledged.dayOfMonth !== shown.dayOfMonth
      );
    case ScheduleType.CRON:
      return acknowledged.cronExpression !== shown.cronExpression;
  }
};
