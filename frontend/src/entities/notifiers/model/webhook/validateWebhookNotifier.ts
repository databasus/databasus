import { isWebhookHeaderNameValid } from './isWebhookHeaderNameValid';
import type { WebhookNotifierSettings } from './WebhookNotifierSettings';

export const validateWebhookNotifier = (settings: WebhookNotifierSettings): boolean =>
  !!settings.webhookUrl &&
  settings.headers.every((header) => isWebhookHeaderNameValid(header.name));
