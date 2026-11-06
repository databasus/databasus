import { describe, expect, it, vi } from 'vitest';

import {
  MIN_SEGMENT_WIDTH_PERCENT,
  getLostStretchTexts,
  getRestoreTimeline,
} from '@/pages/databases/lib/wal-stream-monitoring/restoreTimeline';
import { RestoreTimelineSegmentKind } from '@/pages/databases/model/wal-stream-monitoring/RestoreTimelineSegmentKind';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

const now = new Date('2026-10-20T12:00:00Z');
const at = (time: string) => new Date(time).getTime();

const getWidthTotal = (segments: { widthPercent: number }[]) =>
  segments.reduce((total, segment) => total + segment.widthPercent, 0);

describe('getRestoreTimeline', () => {
  it('draws nothing without a restorable range', () => {
    expect(getRestoreTimeline([], now)).toEqual([]);
  });

  it('fills the bar with one range that reaches now', () => {
    expect(
      getRestoreTimeline([{ from: '2026-10-20T08:00:00Z', to: '2026-10-20T12:00:00Z' }], now),
    ).toEqual([
      {
        kind: RestoreTimelineSegmentKind.RESTORABLE,
        fromMs: at('2026-10-20T08:00:00Z'),
        toMs: at('2026-10-20T12:00:00Z'),
        startPercent: 0,
        widthPercent: 100,
      },
    ]);
  });

  it('marks the stretch between two ranges as lost and the tail after the latest moment as not saved yet', () => {
    const segments = getRestoreTimeline(
      [
        { from: '2026-10-20T00:00:00Z', to: '2026-10-20T04:00:00Z' },
        { from: '2026-10-20T06:00:00Z', to: '2026-10-20T10:00:00Z' },
      ],
      now,
    );

    const expected: [RestoreTimelineSegmentKind, number, number][] = [
      [RestoreTimelineSegmentKind.RESTORABLE, 0, 4],
      [RestoreTimelineSegmentKind.LOST, 4, 2],
      [RestoreTimelineSegmentKind.RESTORABLE, 6, 4],
      [RestoreTimelineSegmentKind.NOT_SAVED_YET, 10, 2],
    ];

    expect(segments.map((segment) => segment.kind)).toEqual(expected.map(([kind]) => kind));
    expected.forEach(([, startHour, widthHours], index) => {
      expect(segments[index].startPercent).toBeCloseTo((startHour / 12) * 100);
      expect(segments[index].widthPercent).toBeCloseTo((widthHours / 12) * 100);
    });
    expect(segments[1]).toMatchObject({
      fromMs: at('2026-10-20T04:00:00Z'),
      toMs: at('2026-10-20T06:00:00Z'),
    });
  });

  it('keeps a lost hour visible on a chain three months long, and the bar still adds up to 100%', () => {
    const segments = getRestoreTimeline(
      [
        { from: '2026-07-20T12:00:00Z', to: '2026-09-01T12:00:00Z' },
        { from: '2026-09-01T13:00:00Z', to: '2026-10-20T12:00:00Z' },
      ],
      now,
    );

    expect(segments[1].kind).toBe(RestoreTimelineSegmentKind.LOST);
    expect(segments[1].widthPercent).toBe(MIN_SEGMENT_WIDTH_PERCENT);
    expect(getWidthTotal(segments)).toBeCloseTo(100);
    expect(segments[2].startPercent).toBeCloseTo(
      segments[0].widthPercent + MIN_SEGMENT_WIDTH_PERCENT,
    );
  });

  it('keeps a restorable single moment visible', () => {
    const segments = getRestoreTimeline(
      [
        { from: '2026-10-20T00:00:00Z', to: '2026-10-20T04:00:00Z' },
        { from: '2026-10-20T08:00:00Z', to: '2026-10-20T08:00:00Z' },
      ],
      new Date('2026-10-20T08:00:00Z'),
    );

    expect(segments[segments.length - 1]).toMatchObject({
      kind: RestoreTimelineSegmentKind.RESTORABLE,
      widthPercent: MIN_SEGMENT_WIDTH_PERCENT,
    });
    expect(getWidthTotal(segments)).toBeCloseTo(100);
  });

  it('does not stretch a short not saved yet tail', () => {
    const segments = getRestoreTimeline(
      [{ from: '2026-10-10T12:00:00Z', to: '2026-10-20T11:59:20Z' }],
      now,
    );

    expect(segments[1].kind).toBe(RestoreTimelineSegmentKind.NOT_SAVED_YET);
    expect(segments[1].widthPercent).toBeLessThan(MIN_SEGMENT_WIDTH_PERCENT);
  });
});

describe('getLostStretchTexts', () => {
  const buildRanges = (lostCount: number) =>
    Array.from({ length: lostCount + 1 }, (_, index) => ({
      from: new Date(Date.parse('2026-10-20T00:00:00Z') + index * 2 * 3600_000).toISOString(),
      to: new Date(Date.parse('2026-10-20T01:00:00Z') + index * 2 * 3600_000).toISOString(),
    }));

  it('says nothing without a lost stretch', () => {
    expect(getLostStretchTexts(getRestoreTimeline(buildRanges(0), now), now)).toEqual([]);
  });

  it('names the start and end of a lost stretch, as a time of day for today and with a date before', () => {
    const segments = getRestoreTimeline(
      [
        { from: '2026-10-10T12:00:00Z', to: '2026-10-10T15:35:00Z' },
        { from: '2026-10-20T04:51:00Z', to: '2026-10-20T11:59:00Z' },
      ],
      now,
    );

    expect(getLostStretchTexts(segments, now)).toEqual([
      {
        key: 'backups.physical.monitoring.restoreTimeline.lostSentence',
        params: {
          from: { kind: 'moment', at: '2026-10-10T15:35:00.000Z', isToday: false },
          to: { kind: 'moment', at: '2026-10-20T04:51:00.000Z', isToday: true },
        },
      },
    ]);
  });

  it('lists three lost stretches and counts the rest', () => {
    const texts = getLostStretchTexts(getRestoreTimeline(buildRanges(5), now), now);

    expect(texts.map((text) => text.key)).toEqual([
      'backups.physical.monitoring.restoreTimeline.lostSentence',
      'backups.physical.monitoring.restoreTimeline.lostSentence',
      'backups.physical.monitoring.restoreTimeline.lostSentence',
      'backups.physical.monitoring.restoreTimeline.moreLost',
    ]);
    expect(texts[3].params).toEqual({ count: { kind: 'count', count: 2 } });
  });
});
