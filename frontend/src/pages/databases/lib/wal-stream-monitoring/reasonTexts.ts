import {
  getRoundedDuration,
  getSecondsBetween,
} from '@/pages/databases/lib/wal-stream-monitoring/roundedDuration';
import type { MonitoringText } from '@/pages/databases/model/wal-stream-monitoring/MonitoringText';
import type { MonitoringTextParam } from '@/pages/databases/model/wal-stream-monitoring/MonitoringTextParam';
import type { ReasonTexts } from '@/pages/databases/model/wal-stream-monitoring/ReasonTexts';
import { type WalStreamNotice } from '@/pages/databases/model/wal-stream-monitoring/WalStreamNotice';
import { WalStreamNoticeCode } from '@/pages/databases/model/wal-stream-monitoring/WalStreamNoticeCode';
import { WAL_STREAM_NOTICE_CODE_LABEL_KEYS } from '@/pages/databases/model/wal-stream-monitoring/WalStreamNoticeCodeLabelKeys';
import { type WalStreamReason } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReason';
import { WalStreamReasonCode } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReasonCode';
import { WAL_STREAM_REASON_CODE_LABEL_KEYS } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReasonCodeLabelKeys';
import { WalStreamReasonLevel } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReasonLevel';
import type { TranslationKey } from '@/shared/i18n';

interface ReasonSentenceKeys {
  whatHappened: TranslationKey;
  whatToDo: TranslationKey;

  // Asked instead of whatToDo while a full backup is requested or running, for the conditions
  // only a new full backup repairs.
  whatToDoWhilePending?: TranslationKey;
}

const REASON_SENTENCE_KEYS: Record<WalStreamReasonCode, ReasonSentenceKeys> = {
  [WalStreamReasonCode.STREAMING_STOPPED]: {
    whatHappened: 'backups.physical.monitoring.reasons.streamingStopped.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.streamingStopped.whatToDo',
  },
  [WalStreamReasonCode.SOURCE_UNREACHABLE]: {
    whatHappened: 'backups.physical.monitoring.reasons.sourceUnreachable.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.sourceUnreachable.whatToDo',
  },
  [WalStreamReasonCode.WAL_CHAIN_BROKEN]: {
    whatHappened: 'backups.physical.monitoring.reasons.walChainBroken.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.walChainBroken.whatToDo',
    whatToDoWhilePending: 'backups.physical.monitoring.reasons.walChainBroken.whatToDoPending',
  },
  [WalStreamReasonCode.SLOT_MISSING]: {
    whatHappened: 'backups.physical.monitoring.reasons.slotMissing.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.slotMissing.whatToDo',
    whatToDoWhilePending: 'backups.physical.monitoring.reasons.slotMissing.whatToDoPending',
  },
  [WalStreamReasonCode.SLOT_LOST]: {
    whatHappened: 'backups.physical.monitoring.reasons.slotLost.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.slotLost.whatToDo',
  },
  [WalStreamReasonCode.ARCHIVE_NOT_ADVANCING]: {
    whatHappened: 'backups.physical.monitoring.reasons.archiveNotAdvancing.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.archiveNotAdvancing.whatToDo',
  },
  [WalStreamReasonCode.ARCHIVE_SLOWER_THAN_SOURCE]: {
    whatHappened: 'backups.physical.monitoring.reasons.archiveSlowerThanSource.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.archiveSlowerThanSource.whatToDo',
  },
  [WalStreamReasonCode.SLOT_KEEPS_EXTRA_WAL]: {
    whatHappened: 'backups.physical.monitoring.reasons.slotKeepsExtraWal.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.slotKeepsExtraWal.whatToDo',
  },
  [WalStreamReasonCode.SLOT_LIMIT_APPROACHING]: {
    whatHappened: 'backups.physical.monitoring.reasons.slotLimitApproaching.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.slotLimitApproaching.whatToDo',
  },
  [WalStreamReasonCode.NEAR_SLOT_REBUILD]: {
    whatHappened: 'backups.physical.monitoring.reasons.nearSlotRebuild.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.nearSlotRebuild.whatToDo',
  },
  [WalStreamReasonCode.LOCAL_QUEUE_FILLING]: {
    whatHappened: 'backups.physical.monitoring.reasons.localQueueFilling.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.localQueueFilling.whatToDo',
  },
  [WalStreamReasonCode.FREQUENT_RESTARTS]: {
    whatHappened: 'backups.physical.monitoring.reasons.frequentRestarts.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.frequentRestarts.whatToDo',
  },
  [WalStreamReasonCode.FORCED_ROTATION_REFUSED]: {
    whatHappened: 'backups.physical.monitoring.reasons.forcedRotationRefused.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.forcedRotationRefused.whatToDo',
  },
};

const NOTICE_SENTENCE_KEYS: Record<WalStreamNoticeCode, ReasonSentenceKeys> = {
  [WalStreamNoticeCode.NO_SLOT_WAL_LIMIT]: {
    whatHappened: 'backups.physical.monitoring.notices.noSlotWalLimit.whatHappened',
    whatToDo: 'backups.physical.monitoring.notices.noSlotWalLimit.whatToDo',
  },
  [WalStreamNoticeCode.SLOT_KEEPS_EXTRA_WAL_STEADILY]: {
    whatHappened: 'backups.physical.monitoring.notices.slotKeepsExtraWalSteadily.whatHappened',
    whatToDo: 'backups.physical.monitoring.notices.slotKeepsExtraWalSteadily.whatToDo',
  },
  [WalStreamNoticeCode.NO_FULL_BACKUP_YET]: {
    whatHappened: 'backups.physical.monitoring.notices.noFullBackupYet.whatHappened',
    whatToDo: 'backups.physical.monitoring.notices.noFullBackupYet.whatToDo',
  },
};

const REASON_LEVEL_ORDER: Record<WalStreamReasonLevel, number> = {
  [WalStreamReasonLevel.BROKEN]: 0,
  [WalStreamReasonLevel.NEEDS_ATTENTION]: 1,
  [WalStreamReasonLevel.INFO]: 2,
};

// Broken reasons first; within a level the backend's order, which follows the documented list.
export const sortReasonsBrokenFirst = (reasons: WalStreamReason[]): WalStreamReason[] =>
  [...reasons].sort(
    (first, second) => REASON_LEVEL_ORDER[first.level] - REASON_LEVEL_ORDER[second.level],
  );

export const getReasonTexts = (
  reason: WalStreamReason,
  isFullBackupPending: boolean,
  now: Date,
): ReasonTexts => {
  const sentenceKeys = REASON_SENTENCE_KEYS[reason.code];
  const whatToDoKey =
    isFullBackupPending && sentenceKeys.whatToDoWhilePending
      ? sentenceKeys.whatToDoWhilePending
      : sentenceKeys.whatToDo;

  return {
    level: reason.level,
    titleKey: WAL_STREAM_REASON_CODE_LABEL_KEYS[reason.code],
    whatHappened: { key: sentenceKeys.whatHappened, params: getReasonParams(reason, now) },
    whatToDo: { key: whatToDoKey },
  };
};

export const getNoticeTexts = (notice: WalStreamNotice): ReasonTexts => {
  const sentenceKeys = NOTICE_SENTENCE_KEYS[notice.code];

  return {
    level: notice.level,
    titleKey: WAL_STREAM_NOTICE_CODE_LABEL_KEYS[notice.code],
    whatHappened: { key: sentenceKeys.whatHappened },
    whatToDo: { key: sentenceKeys.whatToDo },
  };
};

// The banner's restore sentence already says that no full backup has finished, so that notice is no
// tip.
export const getTipTexts = (notices: WalStreamNotice[]): ReasonTexts[] =>
  notices
    .filter((notice) => notice.code !== WalStreamNoticeCode.NO_FULL_BACKUP_YET)
    .map(getNoticeTexts);

const getReasonParams = (reason: WalStreamReason, now: Date): MonitoringText['params'] => {
  const params: Record<string, MonitoringTextParam> = {};

  if (reason.since !== undefined) {
    params.duration = {
      kind: 'text',
      text: getRoundedDuration(getSecondsBetween(reason.since, now) ?? 0),
    };
  }

  if (reason.estimatedSeconds !== undefined) {
    params.duration = { kind: 'text', text: getRoundedDuration(reason.estimatedSeconds) };
  }

  if (reason.valueBytes !== undefined) {
    params.amount = { kind: 'bytes', bytes: reason.valueBytes };
  }

  if (reason.thresholdBytes !== undefined) {
    params.limit = { kind: 'bytes', bytes: reason.thresholdBytes };
  }

  if (reason.count !== undefined) {
    params.count = { kind: 'count', count: reason.count };
  }

  return params;
};
