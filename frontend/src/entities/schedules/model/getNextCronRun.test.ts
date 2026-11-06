import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { getNextCronRun } from '@/entities/schedules/model/getNextCronRun';

describe('getNextCronRun', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-11-01T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it.each([
    ['0 2 * * *', '2026-11-02T02:00:00.000Z'],
    ['0 0 ? * *', '2026-11-02T00:00:00.000Z'],
    ['0 0 * * MON', '2026-11-02T00:00:00.000Z'],
    ['*/5 1-3 1,15 JAN mon-fri', '2027-01-01T01:00:00.000Z'],
    ['*/5 1-3 1,15 JUL mon-fri', '2027-07-01T01:00:00.000Z'],
    ['0 0 29 2 *', '2028-02-29T00:00:00.000Z'],
    [' 0 2 * * * ', '2026-11-02T02:00:00.000Z'],
    ['0 0 * * */7', '2026-11-08T00:00:00.000Z'],
  ])('returns the next run of %s, which the server accepts', (expression, nextRunIso) => {
    expect(getNextCronRun(expression)?.toISOString()).toBe(nextRunIso);
  });

  it.each([
    '',
    '0 0 2 * * *',
    '@daily',
    '0 2 * *',
    '0 0 L * *',
    '0 0 * * 1#2',
    '0 0 H * *',
    '0 0 * * 7',
    '0 0 * * 5-7',
    '0 25 * * *',
    'TZ=Europe/Moscow 0 2 * * *',
    'CRON_TZ=UTC 0 2 * * *',
    '0 0 31 2 *',
    '0 0 31 2,4 *',
    '0 0 31 FEB,APR *',
    'not a cron',
  ])(
    'returns nothing for %j, which the server refuses or the form must not offer',
    (expression) => {
      expect(getNextCronRun(expression)).toBeNull();
    },
  );
});
