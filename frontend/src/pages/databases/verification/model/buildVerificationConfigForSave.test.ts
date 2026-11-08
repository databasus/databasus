import { describe, expect, it } from 'vitest';

import type { Schedule } from '@/entities/schedules';
import { buildVerificationConfigForSave } from '@/pages/databases/verification/model/buildVerificationConfigForSave';
import type { VerificationConfig } from '@/pages/databases/verification/model/verificationContract';

const acknowledgedConfig: VerificationConfig = {
  databaseId: 'database-id',
  isAutomaticVerificationEnabled: true,
  scheduleType: 'INTERVAL',
  schedule: { type: 'WEEKLY', timeOfDay: '04:00', weekday: 1 },
  notificationTriggers: ['VERIFICATION_FAILED'],
};
const invalidCronSchedule: Schedule = { type: 'CRON', cronExpression: '0 0 31 2 *' };

describe('buildVerificationConfigForSave', () => {
  it('sends the draft when verification runs on a schedule', () => {
    const editedSchedule: Schedule = { type: 'DAILY', timeOfDay: '05:00' };

    expect(
      buildVerificationConfigForSave({
        acknowledgedConfig,
        draft: {
          ...acknowledgedConfig,
          schedule: editedSchedule,
          notificationTriggers: ['VERIFICATION_SUCCESS'],
        },
      }),
    ).toEqual({
      isAutomaticVerificationEnabled: true,
      scheduleType: 'INTERVAL',
      schedule: editedSchedule,
      notificationTriggers: ['VERIFICATION_SUCCESS'],
    });
  });

  it('sends the acknowledged schedule when verification runs after a backup', () => {
    expect(
      buildVerificationConfigForSave({
        acknowledgedConfig,
        draft: {
          ...acknowledgedConfig,
          scheduleType: 'AFTER_BACKUP',
          schedule: invalidCronSchedule,
          notificationTriggers: ['VERIFICATION_SUCCESS'],
        },
      }),
    ).toEqual({
      isAutomaticVerificationEnabled: true,
      scheduleType: 'AFTER_BACKUP',
      schedule: acknowledgedConfig.schedule,
      notificationTriggers: ['VERIFICATION_SUCCESS'],
    });
  });

  it('keeps every acknowledged value after a draft chose after-backup and turned verification off', () => {
    expect(
      buildVerificationConfigForSave({
        acknowledgedConfig,
        draft: {
          ...acknowledgedConfig,
          isAutomaticVerificationEnabled: false,
          scheduleType: 'AFTER_BACKUP',
          schedule: invalidCronSchedule,
          notificationTriggers: [],
        },
      }),
    ).toEqual({
      isAutomaticVerificationEnabled: false,
      scheduleType: 'INTERVAL',
      schedule: acknowledgedConfig.schedule,
      notificationTriggers: ['VERIFICATION_FAILED'],
    });
  });
});
