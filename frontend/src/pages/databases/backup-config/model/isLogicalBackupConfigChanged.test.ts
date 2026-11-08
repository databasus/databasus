import { describe, expect, it } from 'vitest';

import { buildLogicalBackupConfig } from '@/pages/databases/backup-config/model/buildLogicalBackupConfig.fixture';
import { isLogicalBackupConfigChanged } from '@/pages/databases/backup-config/model/isLogicalBackupConfigChanged';
import { LogicalNotificationTrigger } from '@/pages/databases/backup-config/model/LogicalNotificationTrigger';
import { LogicalRetentionPolicy } from '@/pages/databases/backup-config/model/LogicalRetentionPolicy';

describe('isLogicalBackupConfigChanged', () => {
  it('compares notification triggers as a set', () => {
    const acknowledged = buildLogicalBackupConfig(
      {},
      {
        notificationTriggers: [
          LogicalNotificationTrigger.BACKUP_FAILED,
          LogicalNotificationTrigger.BACKUP_SUCCESS,
        ],
      },
    );
    const reordered = buildLogicalBackupConfig(
      {},
      {
        notificationTriggers: [
          LogicalNotificationTrigger.BACKUP_SUCCESS,
          LogicalNotificationTrigger.BACKUP_FAILED,
        ],
      },
    );

    expect(isLogicalBackupConfigChanged({ acknowledged, shown: reordered })).toBe(false);
  });

  it('ignores the fields of a policy that is not chosen', () => {
    expect(
      isLogicalBackupConfigChanged({
        acknowledged: buildLogicalBackupConfig(),
        shown: buildLogicalBackupConfig({}, {}, { count: 7, gfsDays: 99 }),
      }),
    ).toBe(false);
  });

  it('counts a change of the chosen policy', () => {
    expect(
      isLogicalBackupConfigChanged({
        acknowledged: buildLogicalBackupConfig(
          {},
          {},
          { policy: LogicalRetentionPolicy.COUNT, count: 5 },
        ),
        shown: buildLogicalBackupConfig({}, {}, { policy: LogicalRetentionPolicy.COUNT, count: 6 }),
      }),
    ).toBe(true);
  });

  it('ignores hidden hours and counts shown ones', () => {
    const gfs = { policy: LogicalRetentionPolicy.GFS, gfsHours: 5 };
    const moreHours = { policy: LogicalRetentionPolicy.GFS, gfsHours: 6 };

    expect(
      isLogicalBackupConfigChanged({
        acknowledged: buildLogicalBackupConfig({}, {}, gfs),
        shown: buildLogicalBackupConfig({}, {}, moreHours),
      }),
    ).toBe(false);
    expect(
      isLogicalBackupConfigChanged({
        acknowledged: buildLogicalBackupConfig({}, { schedule: { type: 'HOURLY' } }, gfs),
        shown: buildLogicalBackupConfig({}, { schedule: { type: 'HOURLY' } }, moreHours),
      }),
    ).toBe(true);
  });

  it('ignores schedule fields the type does not show', () => {
    expect(
      isLogicalBackupConfigChanged({
        acknowledged: buildLogicalBackupConfig(
          {},
          { schedule: { type: 'HOURLY', timeOfDay: '12:00' } },
        ),
        shown: buildLogicalBackupConfig({}, { schedule: { type: 'HOURLY' } }),
      }),
    ).toBe(false);
  });

  it('counts a change of the time period', () => {
    expect(
      isLogicalBackupConfigChanged({
        acknowledged: buildLogicalBackupConfig({}, {}, { period: 'MONTH' }),
        shown: buildLogicalBackupConfig({}, {}, { period: 'WEEK' }),
      }),
    ).toBe(true);
  });

  it('ignores differences hidden with backups off', () => {
    expect(
      isLogicalBackupConfigChanged({
        acknowledged: buildLogicalBackupConfig({ isBackupsEnabled: false }),
        shown: buildLogicalBackupConfig(
          { isBackupsEnabled: false },
          {
            schedule: { type: 'HOURLY' },
            notificationTriggers: [],
            isRetryIfFailed: false,
          },
        ),
      }),
    ).toBe(false);
  });

  it('ignores the number of tries with retry off', () => {
    expect(
      isLogicalBackupConfigChanged({
        acknowledged: buildLogicalBackupConfig(
          {},
          { isRetryIfFailed: false, maxFailedTriesCount: 3 },
        ),
        shown: buildLogicalBackupConfig({}, { isRetryIfFailed: false, maxFailedTriesCount: 7 }),
      }),
    ).toBe(false);
  });

  it('counts a trigger added or removed', () => {
    expect(
      isLogicalBackupConfigChanged({
        acknowledged: buildLogicalBackupConfig({}, { notificationTriggers: [] }),
        shown: buildLogicalBackupConfig(
          {},
          { notificationTriggers: [LogicalNotificationTrigger.BACKUP_FAILED] },
        ),
      }),
    ).toBe(true);
  });

  it('counts a changed number of tries with retry on', () => {
    expect(
      isLogicalBackupConfigChanged({
        acknowledged: buildLogicalBackupConfig({}, { maxFailedTriesCount: 3 }),
        shown: buildLogicalBackupConfig({}, { maxFailedTriesCount: 7 }),
      }),
    ).toBe(true);
  });

  it('counts a retention change with backups off', () => {
    expect(
      isLogicalBackupConfigChanged({
        acknowledged: buildLogicalBackupConfig(
          { isBackupsEnabled: false },
          {},
          { policy: LogicalRetentionPolicy.COUNT, count: 5 },
        ),
        shown: buildLogicalBackupConfig(
          { isBackupsEnabled: false },
          {},
          { policy: LogicalRetentionPolicy.COUNT, count: 6 },
        ),
      }),
    ).toBe(true);
  });
});
