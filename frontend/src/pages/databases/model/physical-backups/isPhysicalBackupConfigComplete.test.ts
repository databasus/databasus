import { describe, expect, it } from 'vitest';

import { buildPhysicalBackupConfigDraft } from './buildPhysicalBackupConfigDraft.fixture';
import { isPhysicalBackupConfigComplete } from './isPhysicalBackupConfigComplete';
import type { PhysicalBackupConfigDraft } from './PhysicalBackupConfigDraft';

const NO_GFS_PERIODS = { gfsHours: 0, gfsDays: 0, gfsWeeks: 0, gfsMonths: 0, gfsYears: 0 };

describe('isPhysicalBackupConfigComplete', () => {
  it('requires a positive chain count when the policy keeps chains', () => {
    const buildChains = (chainsCount: number) =>
      buildPhysicalBackupConfigDraft({ retention: { policy: 'CHAINS', chainsCount } });

    expect(isPhysicalBackupConfigComplete(buildChains(1), 'FULL_INCREMENTAL')).toBe(true);
    expect(isPhysicalBackupConfigComplete(buildChains(0), 'FULL_INCREMENTAL')).toBe(false);
  });

  it('ignores the chain count when the policy keeps only full backups', () => {
    const draft = buildPhysicalBackupConfigDraft({
      retention: { policy: 'FULL_BACKUPS', chainsCount: 0 },
      backupType: 'FULL',
    });

    expect(isPhysicalBackupConfigComplete(draft, 'FULL')).toBe(true);
  });

  it('ignores the full-backup retention when the policy keeps only chains', () => {
    const draft = buildPhysicalBackupConfigDraft({
      retention: { policy: 'CHAINS' },
      fullBackups: NO_GFS_PERIODS,
    });

    expect(isPhysicalBackupConfigComplete(draft, 'FULL_INCREMENTAL')).toBe(true);
  });

  it('requires a positive count for LAST_N', () => {
    const buildLastN = (count: number) =>
      buildPhysicalBackupConfigDraft({
        retention: { policy: 'CHAINS_AND_FULL_BACKUPS' },
        fullBackups: { policy: 'LAST_N', count },
      });

    expect(isPhysicalBackupConfigComplete(buildLastN(1), 'FULL_INCREMENTAL')).toBe(true);
    expect(isPhysicalBackupConfigComplete(buildLastN(0), 'FULL_INCREMENTAL')).toBe(false);
  });

  it.each(['gfsHours', 'gfsDays', 'gfsWeeks', 'gfsMonths', 'gfsYears'] as const)(
    'accepts GFS retention that keeps only %s',
    (period) => {
      const draft = buildPhysicalBackupConfigDraft({
        fullBackups: { ...NO_GFS_PERIODS, policy: 'GFS', [period]: 1 },
      });

      expect(isPhysicalBackupConfigComplete(draft, 'FULL_INCREMENTAL')).toBe(true);
    },
  );

  it('applies the retention rules with backups off', () => {
    const draft = buildPhysicalBackupConfigDraft({
      config: { isBackupsEnabled: false },
      fullBackups: { ...NO_GFS_PERIODS, policy: 'GFS' },
    });

    expect(isPhysicalBackupConfigComplete(draft, 'FULL_INCREMENTAL')).toBe(false);
  });

  it('applies the schedule rules only with backups on', () => {
    const buildWithoutIncrementalSchedule = (isBackupsEnabled: boolean) =>
      buildPhysicalBackupConfigDraft({
        config: { isBackupsEnabled },
        settings: { incrementalBackupSchedule: null },
      });

    expect(
      isPhysicalBackupConfigComplete(buildWithoutIncrementalSchedule(true), 'FULL_INCREMENTAL'),
    ).toBe(false);
    expect(
      isPhysicalBackupConfigComplete(buildWithoutIncrementalSchedule(false), 'FULL_INCREMENTAL'),
    ).toBe(true);
  });

  it.each([
    {
      name: 'an empty encryption',
      change: { encryption: '' as PhysicalBackupConfigDraft['encryption'] },
      settings: {},
    },
    {
      name: 'an incomplete full schedule',
      change: {},
      settings: { fullBackupSchedule: { type: 'DAILY' as const } },
    },
  ])('refuses $name only with backups on', ({ change, settings }) => {
    const buildIncomplete = (isBackupsEnabled: boolean) =>
      buildPhysicalBackupConfigDraft({ config: { ...change, isBackupsEnabled }, settings });

    expect(isPhysicalBackupConfigComplete(buildIncomplete(true), 'FULL_INCREMENTAL')).toBe(false);
    expect(isPhysicalBackupConfigComplete(buildIncomplete(false), 'FULL_INCREMENTAL')).toBe(true);
  });

  it('requires no incremental schedule for a full-only backup type', () => {
    const draft = buildPhysicalBackupConfigDraft({ backupType: 'FULL' });

    expect(isPhysicalBackupConfigComplete(draft, 'FULL')).toBe(true);
  });
});
