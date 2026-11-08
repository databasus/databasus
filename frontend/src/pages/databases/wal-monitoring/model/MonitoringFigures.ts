// The flow right now. A null value is unknown and reads as "not measured", never as zero.
export interface MonitoringFigures {
  lastSavedAt: string | null;
  waitingToBeSavedBytes: number | null;
  writeBytesPerSecond: number | null;
  saveBytesPerSecond: number | null;
}
