import type { WalStreamLive } from '@/pages/databases/model/wal-stream-monitoring/WalStreamLive';
import type { WalStreamStatistics } from '@/pages/databases/model/wal-stream-monitoring/WalStreamStatistics';

const isNewer = (time: string | null, than: string): time is string =>
  time !== null && new Date(time).getTime() > new Date(than).getTime();

const isSameMoment = (a: string | null, b: string | null) =>
  a !== null && b !== null && new Date(a).getTime() === new Date(b).getTime();

// A restore that reached the newest saved change in the sample keeps reaching it as segments land,
// so the restore point follows the live save time instead of lagging the figures by up to one
// statistics refresh. A restore stopped short of it, at a gap, stays where the statistics put it.
const followRestorePoint = (
  statistics: WalStreamStatistics,
  liveLastArchivedWalAt: string | null,
): Pick<WalStreamStatistics, 'latestRestorableAt' | 'restorableRanges'> => {
  const { latestRestorableAt, restorableRanges, current } = statistics;

  if (
    latestRestorableAt === null ||
    !isSameMoment(latestRestorableAt, current?.lastArchivedWalAt ?? null) ||
    !isNewer(liveLastArchivedWalAt, latestRestorableAt)
  ) {
    return { latestRestorableAt, restorableRanges };
  }

  const lastIndex = restorableRanges.length - 1;

  return {
    latestRestorableAt: liveLastArchivedWalAt,
    restorableRanges: restorableRanges.map((range, index) =>
      index === lastIndex && isSameMoment(range.to, latestRestorableAt)
        ? { ...range, to: liveLastArchivedWalAt }
        : range,
    ),
  };
};

// Lays the running streamer's newer values over the newest sample, so the figures, the path of
// changes and the measured label move every second while the verdict, the rates and the charts
// keep the 30-second statistics. Source-side values replace the sample's only when their probe is
// newer than the sample: an older probe's position against a newer progress point would understate
// what is not saved yet.
export const applyLiveValues = (
  statistics: WalStreamStatistics,
  live: WalStreamLive | null,
): WalStreamStatistics => {
  const { current } = statistics;

  if (live === null || current === null || !isNewer(live.measuredAt, current.sampledAt)) {
    return statistics;
  }

  const overlaidCurrent = {
    ...current,
    sampledAt: live.measuredAt,
    localQueueBytes: live.localQueueBytes,
    localQueuePauseBytes: live.localQueuePauseBytes,
    isReceiverRunning: live.isReceiverRunning,
    archiveProgressLsn: live.archiveProgressLsn ?? current.archiveProgressLsn,
    lastArchivedWalAt: live.lastArchivedWalAt ?? current.lastArchivedWalAt,
  };
  let overlaidKeepingUp = statistics.keepingUp;

  if (isNewer(live.sourceProbedAt, current.sampledAt) && live.sourceState !== null) {
    overlaidCurrent.sourceState = live.sourceState;
    overlaidCurrent.sourceCurrentLsn = live.sourceCurrentLsn;
    overlaidCurrent.slotWalStatus = live.slotWalStatus;
    overlaidCurrent.slotLagBytes = live.slotLagBytes;
    overlaidCurrent.slotSafeWalSizeBytes = live.slotSafeWalSizeBytes;
    overlaidCurrent.unarchivedWalBytes = live.unarchivedWalBytes;

    overlaidKeepingUp = {
      ...statistics.keepingUp,
      unarchivedWalBytes: live.unarchivedWalBytes,
      slotLagBytes: live.slotLagBytes,
    };
  }

  return {
    ...statistics,
    ...followRestorePoint(statistics, live.lastArchivedWalAt),
    current: overlaidCurrent,
    keepingUp: overlaidKeepingUp,
  };
};
