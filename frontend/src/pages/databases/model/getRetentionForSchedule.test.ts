import { describe, expect, it } from 'vitest';

import { buildLogicalBackupConfig } from '@/pages/databases/model/buildLogicalBackupConfig.fixture';
import { getRetentionForSchedule } from '@/pages/databases/model/getRetentionForSchedule';
import { LogicalRetentionPolicy } from '@/pages/databases/model/LogicalRetentionPolicy';

describe('getRetentionForSchedule', () => {
  const gfsSettings = buildLogicalBackupConfig(
    {},
    { schedule: { type: 'HOURLY' } },
    { policy: LogicalRetentionPolicy.GFS, gfsHours: 5 },
  ).settings;

  it('resets the hours when a schedule type stops showing them', () => {
    expect(
      getRetentionForSchedule(gfsSettings, { type: 'DAILY', timeOfDay: '04:00' }).gfsHours,
    ).toBe(24);
  });

  it('keeps the hours when the schedule still shows them', () => {
    expect(getRetentionForSchedule(gfsSettings, { type: 'CRON' }).gfsHours).toBe(5);
  });
});
