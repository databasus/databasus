import { MattermostDeliveryMode } from './MattermostDeliveryMode';
import type { MattermostNotifierSettings } from './MattermostNotifierSettings';

const MATTERMOST_CHANNEL_ID_LENGTH = 26;

export const validateMattermostNotifier = (
  isCreate: boolean,
  settings: MattermostNotifierSettings,
): boolean => {
  if (settings.deliveryMode === MattermostDeliveryMode.WEBHOOK) {
    return !isCreate || !!settings.webhookUrl;
  }

  if (!settings.serverUrl) {
    return false;
  }

  if (isCreate && !settings.botToken) {
    return false;
  }

  return settings.targetChannelId?.length === MATTERMOST_CHANNEL_ID_LENGTH;
};
