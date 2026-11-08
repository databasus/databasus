import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type WalStreamStreamerState = ApiComponents['schemas']['WalStreamStreamerState'];

export const WalStreamStreamerState = {
  RUNNING: 'RUNNING',
  NOT_RUNNING: 'NOT_RUNNING',
  NOT_STARTED: 'NOT_STARTED',
} as const satisfies Record<string, WalStreamStreamerState>;

export const WAL_STREAM_STREAMER_STATE_LABEL_KEYS: Record<WalStreamStreamerState, TranslationKey> =
  {
    [WalStreamStreamerState.RUNNING]: 'backups.physical.monitoring.details.streamerStates.running',
    [WalStreamStreamerState.NOT_RUNNING]:
      'backups.physical.monitoring.details.streamerStates.notRunning',
    [WalStreamStreamerState.NOT_STARTED]:
      'backups.physical.monitoring.details.streamerStates.notStarted',
  };
