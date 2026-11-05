import type { DiscordNotifierSettings } from './DiscordNotifierSettings';

// A saved notifier keeps its stored webhook URL when the form leaves it empty.
export const validateDiscordNotifier = (
  isCreate: boolean,
  settings: DiscordNotifierSettings,
): boolean => !isCreate || !!settings.channelWebhookUrl;
