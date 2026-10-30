import type { TranslationKey } from '../../../../shared/i18n';
import { WalStreamStreamerState } from './WalStreamStreamerState';

export const WAL_STREAM_STREAMER_STATE_LABEL_KEYS: Record<WalStreamStreamerState, TranslationKey> =
  {
    [WalStreamStreamerState.RUNNING]: 'backups.physical.monitoring.details.streamerStates.running',
    [WalStreamStreamerState.NOT_RUNNING]:
      'backups.physical.monitoring.details.streamerStates.notRunning',
    [WalStreamStreamerState.NOT_STARTED]:
      'backups.physical.monitoring.details.streamerStates.notStarted',
  };
