import { describe, expect, it } from 'vitest';

import { isVerificationConfigChanged } from '@/pages/databases/model/verification/isVerificationConfigChanged';
import type { VerificationConfig } from '@/pages/databases/model/verification/VerificationConfig';

const acknowledged: VerificationConfig = {
  databaseId: 'database-id',
  isAutomaticVerificationEnabled: true,
  scheduleType: 'INTERVAL',
  schedule: { type: 'WEEKLY', timeOfDay: '04:00', weekday: 1 },
  notificationTriggers: ['VERIFICATION_FAILED', 'VERIFICATION_SUCCESS'],
};

describe('isVerificationConfigChanged', () => {
  it('reports no change for the acknowledged values', () => {
    expect(isVerificationConfigChanged({ acknowledged, shown: { ...acknowledged } })).toBe(false);
  });

  it('reports a turned off switch', () => {
    expect(
      isVerificationConfigChanged({
        acknowledged,
        shown: { ...acknowledged, isAutomaticVerificationEnabled: false },
      }),
    ).toBe(true);
  });

  it('reports a different schedule choice', () => {
    expect(
      isVerificationConfigChanged({
        acknowledged,
        shown: { ...acknowledged, scheduleType: 'AFTER_BACKUP' },
      }),
    ).toBe(true);
  });

  it('reports a different schedule while verification runs on it', () => {
    expect(
      isVerificationConfigChanged({
        acknowledged,
        shown: { ...acknowledged, schedule: { ...acknowledged.schedule, weekday: 2 } },
      }),
    ).toBe(true);
  });

  it('compares notification triggers as a set', () => {
    expect(
      isVerificationConfigChanged({
        acknowledged,
        shown: {
          ...acknowledged,
          notificationTriggers: ['VERIFICATION_SUCCESS', 'VERIFICATION_FAILED'],
        },
      }),
    ).toBe(false);
  });

  it('reports a different notification trigger', () => {
    expect(
      isVerificationConfigChanged({
        acknowledged,
        shown: { ...acknowledged, notificationTriggers: ['VERIFICATION_FAILED'] },
      }),
    ).toBe(true);
  });

  it('ignores a schedule hidden by running after a backup', () => {
    const acknowledgedAfterBackup: VerificationConfig = {
      ...acknowledged,
      scheduleType: 'AFTER_BACKUP',
    };

    expect(
      isVerificationConfigChanged({
        acknowledged: acknowledgedAfterBackup,
        shown: {
          ...acknowledgedAfterBackup,
          schedule: { type: 'CRON', cronExpression: '0 0 31 2 *' },
        },
      }),
    ).toBe(false);
  });

  it('ignores everything hidden when verification stays off', () => {
    const acknowledgedOff: VerificationConfig = {
      ...acknowledged,
      isAutomaticVerificationEnabled: false,
    };

    expect(
      isVerificationConfigChanged({
        acknowledged: acknowledgedOff,
        shown: {
          ...acknowledgedOff,
          scheduleType: 'AFTER_BACKUP',
          schedule: { type: 'HOURLY' },
          notificationTriggers: [],
        },
      }),
    ).toBe(false);
  });
});
