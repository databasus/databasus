import dayjs from 'dayjs';

import type { MonitoringText } from '@/pages/databases/model/wal-stream-monitoring/MonitoringText';
import type { MonitoringTextParam } from '@/pages/databases/model/wal-stream-monitoring/MonitoringTextParam';
import type { RestoreTimelineSegment } from '@/pages/databases/model/wal-stream-monitoring/RestoreTimelineSegment';
import { RestoreTimelineSegmentKind } from '@/pages/databases/model/wal-stream-monitoring/RestoreTimelineSegmentKind';
import type { WalStreamTimeRange } from '@/pages/databases/model/wal-stream-monitoring/WalStreamTimeRange';

// A lost stretch of a few hours on a chain months long would be thinner than a pixel, so every
// stretch except the not saved yet tail keeps this share of the bar.
export const MIN_SEGMENT_WIDTH_PERCENT = 1.5;

// Past three sentences the list stops helping, so the rest is only counted.
const MAX_LOST_SENTENCES = 3;

interface TimelineStretch {
  kind: RestoreTimelineSegmentKind;
  fromMs: number;
  toMs: number;
}

// The bar runs from the start of the first restorable range to now: the ranges, the lost
// stretches between them, and the tail after the latest restorable moment that is not saved yet.
export const getRestoreTimeline = (
  ranges: WalStreamTimeRange[],
  now: Date,
): RestoreTimelineSegment[] => {
  const stretches = getStretches(ranges, now.getTime());

  if (stretches.length === 0) {
    return [];
  }

  const startMs = stretches[0].fromMs;
  const totalMs = stretches[stretches.length - 1].toMs - startMs;

  if (totalMs <= 0) {
    return stretches.map((stretch, index) => ({
      ...stretch,
      startPercent: 0,
      widthPercent: index === 0 ? 100 : 0,
    }));
  }

  const widths = getWidthsWithMinimum(
    stretches.map((stretch) => ((stretch.toMs - stretch.fromMs) / totalMs) * 100),
    stretches.map((stretch) => stretch.kind !== RestoreTimelineSegmentKind.NOT_SAVED_YET),
  );

  let startPercent = 0;

  return stretches.map((stretch, index) => {
    const segment = { ...stretch, startPercent, widthPercent: widths[index] };
    startPercent += widths[index];

    return segment;
  });
};

const getStretches = (ranges: WalStreamTimeRange[], nowMs: number): TimelineStretch[] => {
  const stretches: TimelineStretch[] = [];

  ranges.forEach((range, index) => {
    const fromMs = new Date(range.from).getTime();
    const toMs = new Date(range.to).getTime();

    stretches.push({ kind: RestoreTimelineSegmentKind.RESTORABLE, fromMs, toMs });

    const nextRange = ranges[index + 1];

    if (nextRange) {
      stretches.push({
        kind: RestoreTimelineSegmentKind.LOST,
        fromMs: toMs,
        toMs: new Date(nextRange.from).getTime(),
      });
    }
  });

  const lastStretch = stretches[stretches.length - 1] as TimelineStretch | undefined;

  if (lastStretch && nowMs > lastStretch.toMs) {
    stretches.push({
      kind: RestoreTimelineSegmentKind.NOT_SAVED_YET,
      fromMs: lastStretch.toMs,
      toMs: nowMs,
    });
  }

  return stretches;
};

// Raises the widths that may not shrink below the minimum and takes the difference from the
// others in proportion, so the bar still adds up to 100%.
const getWidthsWithMinimum = (widths: number[], isMinimumApplied: boolean[]): number[] => {
  const isRaised = widths.map(
    (width, index) => isMinimumApplied[index] && width < MIN_SEGMENT_WIDTH_PERCENT,
  );
  const raisedTotal = isRaised.filter(Boolean).length * MIN_SEGMENT_WIDTH_PERCENT;
  const otherTotal = widths.reduce(
    (total, width, index) => total + (isRaised[index] ? 0 : width),
    0,
  );

  if (raisedTotal === 0 || otherTotal === 0) {
    return widths;
  }

  const scale = (100 - raisedTotal) / otherTotal;

  return widths.map((width, index) =>
    isRaised[index] ? MIN_SEGMENT_WIDTH_PERCENT : width * scale,
  );
};

// The text under the bar, so the lost stretches are said in words and not by color alone.
export const getLostStretchTexts = (
  segments: RestoreTimelineSegment[],
  now: Date,
): MonitoringText[] => {
  const lostSegments = segments.filter(
    (segment) => segment.kind === RestoreTimelineSegmentKind.LOST,
  );

  const texts: MonitoringText[] = lostSegments.slice(0, MAX_LOST_SENTENCES).map((segment) => ({
    key: 'backups.physical.monitoring.restoreTimeline.lostSentence',
    params: getRangeParams(segment, now),
  }));

  if (lostSegments.length > MAX_LOST_SENTENCES) {
    texts.push({
      key: 'backups.physical.monitoring.restoreTimeline.moreLost',
      params: { count: { kind: 'count', count: lostSegments.length - MAX_LOST_SENTENCES } },
    });
  }

  return texts;
};

export const getSegmentRangeText = (
  segment: RestoreTimelineSegment,
  now: Date,
): MonitoringText => ({
  key: 'backups.physical.monitoring.restoreTimeline.range',
  params: getRangeParams(segment, now),
});

export const getMomentParam = (
  atMs: number,
  now: Date,
): Extract<MonitoringTextParam, { kind: 'moment' }> => ({
  kind: 'moment',
  at: new Date(atMs).toISOString(),
  isToday: dayjs(atMs).isSame(now, 'day'),
});

const getRangeParams = (
  segment: RestoreTimelineSegment,
  now: Date,
): Record<string, MonitoringTextParam> => ({
  from: getMomentParam(segment.fromMs, now),
  to: getMomentParam(segment.toMs, now),
});
