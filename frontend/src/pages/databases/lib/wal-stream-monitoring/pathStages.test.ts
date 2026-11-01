import { describe, expect, it, vi } from 'vitest';

import { MonitoringTone } from '../../model/wal-stream-monitoring/MonitoringTone';
import { PathStage } from '../../model/wal-stream-monitoring/PathStage';
import { WalStreamHealthLevel } from '../../model/wal-stream-monitoring/WalStreamHealthLevel';
import { WalStreamReasonCode } from '../../model/wal-stream-monitoring/WalStreamReasonCode';
import { WalStreamReasonLevel } from '../../model/wal-stream-monitoring/WalStreamReasonLevel';
import { getAmountProgress, getStageTones } from './pathStages';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

describe('getStageTones', () => {
  it('colors only the Databasus stage when the local queue is filling', () => {
    const stageTones = getStageTones(WalStreamHealthLevel.NEEDS_ATTENTION, [
      {
        code: WalStreamReasonCode.LOCAL_QUEUE_FILLING,
        level: WalStreamReasonLevel.NEEDS_ATTENTION,
      },
    ]);

    expect(stageTones[PathStage.DATABASUS]).toBe(MonitoringTone.WARNING);
    expect(stageTones[PathStage.SOURCE]).toBe(MonitoringTone.NEUTRAL);
    expect(stageTones[PathStage.STORAGE]).toBe(MonitoringTone.NEUTRAL);
  });

  it('leaves every stage neutral when the only reason is a broken chain', () => {
    const stageTones = getStageTones(WalStreamHealthLevel.BROKEN, [
      { code: WalStreamReasonCode.WAL_CHAIN_BROKEN, level: WalStreamReasonLevel.BROKEN },
    ]);

    expect(stageTones[PathStage.SOURCE]).toBe(MonitoringTone.NEUTRAL);
    expect(stageTones[PathStage.DATABASUS]).toBe(MonitoringTone.NEUTRAL);
    expect(stageTones[PathStage.STORAGE]).toBe(MonitoringTone.NEUTRAL);
  });

  it('takes the most severe reason of a stage', () => {
    const stageTones = getStageTones(WalStreamHealthLevel.BROKEN, [
      { code: WalStreamReasonCode.SLOT_MISSING, level: WalStreamReasonLevel.BROKEN },
      { code: WalStreamReasonCode.NEAR_SLOT_REBUILD, level: WalStreamReasonLevel.NEEDS_ATTENTION },
    ]);

    expect(stageTones[PathStage.SOURCE]).toBe(MonitoringTone.ERROR);
  });

  it('leaves every stage neutral without a reason, so a healthy path draws no colored border', () => {
    const stageTones = getStageTones(WalStreamHealthLevel.HEALTHY, []);

    expect(Object.values(stageTones)).toEqual([
      MonitoringTone.NEUTRAL,
      MonitoringTone.NEUTRAL,
      MonitoringTone.NEUTRAL,
    ]);
  });

  it('leaves every stage neutral while streaming is off', () => {
    const stageTones = getStageTones(WalStreamHealthLevel.NOT_STREAMING, []);

    expect(Object.values(stageTones)).toEqual([
      MonitoringTone.NEUTRAL,
      MonitoringTone.NEUTRAL,
      MonitoringTone.NEUTRAL,
    ]);
  });
});

describe('getAmountProgress', () => {
  it('shows an amount within its limit as a share of it', () => {
    expect(getAmountProgress(300, 512)).toEqual({ percent: (300 / 512) * 100, isOverLimit: false });
  });

  it('marks an amount above its limit as over the limit', () => {
    expect(getAmountProgress(600, 512)).toEqual({ percent: 100, isOverLimit: true });
  });

  it('shows no progress without a limit', () => {
    expect(getAmountProgress(600, 0)).toEqual({ percent: 0, isOverLimit: false });
  });
});
