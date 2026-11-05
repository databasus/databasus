import type { RestoreTimelineSegmentKind } from './RestoreTimelineSegmentKind';

// One stretch of the restore timeline, with its place on the bar in percent of the bar's width.
export interface RestoreTimelineSegment {
  kind: RestoreTimelineSegmentKind;
  fromMs: number;
  toMs: number;
  startPercent: number;
  widthPercent: number;
}
