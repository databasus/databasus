import type { MonitoringFigures } from '@/pages/databases/model/wal-stream-monitoring/MonitoringFigures';
import { WalStreamHealthLevel } from '@/pages/databases/model/wal-stream-monitoring/WalStreamHealthLevel';
import { type WalStreamStatistics } from '@/pages/databases/model/wal-stream-monitoring/WalStreamStatistics';

// The amounts come from the keeping-up summary rather than the newest sample, so the figures agree
// with the keeping-up sentence written under them.
export const getMonitoringFigures = (statistics: WalStreamStatistics): MonitoringFigures | null => {
  const { level } = statistics.health;

  if (
    level === WalStreamHealthLevel.COLLECTING_DATA ||
    level === WalStreamHealthLevel.NOT_STREAMING
  ) {
    return null;
  }

  return {
    lastSavedAt: statistics.current?.lastArchivedWalAt ?? null,
    waitingToBeSavedBytes: statistics.keepingUp.unarchivedWalBytes,
    writeBytesPerSecond: statistics.keepingUp.sourceWriteBytesPerSecond,
    saveBytesPerSecond: statistics.keepingUp.archiveBytesPerSecond,
  };
};
