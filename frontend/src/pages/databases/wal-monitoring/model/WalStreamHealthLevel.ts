import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type WalStreamHealthLevel = ApiComponents['schemas']['WalStreamHealthLevel'];

export const WalStreamHealthLevel = {
  NOT_STREAMING: 'NOT_STREAMING',
  COLLECTING_DATA: 'COLLECTING_DATA',
  BROKEN: 'BROKEN',
  NEEDS_ATTENTION: 'NEEDS_ATTENTION',
  HEALTHY: 'HEALTHY',
} as const satisfies Record<string, WalStreamHealthLevel>;

export const WAL_STREAM_HEALTH_LEVEL_LABEL_KEYS: Record<WalStreamHealthLevel, TranslationKey> = {
  [WalStreamHealthLevel.NOT_STREAMING]: 'backups.physical.monitoring.levels.notStreaming',
  [WalStreamHealthLevel.COLLECTING_DATA]: 'backups.physical.monitoring.levels.collectingData',
  [WalStreamHealthLevel.BROKEN]: 'backups.physical.monitoring.levels.broken',
  [WalStreamHealthLevel.NEEDS_ATTENTION]: 'backups.physical.monitoring.levels.needsAttention',
  [WalStreamHealthLevel.HEALTHY]: 'backups.physical.monitoring.levels.healthy',
};
