import { describe, expect, it } from 'vitest';

import { buildLogicalBackupConfig, zeroRetention } from './buildLogicalBackupConfig.fixture';
import { changeLogicalRetentionPolicy } from './changeLogicalRetentionPolicy';
import { LogicalRetentionPolicy } from './LogicalRetentionPolicy';

describe('changeLogicalRetentionPolicy', () => {
  it('keeps entered values when the chosen policy has them', () => {
    const settings = buildLogicalBackupConfig(
      {},
      {},
      { count: 42, gfsDays: 9, gfsWeeks: 0 },
    ).settings;

    expect(changeLogicalRetentionPolicy(settings, LogicalRetentionPolicy.COUNT)).toMatchObject({
      policy: LogicalRetentionPolicy.COUNT,
      count: 42,
      gfsDays: 9,
    });
    expect(changeLogicalRetentionPolicy(settings, LogicalRetentionPolicy.GFS)).toMatchObject({
      policy: LogicalRetentionPolicy.GFS,
      count: 42,
      gfsDays: 9,
      gfsWeeks: 0,
    });
  });

  it('fills the defaults when the chosen policy holds only zeros', () => {
    const settings = buildLogicalBackupConfig({}, { retention: zeroRetention }).settings;

    expect(changeLogicalRetentionPolicy(settings, LogicalRetentionPolicy.COUNT).count).toBe(100);
    expect(changeLogicalRetentionPolicy(settings, LogicalRetentionPolicy.GFS)).toMatchObject({
      gfsHours: 24,
      gfsDays: 7,
      gfsWeeks: 4,
      gfsMonths: 12,
      gfsYears: 3,
    });
  });

  it('fills the defaults when a hidden hourly period is the only one set', () => {
    const settings = buildLogicalBackupConfig(
      {},
      { retention: { ...zeroRetention, gfsHours: 24 } },
    ).settings;

    expect(changeLogicalRetentionPolicy(settings, LogicalRetentionPolicy.GFS).gfsDays).toBe(7);
  });

  it('keeps an hourly period that the schedule shows', () => {
    const settings = buildLogicalBackupConfig(
      {},
      { schedule: { type: 'HOURLY' }, retention: { ...zeroRetention, gfsHours: 5 } },
    ).settings;

    expect(changeLogicalRetentionPolicy(settings, LogicalRetentionPolicy.GFS)).toMatchObject({
      gfsHours: 5,
      gfsDays: 0,
    });
  });
});
