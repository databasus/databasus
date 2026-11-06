import type { AmountProgress } from '@/pages/databases/model/wal-stream-monitoring/AmountProgress';
import { MonitoringTone } from '@/pages/databases/model/wal-stream-monitoring/MonitoringTone';
import { PathStage } from '@/pages/databases/model/wal-stream-monitoring/PathStage';
import { WalStreamHealthLevel } from '@/pages/databases/model/wal-stream-monitoring/WalStreamHealthLevel';
import { type WalStreamReason } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReason';
import { WalStreamReasonCode } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReasonCode';
import { WalStreamReasonLevel } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReasonLevel';

// A broken chain is a hole in the backup history, not a problem of any stage the changes pass
// through, so it colors none.
const REASON_STAGES: Record<WalStreamReasonCode, PathStage | null> = {
  [WalStreamReasonCode.STREAMING_STOPPED]: PathStage.DATABASUS,
  [WalStreamReasonCode.SOURCE_UNREACHABLE]: PathStage.SOURCE,
  [WalStreamReasonCode.WAL_CHAIN_BROKEN]: null,
  [WalStreamReasonCode.SLOT_MISSING]: PathStage.SOURCE,
  [WalStreamReasonCode.SLOT_LOST]: PathStage.SOURCE,
  [WalStreamReasonCode.ARCHIVE_NOT_ADVANCING]: PathStage.STORAGE,
  [WalStreamReasonCode.ARCHIVE_SLOWER_THAN_SOURCE]: PathStage.STORAGE,
  [WalStreamReasonCode.SLOT_KEEPS_EXTRA_WAL]: PathStage.SOURCE,
  [WalStreamReasonCode.SLOT_LIMIT_APPROACHING]: PathStage.SOURCE,
  [WalStreamReasonCode.NEAR_SLOT_REBUILD]: PathStage.SOURCE,
  [WalStreamReasonCode.LOCAL_QUEUE_FILLING]: PathStage.DATABASUS,
  [WalStreamReasonCode.FREQUENT_RESTARTS]: PathStage.DATABASUS,
  [WalStreamReasonCode.FORCED_ROTATION_REFUSED]: PathStage.SOURCE,
};

// Each stage takes the color of the most severe reason about it, and stays neutral without one:
// a healthy color on every stage reads as a selection or an alarm. Notices never color a stage,
// and without a verdict about streaming there is nothing to color.
export const getStageTones = (
  level: WalStreamHealthLevel,
  reasons: WalStreamReason[],
): Record<PathStage, MonitoringTone> => {
  const isVerdictAboutStreaming =
    level !== WalStreamHealthLevel.NOT_STREAMING && level !== WalStreamHealthLevel.COLLECTING_DATA;
  const stageTones: Record<PathStage, MonitoringTone> = {
    [PathStage.SOURCE]: MonitoringTone.NEUTRAL,
    [PathStage.DATABASUS]: MonitoringTone.NEUTRAL,
    [PathStage.STORAGE]: MonitoringTone.NEUTRAL,
  };

  if (!isVerdictAboutStreaming) {
    return stageTones;
  }

  for (const reason of reasons) {
    const stage = REASON_STAGES[reason.code];

    if (stage === null) {
      continue;
    }

    if (reason.level === WalStreamReasonLevel.BROKEN) {
      stageTones[stage] = MonitoringTone.ERROR;
    } else if (
      reason.level === WalStreamReasonLevel.NEEDS_ATTENTION &&
      stageTones[stage] !== MonitoringTone.ERROR
    ) {
      stageTones[stage] = MonitoringTone.WARNING;
    }
  }

  return stageTones;
};

// An amount above its limit is shown as over the limit rather than as a bar that merely looks
// full, so the reader sees that the limit was crossed.
export const getAmountProgress = (valueBytes: number, limitBytes: number): AmountProgress => {
  if (limitBytes <= 0) {
    return { percent: 0, isOverLimit: false };
  }

  if (valueBytes > limitBytes) {
    return { percent: 100, isOverLimit: true };
  }

  return { percent: Math.max(0, (valueBytes / limitBytes) * 100), isOverLimit: false };
};
