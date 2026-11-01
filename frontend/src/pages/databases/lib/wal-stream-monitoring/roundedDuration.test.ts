import { describe, expect, it } from 'vitest';

import { getMeasurementAge, getRoundedDuration, getSecondsBetween } from './roundedDuration';

describe('getRoundedDuration', () => {
  it.each([
    [20, 'backups.physical.monitoring.durations.minutes', { minutes: 1 }],
    [40 * 60 + 20, 'backups.physical.monitoring.durations.minutes', { minutes: 40 }],
    [59 * 60 + 10, 'backups.physical.monitoring.durations.minutes', { minutes: 59 }],
    [59 * 60 + 40, 'backups.physical.monitoring.durations.hoursMinutes', { hours: 1, minutes: 0 }],
    [
      2 * 3600 + 15 * 60 + 10,
      'backups.physical.monitoring.durations.hoursMinutes',
      { hours: 2, minutes: 15 },
    ],
    [
      23 * 3600 + 59 * 60,
      'backups.physical.monitoring.durations.hoursMinutes',
      { hours: 23, minutes: 59 },
    ],
    [
      3 * 86400 + 4 * 3600 + 20 * 60,
      'backups.physical.monitoring.durations.daysHours',
      { days: 3, hours: 4 },
    ],
  ])('rounds %d seconds for reading', (seconds, expectedKey, expectedParams) => {
    expect(getRoundedDuration(seconds)).toEqual({ key: expectedKey, params: expectedParams });
  });
});

describe('getSecondsBetween', () => {
  const now = new Date('2026-10-18T12:00:00Z');

  it('counts the seconds from an earlier time', () => {
    expect(getSecondsBetween('2026-10-18T11:59:20Z', now)).toBe(40);
  });

  it('never goes below zero for a time slightly in the future', () => {
    expect(getSecondsBetween('2026-10-18T12:00:05Z', now)).toBe(0);
  });

  it.each([['not a time'], [''], [undefined as unknown as string]])(
    'is null for a time it cannot read (%j), so no "NaN" reaches the screen',
    (earlier) => {
      expect(getSecondsBetween(earlier, now)).toBeNull();
    },
  );

  it('is null when now itself is invalid', () => {
    expect(getSecondsBetween('2026-10-18T11:59:20Z', new Date(Number.NaN))).toBeNull();
  });
});

describe('getMeasurementAge', () => {
  it('counts seconds under a minute', () => {
    expect(getMeasurementAge(12.7)).toEqual({
      key: 'backups.physical.monitoring.durations.seconds',
      params: { seconds: 12 },
    });
  });

  it('rounds like any other duration from a minute on', () => {
    expect(getMeasurementAge(3 * 60)).toEqual(getRoundedDuration(3 * 60));
  });
});
