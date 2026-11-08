import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type WalStreamReasonLevel = ApiComponents['schemas']['WalStreamReasonLevel'];

export const WalStreamReasonLevel = {
  BROKEN: 'BROKEN',
  NEEDS_ATTENTION: 'NEEDS_ATTENTION',
  INFO: 'INFO',
} as const satisfies Record<string, WalStreamReasonLevel>;

export const WAL_STREAM_REASON_LEVEL_LABEL_KEYS: Record<WalStreamReasonLevel, TranslationKey> = {
  [WalStreamReasonLevel.BROKEN]: 'backups.physical.monitoring.reasonLevels.broken',
  [WalStreamReasonLevel.NEEDS_ATTENTION]: 'backups.physical.monitoring.reasonLevels.needsAttention',
  [WalStreamReasonLevel.INFO]: 'backups.physical.monitoring.reasonLevels.info',
};
