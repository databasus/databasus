import type { SlackNotifierSettings } from './SlackNotifierSettings';

export const validateSlackNotifier = (
  isCreate: boolean,
  settings: SlackNotifierSettings,
): boolean => {
  if (isCreate && !settings.botToken) {
    return false;
  }

  return !!settings.targetChatId;
};
