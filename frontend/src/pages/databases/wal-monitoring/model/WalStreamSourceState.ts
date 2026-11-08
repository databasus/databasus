import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type WalStreamSourceState = ApiComponents['schemas']['WalStreamSourceState'];

export const WalStreamSourceState = {
  REACHABLE: 'REACHABLE',
  UNREACHABLE: 'UNREACHABLE',
  SLOT_MISSING: 'SLOT_MISSING',
} as const satisfies Record<string, WalStreamSourceState>;

export const WAL_STREAM_SOURCE_STATE_LABEL_KEYS: Record<WalStreamSourceState, TranslationKey> = {
  [WalStreamSourceState.REACHABLE]: 'backups.physical.monitoring.details.sourceStates.reachable',
  [WalStreamSourceState.UNREACHABLE]:
    'backups.physical.monitoring.details.sourceStates.unreachable',
  [WalStreamSourceState.SLOT_MISSING]:
    'backups.physical.monitoring.details.sourceStates.slotMissing',
};
