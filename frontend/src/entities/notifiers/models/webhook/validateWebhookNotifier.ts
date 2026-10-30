import type { WebhookNotifierSettings } from './WebhookNotifierSettings';

export const validateWebhookNotifier = (settings: WebhookNotifierSettings): boolean =>
  !!settings.webhookUrl;
