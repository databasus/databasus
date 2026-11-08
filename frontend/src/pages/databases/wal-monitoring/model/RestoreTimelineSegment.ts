// One stretch of the restore timeline, with its place on the bar in percent of the bar's width.
export interface RestoreTimelineSegment {
  kind: RestoreTimelineSegmentKind;
  fromMs: number;
  toMs: number;
  startPercent: number;
  widthPercent: number;
}

export enum RestoreTimelineSegmentKind {
  RESTORABLE = 'RESTORABLE',
  LOST = 'LOST',
  NOT_SAVED_YET = 'NOT_SAVED_YET',
}
