import type { IntervalType } from './IntervalType';

export interface Interval {
  type: IntervalType;
  timeOfDay: string;
  weekday?: number;
  dayOfMonth?: number;
  cronExpression?: string;
}
