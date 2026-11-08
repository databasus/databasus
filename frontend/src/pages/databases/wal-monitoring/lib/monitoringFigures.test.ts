import { describe, expect, it, vi } from 'vitest';

import { getMonitoringFigures } from '@/pages/databases/wal-monitoring/lib/monitoringFigures';
import { WalStreamHealthLevel } from '@/pages/databases/wal-monitoring/model/WalStreamHealthLevel';
import { WalStreamKeepingUpState } from '@/pages/databases/wal-monitoring/model/WalStreamKeepingUpState';
import { createTestWalStreamStatistics } from '@/shared/lib/test/buildWalStreamStatistics';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

const MB = 1000 * 1000;

describe('getMonitoringFigures', () => {
  it('shows when the last change was saved, what waits and both rates for a busy healthy database', () => {
    const statistics = createTestWalStreamStatistics();

    const figures = getMonitoringFigures({
      ...statistics,
      current: statistics.current && {
        ...statistics.current,
        lastArchivedWalAt: '2026-10-18T11:59:45Z',
      },
      keepingUp: {
        ...statistics.keepingUp,
        unarchivedWalBytes: 20 * MB,
        sourceWriteBytesPerSecond: 10 * MB,
        archiveBytesPerSecond: 10 * MB,
      },
    });

    expect(figures).toEqual({
      lastSavedAt: '2026-10-18T11:59:45Z',
      waitingToBeSavedBytes: 20 * MB,
      writeBytesPerSecond: 10 * MB,
      saveBytesPerSecond: 10 * MB,
    });
  });

  it('leaves the rates unknown, not zero, when no sample of the window holds a write value', () => {
    const statistics = createTestWalStreamStatistics();

    const figures = getMonitoringFigures({
      ...statistics,
      keepingUp: {
        ...statistics.keepingUp,
        state: WalStreamKeepingUpState.UNKNOWN,
        sourceWriteBytesPerSecond: null,
        archiveBytesPerSecond: null,
      },
    });

    expect(figures?.writeBytesPerSecond).toBeNull();
    expect(figures?.saveBytesPerSecond).toBeNull();
  });

  it.each([WalStreamHealthLevel.COLLECTING_DATA, WalStreamHealthLevel.NOT_STREAMING])(
    'shows no figures for %s',
    (level) => {
      const statistics = createTestWalStreamStatistics();

      expect(getMonitoringFigures({ ...statistics, health: { ...statistics.health, level } })).toBe(
        null,
      );
    },
  );
});
