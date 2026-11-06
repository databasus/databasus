import type { SlackNotifierSettings } from '@/entities/notifiers/model/slack/SlackNotifierSettings';

export const validateSlackNotifier = (
  isCreate: boolean,
  settings: SlackNotifierSettings,
): boolean => {
  if (isCreate && !settings.botToken) {
    return false;
  }

  return !!settings.targetChatId;
};
