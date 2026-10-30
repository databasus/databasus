import type { TranslationKey } from '../../../../shared/i18n';
import { WalStreamReasonCode } from './WalStreamReasonCode';

export const WAL_STREAM_REASON_CODE_LABEL_KEYS: Record<WalStreamReasonCode, TranslationKey> = {
  [WalStreamReasonCode.STREAMING_STOPPED]:
    'backups.physical.monitoring.reasons.streamingStopped.title',
  [WalStreamReasonCode.SOURCE_UNREACHABLE]:
    'backups.physical.monitoring.reasons.sourceUnreachable.title',
  [WalStreamReasonCode.WAL_CHAIN_BROKEN]:
    'backups.physical.monitoring.reasons.walChainBroken.title',
  [WalStreamReasonCode.SLOT_MISSING]: 'backups.physical.monitoring.reasons.slotMissing.title',
  [WalStreamReasonCode.SLOT_LOST]: 'backups.physical.monitoring.reasons.slotLost.title',
  [WalStreamReasonCode.ARCHIVE_NOT_ADVANCING]:
    'backups.physical.monitoring.reasons.archiveNotAdvancing.title',
  [WalStreamReasonCode.ARCHIVE_SLOWER_THAN_SOURCE]:
    'backups.physical.monitoring.reasons.archiveSlowerThanSource.title',
  [WalStreamReasonCode.SLOT_KEEPS_EXTRA_WAL]:
    'backups.physical.monitoring.reasons.slotKeepsExtraWal.title',
  [WalStreamReasonCode.SLOT_LIMIT_APPROACHING]:
    'backups.physical.monitoring.reasons.slotLimitApproaching.title',
  [WalStreamReasonCode.NEAR_SLOT_REBUILD]:
    'backups.physical.monitoring.reasons.nearSlotRebuild.title',
  [WalStreamReasonCode.LOCAL_QUEUE_FILLING]:
    'backups.physical.monitoring.reasons.localQueueFilling.title',
  [WalStreamReasonCode.FREQUENT_RESTARTS]:
    'backups.physical.monitoring.reasons.frequentRestarts.title',
  [WalStreamReasonCode.FORCED_ROTATION_REFUSED]:
    'backups.physical.monitoring.reasons.forcedRotationRefused.title',
};
