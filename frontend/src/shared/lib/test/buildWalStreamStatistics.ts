/* eslint-disable i18next/no-literal-string -- a test fixture of UTC times and WAL positions, not copy */
import type { ApiComponents } from '../../api';
type WalStreamStatistics = ApiComponents['schemas']['WalStreamStatistics'];

const MiB = 1024 * 1024;

// Statistics of a healthy database at 12:00 UTC on 2026-10-18, for the slice's tests to adjust
// field by field.
export const createTestWalStreamStatistics = (
  overrides: Partial<WalStreamStatistics> = {},
): WalStreamStatistics => ({
  health: {
    level: 'HEALTHY',
    reasons: [],
    notices: [],
    isFullBackupPending: false,
  },
  latestRestorableAt: '2026-10-18T11:59:20Z',
  restorableRanges: [{ from: '2026-10-17T03:00:00Z', to: '2026-10-18T11:59:20Z' }],
  streamer: {
    state: 'RUNNING',
    sessionStartedAt: '2026-10-18T10:00:00Z',
    failingSince: null,
    restartsLastHour: 0,
  },
  current: {
    sampledAt: '2026-10-18T11:59:48Z',
    lastArchivedWalAt: '2026-10-18T11:59:20Z',
    sessionStartedAt: '2026-10-18T10:00:00Z',
    sourceState: 'REACHABLE',
    sourceUnreachableSince: null,
    sourceCurrentLsn: '0/5000000',
    archiveProgressLsn: '0/4F00000',
    sourceWrittenBytes: 16 * MiB,
    archivedSegments: 2,
    archivedWalBytes: 32 * MiB,
    archivedStoredBytes: 8 * MiB,
    walSegmentSizeBytes: 16 * MiB,
    highestConcurrentUploads: 3,
    segmentWriteAvgMillis: 120,
    segmentWriteMaxMillis: 300,
    slotWalStatus: 'reserved',
    slotExtraWalSince: null,
    slotLagBytes: 2 * MiB,
    slotSafeWalSizeBytes: null,
    unarchivedWalBytes: 20 * MiB,
    archiveStalledSince: null,
    localQueueBytes: 0,
    localQueuePauseBytes: 512 * MiB,
    receiverRestarts: 0,
    receiverPauses: 0,
    isReceiverRunning: true,
    isForcedRotationDenied: false,
  },
  keepingUp: {
    state: 'KEEPING_UP',
    sourceWriteBytesPerSecond: 10 * MiB,
    archiveBytesPerSecond: 10 * MiB,
    unarchivedWalBytes: 20 * MiB,
    unarchivedWalChangeBytesPerSecond: 0,
    slotLagBytes: 2 * MiB,
    slotLagChangeBytesPerSecond: 0,
    slotRebuildLagBytes: 256 * MiB,
    estimatedCatchUpSeconds: null,
    estimatedSlotRebuildSeconds: null,
    estimatedSlotLimitSeconds: null,
  },
  series: {
    period: 'HOUR',
    bucketSeconds: 60,
    from: '2026-10-18T11:01:00Z',
    to: '2026-10-18T12:01:00Z',
    points: [],
  },
  ...overrides,
});
