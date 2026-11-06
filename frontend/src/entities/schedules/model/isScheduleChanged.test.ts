import { describe, expect, it } from 'vitest';

import { isScheduleChanged } from '@/entities/schedules/model/isScheduleChanged';
import type { Schedule } from '@/entities/schedules/model/Schedule';

describe('isScheduleChanged', () => {
  it.each<{ scenario: string; acknowledged: Schedule; shown: Schedule; isChanged: boolean }>([
    {
      scenario: 'reports a different type',
      acknowledged: { type: 'HOURLY' },
      shown: { type: 'DAILY' },
      isChanged: true,
    },
    {
      scenario: 'ignores a time left on an hourly schedule',
      acknowledged: { type: 'HOURLY' },
      shown: { type: 'HOURLY', timeOfDay: '04:00' },
      isChanged: false,
    },
    {
      scenario: 'reports a different time of a daily schedule',
      acknowledged: { type: 'DAILY', timeOfDay: '04:00' },
      shown: { type: 'DAILY', timeOfDay: '05:00' },
      isChanged: true,
    },
    {
      scenario: 'reports a different weekday of a weekly schedule',
      acknowledged: { type: 'WEEKLY', timeOfDay: '04:00', weekday: 1 },
      shown: { type: 'WEEKLY', timeOfDay: '04:00', weekday: 2 },
      isChanged: true,
    },
    {
      scenario: 'reports a different day of a monthly schedule',
      acknowledged: { type: 'MONTHLY', timeOfDay: '04:00', dayOfMonth: 1 },
      shown: { type: 'MONTHLY', timeOfDay: '04:00', dayOfMonth: 2 },
      isChanged: true,
    },
    {
      scenario: 'reports a different cron expression',
      acknowledged: { type: 'CRON', cronExpression: '0 2 * * *' },
      shown: { type: 'CRON', cronExpression: '0 3 * * *' },
      isChanged: true,
    },
    {
      scenario: 'ignores a time left on a cron schedule',
      acknowledged: { type: 'CRON', cronExpression: '0 2 * * *' },
      shown: { type: 'CRON', cronExpression: '0 2 * * *', timeOfDay: '04:00' },
      isChanged: false,
    },
  ])('$scenario', ({ acknowledged, shown, isChanged }) => {
    expect(isScheduleChanged({ acknowledged, shown })).toBe(isChanged);
  });
});
