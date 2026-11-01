import { describe, expect, it } from 'vitest';

import type { LogicalBackupConfigInput } from '../../../entities/backup-configs';
import { buildLogicalBackupConfig, zeroRetention } from './buildLogicalBackupConfig.fixture';
import { isLogicalBackupConfigComplete } from './isLogicalBackupConfigComplete';
import { LogicalRetentionPolicy } from './LogicalRetentionPolicy';

describe('isLogicalBackupConfigComplete', () => {
  const hourlyOnlyGfs = { ...zeroRetention, policy: LogicalRetentionPolicy.GFS, gfsHours: 5 };

  it('accepts hourly-only GFS on an hourly schedule', () => {
    expect(
      isLogicalBackupConfigComplete(
        buildLogicalBackupConfig({}, { schedule: { type: 'HOURLY' }, retention: hourlyOnlyGfs }),
      ),
    ).toBe(true);
  });

  it('refuses hourly-only GFS on a daily schedule, where the hourly period is hidden', () => {
    expect(
      isLogicalBackupConfigComplete(buildLogicalBackupConfig({}, { retention: hourlyOnlyGfs })),
    ).toBe(false);
  });

  it('checks retention with backups off', () => {
    expect(
      isLogicalBackupConfigComplete(
        buildLogicalBackupConfig(
          { isBackupsEnabled: false },
          { retention: { ...zeroRetention, policy: LogicalRetentionPolicy.COUNT } },
        ),
      ),
    ).toBe(false);
  });

  it('skips the schedule and encryption with backups off', () => {
    const incompleteConfig = buildLogicalBackupConfig(
      { encryption: '' as LogicalBackupConfigInput['encryption'] },
      { schedule: { type: 'DAILY' } },
    );

    expect(isLogicalBackupConfigComplete(incompleteConfig)).toBe(false);
    expect(isLogicalBackupConfigComplete({ ...incompleteConfig, isBackupsEnabled: false })).toBe(
      true,
    );
  });

  it('refuses an empty encryption only with backups on', () => {
    const configWithoutEncryption = buildLogicalBackupConfig({
      encryption: '' as LogicalBackupConfigInput['encryption'],
    });

    expect(isLogicalBackupConfigComplete(configWithoutEncryption)).toBe(false);
    expect(
      isLogicalBackupConfigComplete({ ...configWithoutEncryption, isBackupsEnabled: false }),
    ).toBe(true);
  });

  it('does not require a storage', () => {
    expect(isLogicalBackupConfigComplete(buildLogicalBackupConfig({ storageId: null }))).toBe(true);
  });
});
