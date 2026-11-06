import { MattermostDeliveryMode } from '@/entities/notifiers/model/mattermost/MattermostDeliveryMode';
import type { MattermostNotifierSettings } from '@/entities/notifiers/model/mattermost/MattermostNotifierSettings';

const MATTERMOST_CHANNEL_ID_LENGTH = 26;

export const validateMattermostNotifier = (
  isCreate: boolean,
  settings: MattermostNotifierSettings,
  isAddressChanged = false,
): boolean => {
  if (settings.deliveryMode === MattermostDeliveryMode.WEBHOOK) {
    return !isCreate || !!settings.webhookUrl;
  }

  if (!settings.serverUrl) {
    return false;
  }

  if ((isCreate || isAddressChanged) && !settings.botToken) {
    return false;
  }

  return settings.targetChannelId?.length === MATTERMOST_CHANNEL_ID_LENGTH;
};
