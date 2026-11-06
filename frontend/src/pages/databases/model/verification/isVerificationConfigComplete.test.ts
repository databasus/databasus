import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { Schedule } from '@/entities/schedules';
import { isVerificationConfigComplete } from '@/pages/databases/model/verification/isVerificationConfigComplete';
import type { VerificationConfig } from '@/pages/databases/model/verification/VerificationConfig';

const buildConfig = (overrides: Partial<VerificationConfig>): VerificationConfig => ({
  databaseId: 'database-id',
  isAutomaticVerificationEnabled: true,
  scheduleType: 'INTERVAL',
  schedule: { type: 'HOURLY' },
  notificationTriggers: [],
  ...overrides,
});

const INVALID_CRON_SCHEDULE: Schedule = { type: 'CRON', cronExpression: '0 0 31 2 *' };

describe('isVerificationConfigComplete', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-10-27T22:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('accepts an invalid schedule when verification is off', () => {
    expect(
      isVerificationConfigComplete(
        buildConfig({ isAutomaticVerificationEnabled: false, schedule: INVALID_CRON_SCHEDULE }),
      ),
    ).toBe(true);
  });

  it('accepts an invalid schedule when verification runs after a backup', () => {
    expect(
      isVerificationConfigComplete(
        buildConfig({ scheduleType: 'AFTER_BACKUP', schedule: INVALID_CRON_SCHEDULE }),
      ),
    ).toBe(true);
  });

  it.each<{ scenario: string; schedule: Schedule; isComplete: boolean }>([
    { scenario: 'accepts an hourly schedule', schedule: { type: 'HOURLY' }, isComplete: true },
    {
      scenario: 'accepts a daily schedule with a time',
      schedule: { type: 'DAILY', timeOfDay: '04:00' },
      isComplete: true,
    },
    {
      scenario: 'refuses a daily schedule without a time',
      schedule: { type: 'DAILY' },
      isComplete: false,
    },
    {
      scenario: 'accepts a weekly schedule with a weekday and a time',
      schedule: { type: 'WEEKLY', timeOfDay: '04:00', weekday: 1 },
      isComplete: true,
    },
    {
      scenario: 'refuses a weekly schedule without a weekday',
      schedule: { type: 'WEEKLY', timeOfDay: '04:00' },
      isComplete: false,
    },
    {
      scenario: 'accepts a monthly schedule with a day and a time',
      schedule: { type: 'MONTHLY', timeOfDay: '04:00', dayOfMonth: 1 },
      isComplete: true,
    },
    {
      scenario: 'refuses a monthly schedule without a day',
      schedule: { type: 'MONTHLY', timeOfDay: '04:00' },
      isComplete: false,
    },
    {
      scenario: 'accepts a valid cron expression',
      schedule: { type: 'CRON', cronExpression: '0 2 * * *' },
      isComplete: true,
    },
    {
      scenario: 'refuses a cron expression with no date to run on',
      schedule: INVALID_CRON_SCHEDULE,
      isComplete: false,
    },
  ])('$scenario when verification runs on a schedule', ({ schedule, isComplete }) => {
    expect(isVerificationConfigComplete(buildConfig({ schedule }))).toBe(isComplete);
  });
});
