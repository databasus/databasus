import { describe, expect, it, vi } from 'vitest';

import {
  getBinaryAxisTicks,
  getChartTimeDomain,
  getDataSinceMs,
  getMissingDataRanges,
  getRowsMaximum,
  hasEnoughChartPoints,
  toChartRows,
} from '@/pages/databases/wal-monitoring/lib/chartRows';
import type {
  WalStreamSeries,
  WalStreamSeriesPoint,
} from '@/pages/databases/wal-monitoring/model/walStreamContract';
import { WalStreamStatisticsPeriod } from '@/pages/databases/wal-monitoring/model/WalStreamStatisticsPeriod';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

const MB = 1000 * 1000;
const seriesStartMs = Date.parse('2026-10-18T11:00:00Z');

const buildEmptyPoint = (minute: number): WalStreamSeriesPoint => ({
  startedAt: new Date(seriesStartMs + minute * 60_000).toISOString(),
  sourceWriteBytesPerSecond: null,
  archiveBytesPerSecond: null,
  archivedStoredBytesPerSecond: null,
  maxSlotLagBytes: null,
  maxUnarchivedWalBytes: null,
  maxLocalQueueBytes: null,
  localQueuePauseBytes: null,
  restarts: null,
  pauses: null,
  sessionStarts: null,
  highestConcurrentUploads: null,
  segmentWriteAvgMillis: null,
  segmentWriteMaxMillis: null,
});

const buildFallingBehindPoint = (minute: number): WalStreamSeriesPoint => ({
  ...buildEmptyPoint(minute),
  sourceWriteBytesPerSecond: 21 * MB,
  archiveBytesPerSecond: 12 * MB,
  maxUnarchivedWalBytes: minute * 9 * MB * 60,
  maxLocalQueueBytes: 64 * MB,
  localQueuePauseBytes: 512 * MB,
  restarts: 0,
  pauses: 0,
});

const buildSeries = (points: WalStreamSeriesPoint[]): WalStreamSeries => ({
  period: WalStreamStatisticsPeriod.HOUR,
  bucketSeconds: 60,
  from: points[0].startedAt,
  to: new Date(seriesStartMs + points.length * 60_000).toISOString(),
  points,
});

describe('toChartRows', () => {
  it('keeps archiving below the source and the backlog rising while archiving falls behind', () => {
    const rows = toChartRows(
      Array.from({ length: 60 }, (_, minute) => buildFallingBehindPoint(minute)),
    );

    for (const row of rows) {
      expect(row.archiveBytesPerSecond ?? 0).toBeLessThan(row.sourceWriteBytesPerSecond ?? 0);
    }

    for (let rowIndex = 1; rowIndex < rows.length; rowIndex++) {
      expect(rows[rowIndex].maxUnarchivedWalBytes ?? 0).toBeGreaterThan(
        rows[rowIndex - 1].maxUnarchivedWalBytes ?? 0,
      );
    }
  });

  it('keeps an interval without samples as a gap, not as zero', () => {
    const [row] = toChartRows([buildEmptyPoint(0)]);

    expect(row.sourceWriteBytesPerSecond).toBeNull();
    expect(row.archiveBytesPerSecond).toBeNull();
    expect(row.maxUnarchivedWalBytes).toBeNull();
    expect(row.maxLocalQueueBytes).toBeNull();
    expect(row.restartMarkerBytes).toBeNull();
  });

  it('puts restart and pause markers on the buffer line', () => {
    const [row] = toChartRows([{ ...buildFallingBehindPoint(0), restarts: 1, pauses: 2 }]);

    expect(row.restartMarkerBytes).toBe(64 * MB);
    expect(row.pauseMarkerBytes).toBe(64 * MB);
  });
});

describe('hasEnoughChartPoints', () => {
  it('draws no chart from three points with values', () => {
    const series = buildSeries([
      buildEmptyPoint(0),
      buildFallingBehindPoint(1),
      buildFallingBehindPoint(2),
      buildFallingBehindPoint(3),
    ]);

    expect(hasEnoughChartPoints(series)).toBe(false);
  });

  it('draws a chart from four points with values', () => {
    const series = buildSeries([0, 1, 2, 3].map(buildFallingBehindPoint));

    expect(hasEnoughChartPoints(series)).toBe(true);
  });
});

describe('getChartTimeDomain', () => {
  it('starts the axis at the first point with values when the data starts inside the period', () => {
    const series = buildSeries([
      buildEmptyPoint(0),
      buildEmptyPoint(1),
      buildFallingBehindPoint(2),
    ]);

    expect(getChartTimeDomain(series)).toEqual([
      seriesStartMs + 2 * 60_000,
      new Date(series.to).getTime(),
    ]);
  });

  it('spans the whole period when the data covers it', () => {
    const series = buildSeries([buildFallingBehindPoint(0), buildFallingBehindPoint(1)]);

    expect(getChartTimeDomain(series)).toEqual([
      new Date(series.from).getTime(),
      new Date(series.to).getTime(),
    ]);
  });
});

const KiB = 1024;
const MiB = 1024 * KiB;
const GiB = 1024 * MiB;

describe('getBinaryAxisTicks', () => {
  it('keeps the minimum top of 1 MiB for a quiet database, with quarter steps', () => {
    expect(getBinaryAxisTicks(2 * KiB, MiB)).toEqual([0, 256 * KiB, 512 * KiB, 768 * KiB, MiB]);
  });

  it('ends on a whole unit above the data', () => {
    expect(getBinaryAxisTicks(3.7 * MiB, MiB)).toEqual([0, MiB, 2 * MiB, 3 * MiB, 4 * MiB]);
  });

  it('steps in round numbers of the unit for a large amount', () => {
    expect(getBinaryAxisTicks(520 * MiB, 512 * MiB)).toEqual([0, 200 * MiB, 400 * MiB, 600 * MiB]);
  });

  it('switches to gigabytes above 1 GiB', () => {
    expect(getBinaryAxisTicks(1.5 * GiB, MiB)).toEqual([0, 0.5 * GiB, GiB, 1.5 * GiB]);
  });

  it('still draws an axis when there is neither data nor a minimum', () => {
    expect(getBinaryAxisTicks(0, 0)).toEqual([0, 0.25, 0.5, 0.75, 1]);
  });

  it('always reaches the data maximum', () => {
    for (const dataMaximum of [1, 999, 1025, 17 * MiB, 3 * GiB + 1]) {
      const ticks = getBinaryAxisTicks(dataMaximum, 0);

      expect(ticks[ticks.length - 1]).toBeGreaterThanOrEqual(dataMaximum);
      expect(ticks.length).toBeLessThanOrEqual(7);
    }
  });
});

describe('getRowsMaximum', () => {
  it('takes the largest value of the given lines and skips gaps', () => {
    const rows = toChartRows([buildFallingBehindPoint(0), buildEmptyPoint(1)]);

    expect(getRowsMaximum(rows, ['sourceWriteBytesPerSecond', 'archiveBytesPerSecond'])).toBe(
      21 * MB,
    );
  });

  it('is zero without values', () => {
    expect(getRowsMaximum(toChartRows([buildEmptyPoint(0)]), ['archiveBytesPerSecond'])).toBe(0);
  });
});

describe('getMissingDataRanges', () => {
  const buildPoints = (emptyMinutes: number[], count = 10) =>
    Array.from({ length: count }, (_, minute) =>
      emptyMinutes.includes(minute) ? buildEmptyPoint(minute) : buildFallingBehindPoint(minute),
    );

  it('finds nothing in a full series', () => {
    expect(getMissingDataRanges(buildSeries(buildPoints([])))).toEqual([]);
  });

  it('finds five empty minutes in the middle, ending where data comes back', () => {
    expect(getMissingDataRanges(buildSeries(buildPoints([3, 4, 5, 6, 7])))).toEqual([
      { fromMs: seriesStartMs + 3 * 60_000, toMs: seriesStartMs + 8 * 60_000 },
    ]);
  });

  it('ignores a single empty point, which is usually the interval still being measured', () => {
    expect(getMissingDataRanges(buildSeries(buildPoints([4, 9])))).toEqual([]);
  });

  it('does not count the time before the data starts', () => {
    expect(getMissingDataRanges(buildSeries(buildPoints([0, 1, 2])))).toEqual([]);
  });

  it('ends a run at the end of the period when the data stops', () => {
    const series = buildSeries(buildPoints([8, 9]));

    expect(getMissingDataRanges(series)).toEqual([
      { fromMs: seriesStartMs + 8 * 60_000, toMs: new Date(series.to).getTime() },
    ]);
  });

  it('finds nothing in a series without values', () => {
    expect(getMissingDataRanges(buildSeries([buildEmptyPoint(0), buildEmptyPoint(1)]))).toEqual([]);
  });
});

describe('getDataSinceMs', () => {
  it('names the first interval with data when the data starts inside the period', () => {
    const series = buildSeries([
      buildEmptyPoint(0),
      buildEmptyPoint(1),
      buildFallingBehindPoint(2),
    ]);

    expect(getDataSinceMs(series)).toBe(seriesStartMs + 2 * 60_000);
  });

  it('says nothing when the data covers the whole period', () => {
    expect(
      getDataSinceMs(buildSeries([buildFallingBehindPoint(0), buildFallingBehindPoint(1)])),
    ).toBeNull();
  });
});
