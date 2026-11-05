import type { TranslationKey } from '../../../../shared/i18n';
import { WalStreamHealthLevel } from './WalStreamHealthLevel';

export const WAL_STREAM_HEALTH_LEVEL_LABEL_KEYS: Record<WalStreamHealthLevel, TranslationKey> = {
  [WalStreamHealthLevel.NOT_STREAMING]: 'backups.physical.monitoring.levels.notStreaming',
  [WalStreamHealthLevel.COLLECTING_DATA]: 'backups.physical.monitoring.levels.collectingData',
  [WalStreamHealthLevel.BROKEN]: 'backups.physical.monitoring.levels.broken',
  [WalStreamHealthLevel.NEEDS_ATTENTION]: 'backups.physical.monitoring.levels.needsAttention',
  [WalStreamHealthLevel.HEALTHY]: 'backups.physical.monitoring.levels.healthy',
};
