import type { TranslationKey } from '../../../../shared/i18n';
import { WalStreamReasonLevel } from './WalStreamReasonLevel';

export const WAL_STREAM_REASON_LEVEL_LABEL_KEYS: Record<WalStreamReasonLevel, TranslationKey> = {
  [WalStreamReasonLevel.BROKEN]: 'backups.physical.monitoring.reasonLevels.broken',
  [WalStreamReasonLevel.NEEDS_ATTENTION]: 'backups.physical.monitoring.reasonLevels.needsAttention',
  [WalStreamReasonLevel.INFO]: 'backups.physical.monitoring.reasonLevels.info',
};
