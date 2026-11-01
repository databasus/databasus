// One chart point. A null value is a gap the charts draw as a gap, never as zero.
export interface WalStreamChartRow {
  timestampMs: number;

  sourceWriteBytesPerSecond: number | null;
  archiveBytesPerSecond: number | null;

  maxUnarchivedWalBytes: number | null;
  maxSlotLagBytes: number | null;

  maxLocalQueueBytes: number | null;
  localQueuePauseBytes: number | null;

  // Where a restart or pause marker sits on the buffer chart, null without one.
  restartMarkerBytes: number | null;
  pauseMarkerBytes: number | null;
}
