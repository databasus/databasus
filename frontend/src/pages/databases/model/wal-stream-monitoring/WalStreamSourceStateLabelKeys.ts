import type { TranslationKey } from '../../../../shared/i18n';
import { WalStreamSourceState } from './WalStreamSourceState';

export const WAL_STREAM_SOURCE_STATE_LABEL_KEYS: Record<WalStreamSourceState, TranslationKey> = {
  [WalStreamSourceState.REACHABLE]: 'backups.physical.monitoring.details.sourceStates.reachable',
  [WalStreamSourceState.UNREACHABLE]:
    'backups.physical.monitoring.details.sourceStates.unreachable',
  [WalStreamSourceState.SLOT_MISSING]:
    'backups.physical.monitoring.details.sourceStates.slotMissing',
};
