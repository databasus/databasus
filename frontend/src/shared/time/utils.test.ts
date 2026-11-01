import dayjs from 'dayjs';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { getLocalDayOfMonth, getUtcDayOfMonth } from './utils';

const zones = ['UTC', 'Europe/Moscow', 'America/New_York'];
const moments = ['2026-10-15T12:00:00Z', '2026-11-15T12:00:00Z', '2027-02-15T12:00:00Z'];
const zoneMoments = zones.flatMap((zone) => moments.map((moment) => [zone, moment] as const));

const pinClock = ({ zone, moment }: { zone: string; moment: string }) => {
  vi.stubEnv('TZ', zone);
  vi.useFakeTimers({ toFake: ['Date'] });
  vi.setSystemTime(new Date(moment));
};

const getLocalTime = ({ hour, minute = 0 }: { hour: number; minute?: number }) =>
  dayjs().hour(hour).minute(minute);

describe('monthly day conversion', () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllEnvs();
  });

  it.each(zoneMoments)('shows a saved day 31 as 31 in %s at %s', (zone, moment) => {
    pinClock({ zone, moment });

    expect(getLocalDayOfMonth(31, '12:00')).toBe(31);
  });

  it.each(zoneMoments)('saves a chosen day 31 as 31 in %s at %s', (zone, moment) => {
    pinClock({ zone, moment });

    expect(getUtcDayOfMonth(31, getLocalTime({ hour: 12 }))).toBe(31);
  });

  it.each(zoneMoments)('keeps the day when the time changes in %s at %s', (zone, moment) => {
    pinClock({ zone, moment });

    const shownDay = getLocalDayOfMonth(31, '12:00');
    const savedDay = getUtcDayOfMonth(shownDay, getLocalTime({ hour: 13, minute: 30 }));

    expect(savedDay).toBe(31);
  });

  it.each(moments)('saves day 1 at 02:00 at UTC+3 as day 31 and shows it back at %s', (moment) => {
    pinClock({ zone: 'Europe/Moscow', moment });

    const localTime = getLocalTime({ hour: 2 });

    expect(localTime.utc().format('HH:mm')).toBe('23:00');
    expect(getUtcDayOfMonth(1, localTime)).toBe(31);
    expect(getLocalDayOfMonth(31, '23:00')).toBe(1);
  });

  it.each(['2026-11-15T12:00:00Z', '2027-02-15T12:00:00Z'])(
    'saves day 31 at 22:00 at UTC-5 as day 1 and shows it back at %s',
    (moment) => {
      pinClock({ zone: 'America/New_York', moment });

      const localTime = getLocalTime({ hour: 22 });

      expect(localTime.utc().format('HH:mm')).toBe('03:00');
      expect(getUtcDayOfMonth(31, localTime)).toBe(1);
      expect(getLocalDayOfMonth(1, '03:00')).toBe(31);
    },
  );

  it.each(zoneMoments)('round-trips every day at several times in %s at %s', (zone, moment) => {
    pinClock({ zone, moment });

    const localHours = [0, 2, 12, 22, 23];
    const days = Array.from({ length: 31 }, (_, index) => index + 1);

    for (const localHour of localHours) {
      const localTime = getLocalTime({ hour: localHour, minute: 30 });
      const utcTime = localTime.utc().format('HH:mm');

      for (const day of days) {
        expect(getLocalDayOfMonth(getUtcDayOfMonth(day, localTime), utcTime)).toBe(day);
      }
    }
  });

  it('round-trips a day on the daylight saving switch day in America/New_York', () => {
    pinClock({ zone: 'America/New_York', moment: '2026-11-01T12:00:00Z' });

    const shownDay = getLocalDayOfMonth(10, '04:30');
    const shownTime = dayjs.utc('04:30', 'HH:mm').local();

    expect(shownDay).toBe(10);
    expect(getUtcDayOfMonth(shownDay, shownTime)).toBe(10);
  });
});
