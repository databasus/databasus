import type { TranslationKey } from '../../../../shared/i18n';
import { MattermostDeliveryMode } from './MattermostDeliveryMode';

export const MATTERMOST_DELIVERY_MODE_LABEL_KEYS: Record<MattermostDeliveryMode, TranslationKey> = {
  [MattermostDeliveryMode.WEBHOOK]: 'notifiers.mattermostDeliveryModes.webhook',
  [MattermostDeliveryMode.BOT]: 'notifiers.mattermostDeliveryModes.bot',
};
