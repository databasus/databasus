import { describe, expect, it } from 'vitest';

import { isScheduleComplete } from './isScheduleComplete';
import type { Schedule } from './Schedule';
import { ScheduleType } from './ScheduleType';

describe('isScheduleComplete', () => {
  it.each<[string, Schedule]>([
    ['an hourly schedule', { type: ScheduleType.HOURLY }],
    ['a daily schedule with a time', { type: ScheduleType.DAILY, timeOfDay: '04:00' }],
    [
      'a weekly schedule with a day and a time',
      { type: ScheduleType.WEEKLY, timeOfDay: '04:00', weekday: 1 },
    ],
    [
      'a weekly schedule on weekday 0',
      { type: ScheduleType.WEEKLY, timeOfDay: '04:00', weekday: 0 },
    ],
    [
      'a monthly schedule with a day and a time',
      { type: ScheduleType.MONTHLY, timeOfDay: '04:00', dayOfMonth: 31 },
    ],
    [
      'a cron schedule the server accepts',
      { type: ScheduleType.CRON, cronExpression: '0 2 * * *' },
    ],
  ])('accepts %s', (_, schedule) => {
    expect(isScheduleComplete(schedule)).toBe(true);
  });

  it.each<[string, Schedule | null]>([
    ['no schedule', null],
    ['a daily schedule without a time', { type: ScheduleType.DAILY }],
    ['a weekly schedule without a time', { type: ScheduleType.WEEKLY, weekday: 1 }],
    ['a weekly schedule without a day', { type: ScheduleType.WEEKLY, timeOfDay: '04:00' }],
    ['a monthly schedule without a time', { type: ScheduleType.MONTHLY, dayOfMonth: 1 }],
    ['a monthly schedule without a day', { type: ScheduleType.MONTHLY, timeOfDay: '04:00' }],
    ['a cron schedule without an expression', { type: ScheduleType.CRON }],
    ['a cron schedule of six fields', { type: ScheduleType.CRON, cronExpression: '0 0 2 * * *' }],
    [
      'a cron schedule the server refuses',
      { type: ScheduleType.CRON, cronExpression: '0 0 L * *' },
    ],
    ['a cron schedule with no date', { type: ScheduleType.CRON, cronExpression: '0 0 31 2 *' }],
  ])('refuses %s', (_, schedule) => {
    expect(isScheduleComplete(schedule)).toBe(false);
  });
});
