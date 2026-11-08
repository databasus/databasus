import { describe, expect, it, vi } from 'vitest';

import { applyLiveValues } from '@/pages/databases/wal-monitoring/lib/liveOverlay';
import { type WalStreamLive } from '@/pages/databases/wal-monitoring/model/walStreamContract';
import { WalStreamSourceState } from '@/pages/databases/wal-monitoring/model/WalStreamSourceState';
import { createTestWalStreamStatistics } from '@/shared/lib/test/buildWalStreamStatistics';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

const MiB = 1024 * 1024;

// The fixture's newest sample was taken at 11:59:48.
const buildLive = (overrides: Partial<WalStreamLive> = {}): WalStreamLive => ({
  measuredAt: '2026-10-18T11:59:59Z',
  sourceProbedAt: '2026-10-18T11:59:55Z',
  sourceState: WalStreamSourceState.REACHABLE,
  sourceCurrentLsn: '0/6000000',
  slotWalStatus: 'reserved',
  slotLagBytes: MiB,
  slotSafeWalSizeBytes: null,
  archiveProgressLsn: '0/5F00000',
  lastArchivedWalAt: '2026-10-18T11:59:57Z',
  unarchivedWalBytes: MiB,
  localQueueBytes: 300 * MiB,
  localQueuePauseBytes: 512 * MiB,
  isReceiverRunning: false,
  ...overrides,
});

describe('applyLiveValues', () => {
  it('keeps the statistics without live values', () => {
    const statistics = createTestWalStreamStatistics();

    expect(applyLiveValues(statistics, null)).toBe(statistics);
  });

  it('lays newer values over the newest sample and moves its time', () => {
    const overlaid = applyLiveValues(createTestWalStreamStatistics(), buildLive());

    expect(overlaid.current).toMatchObject({
      sampledAt: '2026-10-18T11:59:59Z',
      localQueueBytes: 300 * MiB,
      isReceiverRunning: false,
      lastArchivedWalAt: '2026-10-18T11:59:57Z',
      slotLagBytes: MiB,
      unarchivedWalBytes: MiB,
    });
    expect(overlaid.keepingUp.unarchivedWalBytes).toBe(MiB);
  });

  it('keeps the rates, the verdict and the series of the statistics', () => {
    const statistics = createTestWalStreamStatistics();
    const overlaid = applyLiveValues(statistics, buildLive());

    expect(overlaid.keepingUp.sourceWriteBytesPerSecond).toBe(
      statistics.keepingUp.sourceWriteBytesPerSecond,
    );
    expect(overlaid.health).toBe(statistics.health);
    expect(overlaid.series).toBe(statistics.series);
  });

  it('keeps the sample when the live values are older than it', () => {
    const statistics = createTestWalStreamStatistics();

    expect(applyLiveValues(statistics, buildLive({ measuredAt: '2026-10-18T11:59:40Z' }))).toBe(
      statistics,
    );
  });

  it('keeps the source values of the sample when the probe is older than the sample', () => {
    const statistics = createTestWalStreamStatistics();
    const overlaid = applyLiveValues(
      statistics,
      buildLive({ sourceProbedAt: '2026-10-18T11:59:40Z' }),
    );

    expect(overlaid.current?.localQueueBytes).toBe(300 * MiB);
    expect(overlaid.current?.slotLagBytes).toBe(statistics.current?.slotLagBytes);
    expect(overlaid.current?.unarchivedWalBytes).toBe(statistics.current?.unarchivedWalBytes);
    expect(overlaid.keepingUp).toBe(statistics.keepingUp);
  });

  it('keeps the last save of the sample while the streamer has archived nothing yet', () => {
    const statistics = createTestWalStreamStatistics();
    const overlaid = applyLiveValues(statistics, buildLive({ lastArchivedWalAt: null }));

    expect(overlaid.current?.lastArchivedWalAt).toBe(statistics.current?.lastArchivedWalAt);
  });

  it('moves the restore point and the end of the newest stretch with the live save time', () => {
    const overlaid = applyLiveValues(createTestWalStreamStatistics(), buildLive());

    expect(overlaid.latestRestorableAt).toBe('2026-10-18T11:59:57Z');
    expect(overlaid.restorableRanges).toEqual([
      { from: '2026-10-17T03:00:00Z', to: '2026-10-18T11:59:57Z' },
    ]);
  });

  it('keeps the restore point where a gap stopped it, short of the newest saved change', () => {
    const statistics = createTestWalStreamStatistics({
      latestRestorableAt: '2026-10-18T07:50:00Z',
      restorableRanges: [{ from: '2026-10-17T03:00:00Z', to: '2026-10-18T07:50:00Z' }],
    });
    const overlaid = applyLiveValues(statistics, buildLive());

    expect(overlaid.latestRestorableAt).toBe('2026-10-18T07:50:00Z');
    expect(overlaid.restorableRanges).toBe(statistics.restorableRanges);
  });
});
