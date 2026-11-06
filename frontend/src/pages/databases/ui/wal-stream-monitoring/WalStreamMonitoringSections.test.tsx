import type { JSX } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import { getFullBackupAction } from '@/pages/databases/lib/wal-stream-monitoring/fullBackupAction';
import { getMonitoringFigures } from '@/pages/databases/lib/wal-stream-monitoring/monitoringFigures';
import { getStageTones } from '@/pages/databases/lib/wal-stream-monitoring/pathStages';
import { getTipTexts } from '@/pages/databases/lib/wal-stream-monitoring/reasonTexts';
import { getVerdictBanner } from '@/pages/databases/lib/wal-stream-monitoring/verdictBanner';
import { type WalStreamHealth } from '@/pages/databases/model/wal-stream-monitoring/WalStreamHealth';
import { WalStreamHealthLevel } from '@/pages/databases/model/wal-stream-monitoring/WalStreamHealthLevel';
import { WalStreamKeepingUpState } from '@/pages/databases/model/wal-stream-monitoring/WalStreamKeepingUpState';
import { WalStreamNoticeCode } from '@/pages/databases/model/wal-stream-monitoring/WalStreamNoticeCode';
import { WalStreamReasonCode } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReasonCode';
import { WalStreamReasonLevel } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReasonLevel';
import { type WalStreamSeriesPoint } from '@/pages/databases/model/wal-stream-monitoring/WalStreamSeriesPoint';
import { type WalStreamStatistics } from '@/pages/databases/model/wal-stream-monitoring/WalStreamStatistics';
import { ChangesPathComponent } from '@/pages/databases/ui/wal-stream-monitoring/ChangesPathComponent';
import { MonitoringFiguresComponent } from '@/pages/databases/ui/wal-stream-monitoring/MonitoringFiguresComponent';
import { RestoreTimelineComponent } from '@/pages/databases/ui/wal-stream-monitoring/RestoreTimelineComponent';
import { TechnicalDetailsComponent } from '@/pages/databases/ui/wal-stream-monitoring/TechnicalDetailsComponent';
import { VerdictBannerComponent } from '@/pages/databases/ui/wal-stream-monitoring/VerdictBannerComponent';
import { ChartAvailabilityComponent } from '@/pages/databases/ui/wal-stream-monitoring/WalStreamChartsComponent';
import { LocaleProvider } from '@/shared/i18n';
import { createTestWalStreamStatistics } from '@/shared/lib/test/buildWalStreamStatistics';
import { ThemeContext } from '@/shared/theme';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window. LocaleProvider then reads
// localStorage, which here reports no saved language. Times render in the local zone, so the
// tests pin it.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
  vi.stubGlobal('localStorage', { getItem: () => null });
  vi.stubGlobal('navigator', { language: 'en-US', languages: ['en-US'] });
  vi.stubEnv('TZ', 'UTC');
});

const MiB = 1024 * 1024;
const now = new Date('2026-10-18T12:00:00Z');

const brokenChainHealth: WalStreamHealth = {
  level: WalStreamHealthLevel.BROKEN,
  reasons: [
    { code: WalStreamReasonCode.WAL_CHAIN_BROKEN, level: WalStreamReasonLevel.BROKEN },
    {
      code: WalStreamReasonCode.LOCAL_QUEUE_FILLING,
      level: WalStreamReasonLevel.NEEDS_ATTENTION,
      valueBytes: 600 * MiB,
      thresholdBytes: 512 * MiB,
    },
  ],
  notices: [],
  isFullBackupPending: false,
};

const buildPoint = (minute: number, hasValues: boolean): WalStreamSeriesPoint => ({
  startedAt: new Date(Date.parse('2026-10-18T11:01:00Z') + minute * 60_000).toISOString(),
  sourceWriteBytesPerSecond: hasValues ? 10 * MiB : null,
  archiveBytesPerSecond: hasValues ? 10 * MiB : null,
  archivedStoredBytesPerSecond: hasValues ? 3 * MiB : null,
  maxSlotLagBytes: hasValues ? 2 * MiB : null,
  maxUnarchivedWalBytes: hasValues ? 20 * MiB : null,
  maxLocalQueueBytes: hasValues ? 0 : null,
  localQueuePauseBytes: hasValues ? 512 * MiB : null,
  restarts: hasValues ? 0 : null,
  pauses: hasValues ? 0 : null,
  sessionStarts: hasValues ? 0 : null,
  highestConcurrentUploads: hasValues ? 2 : null,
  segmentWriteAvgMillis: hasValues ? 100 : null,
  segmentWriteMaxMillis: hasValues ? 200 : null,
});

// Values only for the last 20 minutes of the hour.
const buildSeriesStartingInsidePeriod = (): WalStreamStatistics['series'] => ({
  ...createTestWalStreamStatistics().series,
  points: Array.from({ length: 60 }, (_, minute) => buildPoint(minute, minute >= 40)),
});

const renderWithLocale = (element: JSX.Element) =>
  renderToStaticMarkup(
    <ThemeContext.Provider value={{ theme: 'light', resolvedTheme: 'light', setTheme: () => {} }}>
      <LocaleProvider>{element}</LocaleProvider>
    </ThemeContext.Provider>,
  );

const renderBanner = (statistics: WalStreamStatistics, isCanManageDBs: boolean) => {
  const banner = getVerdictBanner({ statistics, lastSuccessfulRefreshAt: now, now });

  return renderWithLocale(
    <VerdictBannerComponent
      banner={banner}
      fullBackupAction={getFullBackupAction({
        level: statistics.health.level,
        isStateConfirmed: banner.isStateConfirmed,
        reasons: statistics.health.reasons,
        latestRestorableAt: statistics.latestRestorableAt,
        isFullBackupPending: statistics.health.isFullBackupPending,
        isCanManageDBs,
        isRequestingFullBackup: false,
      })}
      sampledAt={statistics.current?.sampledAt ?? null}
      now={now}
      isReducedMotion
      onStartFullBackup={() => {}}
      onOpenConfig={() => {}}
    />,
  );
};

const getBrokenChainStatistics = (): WalStreamStatistics => {
  const statistics = createTestWalStreamStatistics({
    health: brokenChainHealth,
    latestRestorableAt: '2026-10-18T07:50:00Z',
  });

  return {
    ...statistics,
    current: statistics.current && {
      ...statistics.current,
      lastArchivedWalAt: '2026-10-18T11:59:20Z',
    },
  };
};

describe('the Monitoring tab sections', () => {
  it('says that all changes are saved and to which moment the database can be restored', () => {
    const markup = renderBanner(createTestWalStreamStatistics(), true);

    expect(markup).toContain('All changes are being saved');
    expect(markup).toContain('aria-live="polite"');
    expect(markup).toContain(
      'You can restore the database to any moment up to 11:59 AM (40 s ago).',
    );
    expect(markup).toContain('Measured 12 s ago');
    expect(markup).toMatch(
      /font-semibold">All changes are being saved<\/div><div[^>]*>.*Measured 12 s ago/,
    );
    expect(markup).not.toContain('Start full backup');
    expect(markup).not.toContain('writes almost nothing');
  });

  it('names a broken chain in the headline, limits the restore to the moment before it and offers a full backup instead of the sentence', () => {
    const markup = renderBanner(getBrokenChainStatistics(), true);

    expect(markup).toContain('Changes were lost between backups');
    expect(markup).toContain('Part of the change history never reached storage');
    expect(markup).not.toContain('Start a full backup on the Backups tab.');
    expect(markup).toContain(
      'Until a new full backup finishes, you can restore the database only up to 7:50 AM',
    );
    expect(markup).toContain('Start full backup');
    expect(markup).toContain('The Databasus buffer is filling up');
    expect(markup).toContain('The Databasus buffer holds 600 MB of 512 MB.');
  });

  it('shows the problem without the full backup button to a user who cannot manage the database', () => {
    const markup = renderBanner(getBrokenChainStatistics(), false);

    expect(markup).toContain('Changes were lost between backups');
    expect(markup).toContain('Start a full backup on the Backups tab.');
    expect(markup).not.toContain('Start full backup');
  });

  it('shows the three figures and says not measured instead of zero for unknown rates', () => {
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

    const markup = renderWithLocale(
      <MonitoringFiguresComponent figures={figures!} keepingUpSentences={[]} isReducedMotion />,
    );

    expect(markup).toContain('Last change saved');
    expect(markup).toContain('Waiting to be saved');
    expect(markup).toContain('Database writes');
    expect(markup).toContain('Saved to storage');
    expect(markup).toContain('20 MB');
    expect(markup.match(/Not measured yet/g)).toHaveLength(2);
    expect(markup).not.toContain('0 B/s');
  });

  it('shows each rate as its own card, marked as a 10-minute average, next to the other figures', () => {
    const statistics = createTestWalStreamStatistics();

    const markup = renderWithLocale(
      <MonitoringFiguresComponent
        figures={getMonitoringFigures(statistics)!}
        keepingUpSentences={[]}
        isReducedMotion
      />,
    );

    expect(markup.match(/rounded-lg border/g)).toHaveLength(4);
    expect(markup).toContain('10 MB/s');
    expect(markup.match(/avg\. 10 min/g)).toHaveLength(2);
  });

  it('marks a gap an incremental backup bridged on the restore timeline and says it in words', () => {
    const markup = renderWithLocale(
      <RestoreTimelineComponent
        ranges={[
          { from: '2026-10-10T12:00:00Z', to: '2026-10-10T18:35:00Z' },
          { from: '2026-10-18T07:51:00Z', to: '2026-10-18T11:59:20Z' },
        ]}
        now={now}
      />,
    );

    expect(markup).toContain('What you can restore');
    expect(markup).toContain('Cannot be restored');
    expect(markup).toMatch(
      /You cannot restore the database to moments between .*6:35 PM and 7:51 AM: these changes never reached storage\./,
    );
  });

  it('shows no restore timeline before the first full backup', () => {
    expect(renderWithLocale(<RestoreTimelineComponent ranges={[]} now={now} />)).toBe('');
  });

  it('shows each tip with its title and both sentences inside the technical details', () => {
    const statistics = createTestWalStreamStatistics();

    const markup = renderWithLocale(
      <TechnicalDetailsComponent
        statistics={statistics}
        stageTones={getStageTones(statistics.health.level, statistics.health.reasons)}
        tips={getTipTexts([
          { code: WalStreamNoticeCode.NO_SLOT_WAL_LIMIT, level: WalStreamReasonLevel.INFO },
        ])}
        isReducedMotion
        isChartAnimationActive={false}
      />,
    );

    expect(markup).toContain('<div class="font-semibold">No limit on kept changes</div>');
    expect(markup).toContain('The database keeps changes for Databasus without a limit.');
    expect(markup).toContain('A limit protects the database server');
    expect(markup).toContain('max_slot_wal_keep_size');
  });

  it('shows the path of changes, both extra charts and the raw values without a click', () => {
    const statistics = createTestWalStreamStatistics({
      series: buildSeriesStartingInsidePeriod(),
    });

    const markup = renderWithLocale(
      <TechnicalDetailsComponent
        statistics={statistics}
        stageTones={getStageTones(statistics.health.level, statistics.health.reasons)}
        tips={[]}
        isReducedMotion
        isChartAnimationActive={false}
      />,
    );

    expect(markup).toContain('Technical details');
    expect(markup).toContain('Not received by Databasus yet');
    expect(markup).toContain('Waiting in the buffer');
    expect(markup).toContain('How much is not saved yet?');
    expect(markup).toContain('Databasus buffer');
    expect(markup).toContain('Slot status (wal_status)');
  });

  it('says that there is not enough data instead of drawing a chart from a few points', () => {
    const series = {
      ...createTestWalStreamStatistics().series,
      points: [buildPoint(0, true), buildPoint(1, true)],
    };

    const markup = renderWithLocale(
      <ChartAvailabilityComponent series={series}>
        <div>chart</div>
      </ChartAvailabilityComponent>,
    );

    expect(markup).toContain(
      'Not enough data for this period yet. A shorter period may already have a chart.',
    );
    expect(markup).not.toContain('chart</div>');
  });

  it('says since when it has data when the data starts inside the period', () => {
    const markup = renderWithLocale(
      <ChartAvailabilityComponent series={buildSeriesStartingInsidePeriod()}>
        <div>chart</div>
      </ChartAvailabilityComponent>,
    );

    expect(markup).toContain('Data since 11:41 AM');
    expect(markup).toContain('chart</div>');
  });

  it('draws a healthy path without problem marks, says nothing waits in an empty queue and how much of a limit a filled one uses', () => {
    const statistics = createTestWalStreamStatistics();

    const markup = renderWithLocale(
      <ChangesPathComponent
        statistics={statistics}
        stageTones={getStageTones(statistics.health.level, statistics.health.reasons)}
        isReducedMotion
      />,
    );

    expect(markup).not.toContain('Needs attention');
    expect(markup).not.toContain('Broken');
    expect(markup).toContain('Nothing waiting');
    expect(markup).not.toContain('0 B of 512 MB');
    expect(markup).not.toContain('Receiving pauses at');
    expect(markup).toContain('2 MB of 256 MB');
    expect(markup).toContain('At 256 MB Databasus starts a new chain from a full backup');
    expect(markup).toContain('Not saved to storage yet: 20 MB');
    expect(markup).toContain(
      'Speeds are averages over the last 10 minutes; amounts are right now.',
    );
    expect(markup).toContain('Last change saved');
  });

  it('says nothing waits at both queues while the links still show the averaged rate of a database writing in bursts', () => {
    const statistics = createTestWalStreamStatistics();
    const burstyStatistics: WalStreamStatistics = {
      ...statistics,
      current: statistics.current && {
        ...statistics.current,
        slotLagBytes: 0,
        localQueueBytes: 0,
      },
      keepingUp: {
        ...statistics.keepingUp,
        slotLagBytes: 0,
        sourceWriteBytesPerSecond: 54.6 * 1024,
        archiveBytesPerSecond: 54.6 * 1024,
      },
    };

    const markup = renderWithLocale(
      <ChangesPathComponent
        statistics={burstyStatistics}
        stageTones={getStageTones(statistics.health.level, statistics.health.reasons)}
        isReducedMotion
      />,
    );

    expect(markup.match(/Nothing waiting/g)).toHaveLength(2);
    expect(markup).not.toContain('of 256 MB');
    expect(markup).not.toContain('of 512 MB');
    expect(markup.match(/54\.6 KB\/s/g)).toHaveLength(2);
    expect(markup).toContain(
      'Speeds are averages over the last 10 minutes; amounts are right now.',
    );
  });

  it('shows an amount above its limit as over the limit, and puts each arrow behind its own breakpoint wrapper', () => {
    const statistics = createTestWalStreamStatistics();
    const pausedStatistics: WalStreamStatistics = {
      ...statistics,
      current: statistics.current && {
        ...statistics.current,
        localQueueBytes: 600 * MiB,
        isReceiverRunning: false,
      },
    };

    const markup = renderWithLocale(
      <ChangesPathComponent
        statistics={pausedStatistics}
        stageTones={getStageTones(brokenChainHealth.level, brokenChainHealth.reasons)}
        isReducedMotion
      />,
    );

    expect(markup).toContain('Over the limit');
    expect(markup).toContain('Receiving paused');
    expect(markup).toContain('Receiving pauses at 512 MB');
    expect(markup.match(/Needs attention/g)).toHaveLength(1);
    expect(markup).toMatch(
      /<span class="hidden @2xl:block" aria-hidden="true"><span[^>]*class="anticon anticon-arrow-right"/,
    );
    expect(markup).toMatch(
      /<span class="block text-muted @2xl:hidden" aria-hidden="true"><span[^>]*class="anticon anticon-arrow-down"/,
    );
  });
});
