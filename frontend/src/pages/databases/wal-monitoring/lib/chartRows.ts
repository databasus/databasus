import type { WalStreamChartRow } from '@/pages/databases/wal-monitoring/model/WalStreamChartRow';
import type {
  WalStreamSeries,
  WalStreamSeriesPoint,
} from '@/pages/databases/wal-monitoring/model/walStreamContract';

export const toChartRows = (points: WalStreamSeriesPoint[]): WalStreamChartRow[] =>
  points.map((point) => ({
    timestampMs: new Date(point.startedAt).getTime(),
    sourceWriteBytesPerSecond: point.sourceWriteBytesPerSecond,
    archiveBytesPerSecond: point.archiveBytesPerSecond,
    maxUnarchivedWalBytes: point.maxUnarchivedWalBytes,
    maxSlotLagBytes: point.maxSlotLagBytes,
    maxLocalQueueBytes: point.maxLocalQueueBytes,
    localQueuePauseBytes: point.localQueuePauseBytes,
    restartMarkerBytes: getMarkerBytes(point.restarts, point),
    pauseMarkerBytes: getMarkerBytes(point.pauses, point),
  }));

// A marker sits on the buffer line at its interval, so it needs no axis of its own.
const getMarkerBytes = (eventCount: number | null, point: WalStreamSeriesPoint): number | null =>
  eventCount !== null && eventCount > 0 ? (point.maxLocalQueueBytes ?? 0) : null;

const hasPointValues = (point: WalStreamSeriesPoint): boolean =>
  point.archiveBytesPerSecond !== null ||
  point.sourceWriteBytesPerSecond !== null ||
  point.maxLocalQueueBytes !== null;

// Fewer points than this draw a line that says nothing, such as one start-up burst falling to zero.
const MIN_CHART_POINTS = 4;

export const hasEnoughChartPoints = (series: WalStreamSeries): boolean =>
  series.points.filter(hasPointValues).length >= MIN_CHART_POINTS;

// The axis starts where the data starts, so a few minutes of data are not squeezed against the end
// of a whole hour.
export const getChartTimeDomain = (series: WalStreamSeries): [number, number] => [
  getDataSinceMs(series) ?? new Date(series.from).getTime(),
  new Date(series.to).getTime(),
];

const BYTES_PER_UNIT = 1024;
const TARGET_AXIS_INTERVALS = 4;
const NICE_STEP_FACTORS = [1, 2, 2.5, 5, 10];

// Round ticks for an amount axis, in the binary unit formatBytes shows them in, so the axis reads
// 256 KB, 512 KB, 768 KB rather than 293 KB, 585.9 KB, 878.9 KB. The minimum top is a floor, so a
// quiet database draws a line near the bottom instead of noise scaled up to fill the chart.
export const getBinaryAxisTicks = (dataMaximum: number, minimumTop: number): number[] => {
  const top = Math.max(dataMaximum, minimumTop, 1);

  const unitPower = Math.max(0, Math.floor(Math.log(top) / Math.log(BYTES_PER_UNIT)));
  const unit = BYTES_PER_UNIT ** unitPower;
  const topInUnits = top / unit;

  const roughStep = topInUnits / TARGET_AXIS_INTERVALS;
  const magnitude = 10 ** Math.floor(Math.log10(roughStep));
  const stepFactor =
    NICE_STEP_FACTORS.find((factor) => factor * magnitude >= roughStep) ?? NICE_STEP_FACTORS[4];
  const step = stepFactor * magnitude;

  // Rounding first keeps a float error from adding an interval to an exact top such as 1 MiB; the
  // check in bytes then adds one when the data sits just above the rounded top.
  let intervals = Math.round(topInUnits / step);

  if (intervals * step * unit < top) {
    intervals++;
  }

  return Array.from({ length: intervals + 1 }, (_, index) => index * step * unit);
};

// The largest value the given lines reach, 0 for a chart without values.
export const getRowsMaximum = (
  rows: WalStreamChartRow[],
  dataKeys: (keyof WalStreamChartRow)[],
): number =>
  rows.reduce(
    (maximum, row) =>
      dataKeys.reduce((rowMaximum, dataKey) => Math.max(rowMaximum, row[dataKey] ?? 0), maximum),
    0,
  );

const MIN_MISSING_POINTS = 2;

// Runs of points without values after the data starts. One empty point is usually the interval
// still being measured, so a run needs two. Each range ends where the next point with values
// starts, or at the end of the period.
export const getMissingDataRanges = (series: WalStreamSeries): MissingDataRange[] => {
  const firstIndexWithValues = series.points.findIndex(hasPointValues);

  if (firstIndexWithValues === -1) {
    return [];
  }

  const ranges: MissingDataRange[] = [];
  let runStartIndex: number | null = null;

  const closeRun = (endIndex: number, toMs: number) => {
    if (runStartIndex !== null && endIndex - runStartIndex >= MIN_MISSING_POINTS) {
      ranges.push({ fromMs: new Date(series.points[runStartIndex].startedAt).getTime(), toMs });
    }

    runStartIndex = null;
  };

  for (let index = firstIndexWithValues; index < series.points.length; index++) {
    const point = series.points[index];

    if (!hasPointValues(point)) {
      runStartIndex ??= index;
      continue;
    }

    closeRun(index, new Date(point.startedAt).getTime());
  }

  closeRun(series.points.length, new Date(series.to).getTime());

  return ranges;
};

// Null when the data covers the whole period, so the caption only appears when the charts start
// later than their axis.
export const getDataSinceMs = (series: WalStreamSeries): number | null => {
  const firstPointWithValues = series.points.find(hasPointValues);

  if (!firstPointWithValues || firstPointWithValues === series.points[0]) {
    return null;
  }

  return new Date(firstPointWithValues.startedAt).getTime();
};

// A stretch of the chart's period without measurements, as epoch milliseconds.
export interface MissingDataRange {
  fromMs: number;
  toMs: number;
}
