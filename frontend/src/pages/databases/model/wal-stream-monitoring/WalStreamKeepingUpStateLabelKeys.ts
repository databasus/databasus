import type { TranslationKey } from '../../../../shared/i18n';
import { WalStreamKeepingUpState } from './WalStreamKeepingUpState';

export const WAL_STREAM_KEEPING_UP_STATE_LABEL_KEYS: Record<
  WalStreamKeepingUpState,
  TranslationKey
> = {
  [WalStreamKeepingUpState.UNKNOWN]: 'backups.physical.monitoring.keepingUpStates.unknown',
  [WalStreamKeepingUpState.FALLING_BEHIND]:
    'backups.physical.monitoring.keepingUpStates.fallingBehind',
  [WalStreamKeepingUpState.CATCHING_UP]: 'backups.physical.monitoring.keepingUpStates.catchingUp',
  [WalStreamKeepingUpState.IDLE]: 'backups.physical.monitoring.keepingUpStates.idle',
  [WalStreamKeepingUpState.KEEPING_UP]: 'backups.physical.monitoring.keepingUpStates.keepingUp',
};
