import { describe, expect, it, vi } from 'vitest';

import { getVerdictBanner } from '@/pages/databases/wal-monitoring/lib/verdictBanner';
import { MonitoringTone } from '@/pages/databases/wal-monitoring/model/MonitoringTone';
import { VerdictIcon } from '@/pages/databases/wal-monitoring/model/VerdictBanner';
import { type WalStreamHealth } from '@/pages/databases/wal-monitoring/model/walStreamContract';
import { WalStreamHealthLevel } from '@/pages/databases/wal-monitoring/model/WalStreamHealthLevel';
import { WalStreamReasonCode } from '@/pages/databases/wal-monitoring/model/WalStreamReasonCode';
import { WalStreamReasonLevel } from '@/pages/databases/wal-monitoring/model/WalStreamReasonLevel';
import { createTestWalStreamStatistics } from '@/shared/lib/test/buildWalStreamStatistics';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window. Whether a moment is from today depends
// on the time zone, so the tests pin it.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
  vi.stubEnv('TZ', 'UTC');
});

const now = new Date('2026-10-18T12:00:00Z');

const getBannerFor = (
  health: Partial<WalStreamHealth>,
  overrides: Parameters<typeof createTestWalStreamStatistics>[0] = {},
  lastSuccessfulRefreshAt = now,
) => {
  const statistics = createTestWalStreamStatistics(overrides);

  return getVerdictBanner({
    statistics: { ...statistics, health: { ...statistics.health, ...health } },
    lastSuccessfulRefreshAt,
    now,
  });
};

const brokenChainReason = {
  code: WalStreamReasonCode.WAL_CHAIN_BROKEN,
  level: WalStreamReasonLevel.BROKEN,
};

describe('getVerdictBanner', () => {
  it.each([
    [WalStreamHealthLevel.HEALTHY, MonitoringTone.SUCCESS, VerdictIcon.CHECK],
    [WalStreamHealthLevel.NEEDS_ATTENTION, MonitoringTone.WARNING, VerdictIcon.EXCLAMATION],
    [WalStreamHealthLevel.BROKEN, MonitoringTone.ERROR, VerdictIcon.CLOSE],
    [WalStreamHealthLevel.COLLECTING_DATA, MonitoringTone.NEUTRAL, VerdictIcon.CLOCK],
    [WalStreamHealthLevel.NOT_STREAMING, MonitoringTone.NEUTRAL, VerdictIcon.PAUSE],
  ])('shows %s with its own color and icon', (level, tone, icon) => {
    const banner = getBannerFor({ level });

    expect(banner.tone).toBe(tone);
    expect(banner.icon).toBe(icon);
  });

  it('says that all changes are being saved when healthy', () => {
    const banner = getBannerFor({ level: WalStreamHealthLevel.HEALTHY });

    expect(banner.headlineKey).toBe('backups.physical.monitoring.banner.healthyHeadline');
    expect(banner.headlineReason).toBeUndefined();
    expect(banner.otherReasons).toEqual([]);
  });

  it('names the problem in the headline instead of the verdict', () => {
    const banner = getBannerFor({
      level: WalStreamHealthLevel.BROKEN,
      reasons: [brokenChainReason],
    });

    expect(banner.headlineKey).toBe('backups.physical.monitoring.reasons.walChainBroken.title');
    expect(banner.headlineReason?.whatHappened.key).toBe(
      'backups.physical.monitoring.reasons.walChainBroken.whatHappened',
    );
    expect(banner.headlineReason?.whatToDo.key).toBe(
      'backups.physical.monitoring.reasons.walChainBroken.whatToDo',
    );
  });

  it('heads with a broken reason and lists needs-attention ones after it', () => {
    const banner = getBannerFor({
      level: WalStreamHealthLevel.BROKEN,
      reasons: [
        {
          code: WalStreamReasonCode.LOCAL_QUEUE_FILLING,
          level: WalStreamReasonLevel.NEEDS_ATTENTION,
        },
        { code: WalStreamReasonCode.SLOT_MISSING, level: WalStreamReasonLevel.BROKEN },
      ],
    });

    expect(banner.headlineKey).toBe('backups.physical.monitoring.reasons.slotMissing.title');
    expect(banner.otherReasons.map((reason) => reason.titleKey)).toEqual([
      'backups.physical.monitoring.reasons.localQueueFilling.title',
    ]);
  });

  it.each([
    [WalStreamHealthLevel.COLLECTING_DATA, 'backups.physical.monitoring.levels.collectingData'],
    [WalStreamHealthLevel.NOT_STREAMING, 'backups.physical.monitoring.levels.notStreaming'],
  ])('keeps the headline of %s', (level, headlineKey) => {
    expect(getBannerFor({ level }).headlineKey).toBe(headlineKey);
  });

  it('links the not streaming headline to the configuration', () => {
    expect(getBannerFor({ level: WalStreamHealthLevel.NOT_STREAMING }).isConfigLinkShown).toBe(
      true,
    );
    expect(getBannerFor({ level: WalStreamHealthLevel.HEALTHY }).isConfigLinkShown).toBe(false);
  });

  it('says to which time today the database can be restored, and how long ago that was', () => {
    const banner = getBannerFor({}, { latestRestorableAt: '2026-10-18T11:59:30Z' });

    expect(banner.restoreSentence).toEqual({
      key: 'backups.physical.monitoring.banner.restoreUpTo',
      params: {
        moment: { kind: 'moment', at: '2026-10-18T11:59:30Z', isToday: true },
        age: {
          kind: 'text',
          text: { key: 'backups.physical.monitoring.durations.seconds', params: { seconds: 30 } },
        },
      },
    });
  });

  it('shows a restorable moment from an earlier day with its date', () => {
    const banner = getBannerFor({}, { latestRestorableAt: '2026-10-17T22:00:00Z' });

    expect(banner.restoreSentence.params?.moment).toEqual({
      kind: 'moment',
      at: '2026-10-17T22:00:00Z',
      isToday: false,
    });
  });

  it('says that only an earlier moment can be restored while saved changes lie beyond a broken chain', () => {
    const statistics = createTestWalStreamStatistics();
    const banner = getBannerFor(
      { level: WalStreamHealthLevel.BROKEN, reasons: [brokenChainReason] },
      {
        latestRestorableAt: '2026-10-18T07:50:00Z',
        current: statistics.current && {
          ...statistics.current,
          lastArchivedWalAt: '2026-10-18T07:58:00Z',
        },
      },
    );

    expect(banner.restoreSentence.key).toBe('backups.physical.monitoring.banner.restoreOnlyUpTo');
  });

  it('names where the newest restorable stretch starts when an earlier stretch cannot be restored', () => {
    const banner = getBannerFor(
      {},
      {
        latestRestorableAt: '2026-10-18T11:59:30Z',
        restorableRanges: [
          { from: '2026-10-10T14:51:00Z', to: '2026-10-10T15:35:00Z' },
          { from: '2026-10-17T04:51:00Z', to: '2026-10-18T11:59:30Z' },
        ],
      },
    );

    expect(banner.restoreSentence.key).toBe('backups.physical.monitoring.banner.restoreFromUpTo');
    expect(banner.restoreSentence.params?.from).toEqual({
      kind: 'moment',
      at: '2026-10-17T04:51:00Z',
      isToday: false,
    });
  });

  it('says "any moment up to" without a start while one stretch covers the whole history', () => {
    const banner = getBannerFor({}, { latestRestorableAt: '2026-10-18T11:59:30Z' });

    expect(banner.restoreSentence.params).not.toHaveProperty('from');
  });

  it.each([
    [[], 'backups.physical.monitoring.banner.restoreUpToWithoutAge'],
    [[brokenChainReason], 'backups.physical.monitoring.banner.restoreOnlyUpToWithoutAge'],
  ])(
    'leaves the age out instead of showing NaN when the moment cannot be read (%#)',
    (reasons, restoreKey) => {
      const banner = getBannerFor(
        {
          level: reasons.length ? WalStreamHealthLevel.BROKEN : WalStreamHealthLevel.HEALTHY,
          reasons,
        },
        { latestRestorableAt: 'not a time' },
      );

      expect(banner.restoreSentence.key).toBe(restoreKey);
      expect(banner.restoreSentence.params).not.toHaveProperty('age');
    },
  );

  it.each([
    [false, 'backups.physical.monitoring.banner.restoreNothing'],
    [true, 'backups.physical.monitoring.banner.restoreNothingPending'],
  ])(
    'says that nothing can be restored before the first full backup (pending: %s)',
    (isFullBackupPending, restoreKey) => {
      const banner = getBannerFor({ isFullBackupPending }, { latestRestorableAt: null });

      expect(banner.restoreSentence).toEqual({ key: restoreKey });
    },
  );

  it('says that nothing can be restored yet while a just-enabled database collects data', () => {
    const banner = getBannerFor(
      { level: WalStreamHealthLevel.COLLECTING_DATA },
      { latestRestorableAt: null },
    );

    expect(banner.restoreSentence).toEqual({
      key: 'backups.physical.monitoring.banner.restoreNothing',
    });
  });

  it('cannot confirm the state when refreshes have failed for 3 minutes, and keeps the last known restorable moment', () => {
    const lastSuccessfulRefreshAt = new Date(now.getTime() - 3 * 60 * 1000);

    const banner = getBannerFor({}, {}, lastSuccessfulRefreshAt);

    expect(banner.isStateConfirmed).toBe(false);
    expect(banner.tone).toBe(MonitoringTone.NEUTRAL);
    expect(banner.icon).toBe(VerdictIcon.QUESTION);
    expect(banner.headlineKey).toBe('backups.physical.monitoring.banner.cannotConfirm');
    expect(banner.description).toEqual({
      key: 'backups.physical.monitoring.banner.cannotConfirmDescription',
      params: {
        level: { kind: 'text', text: { key: 'backups.physical.monitoring.levels.healthy' } },
        duration: {
          kind: 'text',
          text: { key: 'backups.physical.monitoring.durations.minutes', params: { minutes: 3 } },
        },
      },
    });
    expect(banner.restoreSentence.key).toBe('backups.physical.monitoring.banner.restoreUpTo');
  });

  it('keeps the verdict while the last refresh is under 2 minutes old', () => {
    const lastSuccessfulRefreshAt = new Date(now.getTime() - 90 * 1000);

    expect(getBannerFor({}, {}, lastSuccessfulRefreshAt).tone).toBe(MonitoringTone.SUCCESS);
  });
});
