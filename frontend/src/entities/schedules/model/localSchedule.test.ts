import dayjs from 'dayjs';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  getLocalScheduleParts,
  withLocalDayOfMonth,
  withLocalTimeOfDay,
  withLocalWeekday,
} from './localSchedule';
import { ScheduleType } from './ScheduleType';

describe('local schedule at UTC+3 with the clock at 01:00 local', () => {
  beforeEach(() => {
    vi.stubEnv('TZ', 'Europe/Moscow');
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-10-27T22:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllEnvs();
  });

  it('keeps a weekday chosen before a time and converts it with the time', () => {
    const chosen = withLocalWeekday({ type: ScheduleType.WEEKLY }, 1);

    expect(chosen.weekday).toBe(1);
    expect(getLocalScheduleParts(chosen).weekday).toBe(1);

    const timed = withLocalTimeOfDay(chosen, dayjs().hour(12).minute(0));

    expect(timed).toEqual({ type: ScheduleType.WEEKLY, timeOfDay: '09:00', weekday: 1 });
  });

  it('keeps a day of the month chosen before a time and converts it with the time', () => {
    const chosen = withLocalDayOfMonth({ type: ScheduleType.MONTHLY }, 10);

    expect(chosen.dayOfMonth).toBe(10);
    expect(getLocalScheduleParts(chosen).dayOfMonth).toBe(10);

    const timed = withLocalTimeOfDay(chosen, dayjs().hour(2).minute(0));

    expect(timed).toEqual({ type: ScheduleType.MONTHLY, timeOfDay: '23:00', dayOfMonth: 9 });
  });

  it('keeps the local day when only the time changes', () => {
    const shownOnMonday = { type: ScheduleType.WEEKLY, timeOfDay: '09:00', weekday: 1 };

    const changed = withLocalTimeOfDay(shownOnMonday, dayjs().hour(2).minute(0));

    expect(changed).toEqual({ type: ScheduleType.WEEKLY, timeOfDay: '23:00', weekday: 7 });
    expect(getLocalScheduleParts(changed).weekday).toBe(1);
    expect(getLocalScheduleParts(changed).localTime?.format('HH:mm')).toBe('02:00');
  });

  it('converts a weekday chosen when a time exists', () => {
    const shownAtNight = { type: ScheduleType.WEEKLY, timeOfDay: '23:00', weekday: 7 };

    expect(withLocalWeekday(shownAtNight, 3).weekday).toBe(2);
  });

  it('shows a saved weekday 0 without a time as Sunday', () => {
    expect(getLocalScheduleParts({ type: ScheduleType.WEEKLY, weekday: 0 }).weekday).toBe(7);
  });

  it('leaves the parts empty for a schedule with none of them', () => {
    expect(getLocalScheduleParts({ type: ScheduleType.HOURLY })).toEqual({
      localTime: undefined,
      weekday: undefined,
      dayOfMonth: undefined,
    });
  });
});
